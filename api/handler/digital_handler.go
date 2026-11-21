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
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	"log"
	"net/http"
	"time"
)

type DigitalHandler struct {
	BaseHandler
	pool      *mj.DigiServicePool
	snowflake *service.Snowflake
	uploader  *oss.UploaderManager
}

//
//func NewDigitalHandler(app *core.AppServer, db *gorm.DB) *DigitalHandler {
//	return &DigitalHandler{BaseHandler: BaseHandler{App: app, DB: db}}
//}

func NewDigitalHandler(app *core.AppServer, db *gorm.DB, snowflake *service.Snowflake, pool *mj.DigiServicePool, manager *oss.UploaderManager) *DigitalHandler {
	return &DigitalHandler{
		snowflake: snowflake,
		pool:      pool,
		uploader:  manager,
		BaseHandler: BaseHandler{
			App: app,
			DB:  db,
		},
	}
}

// 定义一个结构体，映射JSON数据的字段
type DigitalTaskRequest struct {
	BgUrl     string `json:"bg_url"`
	Content   string `json:"content"`
	Language  string `json:"language"`
	RoleName  string `json:"roleName"`
	RoleStyle string `json:"character"`
}

// JobList 获取 数字人 任务列表
func (h *DigitalHandler) JobList(c *gin.Context) {
	status := h.GetBool(c, "status")
	userId := h.GetLoginUserId(c)
	page := h.GetInt(c, "page", 0)
	pageSize := h.GetInt(c, "page_size", 0)
	//publish := h.GetBool(c, "publish")

	//	logger.Info(status)

	err, jobs := h.getData(status, userId, page, pageSize)
	//logger.Info(jobs)
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	resp.SUCCESS(c, jobs)
}

// JobList 获取数字人列表
func (h *DigitalHandler) getData(finish bool, userId uint, page int, pageSize int) (error, []vo.DigiJob) {
	session := h.DB.Session(&gorm.Session{})
	if finish {
		session = session.Where("progress = ?", 100).Order("id DESC")
	} else {
		session = session.Where("progress < ?", 100).Order("id ASC")
	}
	if userId > 0 {
		session = session.Where("user_id = ?", userId)
	}

	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		session = session.Offset(offset).Limit(pageSize)
	}

	var items []model.DigiJob
	res := session.Find(&items)
	if res.Error != nil {
		return res.Error, nil
	}

	var jobs = make([]vo.DigiJob, 0)
	for _, item := range items {
		var job vo.DigiJob
		//item.CreatedAt=job.CreatedAt;

		err := utils.CopyObject(item, &job)
		if err != nil {
			continue
		}

		//if item.Progress < 100 && item.ImgURL == "" && item.OrgURL != "" {
		//	// discord 服务器图片需要使用代理转发图片数据流
		//	if strings.HasPrefix(item.OrgURL, "https://cdn.discordapp.com") {
		//		image, err := utils.DownloadImage(item.OrgURL, h.App.Config.ProxyURL)
		//		if err == nil {
		//			job.ImgURL = "data:image/png;base64," + base64.StdEncoding.EncodeToString(image)
		//		}
		//	} else {
		//		job.ImgURL = job.OrgURL
		//	}
		//}

		jobs = append(jobs, job)
	}
	return nil, jobs
}

// 创建一个数字人任务
func (h *DigitalHandler) Digi(c *gin.Context) {

	var data struct {
		SessionId string `json:"session_id"`
		BgUrl     string `json:"background"` //背景
		Content   string `json:"content"`    //文案
		Language  string `json:"language"`   //语言
		RoleName  string `json:"role_name"`  //角色
		RoleStyle string `json:"character"`  //角色形象
	}
	//检查data是否为json格式
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	if !h.preCheck(c) {
		return
	}

	idValue, _ := c.Get(types.LoginUserID)
	userId := utils.IntValue(utils.InterfaceToString(idValue), 0)
	// generate task id
	taskId, err := h.snowflake.Next(true)
	if err != nil {
		resp.ERROR(c, "error with generate task id: "+err.Error())
		return
	}

	// 形象图
	var preUrl string

	//logger.Info("背景为", data.RoleStyle)

	if data.RoleStyle == "casual-sitting" {
		preUrl = "https://www.ifaceai.com/img/ai/role/video/lisa-casual-sitting.png"
	} else if data.RoleStyle == "graceful-sitting" {
		preUrl = "https://www.ifaceai.com/img/ai/role/video/lisa-graceful-sitting.png"
	} else if data.RoleStyle == "graceful-standing" {
		preUrl = "https://www.ifaceai.com/img/ai/role/video/lisa-graceful-standing.png"
	} else if data.RoleStyle == "business" {
		preUrl = "https://www.ifaceai.com/img/ai/role/video/harry-business-transparent.png"
	} else if data.RoleStyle == "casual" {
		preUrl = "https://www.ifaceai.com/img/ai/role/video/harry-casual-transparent.png"
	} else if data.RoleStyle == "youthful" {
		preUrl = "https://www.ifaceai.com/img/ai/role/video/harry-youthful-transparent.png"
	}

	job := model.DigiJob{
		UserId:    userId,
		TaskId:    taskId,
		RoleName:  data.RoleName,
		RoleStyle: data.RoleStyle,
		Language:  data.Language,
		BgUrl:     data.BgUrl,
		PreUrl:    preUrl,
		ChannelId: "digi_service-0",
		Progress:  0,
		Content:   fmt.Sprintf("%s", data.Content),
		//	Power:     h.App.SysConfig.DigiPower, //后期修改为
		Power:     100, //后期修改为
		CreatedAt: time.Now(),
	}
	if res := h.DB.Create(&job); res.Error != nil || res.RowsAffected == 0 {
		resp.ERROR(c, "添加任务失败："+res.Error.Error())
		return
	}
	//推送job

	h.pool.PushDigiTask(types.DigitalTask{
		Id:        job.Id,
		TaskId:    taskId,
		SessionId: data.SessionId,
		//Type:      types.TaskType(data.TaskType),
		Content:   data.Content,
		BgUrl:     data.BgUrl,
		Language:  data.Language,
		RoleName:  data.RoleName,
		RoleStyle: data.RoleStyle,
		UserId:    userId,
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
			Model:     "数字人",
			Remark:    fmt.Sprintf("%s操作，任务ID：%s", "数字人", job.TaskId),
			CreatedAt: time.Now(),
		})
	}
	resp.SUCCESS(c)
}

// 获取当前用户的所有数字人视频
func (h *DigitalHandler) List(c *gin.Context) {
	var data struct {
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	userId := h.GetLoginUserId(c)
	session := h.DB.Session(&gorm.Session{}).Where("user_id = ? AND status = ?", userId, types.OrderPaidSuccess)
	var total int64
	session.Model(&model.Order{}).Count(&total)
	var items []model.Order
	var list = make([]vo.Order, 0)
	offset := (data.Page - 1) * data.PageSize
	res := session.Order("id DESC").Offset(offset).Limit(data.PageSize).Find(&items)
	if res.Error == nil {
		for _, item := range items {
			var order vo.Order
			err := utils.CopyObject(item, &order)
			if err == nil {
				order.Id = item.Id
				order.CreatedAt = item.CreatedAt.Unix()
				order.UpdatedAt = item.UpdatedAt.Unix()
				list = append(list, order)
			} else {
				logger.Error(err)
			}
		}
	}
	resp.SUCCESS(c, vo.NewPage(total, data.Page, data.PageSize, list))
}

func (h *DigitalHandler) StartTask(c *gin.Context) {
	var request DigitalTaskRequest

	// 解析JSON数据到结构体
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Println("解析JSON数据失败:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON数据解析失败"})
		return
	}

	// 打印接收到的参数
	logger.Info("收到的参数为", request.BgUrl)
	logger.Info("收到的参数为", request.Content)
	logger.Info("收到的参数为", request.Language)
	logger.Info("收到的参数为", request.RoleStyle)

	//// 调用服务处理数据
	//var digiService = mj.DigiClient{}
	//body, err := digiService.StartTask(request.BackgroundImageUrl, request.Content, request.Language, request.RoleName, request.RoleStyle)
	//if err != nil {
	//	log.Println("处理任务失败:", err)
	//	c.JSON(http.StatusInternalServerError, gin.H{"error": "处理任务失败"})
	//	return
	//}
	//
	//// 返回处理结果
	//logger.Info(body)
	resp.SUCCESS(c, nil)
}

// Client WebSocket 客户端，用于通知任务状态变更
func (h *DigitalHandler) Client(c *gin.Context) {
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

func (h *DigitalHandler) preCheck(c *gin.Context) bool {
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return false
	}

	if user.Power < 100 {
		resp.ERROR(c, "当前用户剩余算力不足以完成数字人任务！")
		return false
	}

	if !h.pool.HasAvailableService() {
		resp.ERROR(c, "数字人 池子中没有没有可用的服务！")
		return false
	}

	return true

}
