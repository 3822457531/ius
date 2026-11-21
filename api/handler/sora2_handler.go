package handler

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/service"
	"chatplus/service/oss"
	"chatplus/service/sora2"
	"chatplus/store/model"
	"chatplus/utils"
	"chatplus/utils/resp"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Sora2Handler struct {
	BaseHandler
	sora2Service *sora2.Service
	uploader     *oss.UploaderManager
	userService  *service.UserService
}

func NewSora2Handler(app *core.AppServer, db *gorm.DB, sora2Service *sora2.Service, uploader *oss.UploaderManager, userService *service.UserService) *Sora2Handler {
	return &Sora2Handler{
		BaseHandler: BaseHandler{
			App: app,
			DB:  db,
		},
		sora2Service: sora2Service,
		uploader:     uploader,
		userService:  userService,
	}
}

// Generate 生成Sora2视频
func (h *Sora2Handler) Generate(c *gin.Context) {
	if deny, msg := DenyIfAppDisabled(h.App.SysConfig, AppFeatureVideo); deny {
		resp.ERROR(c, msg)
		return
	}

	var data struct {
		Prompt         string   `json:"prompt" binding:"required"`
		Images         []string `json:"images"`       // 图生视频的图片数组
		Duration       string   `json:"duration"`     // 5, 10, 15, 30
		AspectRatio    string   `json:"aspect_ratio"` // 16:9, 9:16, 1:1
		Quality        string   `json:"quality"`      // standard, hd, uhd
		Style          string   `json:"style"`        // realistic, animated, artistic, cinematic
		NegativePrompt string   `json:"negative_prompt"`
		Seed           int64    `json:"seed"`
		Steps          int      `json:"steps"`
		Model          string   `json:"model"` // sora-2, sora-2-pro
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	// 验证参数
	if data.Prompt == "" {
		resp.ERROR(c, "提示词不能为空")
		return
	}

	// 设置默认值
	// if data.Duration == "" {
	// 	data.Duration = "10"
	// }
	if data.AspectRatio == "" {
		data.AspectRatio = "16:9"
	}
	// if data.Quality == "" {
	// 	data.Quality = "hd"
	// }
	// if data.Style == "" {
	// 	data.Style = "realistic"
	// }
	if data.Model == "" {
		data.Model = "doubao-seedance-2-0-fast-260128"
	}
	// if data.Steps == 0 {
	// 	data.Steps = 30
	// }

	// 计算消耗的算力
	power := h.calculatePower(data.Duration, data.Quality, data.Model)

	if user.Power < float64(power) {
		resp.ERROR(c, "您的算力不足，请充值后再试！")
		return
	}

	userId := int(h.GetLoginUserId(c))

	// 创建任务参数
	params := types.Sora2Params{
		Duration:       data.Duration,
		AspectRatio:    data.AspectRatio,
		Quality:        data.Quality,
		Style:          data.Style,
		NegativePrompt: data.NegativePrompt,
		Seed:           data.Seed,
		Steps:          data.Steps,
		Model:          data.Model,
	}

	task := types.Sora2Task{
		ClientId: utils.RandString(10),
		UserId:   userId,
		Prompt:   data.Prompt,
		Images:   data.Images,
		Params:   params,
		// TranslateModelId: 0, // 如果需要翻译，可以在这里设置模型ID
	}

	// 插入数据库（power 在扣费成功后再写入，避免提交失败被误退款）
	job := model.Sora2Job{
		UserId:   userId,
		Prompt:   data.Prompt,
		Images:   utils.JsonEncode(data.Images),
		Power:    0,
		UpStatus: 0,
		TaskInfo: utils.JsonEncode(task),
		Status:   "pending",
	}

	tx := h.DB.Create(&job)
	if tx.Error != nil {
		resp.ERROR(c, tx.Error.Error())
		return
	}

	// 使用新厂商模式：立即提交上游任务，直接拿到外部 task_id 返回给前端
	task.Id = job.Id
	createRes, err := h.sora2Service.Sora2Create(task)
	if err != nil {
		_ = h.DB.Model(&model.Sora2Job{Id: job.Id}).UpdateColumns(map[string]interface{}{
			"status":  "failed",
			"err_msg": err.Error(),
			"power":   0,
		}).Error
		resp.ERROR(c, "系统维护中")
		return
	}

	// 回写上游 task_id / channel，状态切为 processing
	err = h.DB.Model(&model.Sora2Job{Id: job.Id}).UpdateColumns(map[string]interface{}{
		"task_id":    createRes.TaskId,
		"channel":    createRes.Channel,
		"prompt_ext": task.Prompt,
		"status":     "processing",
	}).Error
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	// 扣除用户算力（在提交成功后扣减）
	err = h.userService.DecreasePower(job.UserId, power, model.PowerLog{
		Type:   types.PowerConsume,
		Model:  service.PowerLogModelVideo,
		Remark: fmt.Sprintf("AI视频生成，任务ID：%d", job.Id),
	})
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}
	_ = h.DB.Model(&model.Sora2Job{Id: job.Id}).UpdateColumn("power", power).Error

	resp.SUCCESS(c, gin.H{
		"task_id": createRes.TaskId,
		"message": "任务已提交，正在生成中...",
	})
}

// List 获取视频列表
func (h *Sora2Handler) List(c *gin.Context) {
	userId := h.GetLoginUserId(c)
	page := h.GetInt(c, "page", 1)
	pageSize := h.GetInt(c, "page_size", 20)
	status := c.Query("status")

	session := h.DB.Session(&gorm.Session{}).Where("user_id", userId)
	if status != "" {
		session = session.Where("status", status)
	}

	// 统计总数
	var total int64
	session.Model(&model.Sora2Job{}).Count(&total)

	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		session = session.Offset(offset).Limit(pageSize)
	}

	var list []model.Sora2Job
	err := session.Order("id desc").Find(&list).Error
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	resp.SUCCESS(c, gin.H{
		"total": total,
		"page":  page,
		"size":  pageSize,
		"data":  list,
	})
}

// ToggleFavorite 切换收藏状态
func (h *Sora2Handler) ToggleFavorite(c *gin.Context) {
	id := h.GetInt(c, "id", 0)
	userId := h.GetLoginUserId(c)

	var job model.Sora2Job
	err := h.DB.Where("id = ? AND user_id = ?", id, userId).First(&job).Error
	if err != nil {
		resp.ERROR(c, "视频不存在")
		return
	}

	// 切换收藏状态
	job.IsFavorite = !job.IsFavorite
	err = h.DB.Model(&job).UpdateColumn("is_favorite", job.IsFavorite).Error
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	message := "已取消收藏"
	if job.IsFavorite {
		message = "已添加到收藏"
	}

	resp.SUCCESS(c, gin.H{
		"is_favorite": job.IsFavorite,
		"message":     message,
	})
}

// Remove 删除视频
func (h *Sora2Handler) Remove(c *gin.Context) {
	id := h.GetInt(c, "id", 0)
	userId := h.GetLoginUserId(c)

	var job model.Sora2Job
	err := h.DB.Where("id = ? AND user_id = ?", id, userId).First(&job).Error
	if err != nil {
		resp.ERROR(c, "视频不存在")
		return
	}

	// 生成中且未超时不可删除；已完成、失败可删除
	if job.Status == "processing" || job.Status == "pending" {
		if time.Now().Before(job.CreatedAt.Add(10 * time.Minute)) {
			resp.ERROR(c, "视频生成中，请稍后再删除")
			return
		}
	}

	// 删除任务
	err = h.DB.Delete(&job).Error
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	// 删除文件
	if job.VideoURL != "" {
		_ = h.uploader.GetUploadHandler().Delete(job.VideoURL)
	}
	if job.CoverURL != "" {
		_ = h.uploader.GetUploadHandler().Delete(job.CoverURL)
	}

	resp.SUCCESS(c)
}

// GetProgress 查询任务进度
func (h *Sora2Handler) GetProgress(c *gin.Context) {
	taskId := c.Param("task_id")
	if taskId == "" {
		resp.ERROR(c, "任务ID不能为空")
		return
	}

	// 查询任务状态：按上游 task_id 查询（与厂商协议一致）
	var job model.Sora2Job
	err := h.DB.Where("task_id = ?", taskId).First(&job).Error
	if err != nil {
		resp.ERROR(c, "任务不存在")
		return
	}

	// 检查用户权限
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.ERROR(c, "请先登录")
		return
	}
	if int(user.Id) != job.UserId {
		resp.ERROR(c, "无权查看该任务")
		return
	}

	// 超时兜底：10分钟仍未完成，直接标记失败并返回失败状态
	if job.Status == "processing" && time.Since(job.CreatedAt) > 10*time.Minute {
		timeoutMsg := "任务超时(10分钟未完成)"
		_ = h.DB.Model(&model.Sora2Job{Id: job.Id}).UpdateColumns(map[string]interface{}{
			"status":  "failed",
			"err_msg": timeoutMsg,
		}).Error
		job.Status = "failed"
		job.ErrMsg = timeoutMsg
	}

	duration := ""
	output := ""
	progress := h.mapStatusToProgress(job.Status)
	statusEnum := h.mapStatusToEnum(job.Status)
	if job.RawData != "" {
		var raw struct {
			Duration   string `json:"duration"`
			Output     string `json:"output"`
			Status     string `json:"status"`
			Progress   string `json:"progress"`
			FailReason string `json:"fail_reason"`
			Data       *struct {
				Duration string `json:"duration"`
				Output   string `json:"output"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(job.RawData), &raw); err == nil {
			duration = raw.Duration
			output = raw.Output
			if raw.Data != nil {
				if raw.Data.Duration != "" {
					duration = raw.Data.Duration
				}
				if raw.Data.Output != "" {
					output = raw.Data.Output
				}
			}
			if raw.Progress != "" {
				progress = strings.TrimSuffix(strings.TrimSpace(raw.Progress), "%")
			}
			if raw.Status != "" && job.Status == "processing" {
				switch strings.ToUpper(raw.Status) {
				case "SUCCESS", "COMPLETED":
					statusEnum = "SUCCESS"
					progress = "100"
				case "FAILURE", "FAILED":
					statusEnum = "FAILURE"
				default:
					statusEnum = "IN_PROGRESS"
				}
			}
			if job.ErrMsg == "" && raw.FailReason != "" {
				job.ErrMsg = raw.FailReason
			}
		}
	}
	if output == "" {
		output = job.VideoURL
	}
	if job.Status == "completed" {
		progress = "100"
		statusEnum = "SUCCESS"
	}

	progressData := gin.H{
		"task_id":    job.TaskId,
		"status":     statusEnum,
		"progress":   progress,
		"duration":   duration,
		"output":     output,
		"video_url":  job.VideoURL,
		"cover_url":  job.CoverURL,
		"water_url":  job.WaterURL,
		"err_msg":    job.ErrMsg,
		"created_at": job.CreatedAt,
		"updated_at": job.UpdatedAt,
	}

	resp.SUCCESS(c, progressData)
}

// calculatePower 计算消耗的算力
func (h *Sora2Handler) calculatePower(duration, quality, model string) int {
	// 使用默认算力值，可以根据 duration、quality、model 计算不同的算力消耗
	// 暂时使用固定值，后续可以根据需要调整
	basePower := 500 // 默认消耗 10 算力
	return basePower
}

// mapStatusToEnum 将数据库状态映射为API枚举状态
func (h *Sora2Handler) mapStatusToEnum(status string) string {
	switch status {
	case "pending":
		return "NOT_START"
	case "processing":
		return "IN_PROGRESS"
	case "completed":
		return "SUCCESS"
	case "failed":
		return "FAILURE"
	default:
		return "NOT_START"
	}
}

// mapStatusToProgress 将数据库状态映射为进度值
func (h *Sora2Handler) mapStatusToProgress(status string) string {
	switch status {
	case "pending":
		return "0"
	case "processing":
		return "50"
	case "completed":
		return "100"
	case "failed":
		return "0"
	default:
		return "0"
	}
}
