package suno

import (
	"chatplus/core/types"
	logger2 "chatplus/logger"
	"chatplus/service"
	"chatplus/store"
	"chatplus/store/model"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

var logger = logger2.GetLogger()

// TaskStatus 定义任务状态结构
type TaskStatus struct {
	Status   string
	Progress int
	AudioURL string
	VideoURL string
	ImageURL string
	Lyrics   string
	Title    string
	Duration float64
	Error    string
}

type sunoTrackItem struct {
	Title        string  `json:"title"`
	Prompt       string  `json:"prompt"`
	Cld2AudioUrl string  `json:"cld2AudioUrl"`
	Cld2ImageUrl string  `json:"cld2ImageUrl"`
	Cld2VideoUrl string  `json:"cld2VideoUrl"`
	Duration     float64 `json:"duration"`
	Progress     int     `json:"progress"`
}

func pickSunoTrackItem(items []sunoTrackItem, preferAudio string) *sunoTrackItem {
	if len(items) == 0 {
		return nil
	}
	if preferAudio != "" {
		for i := range items {
			if items[i].Cld2AudioUrl == preferAudio {
				return &items[i]
			}
		}
	}
	return &items[0]
}

func applySunoTrackItem(status *TaskStatus, item *sunoTrackItem) {
	if item == nil {
		return
	}
	status.AudioURL = item.Cld2AudioUrl
	status.VideoURL = item.Cld2VideoUrl
	status.ImageURL = item.Cld2ImageUrl
	status.Lyrics = item.Prompt
	status.Title = item.Title
	status.Duration = item.Duration
}

// extractSunoTrackItems 兼容上游两种 data 结构：
// 新格式 data.data 为曲目数组；旧格式 data.data.data.items 为曲目数组。
func extractSunoTrackItems(raw json.RawMessage) (items []sunoTrackItem, nestedTaskStatus string) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, ""
	}

	var arr []sunoTrackItem
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, ""
	}

	var legacyWrap struct {
		Data struct {
			Items      []sunoTrackItem `json:"items"`
			TaskStatus string          `json:"taskStatus"`
		} `json:"data"`
		Items      []sunoTrackItem `json:"items"`
		TaskStatus string          `json:"taskStatus"`
	}
	if err := json.Unmarshal(raw, &legacyWrap); err != nil {
		return nil, ""
	}
	if len(legacyWrap.Data.Items) > 0 {
		return legacyWrap.Data.Items, legacyWrap.Data.TaskStatus
	}
	return legacyWrap.Items, legacyWrap.TaskStatus
}

func buildTaskStatusFromFetchBody(respBody []byte, preferAudio string) (*TaskStatus, error) {
	var result struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Status     string          `json:"status"`
			Progress   string          `json:"progress"`
			FailReason string          `json:"fail_reason"`
			Data       json.RawMessage `json:"data"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	if result.Code != "success" {
		return nil, fmt.Errorf("查询失败: %s", result.Message)
	}

	items, nestedTaskStatus := extractSunoTrackItems(result.Data.Data)

	progress := 0
	if result.Data.Progress != "" {
		progress, _ = strconv.Atoi(strings.TrimSuffix(result.Data.Progress, "%"))
	}
	if progress == 0 && len(items) > 0 {
		progress = items[0].Progress
	}

	status := &TaskStatus{
		Progress: progress,
		Error:    result.Data.FailReason,
	}

	switch strings.ToUpper(result.Data.Status) {
	case "SUCCESS":
		status.Status = "Succeeded"
		applySunoTrackItem(status, pickSunoTrackItem(items, preferAudio))
		logger.Debugf("获取到的 URL - Audio: %s, Video: %s, Image: %s, LyricsLen: %d",
			status.AudioURL, status.VideoURL, status.ImageURL, len(status.Lyrics))
	case "FAILED":
		status.Status = "Failed"
	case "PROCESSING", "IN_PROGRESS", "PENDING":
		status.Status = "Processing"
	default:
		if nestedTaskStatus == "finished" && progress >= 100 {
			status.Status = "Succeeded"
			applySunoTrackItem(status, pickSunoTrackItem(items, preferAudio))
		} else {
			status.Status = result.Data.Status
		}
	}

	return status, nil
}

type SunoServicePool struct {
	config      *types.SunoConfig
	taskQueue   *store.RedisQueue
	notifyQueue *store.RedisQueue
	db          *gorm.DB
	Clients     *types.LMap[uint, *types.WsClient]
}

func NewSunoServicePool(db *gorm.DB, redisCli *redis.Client, appConfig *types.AppConfig) *SunoServicePool {
	taskQueue := store.NewRedisQueue("suno_task_queue", redisCli)
	notifyQueue := store.NewRedisQueue("suno_notify_queue", redisCli)

	pool := &SunoServicePool{
		config:      &appConfig.SunoConfig,
		taskQueue:   taskQueue,
		notifyQueue: notifyQueue,
		db:          db,
		Clients:     types.NewLMap[uint, *types.WsClient](),
	}

	return pool
}

// refundFailedJob 失败任务退还算力（幂等：同一任务 power>0 时仅退一次）
func (p *SunoServicePool) refundFailedJob(job *model.MusicJob, errMsg string) {
	if job == nil || job.Id == 0 || job.Power <= 0 {
		return
	}
	if errMsg == "" {
		errMsg = "任务失败"
	}
	refundPower := job.Power

	tx := p.db.Begin()
	res := tx.Model(&model.MusicJob{}).
		Where("id = ? AND power > 0", job.Id).
		Updates(map[string]interface{}{
			"power":    0,
			"progress": 100,
			"status":   "FAILED",
			"err_msg":  errMsg,
		})
	if res.Error != nil || res.RowsAffected == 0 {
		tx.Rollback()
		return
	}

	if err := tx.Model(&model.User{}).Where("id = ?", job.UserId).
		UpdateColumn("power", gorm.Expr("power + ?", refundPower)).Error; err != nil {
		tx.Rollback()
		logger.Errorf("音乐任务退还算力失败 user=%d job=%d: %v", job.UserId, job.Id, err)
		return
	}

	var user model.User
	if err := tx.Where("id = ?", job.UserId).First(&user).Error; err != nil {
		tx.Rollback()
		return
	}

	if err := tx.Create(&model.PowerLog{
		UserId:    user.Id,
		Username:  user.Username,
		Type:      types.PowerRefund,
		Amount:    refundPower,
		Balance:   user.Power,
		Mark:      types.PowerAdd,
		Model:     service.PowerLogModelMusic,
		Remark:    fmt.Sprintf("音乐生成任务失败，退回算力。任务ID：%s", job.TaskId),
		CreatedAt: time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		logger.Errorf("音乐任务退还算力日志失败 user=%d job=%d: %v", job.UserId, job.Id, err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		logger.Errorf("音乐任务退还算力提交失败 job=%d: %v", job.Id, err)
	}
}

// SyncTaskProgress 异步监控任务进度
func (p *SunoServicePool) SyncTaskProgress() {
	go func() {
		var items []model.MusicJob
		for {
			res := p.db.Where("progress < ? AND power > ?", 100, 0).Find(&items)
			if res.Error != nil {
				continue
			}

			for _, job := range items {
				// 已标记失败但未退费的遗留任务
				if job.Progress == -1 || strings.EqualFold(job.Status, "FAILED") {
					p.refundFailedJob(&job, job.ErrMsg)
					continue
				}

				// 超时任务
				if time.Now().Sub(job.CreatedAt) > time.Minute*30 {
					p.refundFailedJob(&job, "任务超时或失败")
					continue
				}

				// 查询任务状态
				status, err := p.queryTaskStatus(job.TaskId, job.AudioUrl)
				if err != nil {
					logger.Errorf("查询任务状态失败: %v", err)
					continue
				}

				if status.Status == "Failed" {
					errMsg := status.Error
					if errMsg == "" {
						errMsg = "任务失败"
					}
					p.refundFailedJob(&job, errMsg)
					continue
				}

				// 更新任务状态
				updates := map[string]interface{}{
					"progress": status.Progress,
					"status":   status.Status,
				}

				if status.Status == "Succeeded" {
					updates["progress"] = 100
					updates["audio_url"] = status.AudioURL
					updates["video_url"] = status.VideoURL
					updates["image_url"] = status.ImageURL
					if status.Lyrics != "" {
						updates["lyrics"] = status.Lyrics
					}
					if status.Duration > 0 {
						updates["duration"] = status.Duration
					}
					if status.Title != "" {
						updates["content"] = status.Title
					}

					if job.Progress < 100 && job.Power > 0 {
						service.NotifyAppPlazaPowerConsumed(
							p.db,
							uint(job.UserId),
							job.Power,
							service.PowerLogModelMusic,
							fmt.Sprintf("music:%d", job.Id),
						)
					}

					logger.Debugf("更新任务 %s 的 URL - Audio: %s, Video: %s, Image: %s",
						job.TaskId, status.AudioURL, status.VideoURL, status.ImageURL)
				}

				result := p.db.Model(&job).Updates(updates)
				if result.Error != nil {
					logger.Errorf("更新任务状态失败: %v", result.Error)
				} else {
					logger.Debugf("成功更新任务 %s 的状态，影响行数: %d", job.TaskId, result.RowsAffected)
				}

				// 通知客户端
				if cli := p.Clients.Get(uint(job.UserId)); cli != nil {
					cli.Send([]byte("Task Updated"))
				}
			}

			time.Sleep(time.Second * 3)
		}
	}()
}

// queryTaskStatus 查询单个任务状态（weda.cc: GET /suno/fetch/{task_id}）
func (p *SunoServicePool) queryTaskStatus(taskID string, preferAudio string) (*TaskStatus, error) {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/suno/fetch/%s", strings.TrimRight(p.config.ApiURL, "/"), taskID), nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	request.Header.Set("Authorization", "Bearer "+p.config.ApiKey)
	request.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}
	logger.Debugf("任务状态响应: %s", string(respBody))
	return buildTaskStatusFromFetchBody(respBody, preferAudio)
}

// BuildTaskStatusFromFetchBody 解析 fetch 接口 JSON（供 handler 复用）
func BuildTaskStatusFromFetchBody(respBody []byte, preferAudio string) (*TaskStatus, error) {
	return buildTaskStatusFromFetchBody(respBody, preferAudio)
}

// RefreshMusicJob 从供应商同步已完成任务的歌词与媒体信息
func RefreshMusicJob(db *gorm.DB, config *types.SunoConfig, job *model.MusicJob) error {
	if job == nil || config == nil || job.TaskId == "" {
		return fmt.Errorf("invalid job")
	}
	if strings.EqualFold(job.Status, "Succeeded") && job.AudioUrl != "" && job.Lyrics != "" {
		return nil
	}

	pool := &SunoServicePool{config: config, db: db}
	status, err := pool.queryTaskStatus(job.TaskId, job.AudioUrl)
	if err != nil {
		return err
	}
	if status.Status != "Succeeded" || status.AudioURL == "" {
		return nil
	}

	updates := map[string]interface{}{
		"progress": 100,
		"status":   "Succeeded",
		"err_msg":  "",
	}
	if status.Progress > 0 {
		updates["progress"] = status.Progress
	}
	if status.AudioURL != "" {
		updates["audio_url"] = status.AudioURL
	}
	if status.ImageURL != "" {
		updates["image_url"] = status.ImageURL
	}
	if status.VideoURL != "" {
		updates["video_url"] = status.VideoURL
	}
	if status.Lyrics != "" {
		updates["lyrics"] = status.Lyrics
	}
	if status.Duration > 0 {
		updates["duration"] = status.Duration
	}
	if status.Title != "" {
		updates["content"] = status.Title
	}

	return db.Model(job).Updates(updates).Error
}

// RepairFailedMusicJobsResult 批量修复失败音乐任务的结果
type RepairFailedMusicJobsResult struct {
	Total    int                          `json:"total"`
	Repaired int                          `json:"repaired"`
	Skipped  int                          `json:"skipped"`
	Errors   []string                     `json:"errors"`
	Items    []RepairFailedMusicJobItem   `json:"items"`
}

type RepairFailedMusicJobItem struct {
	Id     uint   `json:"id"`
	TaskId string `json:"task_id"`
	UserId int    `json:"user_id"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// RepairFailedMusicJobs 从上游重新拉取本地标记失败、但上游可能已成功的音乐任务
func RepairFailedMusicJobs(db *gorm.DB, config *types.SunoConfig) RepairFailedMusicJobsResult {
	result := RepairFailedMusicJobsResult{Items: make([]RepairFailedMusicJobItem, 0)}
	if db == nil || config == nil || config.ApiKey == "" {
		result.Errors = append(result.Errors, "Suno 服务未配置")
		return result
	}

	var jobs []model.MusicJob
	if err := db.Where("UPPER(status) = ? AND (audio_url = '' OR audio_url IS NULL)", "FAILED").
		Order("id ASC").
		Find(&jobs).Error; err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result
	}

	result.Total = len(jobs)
	for _, job := range jobs {
		item := RepairFailedMusicJobItem{
			Id:     job.Id,
			TaskId: job.TaskId,
			UserId: job.UserId,
		}
		if job.TaskId == "" {
			item.Status = "skipped"
			item.Reason = "缺少 task_id"
			result.Items = append(result.Items, item)
			result.Skipped++
			continue
		}

		if err := RefreshMusicJob(db, config, &job); err != nil {
			item.Status = "skipped"
			item.Reason = err.Error()
			result.Items = append(result.Items, item)
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("id=%d task=%s: %v", job.Id, job.TaskId, err))
			continue
		}

		var fresh model.MusicJob
		if err := db.Where("id = ?", job.Id).First(&fresh).Error; err != nil {
			item.Status = "skipped"
			item.Reason = "读取更新结果失败"
			result.Items = append(result.Items, item)
			result.Skipped++
			continue
		}
		if fresh.AudioUrl == "" || !strings.EqualFold(fresh.Status, "Succeeded") {
			item.Status = "skipped"
			item.Reason = "上游仍未成功或无音频"
			result.Items = append(result.Items, item)
			result.Skipped++
			continue
		}

		item.Status = "repaired"
		result.Items = append(result.Items, item)
		result.Repaired++
		time.Sleep(200 * time.Millisecond)
	}

	return result
}

// HasAvailableService 检查是否有可用服务
func (p *SunoServicePool) HasAvailableService() bool {
	return p.config != nil && p.config.ApiKey != ""
}
