package handler

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/service"
	"chatplus/service/mj"
	"chatplus/service/oss"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils"
	"chatplus/utils/resp"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

const logoPromptMarker = "[AI-LOGO]"

func applyMjAppSourceFilter(session *gorm.DB, appSource string) *gorm.DB {
	switch strings.ToLower(strings.TrimSpace(appSource)) {
	case "logo":
		return session.Where("prompt LIKE ?", "%"+logoPromptMarker+"%")
	case "image":
		return session.Where("prompt NOT LIKE ?", "%"+logoPromptMarker+"%")
	default:
		return session
	}
}

type MidJourneyHandler struct {
	BaseHandler
	pool        *mj.ServicePool
	snowflake   *service.Snowflake
	uploader    *oss.UploaderManager
	userService *service.UserService
}

func NewMidJourneyHandler(app *core.AppServer, db *gorm.DB, snowflake *service.Snowflake, pool *mj.ServicePool, manager *oss.UploaderManager, userService *service.UserService) *MidJourneyHandler {
	return &MidJourneyHandler{
		snowflake:   snowflake,
		pool:        pool,
		uploader:    manager,
		userService: userService,
		BaseHandler: BaseHandler{
			App: app,
			DB:  db,
		},
	}
}

func (h *MidJourneyHandler) preCheck(c *gin.Context) bool {
	return h.preCheckPower(c, int(h.App.SysConfig.MjPower))
}

func (h *MidJourneyHandler) preCheckPower(c *gin.Context, requiredPower int) bool {
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return false
	}

	if user.Power < float64(requiredPower) {
		resp.ERROR(c, "当前用户剩余算力不足以完成本次绘画！")
		return false
	}

	if !h.pool.HasAvailableService() {
		resp.ERROR(c, "MidJourney 池子中没有没有可用的服务！")
		return false
	}

	return true
}

// Client WebSocket 客户端，用于通知任务状态变更
func (h *MidJourneyHandler) Client(c *gin.Context) {
	ws, err := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error(err)
		c.Abort()
		return
	}

	userId := h.GetInt(c, "user_id", 0)
	if userId == 0 {
		logger.Info("Invalid user ID")
		c.Abort()
		return
	}

	client := types.NewWsClient(ws)
	h.pool.Clients.Put(uint(userId), client)
	logger.Infof("New websocket connected, IP: %s", c.RemoteIP())
}

// Image 创建一个绘画任务
func (h *MidJourneyHandler) Image(c *gin.Context) {
	var data struct {
		SessionId string   `json:"session_id"`
		ClientId  string   `json:"client_id"`
		TaskType  string   `json:"task_type"`
		NanoMode  string   `json:"nano_mode"`
		Provider  string   `json:"provider"` // image2 | nano（默认 nano-service）
		AppSource string   `json:"app_source"` // image | logo
		Prompt    string   `json:"prompt"`
		NegPrompt string   `json:"neg_prompt"`
		Rate      string   `json:"rate"`
		Model     string   `json:"model"`
		Chaos     int      `json:"chaos"`
		Raw       bool     `json:"raw"`
		Seed      int64    `json:"seed"`
		Stylize   int      `json:"stylize"`
		ImgArr    []string `json:"img_arr"`
		Tile      bool     `json:"tile"`
		Quality   float32  `json:"quality"`
		Iw        float32  `json:"iw"`
		CRef      string   `json:"cref"` //生成角色一致的图像
		SRef      string   `json:"sref"` //生成风格一致的图像
		Cw        int      `json:"cw"`   // 参考程度
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	feature := AppFeatureImage
	if strings.EqualFold(strings.TrimSpace(data.AppSource), "logo") {
		feature = AppFeatureLogo
		if !strings.Contains(data.Prompt, logoPromptMarker) {
			data.Prompt = strings.TrimSpace(logoPromptMarker + " " + data.Prompt)
		}
	}
	if deny, msg := DenyIfAppDisabled(h.App.SysConfig, feature); deny {
		resp.ERROR(c, msg)
		return
	}

	if data.TaskType != types.TaskImage.String() {
		resp.ERROR(c, "当前模型仅支持文生图和图生图")
		return
	}

	if data.NanoMode == "" {
		data.NanoMode = "txt2img"
	}

	channelID := "nano-service"
	useImage2 := data.Provider == "image2" || data.Model == "gpt-image-2" || strings.HasPrefix(data.Model, "gpt-image")
	if useImage2 {
		channelID = "image2-service"
		if data.Model == "" {
			data.Model = "gpt-image-2"
		}
	} else if data.Model == "" {
		data.Model = "nano-banana"
	}

	// 根据模型选择不同的算力值进行预检查
	requiredPower := func() int {
		// 使用统一的 MjPower 配置
		return int(h.App.SysConfig.MjPower)
	}()
	if !h.preCheckPower(c, requiredPower) {
		return
	}

	// 任务参数（nano / gpt-image-2 使用 JSON）
	type nanoParams struct {
		AspectRatio string `json:"aspect_ratio,omitempty"`
		Mode        string `json:"mode,omitempty"`
		Model       string `json:"model,omitempty"`
		Resolution  string `json:"resolution,omitempty"`
		Quality     string `json:"quality,omitempty"`
		Size        string `json:"size,omitempty"`
	}
	opts := nanoParams{
		AspectRatio: data.Rate,
		Mode:        data.NanoMode,
		Model:       data.Model,
		Resolution:  "2K",
		Quality:     "medium",
	}
	if useImage2 {
		opts.Resolution = ""
		opts.Quality = "low"
		if data.NanoMode == "img2img" {
			opts.Quality = "auto"
		}
	}
	paramsBytes, err := json.Marshal(opts)
	if err != nil {
		resp.ERROR(c, "参数序列化失败："+err.Error())
		return
	}
	params := string(paramsBytes)

	// 保留相对路径交给下游（Apilio）用 API KEY 的 public_base 拼接公网域名；
	// 已是 http(s) 的保持不变。纠正误拼 /static/static/。
	for k, v := range data.ImgArr {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		for strings.Contains(v, "/static/static/") {
			v = strings.ReplaceAll(v, "/static/static/", "/static/")
		}
		data.ImgArr[k] = v
	}

	idValue, _ := c.Get(types.LoginUserID)
	userId := utils.IntValue(utils.InterfaceToString(idValue), 0)
	// generate task id
	taskId, err := h.snowflake.Next(true)
	if err != nil {
		resp.ERROR(c, "error with generate task id: "+err.Error())
		return
	}
	// 创建任务参数
	task := types.MjTask{
		TaskId:    taskId,
		Type:      types.TaskType(data.TaskType),
		Prompt:    data.Prompt,
		NegPrompt: data.NegPrompt,
		Params:    params,
		UserId:    userId,
		ImgArr:    data.ImgArr,
		ChannelId: channelID,
	}

	job := model.MidJourneyJob{
		Type:      data.TaskType,
		UserId:    userId,
		TaskId:    taskId,
		Progress:  0,
		Prompt:    data.Prompt,
		Power:     h.App.SysConfig.MjPower,
		CreatedAt: time.Now(),
	}
	opt := "绘图"

	if res := h.DB.Create(&job); res.Error != nil || res.RowsAffected == 0 {
		resp.ERROR(c, "添加任务失败："+res.Error.Error())
		return
	}

	task.Id = job.Id
	h.pool.PushTask(task)

	client := h.pool.Clients.Get(uint(job.UserId))
	if client != nil {
		_ = client.Send([]byte("Task Updated"))
	}

	// update user's power
	err = h.userService.DecreasePower(job.UserId, int(job.Power), model.PowerLog{
		Type:   types.PowerConsume,
		Model:  mj.ResolveImagePowerLogModel(data.Prompt, data.Model),
		Remark: fmt.Sprintf("%s操作，任务ID：%s", opt, job.TaskId),
	})
	if err != nil {
		h.DB.Delete(&model.MidJourneyJob{}, job.Id)
		resp.ERROR(c, err.Error())
		return
	}
	resp.SUCCESS(c)
}

type reqVo struct {
	Index       int    `json:"index"`
	ChannelId   string `json:"channel_id"`
	MessageId   string `json:"message_id"`
	MessageHash string `json:"message_hash"`
	SessionId   string `json:"session_id"`
	Prompt      string `json:"prompt"`
	ChatId      string `json:"chat_id"`
	RoleId      int    `json:"role_id"`
	Icon        string `json:"icon"`
}

// Upscale send upscale command to MidJourney Bot
func (h *MidJourneyHandler) Upscale(c *gin.Context) {
	var data reqVo
	if err := c.ShouldBindJSON(&data); err != nil || data.SessionId == "" {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	if !h.preCheck(c) {
		return
	}

	idValue, _ := c.Get(types.LoginUserID)
	userId := utils.IntValue(utils.InterfaceToString(idValue), 0)
	taskId, _ := h.snowflake.Next(true)
	job := model.MidJourneyJob{
		Type:        types.TaskUpscale.String(),
		ReferenceId: data.MessageId,
		UserId:      userId,
		TaskId:      taskId,
		Progress:    0,
		Prompt:      data.Prompt,
		Power:       h.App.SysConfig.MjActionPower,
		CreatedAt:   time.Now(),
	}
	if res := h.DB.Create(&job); res.Error != nil || res.RowsAffected == 0 {
		resp.ERROR(c, "添加任务失败："+res.Error.Error())
		return
	}

	h.pool.PushTask(types.MjTask{
		Id:          job.Id,
		SessionId:   data.SessionId,
		Type:        types.TaskUpscale,
		Prompt:      data.Prompt,
		UserId:      userId,
		ChannelId:   data.ChannelId,
		Index:       data.Index,
		MessageId:   data.MessageId,
		MessageHash: data.MessageHash,
	})

	client := h.pool.Clients.Get(uint(job.UserId))
	if client != nil {
		_ = client.Send([]byte("Task Updated"))
	}
	// update user's power
	tx := h.DB.Model(&model.User{}).Where("id = ?", job.UserId).UpdateColumn("power", gorm.Expr("power - ?", job.Power))
	// 记录算力变化日志
	if tx.Error == nil && tx.RowsAffected > 0 {
		user, _ := h.GetLoginUser(c)
		h.DB.Create(&model.PowerLog{
			UserId:    user.Id,
			Username:  user.Username,
			Type:      types.PowerConsume,
			Amount:    job.Power,
			Balance:   user.Power - job.Power,
			Mark:      types.PowerSub,
			Model:     mj.ResolveImagePowerLogModel(data.Prompt, ""),
			Remark:    fmt.Sprintf("Upscale 操作，任务ID：%s", job.TaskId),
			CreatedAt: time.Now(),
		})
	}
	resp.SUCCESS(c)
}

// Variation send variation command to MidJourney Bot
func (h *MidJourneyHandler) Variation(c *gin.Context) {
	var data reqVo
	if err := c.ShouldBindJSON(&data); err != nil || data.SessionId == "" {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	if !h.preCheck(c) {
		return
	}

	idValue, _ := c.Get(types.LoginUserID)
	userId := utils.IntValue(utils.InterfaceToString(idValue), 0)
	taskId, _ := h.snowflake.Next(true)
	job := model.MidJourneyJob{
		Type:        types.TaskVariation.String(),
		ChannelId:   data.ChannelId,
		ReferenceId: data.MessageId,
		UserId:      userId,
		TaskId:      taskId,
		Progress:    0,
		Prompt:      data.Prompt,
		Power:       h.App.SysConfig.MjActionPower,
		CreatedAt:   time.Now(),
	}
	if res := h.DB.Create(&job); res.Error != nil || res.RowsAffected == 0 {
		resp.ERROR(c, "添加任务失败："+res.Error.Error())
		return
	}

	h.pool.PushTask(types.MjTask{
		Id:          job.Id,
		SessionId:   data.SessionId,
		Type:        types.TaskVariation,
		Prompt:      data.Prompt,
		UserId:      userId,
		Index:       data.Index,
		ChannelId:   data.ChannelId,
		MessageId:   data.MessageId,
		MessageHash: data.MessageHash,
	})

	client := h.pool.Clients.Get(uint(job.UserId))
	if client != nil {
		_ = client.Send([]byte("Task Updated"))
	}

	// update user's power
	tx := h.DB.Model(&model.User{}).Where("id = ?", job.UserId).UpdateColumn("power", gorm.Expr("power - ?", job.Power))
	// 记录算力变化日志
	if tx.Error == nil && tx.RowsAffected > 0 {
		user, _ := h.GetLoginUser(c)
		h.DB.Create(&model.PowerLog{
			UserId:    user.Id,
			Username:  user.Username,
			Type:      types.PowerConsume,
			Amount:    job.Power,
			Balance:   user.Power - job.Power,
			Mark:      types.PowerSub,
			Model:     mj.ResolveImagePowerLogModel(data.Prompt, ""),
			Remark:    fmt.Sprintf("Variation 操作，任务ID：%s", job.TaskId),
			CreatedAt: time.Now(),
		})
	}
	resp.SUCCESS(c)
}

// ImgWall 照片墙
func (h *MidJourneyHandler) ImgWall(c *gin.Context) {
	page := h.GetInt(c, "page", 0)
	pageSize := h.GetInt(c, "page_size", 0)
	err, jobs := h.getData(true, 0, page, pageSize, true, "image")
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	resp.SUCCESS(c, jobs)
}

// JobList 获取 MJ 任务列表
func (h *MidJourneyHandler) JobList(c *gin.Context) {
	// 支持 finish 和 status 两个参数名
	finish := h.GetBool(c, "finish") || h.GetBool(c, "status")
	userId := h.GetLoginUserId(c)
	page := h.GetInt(c, "page", 0)
	pageSize := h.GetInt(c, "page_size", 0)
	publish := h.GetBool(c, "publish")
	appSource := h.GetTrim(c, "app_source")

	err, jobs := h.getData(finish, userId, page, pageSize, publish, appSource)
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	resp.SUCCESS(c, jobs)
}

// JobList 获取 MJ 任务列表
func (h *MidJourneyHandler) getData(finish bool, userId uint, page int, pageSize int, publish bool, appSource string) (error, []vo.MidJourneyJob) {
	session := h.DB.Session(&gorm.Session{})
	if finish {
		session = session.Where("progress >= ?", 100).Order("id DESC")
	} else {
		session = session.Where("progress < ?", 100).Order("id ASC")
	}
	if userId > 0 {
		session = session.Where("user_id = ?", userId)
	}
	if publish {
		session = session.Where("publish = ?", publish)
	}
	session = applyMjAppSourceFilter(session, appSource)
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		session = session.Offset(offset).Limit(pageSize)
	}

	var items []model.MidJourneyJob
	res := session.Find(&items)
	if res.Error != nil {
		return res.Error, nil
	}

	var jobs = make([]vo.MidJourneyJob, 0)
	for _, item := range items {
		var job vo.MidJourneyJob
		err := utils.CopyObject(item, &job)
		if err != nil {
			continue
		}

		if item.Progress < 100 && item.ImgURL == "" && item.OrgURL != "" {
			// discord 服务器图片需要使用代理转发图片数据流
			if strings.HasPrefix(item.OrgURL, "https://cdn.discordapp.com") {
				image, err := utils.DownloadImage(item.OrgURL, h.App.Config.ProxyURL)
				if err == nil {
					job.ImgURL = "data:image/png;base64," + base64.StdEncoding.EncodeToString(image)
				}
			} else {
				job.ImgURL = job.OrgURL
			}
		}

		jobs = append(jobs, job)
	}
	return nil, jobs
}

// Remove remove task image
func (h *MidJourneyHandler) Remove(c *gin.Context) {
	var data struct {
		Id     uint   `json:"id"`
		UserId uint   `json:"user_id"`
		ImgURL string `json:"img_url"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	// remove job recode
	res := h.DB.Delete(&model.MidJourneyJob{Id: data.Id})
	if res.Error != nil {
		resp.ERROR(c, res.Error.Error())
		return
	}

	// remove image
	err := h.uploader.GetUploadHandler().Delete(data.ImgURL)
	if err != nil {
		logger.Error("remove image failed: ", err)
	}

	client := h.pool.Clients.Get(data.UserId)
	if client != nil {
		_ = client.Send([]byte("Task Updated"))
	}

	resp.SUCCESS(c)
}

// Publish 发布图片到画廊显示
func (h *MidJourneyHandler) Publish(c *gin.Context) {
	var data struct {
		Id     uint `json:"id"`
		Action bool `json:"action"` // 发布动作，true => 发布，false => 取消分享
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	res := h.DB.Model(&model.MidJourneyJob{Id: data.Id}).UpdateColumn("publish", data.Action)
	if res.Error != nil {
		resp.ERROR(c, "更新数据库失败")
		return
	}

	resp.SUCCESS(c)
}
