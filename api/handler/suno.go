package handler

import (
	"bytes"
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/service"
	"chatplus/store/model"
	sunosvc "chatplus/service/suno"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SunoHandler 继承 BaseHandler
type SunoHandler struct {
	*BaseHandler
	client      *http.Client
	config      *types.SunoConfig
	db          *gorm.DB
	userService *service.UserService
}

// 修改构造函数，确保正确初始化 BaseHandler
func NewSunoHandler(app *core.AppServer, config *types.AppConfig, db *gorm.DB, userService *service.UserService) *SunoHandler {
	return &SunoHandler{
		BaseHandler: &BaseHandler{
			App: app,
			DB:  db,
		},
		client:      &http.Client{},
		config:      &config.SunoConfig,
		db:          db,
		userService: userService,
	}
}

func (h *SunoHandler) resolveMusicPower() int {
	if p := h.App.SysConfig.MusicPower; p > 0 {
		return int(p)
	}
	return 100
}

func parseSunoSubmitTaskID(respBody []byte) (string, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return "", err
	}
	code := raw["code"]
	ok := false
	switch v := code.(type) {
	case string:
		ok = strings.EqualFold(strings.TrimSpace(v), "success")
	case float64:
		ok = v == 0
	case json.Number:
		n, _ := v.Int64()
		ok = n == 0
	}
	if !ok {
		msg, _ := raw["message"].(string)
		if msg == "" {
			msg, _ = raw["msg"].(string)
		}
		if msg == "" {
			msg = "上游返回失败"
		}
		return "", fmt.Errorf(msg)
	}

	data := raw["data"]
	switch v := data.(type) {
	case string:
		if id := strings.TrimSpace(v); id != "" {
			return id, nil
		}
	case map[string]interface{}:
		for _, key := range []string{"task_id", "taskId", "id", "data"} {
			if s, ok := v[key].(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s), nil
			}
		}
	}
	return "", fmt.Errorf("上游未返回任务 ID")
}


// Generate 处理音乐生成请求
func (h *SunoHandler) Generate(c *gin.Context) {
	if deny, msg := DenyIfAppDisabled(h.App.SysConfig, AppFeatureMusic); deny {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  msg,
		})
		return
	}

	// 获取用户ID
	userId := h.GetLoginUserId(c)
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "请先登录",
		})
		return
	}

	// 获取用户信息用于预检查
	user, err := h.GetLoginUser(c)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "获取用户信息失败",
		})
		return
	}

	// 检查用户算力是否足够
	powerCost := h.resolveMusicPower()
	if user.Power < float64(powerCost) {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "你当前可用算力余额不足，请充值后再操作！",
		})
		return
	}

	// 定义通用请求结构体
	var req struct {
		// 灵感模式字段
		GptDescriptionPrompt string `json:"gpt_description_prompt"`
		MakeInstrumental     bool   `json:"make_instrumental"`
		MV                   string `json:"mv"`
		Prompt               string `json:"prompt"`

		// 自定义模式字段
		Title string `json:"title"`
		Tags  string `json:"tags"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "参数错误",
		})
		return
	}

	if strings.TrimSpace(req.MV) == "" {
		req.MV = "chirp-v4"
	}

	var apiReq map[string]interface{}

	// 根据请求参数判断模式
	if req.GptDescriptionPrompt != "" {
		// 灵感模式
		apiReq = map[string]interface{}{
			"gpt_description_prompt": req.GptDescriptionPrompt,
			"make_instrumental":      req.MakeInstrumental,
			"mv":                     req.MV,
			"prompt":                 req.Prompt,
		}
	} else {
		// 自定义模式
		apiReq = map[string]interface{}{
			"continue_at":      120,
			"continue_clip_id": "",
			"mv":               req.MV,
			"prompt":           req.Prompt,
			"tags":             req.Tags,
			"task":             "",
			"title":            req.Title,
		}
	}

	// 序列化请求体
	body, err := json.Marshal(apiReq)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "序列化请求失败",
		})
		return
	}

	// 创建请求
	apiBase := strings.TrimRight(h.config.ApiURL, "/")
	request, err := http.NewRequest("POST", apiBase+"/suno/submit/music", bytes.NewBuffer(body))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "创建请求失败",
		})
		return
	}

	// 设置请求头
	request.Header.Set("Authorization", "Bearer "+h.config.ApiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	// 发送请求
	resp, err := h.client.Do(request)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "请求失败",
		})
		return
	}
	defer resp.Body.Close()

	// 读取完整的响应内容
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应失败: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "读取响应失败",
		})
		return
	}
	log.Printf("完整响应内容: %s", string(respBody))

	taskID, submitErr := parseSunoSubmitTaskID(respBody)
	if submitErr != nil {
		log.Printf("解析提交结果失败: %v, 响应内容: %s", submitErr, string(respBody))
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "生成失败: " + submitErr.Error(),
		})
		return
	}

	content := req.Title
	if content == "" {
		content = req.Prompt
	}
	lyrics := ""
	if req.Title != "" && req.Prompt != "" && req.GptDescriptionPrompt == "" {
		lyrics = req.Prompt
	}
	job := &model.MusicJob{
		UserId:    int(userId),
		TaskId:    taskID,
		Content:   content,
		Lyrics:    lyrics,
		Progress:  0,
		Status:    "Processing",
		StatusId:  fmt.Sprintf("%d", time.Now().UnixNano()),
		ChannelId: "suno_service-0",
		Power:     float64(powerCost),
		CreatedAt: time.Now(),
	}

	if err := h.db.Create(job).Error; err != nil {
		log.Printf("保存任务失败: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "保存任务失败",
		})
		return
	}

	if err := h.userService.DecreasePower(int(userId), powerCost, model.PowerLog{
		Type:   types.PowerConsume,
		Model:  service.PowerLogModelMusic,
		Remark: fmt.Sprintf("音乐生成操作，任务ID：%s", taskID),
	}); err != nil {
		_ = h.db.Delete(&model.MusicJob{}, job.Id)
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": job,
	})
}

// 后台轮询任务状态
func (h *SunoHandler) pollTaskStatus(taskID string) {
	apiBase := strings.TrimRight(h.config.ApiURL, "/")
	for i := 0; i < 100; i++ {
		time.Sleep(3 * time.Second)

		req, _ := http.NewRequest("GET", fmt.Sprintf("%s/suno/fetch/%s", apiBase, taskID), nil)
		req.Header.Set("Authorization", "Bearer "+h.config.ApiKey)
		req.Header.Set("Accept", "application/json")

		resp, err := h.client.Do(req)
		if err != nil {
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		taskStatus, err := sunosvc.BuildTaskStatusFromFetchBody(respBody, "")
		if err != nil {
			continue
		}

		status := taskStatus.Status
		if strings.EqualFold(status, "Succeeded") {
			status = "Succeeded"
		} else if strings.EqualFold(status, "Failed") {
			status = "Failed"
		}

		updates := map[string]interface{}{
			"progress": taskStatus.Progress,
			"status":   status,
		}

		if strings.EqualFold(taskStatus.Status, "Succeeded") && taskStatus.AudioURL != "" {
			updates["progress"] = 100
			updates["audio_url"] = taskStatus.AudioURL
			updates["image_url"] = taskStatus.ImageURL
			updates["video_url"] = taskStatus.VideoURL
			if taskStatus.Lyrics != "" {
				updates["lyrics"] = taskStatus.Lyrics
			}
			if taskStatus.Duration > 0 {
				updates["duration"] = taskStatus.Duration
			}
			if taskStatus.Title != "" {
				updates["content"] = taskStatus.Title
			}
		} else if strings.EqualFold(taskStatus.Status, "Failed") {
			updates["err_msg"] = taskStatus.Error
		}

		h.db.Model(&model.MusicJob{}).Where("task_id = ?", taskID).Updates(updates)

		if strings.EqualFold(taskStatus.Status, "Succeeded") || strings.EqualFold(taskStatus.Status, "Failed") {
			break
		}
	}
}

// GenerateByDescription 通过描述生成音乐
func (h *SunoHandler) GenerateByDescription(c *gin.Context) {
	var req struct {
		Title            string `json:"title"`
		Description      string `json:"gpt_description_prompt"`
		MakeInstrumental bool   `json:"make_instrumental"`
		ModelVersion     string `json:"mv"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "参数错误",
		})
		return
	}

	//调用preCheck 方法

	// 调用 Suno API 的描述模式生成
	body, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "请求失败",
		})
		return
	}

	request, err := http.NewRequest("POST", h.config.ApiURL+"/generate/description-mode", bytes.NewBuffer(body))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "请求失败",
		})
		return
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+h.config.ApiKey)

	resp, err := h.client.Do(request)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "生成失败",
		})
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "解析响应失败",
		})
		return
	}

	//记录日志
	//
	//tx := h.DB.Model(&model.User{}).Where("id = ?", job.UserId).UpdateColumn("power", gorm.Expr("power - ?", job.Power))
	//// 记录算力变化日志
	//if tx.Error == nil && tx.RowsAffected > 0 {
	//	user, _ := h.GetLoginUser(c)
	//	h.DB.Create(&model.PowerLog{
	//		UserId:    user.Id,
	//		Username:  user.Username,
	//		Type:      types.PowerConsume,
	//		Amount:    job.Power,
	//		Balance:   user.Power - job.Power,
	//		Mark:      types.PowerSub,
	//		Model:     "音乐生成",
	//		Remark:    fmt.Sprintf("%s操作，任务ID：%s", "音乐生成", job.TaskId),
	//		CreatedAt: time.Now(),
	//	})
	//}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
	})
}

// GetStatus 获取音乐生成状态
func (h *SunoHandler) GetStatus(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "参数错误",
		})
		return
	}

	var job model.MusicJob
	if err := h.db.Where("task_id = ?", taskID).First(&job).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "任务不存在",
		})
		return
	}

	needRefresh := c.Query("refresh") == "1"
	if needRefresh && job.TaskId != "" {
		if strings.EqualFold(job.Status, "FAILED") || job.AudioUrl == "" {
			_ = sunosvc.RefreshMusicJob(h.db, h.config, &job)
			_ = h.db.Where("task_id = ?", taskID).First(&job).Error
		} else if job.Lyrics == "" && job.Progress >= 100 {
			_ = sunosvc.RefreshMusicJob(h.db, h.config, &job)
			_ = h.db.Where("task_id = ?", taskID).First(&job).Error
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": job,
	})
}

// GenerateLyrics 生成歌词
func (h *SunoHandler) GenerateLyrics(c *gin.Context) {
	var req struct {
		Prompt string `json:"prompt"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "参数错误",
		})
		return
	}

	//调用preCheck 方法

	// 调用 Suno API 生成歌词
	body, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "请求失败",
		})
		return
	}

	request, err := http.NewRequest("POST", h.config.ApiURL+"/generate/lyrics", bytes.NewBuffer(body))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "请求失败",
		})
		return
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+h.config.ApiKey)

	resp, err := h.client.Do(request)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "生成失败",
		})
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "解析响应失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
	})
}

// GetList 获取音乐列表
func (h *SunoHandler) GetList(c *gin.Context) {
	// 获取用户ID
	userId := h.GetLoginUserId(c)
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "请先登录",
		})
		return
	}

	var jobs []model.MusicJob
	if err := h.db.Where("user_id = ?", userId).
		Order("id desc").
		Find(&jobs).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "获取列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": jobs,
	})
}

// List 获取音乐列表
func (h *SunoHandler) List(c *gin.Context) {
	// 获取用户ID
	userId := h.GetLoginUserId(c)
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "请先登录",
		})
		return
	}

	var jobs []model.MusicJob
	if err := h.db.Where("user_id = ?", userId).
		Order("id desc").
		Find(&jobs).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "获取列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": jobs,
	})
}

// Status 获取任务状态
func (h *SunoHandler) Status(c *gin.Context) {
	// 获取用户ID
	userId := h.GetLoginUserId(c)
	if userId == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "请先登录",
		})
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "参数错误",
		})
		return
	}

	var job model.MusicJob
	if err := h.db.Where("task_id = ?", taskID).First(&job).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "任务不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": job,
	})
}

// Download 同源代理下载音频，避免外链 CORS / 移动端无法触发 download
func (h *SunoHandler) Download(c *gin.Context) {
	userId := h.GetLoginUserId(c)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": -1, "msg": "请先登录"})
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": -1, "msg": "参数错误"})
		return
	}

	var job model.MusicJob
	if err := h.db.Where("id = ? AND user_id = ?", id, userId).First(&job).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "msg": "作品不存在"})
		return
	}
	audioURL := strings.TrimSpace(job.AudioUrl)
	if audioURL == "" {
		c.JSON(http.StatusNotFound, gin.H{"code": -1, "msg": "暂无可下载音频"})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, audioURL, nil)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": -1, "msg": "构造下载请求失败"})
		return
	}
	client := h.client
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	up, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": -1, "msg": "拉取音频失败"})
		return
	}
	defer up.Body.Close()
	if up.StatusCode < 200 || up.StatusCode >= 300 {
		c.JSON(http.StatusBadGateway, gin.H{"code": -1, "msg": fmt.Sprintf("上游返回 %d", up.StatusCode)})
		return
	}

	ct := up.Header.Get("Content-Type")
	if ct == "" {
		ct = "audio/mpeg"
	}
	ext := "mp3"
	lct := strings.ToLower(ct)
	switch {
	case strings.Contains(lct, "wav"):
		ext = "wav"
	case strings.Contains(lct, "ogg"):
		ext = "ogg"
	case strings.Contains(lct, "mp4"), strings.Contains(lct, "m4a"):
		ext = "m4a"
	}
	filename := fmt.Sprintf("music_%d.%s", job.Id, ext)
	c.Header("Content-Type", ct)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	if cl := up.Header.Get("Content-Length"); cl != "" {
		c.Header("Content-Length", cl)
	}
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, up.Body)
}
