package mj

import (
	"chatplus/core/types"
	"chatplus/service"
	"chatplus/service/oss"
	"chatplus/store"
	"chatplus/store/model"
	"chatplus/utils"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Service MJ 绘画服务
type Service struct {
	Name        string // service Name
	Client      Client // MJ Client
	taskQueue   *store.RedisQueue
	notifyQueue *store.RedisQueue
	db          *gorm.DB
	uploader    *oss.UploaderManager // nano：chat 接口返回 base64/data URL 时需落盘，避免 message_id/org_url 超长
}

func NewService(name string, taskQueue *store.RedisQueue, notifyQueue *store.RedisQueue, db *gorm.DB, cli Client, uploader *oss.UploaderManager) *Service {
	return &Service{
		Name:        name,
		db:          db,
		taskQueue:   taskQueue,
		notifyQueue: notifyQueue,
		Client:      cli,
		uploader:    uploader,
	}
}

// persistNanoGenerationImage 将网关返回的 data URL、纯 base64 或远程图转为站内可访问的短 URL（与 DownloadImages 链路一致）
func persistNanoGenerationImage(um *oss.UploaderManager, raw string) (string, error) {
	if um == nil {
		return "", fmt.Errorf("uploader 未配置")
	}
	h := um.GetUploadHandler()
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("空图片数据")
	}
	if strings.HasPrefix(raw, "data:image") {
		i := strings.Index(raw, ",")
		if i <= 0 || i >= len(raw)-1 {
			return "", fmt.Errorf("无效的 data URL")
		}
		return h.PutBase64(strings.TrimSpace(raw[i+1:]))
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return h.PutImg(raw, true)
	}
	return h.PutBase64(raw)
}

func isRemoteImageURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	return strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://")
}

func shouldPersistNanoPayload(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	if strings.HasPrefix(raw, "data:image") {
		return true
	}
	// 远程图必须落盘到 img_url；超长 base64 也需落盘
	if isRemoteImageURL(raw) {
		return true
	}
	return len(raw) > 500
}

// normalizeLocalImgURL 统一为站内路径，如 /static/upload/2026/5/xxx.jpeg
func normalizeLocalImgURL(stored string) string {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return stored
	}
	if idx := strings.Index(stored, "/static/upload/"); idx >= 0 {
		return stored[idx:]
	}
	return stored
}

func (s *Service) Run() {
	logger.Infof("Starting MidJourney job consumer for %s", s.Name)
	for {
		var task types.MjTask
		err := s.taskQueue.LPop(&task)
		if err != nil {
			logger.Errorf("taking task with error: %v", err)
			continue
		}

		//  如果配置了多个中转平台的 API KEY
		// U,V 操作必须和 Image 操作属于同一个平台，否则找不到关联任务，需重新放回任务列表
		if task.ChannelId != "" && task.ChannelId != s.Name {
			logger.Debugf("handle other service task, name: %s, channel_id: %s, drop it.", s.Name, task.ChannelId)
			s.taskQueue.RPush(task)
			time.Sleep(time.Second)
			continue
		}

		// translate prompt
		//if utils.HasChinese(task.Prompt) {
		//	content, err := utils.OpenAIRequest(s.db, fmt.Sprintf(service.TranslatePromptTemplate, task.Prompt))
		//	if err == nil {
		//		task.Prompt = content
		//	} else {
		//		logger.Warnf("error with translate prompt: %v", err)
		//	}
		//}
		//// translate negative prompt
		//if task.NegPrompt != "" && utils.HasChinese(task.NegPrompt) {
		//	content, err := utils.OpenAIRequest(s.db, fmt.Sprintf(service.TranslatePromptTemplate, task.NegPrompt))
		//	if err == nil {
		//		task.NegPrompt = content
		//	} else {
		//		logger.Warnf("error with translate prompt: %v", err)
		//	}
		//}

		var job model.MidJourneyJob
		tx := s.db.Where("id = ?", task.Id).First(&job)
		if tx.Error != nil {
			logger.Error("任务不存在，任务ID：", task.TaskId)
			continue
		}

		logger.Infof("%s handle a new MidJourney task: %+v", s.Name, task)
		var res ImageRes
		switch task.Type {
		case types.TaskImage:
			res, err = s.Client.Imagine(task)
			break
		case types.TaskUpscale:
			res, err = s.Client.Upscale(task)
			break
		case types.TaskVariation:
			res, err = s.Client.Variation(task)
			break
		case types.TaskBlend:
			res, err = s.Client.Blend(task)
			break
		case types.TaskSwapFace:
			res, err = s.Client.SwapFace(task)
			break
		}

		if err != nil || (res.Code != 1 && res.Code != 22) {
			errMsg := fmt.Sprintf("%v,%s", err, res.Description)
			logger.Error("绘画任务执行失败：", errMsg)
			job.Progress = -1
			job.ErrMsg = errMsg
			// update the task progress
			s.db.Updates(&job)
			// 任务失败，通知前端
			s.notifyQueue.RPush(task.UserId)
			continue
		}
		logger.Infof("任务提交成功：%+v", res)

		// 对于 nano / image2，同步返回结果后直接完成
		if res.Result != "" {
			logger.Infof("%s 任务完成：%+v", s.Name, res)
			upstreamURL := strings.TrimSpace(res.Result)
			imgURL := upstreamURL
			orgURL := upstreamURL

			if (s.Name == "nano-service" || s.Name == "image2-service") && shouldPersistNanoPayload(upstreamURL) {
				if s.uploader == nil {
					logger.Error(s.Name + ": 未注入 OSS，无法保存返回的图片")
					job.Progress = -1
					job.ErrMsg = "服务器未配置图片存储，无法保存生成结果"
					s.db.Updates(&job)
					s.notifyQueue.RPush(task.UserId)
					continue
				}
				stored, errStore := persistNanoGenerationImage(s.uploader, upstreamURL)
				if errStore != nil {
					logger.Errorf("%s 图片持久化失败：%v", s.Name, errStore)
					job.Progress = -1
					job.ErrMsg = "图片保存失败: " + errStore.Error()
					s.db.Updates(&job)
					s.notifyQueue.RPush(task.UserId)
					continue
				}
				imgURL = normalizeLocalImgURL(stored)
				if isRemoteImageURL(upstreamURL) {
					orgURL = upstreamURL
				} else {
					orgURL = imgURL
				}
			} else if isRemoteImageURL(upstreamURL) {
				// 其它服务若返回外链，仍保留 org 为上游（img 由 DownloadImages 异步补全）
				orgURL = upstreamURL
			}

			err = s.db.Model(&model.MidJourneyJob{Id: job.Id}).UpdateColumns(map[string]interface{}{
				"progress":   100,
				"org_url":    orgURL,
				"img_url":    imgURL,
				"message_id": imgURL,
				"channel_id": s.Name,
				"err_msg":    "",
			}).Error
			if err != nil {
				logger.Errorf("更新任务状态失败：%v", err)
			} else {
				logger.Infof("任务状态更新成功，org_url=%s img_url=%s", orgURL, imgURL)
				service.NotifyAppPlazaPowerConsumed(
					s.db,
					uint(job.UserId),
					job.Power,
					ResolveImagePowerLogModel(job.Prompt, "gemini-mj"),
					fmt.Sprintf("mj:%d", job.Id),
				)
			}
			// 通知前端任务完成
			s.notifyQueue.RPush(task.UserId)
		} else {
			// 更新任务 ID/频道（传统 MJ 模式）
			job.TaskId = res.Result
			job.MessageId = res.Result
			job.ChannelId = s.Name
			s.db.Updates(&job)
		}
	}
}

type CBReq struct {
	Id          string      `json:"id"`
	Action      string      `json:"action"`
	Status      string      `json:"status"`
	Prompt      string      `json:"prompt"`
	PromptEn    string      `json:"promptEn"`
	Description string      `json:"description"`
	SubmitTime  int64       `json:"submitTime"`
	StartTime   int64       `json:"startTime"`
	FinishTime  int64       `json:"finishTime"`
	Progress    string      `json:"progress"`
	ImageUrl    string      `json:"imageUrl"`
	FailReason  interface{} `json:"failReason"`
	Properties  struct {
		FinalPrompt string `json:"finalPrompt"`
	} `json:"properties"`
}

func (s *Service) Notify(job model.MidJourneyJob) error {
	task, err := s.Client.QueryTask(job.TaskId)

	logger.Info("获取绘画进度条Notify")

	logger.Info("TaskId", job.TaskId)
	if err != nil {
		return err
	}

	// 任务执行失败了
	if task.FailReason != "" {
		s.db.Model(&model.MidJourneyJob{Id: job.Id}).UpdateColumns(map[string]interface{}{
			"progress": -1,
			"err_msg":  task.FailReason,
		})
		s.notifyQueue.RPush(job.UserId)
		return fmt.Errorf("task failed: %v", task.FailReason)
	}

	if len(task.Buttons) > 0 {
		job.Hash = GetImageHash(task.Buttons[0].CustomId)
	}
	oldProgress := job.Progress
	logger.Info("进度条", oldProgress)

	job.Progress = utils.IntValue(strings.Replace(task.Progress, "%", "", 1), 0)
	job.Prompt = task.PromptEn
	if task.ImageUrl != "" {
		job.OrgURL = task.ImageUrl
	}
	tx := s.db.Updates(&job)
	if tx.Error != nil {
		return fmt.Errorf("error with update database: %v", tx.Error)
	}
	if job.Progress >= 100 && oldProgress < 100 {
		service.NotifyAppPlazaPowerConsumed(
			s.db,
			uint(job.UserId),
			job.Power,
			ResolveImagePowerLogModel(job.Prompt, "gemini-mj"),
			fmt.Sprintf("mj:%d", job.Id),
		)
	}
	// 通知前端更新任务进度
	if oldProgress != job.Progress {
		s.notifyQueue.RPush(job.UserId)
	}
	return nil
}

func GetImageHash(action string) string {
	split := strings.Split(action, "::")
	if len(split) > 5 {
		return split[4]
	}
	return split[len(split)-1]
}
