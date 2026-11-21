package admin

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/handler"
	"chatplus/service"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils"
	"chatplus/utils/resp"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserHandler struct {
	handler.BaseHandler
}

func NewUserHandler(app *core.AppServer, db *gorm.DB) *UserHandler {
	return &UserHandler{BaseHandler: handler.BaseHandler{App: app, DB: db}}
}

// List 用户列表
func (h *UserHandler) List(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		resp.NotPermission(c)
		return
	}

	page := h.GetInt(c, "page", 1)
	pageSize := h.GetInt(c, "page_size", 20)
	username := h.GetTrim(c, "username")

	offset := (page - 1) * pageSize
	var items []model.User
	var users = make([]vo.User, 0)
	var total int64

	session := h.DB.Session(&gorm.Session{})
	if username != "" {
		session = session.Where("username LIKE ?", "%"+username+"%")
	}

	//增加邀请码到前端页面中去 //chatgpt_invite_codes

	session.Model(&model.User{}).Count(&total)
	res := session.Offset(offset).Limit(pageSize).Find(&items)
	if res.Error == nil {
		for _, item := range items {
			var user vo.User
			err := utils.CopyObject(item, &user)
			var inviteCode model.InviteCode
			h.DB.Where("user_id = ?", item.Id).First(&inviteCode)
			if err == nil {
				user.Id = item.Id
				user.CreatedAt = item.CreatedAt.Unix()
				user.UpdatedAt = item.UpdatedAt.Unix()
				user.Code = inviteCode.Code
				users = append(users, user)
			} else {
				logger.Error(err)
			}
		}
	}
	pageVo := vo.NewPage(total, page, pageSize, users)
	resp.SUCCESS(c, pageVo)
}

func (h *UserHandler) Save(c *gin.Context) {
	var data struct {
		Id          uint     `json:"id"`
		Password    string   `json:"password"`
		Username    string   `json:"username"`
		ChatRoles   []string `json:"chat_roles"`
		ChatModels  []int    `json:"chat_models"`
		ExpiredTime string   `json:"expired_time"`
		Status      bool     `json:"status"`
		AllowTrade  bool     `json:"allow_trade"`
	AllowQ2     bool     `json:"allow_q2"`
	AllowMint   bool     `json:"allow_mint"`
	AllowPlacement bool  `json:"allow_placement"`
	Vip         bool     `json:"vip"`
		Power       float64  `json:"power"`
		IsAuditor   bool     `json:"is_auditor"`
		IsCs        bool     `json:"is_cs"`
		Lev         int64    `json:"lev"`
		Floor       int64    `json:"floor"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	var user = model.User{}
	var res *gorm.DB
	var userVo vo.User
	if data.Id > 0 { // 更新
		res = h.DB.Where("id", data.Id).First(&user)
		if res.Error != nil {
			resp.ERROR(c, "user not found")
			return
		}
		var oldPower = user.Power
		oldLev := user.Lev
		user.Username = data.Username
		user.Status = data.Status
		user.Vip = data.Vip
		user.AllowTrade = data.AllowTrade
		user.AllowQ2 = data.AllowQ2
		user.AllowMint = data.AllowMint
		user.AllowPlacement = data.AllowPlacement
		user.Power = data.Power
		user.IsAuditor = data.IsAuditor
		user.IsCs = data.IsCs
		user.Lev = data.Lev
		user.Floor = data.Floor
		if oldLev != data.Lev {
			t := time.Now()
			user.LevSinceAt = &t
		}
		user.ChatRoles = utils.JsonEncode(data.ChatRoles)
		user.ChatModels = utils.JsonEncode(data.ChatModels)
		user.ExpiredTime = utils.Str2stamp(data.ExpiredTime)

		res = h.DB.Select("username", "status", "vip", "allow_trade", "allow_q2", "allow_mint", "allow_placement", "power", "is_auditor", "is_cs", "lev", "floor", "lev_since_at", "chat_roles_json", "chat_models_json", "expired_time").Updates(&user)
		if res.Error != nil {
			resp.ERROR(c, "更新数据库失败！")
			return
		}
		// 记录算力日志
		if oldPower != user.Power {
			mark := types.PowerAdd
			amount := user.Power - oldPower
			if oldPower > user.Power {
				mark = types.PowerSub
				amount = oldPower - user.Power
			}
			h.DB.Create(&model.PowerLog{
				UserId:    user.Id,
				Username:  user.Username,
				Type:      types.PowerGift,
				Amount:    float64(amount),
				Balance:   float64(user.Power),
				Mark:      mark,
				Model:     "管理员",
				Remark:    fmt.Sprintf("后台管理员强制修改用户算力，修改前：%.2f,修改后:%.2f, 管理员ID：%d", oldPower, user.Power, h.GetLoginUserId(c)),
				CreatedAt: time.Now(),
			})
		}
	} else {
		salt := utils.RandString(8)
		u := model.User{
			Username:    data.Username,
			Nickname:    fmt.Sprintf("gptplus@%d", utils.RandomNumber(6)),
			Password:    utils.GenPassword(data.Password, salt),
			Avatar:      "/images/avatar/user.png",
			Salt:        salt,
			Power:       data.Power,
			AgentNum:    1, //默认一代
			Status:      true,
			Vip:         data.Vip,
			AllowTrade:  data.AllowTrade,
			AllowQ2:     data.AllowQ2,
			AllowMint:   data.AllowMint,
			AllowPlacement: data.AllowPlacement,
			IsAuditor:   data.IsAuditor,
			IsCs:        data.IsCs,
			ChatRoles:   utils.JsonEncode(data.ChatRoles),
			ChatModels:  utils.JsonEncode(data.ChatModels),
			ExpiredTime: utils.Str2stamp(data.ExpiredTime),
		}
		res = h.DB.Create(&u)
		_ = utils.CopyObject(u, &userVo)
		userVo.Id = u.Id
		userVo.CreatedAt = u.CreatedAt.Unix()
		userVo.UpdatedAt = u.UpdatedAt.Unix()
	}

	if res.Error != nil {
		resp.ERROR(c, "更新数据库失败")
		return
	}

	resp.SUCCESS(c, userVo)
}

// ResetPass 重置密码
func (h *UserHandler) ResetPass(c *gin.Context) {
	var data struct {
		Id       uint
		Password string
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	var user model.User
	res := h.DB.First(&user, data.Id)
	if res.Error != nil {
		resp.ERROR(c, "No user found")
		return
	}

	password := utils.GenPassword(data.Password, user.Salt)
	user.Password = password
	res = h.DB.Updates(&user)
	if res.Error != nil {
		resp.ERROR(c)
	} else {
		resp.SUCCESS(c)
	}
}

func (h *UserHandler) Remove(c *gin.Context) {
	id := h.GetInt(c, "id", 0)
	if id <= 0 {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	// 删除用户
	res := h.DB.Where("id = ?", id).Delete(&model.User{})
	if res.Error != nil {
		resp.ERROR(c, "删除失败")
		return
	}

	// 删除聊天记录
	h.DB.Where("user_id = ?", id).Delete(&model.ChatItem{})
	// 删除聊天历史记录
	h.DB.Where("user_id = ?", id).Delete(&model.ChatMessage{})
	// 删除登录日志
	h.DB.Where("user_id = ?", id).Delete(&model.UserLoginLog{})
	// 删除算力日志
	h.DB.Where("user_id = ?", id).Delete(&model.PowerLog{})
	// 删除众筹日志
	h.DB.Where("user_id = ?", id).Delete(&model.Reward{})
	// 删除绘图任务
	h.DB.Where("user_id = ?", id).Delete(&model.MidJourneyJob{})
	h.DB.Where("user_id = ?", id).Delete(&model.SdJob{})
	//  删除订单
	h.DB.Where("user_id = ?", id).Delete(&model.Order{})
	resp.SUCCESS(c)
}

func (h *UserHandler) LoginLog(c *gin.Context) {
	page := h.GetInt(c, "page", 1)
	pageSize := h.GetInt(c, "page_size", 20)
	var total int64
	h.DB.Model(&model.UserLoginLog{}).Count(&total)
	offset := (page - 1) * pageSize
	var items []model.UserLoginLog
	res := h.DB.Offset(offset).Limit(pageSize).Order("id DESC").Find(&items)
	if res.Error != nil {
		resp.ERROR(c, "获取数据失败")
		return
	}
	var logs []vo.UserLoginLog
	for _, v := range items {
		var log vo.UserLoginLog
		err := utils.CopyObject(v, &log)
		if err == nil {
			log.Id = v.Id
			log.CreatedAt = v.CreatedAt.Unix()
			logs = append(logs, log)
		}
	}

	resp.SUCCESS(c, vo.NewPage(total, page, pageSize, logs))
}

// SetAuditor 批量设置审核用户
func (h *UserHandler) SetAuditor(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		resp.NotPermission(c)
		return
	}

	var data struct {
		Ids []uint `json:"ids"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	if len(data.Ids) == 0 {
		resp.ERROR(c, "请选择要设置的用户")
		return
	}

	// 批量更新用户为审核用户
	res := h.DB.Model(&model.User{}).Where("id IN ?", data.Ids).Update("is_auditor", true)
	if res.Error != nil {
		resp.ERROR(c, "设置失败："+res.Error.Error())
		return
	}

	resp.SUCCESS(c, fmt.Sprintf("成功设置 %d 个用户为审核用户", res.RowsAffected))
}

// SetCs 批量设置/取消客服
func (h *UserHandler) SetCs(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		resp.NotPermission(c)
		return
	}

	var data struct {
		Ids   []uint `json:"ids"`
		Allow bool   `json:"allow"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	if len(data.Ids) == 0 {
		resp.ERROR(c, "请选择要设置的用户")
		return
	}

	res := h.DB.Model(&model.User{}).Where("id IN ?", data.Ids).Update("is_cs", data.Allow)
	if res.Error != nil {
		resp.ERROR(c, "设置失败："+res.Error.Error())
		return
	}
	action := "设为客服"
	if !data.Allow {
		action = "取消客服"
	}
	resp.SUCCESS(c, fmt.Sprintf("成功%s %d 个用户", action, res.RowsAffected))
}

func (h *UserHandler) SetAllowMint(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		resp.NotPermission(c)
		return
	}

	var data struct {
		Ids   []uint `json:"ids"`
		Allow bool   `json:"allow"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	if len(data.Ids) == 0 {
		resp.ERROR(c, "请选择要设置的用户")
		return
	}

	res := h.DB.Model(&model.User{}).Where("id IN ?", data.Ids).Update("allow_mint", data.Allow)
	if res.Error != nil {
		resp.ERROR(c, "设置失败："+res.Error.Error())
		return
	}

	action := "关闭"
	if data.Allow {
		action = "开通"
	}
	resp.SUCCESS(c, fmt.Sprintf("成功%s %d 个用户的算力机抢购权限", action, res.RowsAffected))
}

// SetAllowPlacement 批量开通/关闭大小区节点显示
func (h *UserHandler) SetAllowPlacement(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		resp.NotPermission(c)
		return
	}

	var data struct {
		Ids   []uint `json:"ids"`
		Allow bool   `json:"allow"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	if len(data.Ids) == 0 {
		resp.ERROR(c, "请选择要设置的用户")
		return
	}

	res := h.DB.Model(&model.User{}).Where("id IN ?", data.Ids).Update("allow_placement", data.Allow)
	if res.Error != nil {
		resp.ERROR(c, "设置失败："+res.Error.Error())
		return
	}

	action := "关闭"
	if data.Allow {
		action = "开通"
	}
	resp.SUCCESS(c, fmt.Sprintf("成功%s %d 个用户的大小区显示", action, res.RowsAffected))
}

// PreviewMusicRefundRollback 预览音乐失败重复退款回滚
func (h *UserHandler) PreviewMusicRefundRollback(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		resp.NotPermission(c)
		return
	}
	userId := uint(h.GetInt(c, "user_id", 0))
	if userId == 0 {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	preview, _, err := service.FindMusicRefundDuplicates(h.DB, userId)
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}
	resp.SUCCESS(c, preview)
}

// RollbackMusicRefundExploit 回滚音乐失败重复退款（薅羊毛）
func (h *UserHandler) RollbackMusicRefundExploit(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		resp.NotPermission(c)
		return
	}
	var data struct {
		UserId uint `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil || data.UserId == 0 {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	preview, err := service.RollbackMusicRefundDuplicates(h.DB, data.UserId, h.GetLoginUserId(c))
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}
	resp.SUCCESS(c, preview)
}

// PreviewPowerPointRollback 预览算力时间点回滚
func (h *UserHandler) PreviewPowerPointRollback(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		resp.NotPermission(c)
		return
	}
	userId := uint(h.GetInt(c, "user_id", 0))
	rollbackAtRaw := c.Query("rollback_at")
	if userId == 0 {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	rollbackAt, err := service.ParseRollbackTime(rollbackAtRaw)
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}
	preview, err := service.PreviewPowerPointRollback(h.DB, userId, rollbackAt)
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}
	resp.SUCCESS(c, preview)
}

// ExecutePowerPointRollback 执行算力时间点回滚（软隐藏流水并重算余额）
func (h *UserHandler) ExecutePowerPointRollback(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		resp.NotPermission(c)
		return
	}
	var data struct {
		UserId     uint   `json:"user_id"`
		RollbackAt string `json:"rollback_at"`
	}
	if err := c.ShouldBindJSON(&data); err != nil || data.UserId == 0 {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	rollbackAt, err := service.ParseRollbackTime(data.RollbackAt)
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}
	preview, err := service.ExecutePowerPointRollback(h.DB, data.UserId, rollbackAt, h.GetLoginUserId(c))
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}
	resp.SUCCESS(c, preview)
}
