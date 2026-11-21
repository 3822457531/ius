package handler

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/service"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils"
	"chatplus/utils/resp"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"

	"github.com/gin-gonic/gin"
	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"gorm.io/gorm"
)

type UserHandler struct {
	BaseHandler
	searcher      *xdb.Searcher
	redis         *redis.Client
	userService   *service.UserService
	creditService *service.CreditService
}

type imageHubCustomApp struct {
	Id       string `json:"id"`
	Label    string `json:"label"`
	Url      string `json:"url"`
	OpenMode string `json:"openMode"`
	Icon     string `json:"icon"`
}

func NewUserHandler(
	app *core.AppServer,
	db *gorm.DB,
	searcher *xdb.Searcher,
	client *redis.Client,
	userService *service.UserService,
	creditService *service.CreditService) *UserHandler {
	return &UserHandler{
		BaseHandler:   BaseHandler{DB: db, App: app},
		searcher:      searcher,
		redis:         client,
		userService:   userService,
		creditService: creditService,
	}
}

// PptConsume 扣减用户算力（供 PPT 子系统调用）
func (h *UserHandler) PptConsume(c *gin.Context) {
	var data struct {
		Amount    int    `json:"amount"`
		ProjectId string `json:"project_id"`
		Action    string `json:"action"`
		Remark    string `json:"remark"`
		Settled   bool   `json:"settled"` // true：算力已确定消耗，不会退回（才触发团队分红）
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	if data.Amount <= 0 {
		resp.ERROR(c, "amount must be greater than 0")
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}
	if user.Power < float64(data.Amount) {
		resp.ERROR(c, "当前用户剩余算力不足")
		return
	}

	action := strings.TrimSpace(data.Action)
	if action == "" {
		action = "ppt_action"
	}
	remark := strings.TrimSpace(data.Remark)
	if remark == "" {
		remark = fmt.Sprintf("PPT 算力扣减，project_id=%s, action=%s", data.ProjectId, action)
	}

	err = h.userService.DecreasePower(int(user.Id), data.Amount, model.PowerLog{
		Type:   types.PowerConsume,
		Model:  service.PowerLogModelPPT,
		Remark: remark,
	})
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	if data.Settled {
		service.NotifyAppPlazaPowerConsumed(
			h.DB,
			user.Id,
			float64(data.Amount),
			service.PowerLogModelPPT,
			fmt.Sprintf("ppt:%d:%s:%s:%d", user.Id, data.ProjectId, action, time.Now().UnixNano()),
		)
	}

	resp.SUCCESS(c, gin.H{
		"user_id":    user.Id,
		"amount":     data.Amount,
		"action":     action,
		"project_id": data.ProjectId,
	})
}

// PptRefund 退还用户算力（供 PPT 子系统在图片生成失败时调用；幂等，同一 request_id 仅退一次）
func (h *UserHandler) PptRefund(c *gin.Context) {
	var data struct {
		Amount    int    `json:"amount"`
		ProjectId string `json:"project_id"`
		Action    string `json:"action"`
		Remark    string `json:"remark"`
		RequestId string `json:"request_id"` // 幂等键，建议 PPT 侧每次失败退款的唯一 ID
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	if data.Amount <= 0 {
		resp.ERROR(c, "amount must be greater than 0")
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	action := strings.TrimSpace(data.Action)
	if action == "" {
		action = "ppt_refund"
	}
	remark := strings.TrimSpace(data.Remark)
	if remark == "" {
		remark = fmt.Sprintf("PPT 算力退还，project_id=%s, action=%s", data.ProjectId, action)
	}

	refundKey := strings.TrimSpace(data.RequestId)
	if refundKey == "" {
		refundKey = fmt.Sprintf("%s:%s", strings.TrimSpace(data.ProjectId), action)
	}

	alreadyRefunded, err := service.RefundPowerIdempotent(
		h.DB,
		user.Id,
		data.Amount,
		"ppt",
		refundKey,
		service.PowerLogModelPPT,
		remark,
	)
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	resp.SUCCESS(c, gin.H{
		"user_id":          user.Id,
		"amount":           data.Amount,
		"action":           action,
		"project_id":       data.ProjectId,
		"request_id":       refundKey,
		"already_refunded": alreadyRefunded,
	})
}

// PptAiSettings 返回管理后台 key=ppt 的 PPT 大模型配置（需登录），供 Banana 子系统在 IUS 模式下覆盖本地 SQLite settings
func (h *UserHandler) PptAiSettings(c *gin.Context) {
	if _, err := h.GetLoginUser(c); err != nil {
		resp.NotAuth(c)
		return
	}
	var cfg model.Config
	res := h.DB.Where("marker", "ppt").First(&cfg)
	if res.Error != nil {
		resp.SUCCESS(c, gin.H{})
		return
	}
	var value map[string]interface{}
	if err := utils.JsonDecode(cfg.Config, &value); err != nil {
		resp.ERROR(c, err.Error())
		return
	}
	resp.SUCCESS(c, value)
}

// Register user register
func (h *UserHandler) Register(c *gin.Context) {
	// parameters process

	var data struct {
		RegWay     string `json:"reg_way"`
		Username   string `json:"username"`
		Mail       string `json:"mail"`
		Password   string `json:"password"`
		Code       string `json:"code"`
		InviteCode string `json:"invite_code"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	if !h.App.SysConfig.EnabledRegister {
		resp.ERROR(c, "系统暂未开放注册")
		return
	}
	data.Username = strings.TrimSpace(data.Username)
	data.Mail = strings.TrimSpace(data.Mail)
	data.InviteCode = strings.TrimSpace(data.InviteCode)
	data.Code = strings.TrimSpace(data.Code)
	data.Password = strings.TrimSpace(data.Password)
	if data.Username == "" || data.Mail == "" {
		resp.ERROR(c, "请填写完整信息")
		return
	}
	if len(data.Password) < 8 {
		resp.ERROR(c, "密码长度不能少于8个字符")
		return
	}

	// 邀请码强制必填（前端校验可绕过，必须以服务端为准）
	if data.InviteCode == "" {
		resp.ERROR(c, "请填写邀请码")
		return
	}
	inviteCode := model.InviteCode{}
	resInvite := h.DB.Where("code = ?", data.InviteCode).First(&inviteCode)
	if resInvite.Error != nil || inviteCode.Id == 0 {
		resp.ERROR(c, "无效的邀请码")
		return
	}

	// 仅在邮箱/手机注册渠道，或请求显式带了 code 时校验验证码
	// （当前注册页未接验证码输入时，不因 RegisterWays 配置误伤正常注册）
	var key string
	if data.RegWay == "email" || data.RegWay == "mobile" || data.Code != "" {
		key = CodeStorePrefix + data.Username
		code, err := h.redis.Get(c, key).Result()
		if err != nil && data.Mail != "" {
			key = CodeStorePrefix + data.Mail
			code, err = h.redis.Get(c, key).Result()
		}
		if err != nil || data.Code == "" || subtle.ConstantTimeCompare([]byte(code), []byte(data.Code)) != 1 {
			resp.ERROR(c, "验证码错误")
			return
		}
	}

	// check if the username is exists
	var item model.User
	res := h.DB.Where("username = ?", data.Username).First(&item)
	if item.Id > 0 {
		resp.ERROR(c, "该用户名已经被注册")
		return
	}

	var mailItem model.User
	//邮箱
	h.DB.Where("mail = ?", data.Mail).First(&mailItem)
	if mailItem.Id > 0 {
		resp.ERROR(c, "该邮箱已经其他用户绑定")
		return
	}

	// 同 IP 注册数量限制（后台可开关及配置上限）
	if err := service.CheckRegisterIPLimit(h.DB, c.ClientIP(), h.App.SysConfig); err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	salt := utils.RandString(8)
	user := model.User{
		Username:   data.Username,
		Mail:       data.Mail,
		Password:   utils.GenPassword(data.Password, salt),
		Nickname:   fmt.Sprintf("gptplus@%d", utils.RandomNumber(6)),
		Avatar:     "/images/avatar/user.png",
		Salt:       salt,
		Status:     true,
		AllowTrade: true,
		AllowQ2:    true, // 默认允许 SOHA 交易 / 抢购算力机
		AllowMint:  true,
		AllowPlacement: false,
		ChatRoles:  utils.JsonEncode([]string{"gpt"}),               // 默认只订阅通用助手角色
		ChatModels: utils.JsonEncode(h.App.SysConfig.DefaultModels), // 默认开通的模型
		Power:      0,                                               //h.App.SysConfig.InitPower,
		AgentNum:   1,                                               //默认直接给一代
		//记录ip信息
		LoginIp:      c.ClientIP(),
		LoginAddress: utils.Ip2Region(h.searcher, c.ClientIP()),
	}

	res = h.DB.Create(&user)
	if res.Error != nil {
		resp.ERROR(c, "保存数据失败")
		logger.Error(res.Error)
		return
	}

	// 记录邀请关系
	if data.InviteCode != "" {
		// 增加邀请数量
		h.DB.Model(&model.InviteCode{}).Where("code = ?", data.InviteCode).UpdateColumn("reg_num", gorm.Expr("reg_num + ?", 1))

		if h.App.SysConfig.InvitePower >= 0 {

			//邀请赠送去掉

			//logger.Info("进入到邀请赠送环节")
			//logger.Info("算力为", h.App.SysConfig.InvitePower)
			//
			//h.DB.Model(&model.User{}).Where("id = ?", inviteCode.UserId).UpdateColumn("power", gorm.Expr("power + ?", h.App.SysConfig.InvitePower))
			//// 记录邀请算力充值日志
			//var inviter model.User
			//h.DB.Where("id", inviteCode.UserId).First(&inviter)
			//h.DB.Create(&model.PowerLog{
			//	UserId:    inviter.Id,
			//	Username:  inviter.Username,
			//	Type:      types.PowerInvite,
			//	Amount:    h.App.SysConfig.InvitePower,
			//	Balance:   inviter.Power,
			//	Mark:      types.PowerAdd,
			//	Model:     "邀请奖励",
			//	Remark:    fmt.Sprintf("邀请用户注册奖励，金额：%d，邀请码：%s，新用户：%s", h.App.SysConfig.InvitePower, inviteCode.Code, user.Username),
			//	CreatedAt: time.Now(),
			//})

			//记录邀请上下级关系
			//h.DB.Model(&model.InviteCode{}).Where("code = ?", data.InviteCode).UpdateColumn("f_id", inviter.Id)
			//直接上级
			//h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumn("zj_fid", inviteCode.UserId)
			//inviteCode.UserId 直接上级

			//	h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumns(model.User{ZjFid: int(inviteCode.UserId), JjFid: otherUser.ZjFid})

			//数组容器
			//	var ids [10]int
			var ids [20]int

			//直接id
			var zjid = int(inviteCode.UserId)
			for i := 0; i < 10; i++ {

				//间接上级
				var otherUser model.User
				h.DB.Where("id", zjid).First(&otherUser)

				//间接id
				var jjid = otherUser.ZjFid

				logger.Info("直接id", zjid)
				logger.Info("间接id", jjid)
				//if zjid > 0 {
				ids[i] = zjid
				//}
				//
				//	user.ChatModels = utils.JsonEncode(data.ChatModels)
				//h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumns(model.User{ZjFid: int(zjid), JjFid: jjid})
				zjid = jjid

			}

			start := 4
			end := 9
			// 截取数组
			subArray := ids[start : end+1]
			var idsofString = utils.JsonEncode(subArray)

			h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumns(model.User{ZjFid: ids[0], JjFid: ids[1], SanFid: ids[2], SiFid: ids[3], FidsJson: idsofString})

			logger.Info("邀请ids为")

			logger.Info(ids)

			//间接上级
			//h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumn("jj_fid", otherUser.ZjFid)

		}

		// 添加邀请记录
		h.DB.Create(&model.InviteLog{
			InviterId:  inviteCode.UserId,
			UserId:     user.Id,
			Username:   user.Username,
			InviteCode: inviteCode.Code,
			Remark:     fmt.Sprintf("奖励 %.2f 算力", h.App.SysConfig.InvitePower),
		})
	}

	_ = h.redis.Del(c, key) // 注册成功，删除短信验证码

	// 自动登录创建 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.Id,
		"expired": time.Now().Add(time.Second * time.Duration(h.App.Config.Session.MaxAge)).Unix(),
	})
	tokenString, err := token.SignedString([]byte(h.App.Config.Session.SecretKey))
	if err != nil {
		resp.ERROR(c, "Failed to generate token, "+err.Error())
		return
	}
	// 保存到 redis
	key = fmt.Sprintf("users/%d", user.Id)
	if _, err := h.redis.Set(c, key, tokenString, 0).Result(); err != nil {
		resp.ERROR(c, "error with save token: "+err.Error())
		return
	}
	resp.SUCCESS(c, tokenString)
}

// Login 用户登录
func (h *UserHandler) Login(c *gin.Context) {
	var data struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	var user model.User
	res := h.DB.Where("username = ?", data.Username).First(&user)
	loginFailMsg := "用户名或密码错误"
	if res.Error != nil {
		// 假哈希抹平时序差异，避免用户名枚举
		_ = utils.GenPassword(data.Password, "01234567")
		resp.ERROR(c, loginFailMsg)
		return
	}

	password := utils.GenPassword(data.Password, user.Salt)
	if password != user.Password {
		resp.ERROR(c, loginFailMsg)
		return
	}

	if user.Status == false {
		resp.ERROR(c, "")
		return
	}

	// 更新最后登录时间和IP
	user.LastLoginIp = c.ClientIP()
	user.LastLoginAt = time.Now().Unix()
	h.DB.Model(&user).Updates(user)

	h.DB.Create(&model.UserLoginLog{
		UserId:       user.Id,
		Username:     user.Username,
		LoginIp:      c.ClientIP(),
		LoginAddress: utils.Ip2Region(h.searcher, c.ClientIP()),
	})

	// 创建 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.Id,
		"expired": time.Now().Add(time.Second * time.Duration(h.App.Config.Session.MaxAge)).Unix(),
	})
	tokenString, err := token.SignedString([]byte(h.App.Config.Session.SecretKey))
	if err != nil {
		resp.ERROR(c, "Failed to generate token, "+err.Error())
		return
	}
	// 保存到 redis
	key := fmt.Sprintf("users/%d", user.Id)
	if _, err := h.redis.Set(c, key, tokenString, 0).Result(); err != nil {
		resp.ERROR(c, "error with save token: "+err.Error())
		return
	}
	resp.SUCCESS(c, tokenString)
}

// Logout 注 销
func (h *UserHandler) Logout(c *gin.Context) {
	sessionId := c.GetHeader(types.ChatTokenHeader)
	key := h.GetUserKey(c)
	if _, err := h.redis.Del(c, key).Result(); err != nil {
		logger.Error("error with delete session: ", err)
	}
	// 删除 websocket 会话列表
	h.App.ChatSession.Delete(sessionId)
	// 关闭 socket 连接
	client := h.App.ChatClients.Get(sessionId)
	if client != nil {
		client.Close()
	}
	resp.SUCCESS(c)
}

// Session 获取/验证会话
func (h *UserHandler) Session(c *gin.Context) {
	user, err := h.GetLoginUser(c)
	if err == nil {
		var userVo vo.User
		err := utils.CopyObject(user, &userVo)
		if err != nil {
			resp.ERROR(c)
		}
		userVo.Id = user.Id
		resp.SUCCESS(c, userVo)
	} else {
		resp.NotAuth(c)
	}

}

type userProfile struct {
	Id                 uint    `json:"id"`
	Nickname           string  `json:"nickname"`
	Username           string  `json:"username"`
	Mail               string  `json:"mail"`
	Avatar             string  `json:"avatar"`
	Power              float64 `json:"power"`
	EthAddress         string  `json:"eth_address"`
	AlipayAccount      string  `json:"alipay_account"`
	DefaultPaymentType string  `json:"default_payment_type"`
	Coin               float64 `json:"coin"`
	DjCoin             float64 `json:"dj_coin"`
	ExpiredTime        int64   `json:"expired_time"`
	IsPartner          int     `json:"is_partner"`
	SignInStreak       int     `json:"sign_in_streak"` // 连续签到天数
	SignedToday        bool    `json:"signed_today"`   // 今日是否已签到
	Vip                bool    `json:"vip"`
	Lev                int64   `json:"lev"`   // 用户等级（0-9，对应0钻到9钻）
	Floor              int64   `json:"floor"` // 用户层数（10, 20, 30... 100，对应层级限制）
	Div                string  `json:"div"`
	LeftPerf           float64 `json:"left_perf"`      // 左区业绩
	RightPerf          float64 `json:"right_perf"`     // 右区业绩
	ProducedPower      float64 `json:"produced_power"` // 已产出（所有订单js_power之和）
	PendingPower       float64 `json:"pending_power"`  // 待产出（所有订单now_power之和）
	TotalRevenue       float64 `json:"total_revenue"`  // 总收益（所有订单power之和 - 所有订单sales之和）
	ReferrerName       string  `json:"referrer_name"`  // 推荐人名称（优先用户名）
	InviterName        string  `json:"inviter_name"`   // 邀请人名称（与referrer_name相同，用于兼容前端）
	IsAuditor          bool    `json:"is_auditor"`     // 是否为审核用户
	IsCs               bool    `json:"is_cs"`          // 是否为客服（可接入接线台）
	AllowQ2            bool    `json:"allow_q2"`       // 是否允许 SOHA 交易
	AllowMint          bool    `json:"allow_mint"`     // 是否允许购买算力机
	AllowPlacement     bool    `json:"allow_placement"` // 是否允许查看大小区节点
	SohaMarketEnabled  bool    `json:"soha_market_enabled"`
	SohaMarketOperable bool    `json:"soha_market_operable"`
	SohaKlineEnabled   bool    `json:"soha_kline_enabled"`
	MemberTier         string  `json:"member_tier"`       // inactive|normal|vip|legacy_vip
	MemberTierLabel    string  `json:"member_tier_label"` // 展示文案
	IsJh               int     `json:"is_jh"`             // 0 未激活 1 已激活
	MaxOrderSales      float64 `json:"max_order_sales"`
	JihuoExMinPower    float64 `json:"jihuo_ex_min_power"`
	MemberNormalMinPower float64 `json:"member_normal_min_power"`
	MemberNormalMaxPower float64 `json:"member_normal_max_power"`
	MemberVipMinPower    float64 `json:"member_vip_min_power"`
	CreditMax            float64 `json:"credit_max"`
	CreditAvailable      float64 `json:"credit_available"`
	CreditDebt           float64 `json:"credit_debt"`
	// CreditLegacy 是否为老会员（分界前已达标，额度上限走固定基数）
	// CreditLegacyBase 老会员固定上限基数，供前端解释额度构成
	// CreditLegacyMin 分界后投入门槛：不足该值时新增投入不计入额度
	CreditLegacy     bool    `json:"credit_legacy"`
	CreditLegacyBase float64 `json:"credit_legacy_base"`
	CreditLegacyMin  float64 `json:"credit_legacy_min"`
}

func (h *UserHandler) Profile(c *gin.Context) {
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}
	// 刷新用户；列名交给 GORM 自动加反引号（div/floor 等为 MySQL 保留字，手写 Select 易失败）
	if err := h.DB.First(&user, user.Id).Error; err != nil {
		resp.ERROR(c, "获取用户信息失败")
		return
	}

	var profile userProfile
	if err := utils.CopyObject(user, &profile); err != nil {
		resp.ERROR(c, "获取用户信息失败")
		return
	}

	profile.Id = user.Id
	profile.IsJh = user.IsJh
	profile.AllowQ2 = user.AllowQ2
	// 授信字段：读库前按自投重算一次 Max（轻量 SUM，失败不影响资料返回）
	if h.creditService != nil {
		snap, snapErr := h.creditService.RecalcMaxAndSnapshot(h.DB, user.Id, h.App.SysConfig)
		var cu model.User
		if err := h.DB.Select("credit_max", "credit_available", "credit_debt").First(&cu, user.Id).Error; err == nil {
			profile.CreditMax = cu.CreditMax
			profile.CreditAvailable = cu.CreditAvailable
			profile.CreditDebt = cu.CreditDebt
		}
		// 老会员判定与额度同源，由投入切分推导，不读任何快照字段
		if snapErr == nil {
			profile.CreditLegacy = snap.Legacy
			profile.CreditLegacyBase = snap.Base
			profile.CreditLegacyMin = service.CreditMaxParamsFrom(h.App.SysConfig).LegacyMin
		}
	} else {
		profile.CreditMax = user.CreditMax
		profile.CreditAvailable = user.CreditAvailable
		profile.CreditDebt = user.CreditDebt
	}
	profile.SohaMarketEnabled = SohaMarketGloballyEnabled(h.App.SysConfig)
	profile.SohaMarketOperable = profile.SohaMarketEnabled && user.AllowQ2
	profile.SohaKlineEnabled = SohaKlineDisplayEnabled(h.App.SysConfig)
	profile.IsAuditor = user.IsAuditor
	profile.IsCs = user.IsCs
	profile.AllowMint = user.AllowMint
	profile.AllowPlacement = user.AllowPlacement

	// 签到状态：连续天数 + 今日是否已签到
	profile.SignInStreak = user.SignInStreak
	if user.LastSignInDate != nil {
		now := time.Now()
		loc := now.Location()
		last := user.LastSignInDate.In(loc)
		profile.SignedToday = time.Date(last.Year(), last.Month(), last.Day(), 0, 0, 0, 0, loc).Equal(
			time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc))
	}

	// 等级：优先用库里已存的 lev/floor，避免每次同步递归钻级（远程库会拖过 30s）
	if user.IsJh == 1 {
		profile.Lev = user.Lev
		profile.Floor = user.Floor
		if profile.Floor <= 0 {
			profile.Floor = 10
			profile.Lev = 0
		}
		// 后台异步刷新等级，不阻塞本次响应
		go h.refreshUserLevelAsync(user.Id)
	} else {
		profile.Lev = 0
		profile.Floor = 0
	}

	var (
		leftPerf, rightPerf        float64
		produced, pending, revenue float64
		maxSales                   float64
		referrerName               string
		wg                         sync.WaitGroup
	)

	wg.Add(3)

	go func() {
		defer wg.Done()
		var placementNode struct {
			LeftPerf  float64 `gorm:"column:left_perf"`
			RightPerf float64 `gorm:"column:right_perf"`
		}
		if err := h.DB.Table("chatgpt_placement_nodes").
			Select("left_perf, right_perf").
			Where("user_id = ?", user.Id).
			Take(&placementNode).Error; err == nil {
			leftPerf = placementNode.LeftPerf
			rightPerf = placementNode.RightPerf
		}
	}()

	go func() {
		defer wg.Done()
		var orderStats struct {
			TotalPower     float64 `gorm:"column:total_power"`
			TotalSales     float64 `gorm:"column:total_sales"`
			TotalJsPower   float64 `gorm:"column:total_js_power"`
			TotalNowPower  float64 `gorm:"column:total_now_power"`
			MaxSalesActive float64 `gorm:"column:max_sales_active"`
		}
		// 一次扫描拿到产出统计 + 在投最大单笔（会员档位用）
		_ = h.DB.Table("chatgpt_orders").
			Select(`
				COALESCE(SUM(CASE WHEN status IN (8, 9) THEN power ELSE 0 END), 0) as total_power,
				COALESCE(SUM(CASE WHEN status IN (8, 9) THEN sales ELSE 0 END), 0) as total_sales,
				COALESCE(SUM(CASE WHEN status IN (8, 9) THEN js_power ELSE 0 END), 0) as total_js_power,
				COALESCE(SUM(CASE WHEN status IN (8, 9) THEN now_power ELSE 0 END), 0) as total_now_power,
				COALESCE(MAX(CASE WHEN status = 8 THEN sales ELSE NULL END), 0) as max_sales_active
			`).
			Where("user_id = ? AND status IN (8, 9)", user.Id).
			Scan(&orderStats).Error
		produced = orderStats.TotalJsPower
		pending = orderStats.TotalNowPower
		revenue = orderStats.TotalPower - orderStats.TotalSales
		maxSales = orderStats.MaxSalesActive
	}()

	go func() {
		defer wg.Done()
		if user.ZjFid <= 0 {
			return
		}
		var username string
		if err := h.DB.Model(&model.User{}).
			Select("username").
			Where("id = ?", user.ZjFid).
			Scan(&username).Error; err == nil {
			referrerName = username
		}
	}()

	wg.Wait()

	profile.LeftPerf = leftPerf
	profile.RightPerf = rightPerf
	profile.ProducedPower = produced
	profile.PendingPower = pending
	profile.TotalRevenue = revenue
	profile.ReferrerName = referrerName
	profile.InviterName = referrerName
	profile.MaxOrderSales = maxSales

	tier := service.ResolveMemberTier(&user, h.App.SysConfig, maxSales, func() bool {
		if h.App.SysConfig == nil {
			return false
		}
		cutoff, ok := service.ParseTeamRewardNormalCutoff(h.App.SysConfig.TeamRewardNormalCutoff)
		if !ok {
			return false
		}
		_, _, _, legacyMin := service.MemberTierThresholds(h.App.SysConfig)
		return service.HasLegacyVipOrderBeforeCutoff(h.DB, int(user.Id), legacyMin, cutoff)
	})
	profile.MemberTier = tier.String()
	profile.MemberTierLabel = service.MemberTierLabel(tier)

	if h.App.SysConfig != nil {
		cfg := h.App.SysConfig
		profile.JihuoExMinPower = cfg.JihuoExMinPower
		if profile.JihuoExMinPower <= 0 {
			profile.JihuoExMinPower = 1000
		}
		profile.MemberNormalMinPower = cfg.MemberNormalMinPower
		if profile.MemberNormalMinPower <= 0 {
			profile.MemberNormalMinPower = 1000
		}
		profile.MemberNormalMaxPower = cfg.MemberNormalMaxPower
		if profile.MemberNormalMaxPower <= 0 {
			profile.MemberNormalMaxPower = 9999
		}
		profile.MemberVipMinPower = cfg.MemberVipMinPower
		if profile.MemberVipMinPower <= 0 {
			profile.MemberVipMinPower = 10000
		}
	}

	resp.SUCCESS(c, profile)
}

// refreshUserLevelAsync 异步重算并写回 lev/floor，供下次 Profile 直接使用
func (h *UserHandler) refreshUserLevelAsync(userID uint) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("refreshUserLevelAsync panic:", r)
		}
	}()

	var user model.User
	if err := h.DB.Select("id, is_jh, lev, floor").First(&user, userID).Error; err != nil {
		return
	}
	if user.IsJh != 1 {
		return
	}

	floor := GetUserRewardLayerLimit(h.DB, int(userID))
	if floor < 0 {
		return
	}

	lev := floorToLev(floor)
	if user.Lev == lev && user.Floor == int64(floor) {
		return
	}
	updates := map[string]interface{}{
		"lev":   lev,
		"floor": floor,
	}
	if user.Lev != lev {
		updates["lev_since_at"] = time.Now()
	}
	_ = h.DB.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error
}

func floorToLev(floor int) int64 {
	switch floor {
	case 10:
		return 0
	case 20:
		return 1
	case 30:
		return 2
	case 40:
		return 3
	case 50:
		return 4
	case 60:
		return 5
	case 70:
		return 6
	case 80:
		return 7
	case 90:
		return 8
	case 100:
		return 9
	default:
		return 0
	}
}

func (h *UserHandler) ProfileUpdate(c *gin.Context) {
	var data userProfile
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}
	h.DB.First(&user, user.Id)
	user.Avatar = data.Avatar
	user.Nickname = data.Nickname
	res := h.DB.Updates(&user)
	if res.Error != nil {
		resp.ERROR(c, "更新用户信息失败")
		return
	}

	resp.SUCCESS(c)
}

// UpdatePass 更新密码
func (h *UserHandler) UpdatePass(c *gin.Context) {
	var data struct {
		OldPass  string `json:"old_pass"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	if len(data.Password) < 8 {
		resp.ERROR(c, "密码长度不能少于8个字符")
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	password := utils.GenPassword(data.OldPass, user.Salt)
	logger.Debugf(user.Salt, ",", user.Password, ",", password, ",", data.OldPass)
	if password != user.Password {
		resp.ERROR(c, "原密码错误")
		return
	}

	newPass := utils.GenPassword(data.Password, user.Salt)
	res := h.DB.Model(&user).UpdateColumn("password", newPass)
	if res.Error != nil {
		logger.Error("更新数据库失败: ", res.Error)
		resp.ERROR(c, "更新数据库失败")
		return
	}

	resp.SUCCESS(c)
}

// ResetPass 重置密码（必须校验邮箱验证码，防任意账号接管）
func (h *UserHandler) ResetPass(c *gin.Context) {
	var data struct {
		Username string `json:"username"`
		Mail     string `json:"mail"`
		Code     string `json:"code"`     // 验证码
		Password string `json:"password"` // 新密码
		Repass   string `json:"repass"`   // 确认密码
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	data.Username = strings.TrimSpace(data.Username)
	data.Mail = strings.TrimSpace(data.Mail)
	data.Code = strings.TrimSpace(data.Code)
	if data.Username == "" || data.Mail == "" || data.Code == "" || data.Password == "" {
		resp.ERROR(c, "请填写完整信息")
		return
	}
	if data.Repass != "" && data.Repass != data.Password {
		resp.ERROR(c, "两次输入的密码不一致")
		return
	}

	// 统一失败文案，避免 username+mail 配对 oracle
	const failMsg = "验证码无效或已过期"
	var user model.User
	res := h.DB.Where("username = ? and mail = ?", data.Username, data.Mail).First(&user)
	userExists := res.Error == nil

	// 验证码按邮箱存储（与 /api/sms/code 的 receiver 一致）；同时兼容旧的 username 键
	mailKey := CodeStorePrefix + data.Mail
	userKey := CodeStorePrefix + data.Username
	code, err := h.redis.Get(c, mailKey).Result()
	codeKey := mailKey
	if err != nil {
		code, err = h.redis.Get(c, userKey).Result()
		codeKey = userKey
	}
	// 无论账号是否存在，都校验验证码；不存在时用假比较抹平差异
	codeOK := err == nil && code != "" && subtle.ConstantTimeCompare([]byte(code), []byte(data.Code)) == 1
	if !userExists || !codeOK {
		if err == nil {
			_ = h.redis.Del(c, codeKey) // 错误也消耗，防爆破重放
		}
		resp.ERROR(c, failMsg)
		return
	}
	_ = h.redis.Del(c, mailKey)
	_ = h.redis.Del(c, userKey)

	password := utils.GenPassword(data.Password, user.Salt)
	user.Password = password
	res = h.DB.Updates(&user)
	if res.Error != nil {
		resp.ERROR(c, "重置失败，请稍后重试")
		return
	}
	// 改密后踢掉前台会话
	_ = h.redis.Del(c, fmt.Sprintf("users/%d", user.Id))
	resp.SUCCESS(c)
}

// BindUsername 重置账号
func (h *UserHandler) BindUsername(c *gin.Context) {
	var data struct {
		Username string `json:"username"`
		Code     string `json:"code"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	// 检查验证码
	key := CodeStorePrefix + data.Username
	code, err := h.redis.Get(c, key).Result()
	if err != nil || code != data.Code {
		resp.ERROR(c, "验证码错误")
		return
	}

	// 检查手机号是否被其他账号绑定
	var item model.User
	res := h.DB.Where("username = ?", data.Username).First(&item)
	if res.Error == nil {
		resp.ERROR(c, "该账号已经被其他账号绑定")
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	res = h.DB.Model(&user).UpdateColumn("username", data.Username)
	if res.Error != nil {
		resp.ERROR(c, "更新数据库失败")
		return
	}

	_ = h.redis.Del(c, key) // 删除短信验证码
	resp.SUCCESS(c)
}

// 绑定邮箱
func (h *UserHandler) BindMail(c *gin.Context) {
	var data struct {
		Mail string `json:"mail"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	// 检查手机号是否被其他账号绑定
	var item model.User
	res := h.DB.Where("mail = ?", data.Mail).First(&item)
	if res.Error == nil {
		resp.ERROR(c, "该账号已经被其他账号绑定")
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}
	res = h.DB.Model(&user).UpdateColumn("mail", data.Mail)
	if res.Error != nil {
		resp.ERROR(c, "更新数据库失败")
		return
	}

	//_ = h.redis.Del(c, key) // 删除短信验证码
	resp.SUCCESS(c)
}

// 保存品牌
func (h *UserHandler) SaveDiv(c *gin.Context) {
	var data struct {
		Div string `json:"div"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	// 检查手机号是否被其他账号绑定
	var item model.User
	res := h.DB.Where("div = ?", data.Div).First(&item)
	if res.Error == nil {
		resp.ERROR(c, "该品牌已经被其他账号绑定")
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}
	res = h.DB.Model(&user).UpdateColumn("div", data.Div)
	if res.Error != nil {
		resp.ERROR(c, "更新数据库失败")
		return
	}

	//_ = h.redis.Del(c, key) // 删除短信验证码
	resp.SUCCESS(c)
}

// ImageHubCustomApps 获取用户图片导航的自定义应用
func (h *UserHandler) ImageHubCustomApps(c *gin.Context) {
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	var apps []imageHubCustomApp
	if strings.TrimSpace(user.ImageHubCustomApps) == "" {
		resp.SUCCESS(c, apps)
		return
	}
	if err := json.Unmarshal([]byte(user.ImageHubCustomApps), &apps); err != nil {
		resp.SUCCESS(c, []imageHubCustomApp{})
		return
	}

	resp.SUCCESS(c, apps)
}

// UpdateImageHubCustomApps 更新用户图片导航的自定义应用
func (h *UserHandler) UpdateImageHubCustomApps(c *gin.Context) {
	var data struct {
		Apps []imageHubCustomApp `json:"apps"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	if len(data.Apps) > 15 {
		resp.ERROR(c, "最多允许 15 个自定义应用")
		return
	}

	apps := make([]imageHubCustomApp, 0, len(data.Apps))
	for _, item := range data.Apps {
		id := strings.TrimSpace(item.Id)
		label := strings.TrimSpace(item.Label)
		url := strings.TrimSpace(item.Url)
		if id == "" || label == "" || url == "" {
			continue
		}
		openMode := "embed"
		if item.OpenMode == "new" {
			openMode = "new"
		}
		icon := strings.TrimSpace(item.Icon)
		if icon == "" {
			icon = "link-o"
		}
		apps = append(apps, imageHubCustomApp{
			Id:       id,
			Label:    label,
			Url:      url,
			OpenMode: openMode,
			Icon:     icon,
		})
	}

	value, err := json.Marshal(apps)
	if err != nil {
		resp.ERROR(c, "数据序列化失败")
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}
	res := h.DB.Model(&user).UpdateColumn("image_hub_custom_apps_json", string(value))
	if res.Error != nil {
		resp.ERROR(c, "更新数据库失败")
		return
	}

	resp.SUCCESS(c, apps)
}

// 更新收款地址
func (h *UserHandler) BindEthAddress(c *gin.Context) {
	var data struct {
		EthAddress string `json:"eth_address"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	// 检查手机号是否被其他账号绑定
	var item model.User
	res := h.DB.Where("eth_address = ?", data.EthAddress).First(&item)
	if res.Error == nil {
		resp.ERROR(c, "该账号已经被其他账号绑定")
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}
	res = h.DB.Model(&user).UpdateColumn("eth_address", data.EthAddress)
	if res.Error != nil {
		resp.ERROR(c, "更新数据库失败")
		return
	}

	resp.SUCCESS(c)
}

// 更新收款方式
func (h *UserHandler) UpdatePaymentMethods(c *gin.Context) {
	var data struct {
		EthAddress         string `json:"eth_address"`
		AlipayAccount      string `json:"alipay_account"`
		DefaultPaymentType string `json:"default_payment_type"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	// 验证默认收款方式
	if data.DefaultPaymentType != "bsc" && data.DefaultPaymentType != "alipay" {
		resp.ERROR(c, "默认收款方式必须是 bsc 或 alipay")
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	// 更新用户收款方式信息
	updates := map[string]interface{}{
		"eth_address":          data.EthAddress,
		"alipay_account":       data.AlipayAccount,
		"default_payment_type": data.DefaultPaymentType,
	}

	res := h.DB.Model(&user).Updates(updates)
	if res.Error != nil {
		logger.Error("更新收款方式失败: ", res.Error)
		resp.ERROR(c, "更新数据库失败")
		return
	}

	resp.SUCCESS(c)
}

// 转账逻辑处理 更新密码
func (h *UserHandler) Transfer(c *gin.Context) {
	//uid指的是用户的推广id；power 兼容字符串与数字
	var data struct {
		Uid      string      `json:"uid"`
		Power    interface{} `json:"power"`
		Password string      `json:"password"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		logger.Error("transfer bind json failed: ", err)
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	powerStr := ""
	switch v := data.Power.(type) {
	case nil:
		powerStr = ""
	case string:
		powerStr = strings.TrimSpace(v)
	case float64:
		powerStr = strconv.FormatFloat(v, 'f', -1, 64)
	case json.Number:
		powerStr = v.String()
	default:
		powerStr = strings.TrimSpace(fmt.Sprintf("%v", v))
	}
	if strings.TrimSpace(data.Uid) == "" || powerStr == "" || powerStr == "<nil>" || strings.TrimSpace(data.Password) == "" {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	data.Uid = strings.TrimSpace(data.Uid)

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	//校验密码
	password := utils.GenPassword(data.Password, user.Salt)
	if password != user.Password {
		resp.ERROR(c, "密码输入错误")
		return
	}

	//判断是否存在这样的用户

	inviteCode := model.InviteCode{}
	res0 := h.DB.Where("code = ?", data.Uid).First(&inviteCode)
	logger.Info("res0------> ", res0)

	//userid
	UserId := inviteCode.UserId

	// UserId 对应的model
	var userModel model.User
	h.DB.Where("id = ? ", UserId).Find(&userModel)

	if res0.Error != nil {
		resp.ERROR(c, "用户id不存在，请核对后再操作")
		return
	}

	currentUserId := h.GetLoginUserId(c)

	//判断只有激活才能进行转账操作
	var items []model.Order
	h.DB.Where("user_id = ? ", currentUserId).Find(&items)
	logger.Info("当前用户已经投入的产品为", items)
	if len(items) == 0 {
		resp.ERROR(c, "请先激活产品才能转账")
		return
	}

	// 先将字符串类型转换为浮点数类型
	dataPower, err := strconv.ParseFloat(powerStr, 64)
	if err != nil {
		// 处理转换失败的情况
		resp.ERROR(c, "数据转换失败")
		return
	}

	//获取当前系统最小转账额度
	//最低金额
	var miniPower = h.App.SysConfig.VipMonthPower

	logger.Info(miniPower)
	logger.Info(dataPower)
	logger.Info(dataPower < miniPower)

	if dataPower < miniPower {
		resp.ERROR(c, fmt.Sprintf("最低转账额度为 %d 算力", int(miniPower)))
		return
	}

	feePct := 0.0
	if h.App.SysConfig.EnabledTransferFee {
		feePct = h.App.SysConfig.TransferFeePercent
		if feePct < 0 {
			feePct = 0
		}
		if feePct > 100 {
			feePct = 100
		}
	}
	// 手续费按转出金额另计：对方实收=用户填写的转出额；发送方再额外扣 feeAmount
	feeAmount := math.Round(dataPower*feePct/100.0*1e6) / 1e6
	receiveAmount := math.Round(dataPower*1e6) / 1e6
	totalDebit := math.Round((dataPower+feeAmount)*1e6) / 1e6
	if totalDebit > user.Power {
		if feeAmount > 0 {
			resp.ERROR(c, "用户余额不足，无法支付转出算力及手续费")
		} else {
			resp.ERROR(c, "用户余额不足")
		}
		return
	}

	// 更新转账人余额：转出额 + 额外手续费
	res := h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumn("power", gorm.Expr("power - ?", totalDebit))

	// 收款人按填写的全额到账
	h.DB.Model(&model.User{}).Where("id = ?", UserId).UpdateColumn("power", gorm.Expr("power + ?", receiveAmount))

	//添加logs
	//str := data.Power
	//num, err := strconv.Atoi(str)
	//if err != nil {
	//	fmt.Printf("转换出错: %v\n", err)
	//	return
	//}
	//fmt.Printf("转换后的整数为: %d\n", num)

	//res := h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumn("power", gorm.Expr("power - ?", power))
	if res.Error == nil {
		var u model.User
		h.DB.Where("id = ?", user.Id).First(&u)

		var u2 model.User
		h.DB.Where("id = ?", UserId).First(&u2)

		// 转出方两条明细：① 转账金额 ② 手续费（手续费为 0 时不记第二条）
		balanceAfterTransfer := math.Round((u.Power+feeAmount)*1e6) / 1e6
		t1 := time.Now()
		h.DB.Create(&model.PowerLog{
			UserId:    user.Id,
			Username:  user.Username,
			Type:      types.PowerFk,
			Amount:    dataPower,
			Mark:      types.PowerSub,
			Balance:   balanceAfterTransfer,
			Model:     fmt.Sprintf("转账:收款人(%s)", u2.Username),
			Remark:    fmt.Sprintf("转账支出 %.6f，收款方到账 %.6f", dataPower, receiveAmount),
			CreatedAt: t1,
		})
		if feeAmount > 0 {
			h.DB.Create(&model.PowerLog{
				UserId:    user.Id,
				Username:  user.Username,
				Type:      types.PowerFk,
				Amount:    feeAmount,
				Mark:      types.PowerSub,
				Balance:   u.Power,
				Model:     fmt.Sprintf("转账手续费:收款人(%s)", u2.Username),
				Remark:    fmt.Sprintf("转账手续费 %.6f（%.4f%%）", feeAmount, feePct),
				CreatedAt: t1.Add(time.Millisecond),
			})
			// 手续费按配置比例静默回流认购池（不写用户可见额外流水）
			poolPct := 100.0
			if h.App.SysConfig != nil {
				poolPct = h.App.SysConfig.TransferFeeToPoolPercent
				if poolPct < 0 {
					poolPct = 0
				}
				if poolPct > 100 {
					poolPct = 100
				}
			}
			poolIn := math.Round(feeAmount*poolPct/100.0*1e6) / 1e6
			if poolIn > 0 {
				var pool model.PowerPool
				_ = h.DB.Where("id = ?", 1).FirstOrCreate(&pool, model.PowerPool{
					BaseModel:  model.BaseModel{Id: 1},
					TotalPower: 0,
					TotalUsdt:  0,
				})
				_ = h.DB.Model(&model.PowerPool{}).Where("id = ?", 1).
					UpdateColumn("total_power", gorm.Expr("total_power + ?", poolIn)).Error
			}
		}

		recvRemark := fmt.Sprintf("用户收到算力转账：%.6f", receiveAmount)
		if feeAmount > 0 {
			recvRemark = fmt.Sprintf("用户收到算力转账：%.6f（转出方已另付手续费 %.6f）", receiveAmount, feeAmount)
		}
		h.DB.Create(&model.PowerLog{
			UserId:    u2.Id,
			Username:  u2.Username,
			Type:      types.PowerSk,
			Amount:    receiveAmount,
			Mark:      types.PowerAdd,
			Balance:   u2.Power,
			Model:     fmt.Sprintf("收款:转账人(%s)", user.Username),
			Remark:    recvRemark,
			CreatedAt: time.Now(),
		})

	} else {
		logger.Error("更新数据库失败: ", res.Error)
		resp.ERROR(c, "更新数据库失败")
		return

	}

	resp.SUCCESS(c)
}

// 通过id 获取以上3-10代所有人的user id  id为推荐人的用户id
func GetAllUserIdByRel(h *UserHandler, id int) []int {
	response := make([]int, 10)
	var otherUser model.User
	h.DB.Where("id", id).First(&otherUser)
	if otherUser.ZjFid >= 0 {
		//获取直接上级
		//otherUser.ZjFid
		for i := 0; i < 10; i++ {
			response[i] = i + 1
		}
		//response
		//	return GetAllUserIdByRel(h, id)
	}
	return response

}

// 领取每日免费福利
func (h *UserHandler) GetFuli(c *gin.Context) {

	userId := h.GetLoginUserId(c)
	var user model.User
	logger.Info("userId--->", userId)
	res := h.DB.Where("id = ?  And  status=1  ", userId).First(&user)

	if res.Error != nil {
		resp.ERROR(c, "用户不存在！")
		return
	}

	if user.IsPartner == 1 {
		resp.ERROR(c, "对不起，你今天已经领取了每日福利")
		return
	}

	//判断只有激活才能进行领取免费福利
	var items []model.Order
	h.DB.Where("user_id = ?  ", userId).Find(&items)
	logger.Info("当前用户已经投入的产品为", items)
	if len(items) == 0 {
		resp.ERROR(c, "请先激活产品才能领取每日福利")
		return
	}

	// 连续签到：按上一次签到自然日推进连续天数；断签（间隔 > 1 天）则重置回第 1 天
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	streak := 1
	if user.LastSignInDate != nil {
		last := user.LastSignInDate.In(now.Location())
		lastDay := time.Date(last.Year(), last.Month(), last.Day(), 0, 0, 0, 0, now.Location())
		if lastDay.Equal(today) {
			// 今日已签到（is_partner 未及时重置时兜底）
			resp.ERROR(c, "对不起，你今天已经领取了每日福利")
			return
		}
		if int(today.Sub(lastDay).Hours()/24) == 1 {
			streak = user.SignInStreak + 1
		}
	}
	if streak > 10 {
		streak = 10
	}
	dailyPower := types.SignInRewardByStreak(streak)
	if dailyPower <= 0 {
		resp.ERROR(c, "签到福利未开启")
		return
	}

	logger.Info("签到算力为", dailyPower, "连续签到天数", streak)

	//更新用户余额
	h.DB.Model(&model.User{}).Where("id = ?", userId).UpdateColumn("power", gorm.Expr("power + ?", dailyPower))
	var banlance float64 = user.Power + dailyPower

	logger.Info("banlance", banlance)

	//更新领取状态 + 连续签到信息
	h.DB.Model(&model.User{}).Where("id = ?", userId).Updates(map[string]interface{}{
		"is_partner":        1,
		"sign_in_streak":    streak,
		"last_sign_in_date": today,
	})

	//	//上下级用户流水
	//	h.DB.Create(&model.PowerLog{
	//		UserId:   zjOrderObj.UserId,
	//		Username: zjOrderObj.Username,
	//		Type:     types.PowerJT, //释放
	//		Amount:   turesubPower,
	//		Balance:  orderUser2.Power,
	//		Mark:     types.PowerAdd,
	//		//	Model:    fmt.Sprintf("团队用户  充值:%s 产品  加速释放算力：%f，", product.Name, turesubPower),
	//		Model:   "推荐奖励释放",
	//		OrderNo: zjOrderObj.OrderNo,
	//		//Remark:    fmt.Sprintf("%s，算力：%f，订单号：%s", "", user.Power, item.OrderNo),
	//		//Remark:    fmt.Sprintf("下级用户充值加速释放算力：%f，用户名：%s", turesubPower, user.Username),
	//		Remark:    fmt.Sprintf("下级用户 用户名：%s 充值:%s 产品  加速释放算力：%f 比例为%f ，", user.Username, product.Name, turesubPower, 0.3),
	//		CreatedAt: time.Now(),
	//	})
	var orderModel model.Order

	res = h.DB.Model(&model.Order{}).Where("user_id = ? and status=8 ", userId).Order("now_power DESC").First(&orderModel)
	if res.Error != nil {
		logger.Error("查询用户订单失败: ", res.Error)
		resp.ERROR(c, "未找到有效的订单")
		return
	}
	logger.Info("找到订单: ", orderModel.OrderNo)
	//
	res = h.DB.Model(&model.Order{}).Where("order_no = ?", orderModel.OrderNo).UpdateColumns(map[string]interface{}{
		"js_power":  gorm.Expr("js_power + ?", dailyPower),
		"now_power": gorm.Expr("now_power - ?", dailyPower),
	})
	//
	//res = h.DB.Model(&model.Order{}).Where("order_no = ?", orderModel.OrderNo).UpdateColumn("js_power", gorm.Expr("js_power + ?", config.DailyPower))
	//res = h.DB.Model(&model.Order{}).Where("order_no = ?", orderModel.OrderNo).UpdateColumn("now_power", gorm.Expr("js_power - ?", config.DailyPower))

	h.DB.Create(&model.PowerLog{
		UserId:    userId,
		Username:  user.Username,
		Type:      types.PowerGift,
		Amount:    dailyPower,
		Mark:      types.PowerAdd,
		OrderNo:   orderModel.OrderNo,
		Balance:   banlance,
		Model:     "用户领取每日福利",
		Remark:    fmt.Sprintf("系统每日福利加速释放，今日额度：%f，连续签到第%d天，订单号：%s", dailyPower, streak, orderModel.OrderNo),
		CreatedAt: time.Now(),
	})

	// 有授信欠款时：先入账再立刻扣还款
	if h.creditService != nil {
		if _, err := h.creditService.RepayFromIncome(h.DB, userId, user.Username, dailyPower, orderModel.OrderNo); err != nil {
			logger.Error("签到授信还款失败:", err)
		}
	}

	resp.SUCCESS(c)

}

// 将SOHA兑换成算力
// SignInCalendar 当月签到日历：已签到日期列表 + 今日是否已签到 + 当前连续天数 + 下一次可得算力
func (h *UserHandler) SignInCalendar(c *gin.Context) {
	userId := h.GetLoginUserId(c)

	var data struct {
		Year  int `json:"year"`
		Month int `json:"month"`
	}
	_ = c.ShouldBindJSON(&data)

	now := time.Now()
	loc := now.Location()
	year, month := now.Year(), int(now.Month())
	if data.Year > 0 && data.Month >= 1 && data.Month <= 12 {
		year, month = data.Year, data.Month
	}

	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, loc)
	nextMonthStart := monthStart.AddDate(0, 1, 0)

	// 该月已签到的自然日：直接取 created_at 再按本地自然日去重
	// （避免 DATE()/字符串解析在不同 MySQL 驱动下拿不到天数）
	var createdAts []time.Time
	service.ActivePowerLogScope(h.DB).Model(&model.PowerLog{}).
		Where("user_id = ? AND model = ? AND created_at >= ? AND created_at < ?",
			userId, "用户领取每日福利", monthStart, nextMonthStart).
		Pluck("created_at", &createdAts)

	daySet := make(map[int]struct{}, len(createdAts))
	for _, at := range createdAts {
		daySet[at.In(loc).Day()] = struct{}{}
	}

	var user model.User
	_ = h.DB.Select("last_sign_in_date, sign_in_streak").First(&user, userId).Error

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	signedToday := false
	if user.LastSignInDate != nil {
		last := user.LastSignInDate.In(loc)
		lastDay := time.Date(last.Year(), last.Month(), last.Day(), 0, 0, 0, 0, loc)
		signedToday = lastDay.Equal(today)
		// 兜底：今日已签但流水尚未写入/时区偏差时，仍把当天计入日历
		if last.Year() == year && int(last.Month()) == month {
			daySet[last.Day()] = struct{}{}
		}
	}

	days := make([]int, 0, len(daySet))
	for d := range daySet {
		days = append(days, d)
	}
	sort.Ints(days)

	// 下一次签到对应的连续天数（今日已签则为当天值）
	previewStreak := user.SignInStreak
	if !signedToday {
		previewStreak = 1
		if user.LastSignInDate != nil {
			last := user.LastSignInDate.In(loc)
			lastDay := time.Date(last.Year(), last.Month(), last.Day(), 0, 0, 0, 0, loc)
			if int(today.Sub(lastDay).Hours()/24) == 1 {
				previewStreak = user.SignInStreak + 1
			}
		}
	}
	if previewStreak < 1 {
		previewStreak = 1
	}
	if previewStreak > 10 {
		previewStreak = 10
	}

	resp.SUCCESS(c, gin.H{
		"year":            year,
		"month":           month,
		"days":            days,
		"signed_today":    signedToday,
		"streak":          user.SignInStreak,
		"next_streak_day": previewStreak,
		"next_reward":     types.SignInRewardByStreak(previewStreak),
	})
}

func (h *UserHandler) SwapQ2ToPower(c *gin.Context) {
	// 获取参数
	var data struct {
		PowerAmount float64 `json:"power_amount"`
		Q2Amount    float64 `json:"q2_amount"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	// 参数验证
	if data.PowerAmount <= 0 || data.Q2Amount <= 0 {
		resp.ERROR(c, "兑换数量必须大于0")
		return
	}

	// 获取当前用户ID
	userId := h.GetLoginUserId(c)
	if userId == 0 {
		resp.ERROR(c, "用户未登录")
		return
	}

	// 获取当前用户的算力余额和SOHA余额
	var user model.User
	res := h.DB.First(&user, userId)
	if res.Error != nil {
		resp.ERROR(c, "用户不存在")
		return
	}

	if ok, msg := CanOperateSohaMarket(h.App.SysConfig, &user); !ok {
		resp.ERROR(c, msg)
		return
	}

	// 检查算力余额是否足够
	if user.Coin < data.Q2Amount {
		resp.ERROR(c, "SOHA余额不足")
		return
	}

	if data.Q2Amount > 500000 {
		resp.ERROR(c, "单次兑换最多支持50万个SOHA")
		return
	}

	// 校验动态兑换比例：算力 = SOHA数量 × (当前SOHA价 / 锚定价)
	swapPowerPrice := h.App.SysConfig.SwapPowerPrice
	if swapPowerPrice <= 0 {
		swapPowerPrice = 0.01
	}
	var latestPrice model.PriceHistory
	q2Price := 0.01
	if err := h.DB.Order("date DESC").First(&latestPrice).Error; err == nil && latestPrice.Price > 0 {
		q2Price = latestPrice.Price
	}
	expectedPower := data.Q2Amount * (q2Price / swapPowerPrice)
	if expectedPower <= 0 {
		resp.ERROR(c, "当前无法计算兑换比例，请稍后再试")
		return
	}
	// 允许 1% 浮点误差
	if math.Abs(data.PowerAmount-expectedPower) > expectedPower*0.01 {
		resp.ERROR(c, "兑换比例已变动，请刷新页面后重试")
		return
	}

	// 使用事务处理兑换操作
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 增加算力余额
		err := tx.Model(&model.User{}).Where("id = ?", userId).
			UpdateColumn("power", gorm.Expr("power + ?", data.PowerAmount)).Error
		if err != nil {
			return err
		}

		// 减少SOHA余额
		err = tx.Model(&model.User{}).Where("id = ?", userId).
			UpdateColumn("coin", gorm.Expr("coin - ?", data.Q2Amount)).Error
		if err != nil {
			return err
		}

		//更新销毁数量
		//获取当前的配置
		// 3. 获取当前系统配置
		var config model.Config
		err = h.DB.Where("marker = ?", "system").First(&config).Error
		if err != nil {
			return fmt.Errorf("获取系统配置失败: %v", err)
		}

		// 4. 解析当前配置JSON
		var currentConfig types.SystemConfig
		err = utils.JsonDecode(config.Config, &currentConfig)
		if err != nil {
			return fmt.Errorf("解析系统配置失败: %v", err)
		}

		// 旧值
		oldLimitCancel := currentConfig.LimitCancel

		//新值
		newLimitCancel := oldLimitCancel + int64(data.Q2Amount)
		// 5. 更新limit_cancel字段
		currentConfig.LimitCancel = newLimitCancel

		updatedConfigJSON := utils.JsonEncode(&currentConfig)

		// 7. 更新数据库中的配置
		err = h.DB.Model(&config).Update("config_json", updatedConfigJSON).Error
		if err != nil {
			return fmt.Errorf("更新数据库配置失败: %v", err)
		}

		// 8. 更新AppServer缓存
		err = utils.JsonDecode(updatedConfigJSON, &h.App.SysConfig)
		if err != nil {
			return fmt.Errorf("更新AppServer缓存失败: %v", err)
		}

		return nil
	})

	if err != nil {
		logger.Error("兑换操作失败：", err)
		resp.ERROR(c, "兑换失败")
		return
	}

	// 获取更新后的用户信息
	var updatedUser model.User
	h.DB.First(&updatedUser, userId)

	// 添加用户算力日志
	h.DB.Create(&model.PowerLog{
		UserId:    userId,
		Username:  user.Username,
		Type:      types.PowerDh,
		Amount:    data.PowerAmount,
		Mark:      types.PowerAdd,
		Balance:   updatedUser.Power,
		Model:     "SOHA兑换算力",
		Remark:    fmt.Sprintf("用户SOHA兑换成算力，消耗SOHA：%f，获得算力：%f", data.Q2Amount, data.PowerAmount),
		CreatedAt: time.Now(),
	})

	// 添加用户的Q2变化日志
	h.DB.Create(&model.Q2Log{
		UserId:   userId,
		Username: user.Username,
		Type:     types.CoinDh,
		Amount:   data.Q2Amount,
		//Mark:     types.PowerAdd,
		Mark:      types.PowerSub,
		Balance:   updatedUser.Coin,
		Model:     "SOHA兑换算力",
		Remark:    fmt.Sprintf("用户SOHA兑换成算力，消耗SOHA：%f，获得算力：%f", data.Q2Amount, data.PowerAmount),
		CreatedAt: time.Now(),
	})

	logger.Info("用户兑换成功：", map[string]interface{}{
		"user_id":      userId,
		"username":     user.Username,
		"power_amount": data.PowerAmount,
		"q2_amount":    data.Q2Amount,
		"new_power":    updatedUser.Power,
		"new_coin":     updatedUser.Coin,
	})

	resp.SUCCESS(c, gin.H{
		"message":   "兑换成功",
		"new_power": updatedUser.Power,
		"new_coin":  updatedUser.Coin,
		"exchanged": gin.H{
			"power_amount": data.PowerAmount,
			"q2_amount":    data.Q2Amount,
		},
	})
}

// Buyback 算力回购
func (h *UserHandler) Buyback(c *gin.Context) {
	// 获取参数
	var data struct {
		Power      float64 `json:"power"`
		EthAddress string  `json:"eth_address"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	// 参数验证
	if data.Power <= 0 {
		resp.ERROR(c, "回购算力数量必须大于0")
		return
	}

	// 获取当前用户
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	// 检查算力余额是否足够
	if data.Power > user.Power {
		resp.ERROR(c, "算力余额不足")
		return
	}

	// 验证BSC地址
	if data.EthAddress == "" || strings.TrimSpace(data.EthAddress) == "" {
		resp.ERROR(c, "请先设置BSC付款地址")
		return
	}

	// 验证用户设置的BSC地址是否匹配
	if user.EthAddress == "" || strings.TrimSpace(user.EthAddress) == "" {
		resp.ERROR(c, "请先在个人中心设置BSC付款地址")
		return
	}

	if strings.TrimSpace(user.EthAddress) != strings.TrimSpace(data.EthAddress) {
		resp.ERROR(c, "BSC地址与用户设置的不匹配")
		return
	}

	// 计算回购金额（1.5折：100算力正常=1U，回购时=0.15U，即1算力=0.0015U）
	buybackAmount := data.Power * 0.0015

	// 生成回购订单号：BUYBACK + 时间戳 + 用户ID
	orderNo := fmt.Sprintf("BUYBACK%d%d", time.Now().Unix(), user.Id)

	var trade model.Buyback
	var updatedUser model.User

	// 使用事务处理回购操作
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		// 在事务中再次查询用户算力余额，防止并发问题
		var currentUser model.User
		if err := tx.First(&currentUser, user.Id).Error; err != nil {
			return fmt.Errorf("获取用户信息失败: %v", err)
		}

		// 再次校验算力余额是否足够
		if currentUser.Power < data.Power {
			return fmt.Errorf("算力余额不足，当前可用算力：%.2f", currentUser.Power)
		}

		// 扣除用户算力（使用乐观锁，确保余额足够才扣减）
		updateResult := tx.Model(&model.User{}).
			Where("id = ? AND power >= ?", user.Id, data.Power).
			UpdateColumn("power", gorm.Expr("power - ?", data.Power))

		if updateResult.Error != nil {
			return updateResult.Error
		}

		// 检查是否成功扣减（如果受影响行数为0，说明余额不足）
		if updateResult.RowsAffected == 0 {
			return fmt.Errorf("算力余额不足，无法完成回购")
		}

		// 获取更新后的用户信息
		err = tx.First(&updatedUser, user.Id).Error
		if err != nil {
			return err
		}

		//使用chatgpt_buyback 这个表
		// 创建回购订单
		trade = model.Buyback{
			OrderNumber: orderNo,
			UserId:      user.Id,
			Number:      data.Power, // 回购的算力数量
			Price:       0.0015,     // 回购金额
			CreateTime:  time.Now().Unix(),
			Status:      1,             // 初始化状态
			Types:       1,             // 卖单（回购）
			TotalPrice:  buybackAmount, // 总价值
			UserName:    user.Username,
			EthAddress:  data.EthAddress, // BSC收款地址
		}
		err = tx.Create(&trade).Error
		if err != nil {
			return err
		}

		// 记录算力日志（关联订单号）
		err = tx.Create(&model.PowerLog{
			UserId:    user.Id,
			Username:  user.Username,
			Type:      types.PowerBuyback,
			Amount:    data.Power,
			Mark:      types.PowerSub,
			Balance:   updatedUser.Power,
			OrderNo:   orderNo,
			Model:     "算力回购",
			Remark:    fmt.Sprintf("算力回购，回购算力：%f，回购金额：%.2f美金，BSC地址：%s，订单号：%s", data.Power, buybackAmount, data.EthAddress, orderNo),
			CreatedAt: time.Now(),
		}).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		logger.Error("回购操作失败：", err)
		// 如果是算力余额不足的错误，返回具体错误信息
		if strings.Contains(err.Error(), "算力余额不足") {
			resp.ERROR(c, err.Error())
		} else {
			resp.ERROR(c, "回购失败，请稍后重试")
		}
		return
	}

	logger.Info("用户回购成功：", map[string]interface{}{
		"user_id":        user.Id,
		"username":       user.Username,
		"power":          data.Power,
		"buyback_amount": buybackAmount,
		"eth_address":    data.EthAddress,
		"new_power":      updatedUser.Power,
		"order_no":       orderNo,
	})

	resp.SUCCESS(c, gin.H{
		"message":        "回购申请提交成功",
		"power":          data.Power,
		"buyback_amount": buybackAmount,
		"eth_address":    data.EthAddress,
		"new_power":      updatedUser.Power,
		"order_no":       orderNo,
	})
}

// PlacementTree 获取当前登录用户的安置子树（含左右区/大小区节点）
// 远程库场景下避免逐节点查询：优先递归 CTE 一次取出，失败则按左右子 ID 分层批量取。
func (h *UserHandler) PlacementTree(c *gin.Context) {
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}
	if err := h.DB.Select("id, allow_placement").First(&user, user.Id).Error; err != nil {
		resp.ERROR(c, "获取用户信息失败")
		return
	}
	if !user.AllowPlacement {
		resp.ERROR(c, "未开通大小区查看权限")
		return
	}

	type placementRow struct {
		UserID       int     `json:"user_id"`
		ParentID     int     `json:"parent_id"`
		LeftChildID  int     `json:"left_child_id"`
		RightChildID int     `json:"right_child_id"`
		LeftPerf     float64 `json:"left_perf"`
		RightPerf    float64 `json:"right_perf"`
		TotalPerf    float64 `json:"total_perf"`
	}

	const maxDepth = 64
	const maxNodes = 3000

	var root placementRow
	if err := h.DB.Table("chatgpt_placement_nodes").Where("user_id = ?", user.Id).First(&root).Error; err != nil {
		resp.SUCCESS(c, gin.H{"nodes": []any{}, "root_user_id": user.Id})
		return
	}

	var allNodes []placementRow
	// MySQL 8+：一次递归查出整棵子树（远程库只 1 次往返）
	cteErr := h.DB.Raw(`
WITH RECURSIVE subtree AS (
  SELECT user_id, parent_id, left_child_id, right_child_id, left_perf, right_perf, total_perf, 0 AS depth
  FROM chatgpt_placement_nodes
  WHERE user_id = ?
  UNION ALL
  SELECT n.user_id, n.parent_id, n.left_child_id, n.right_child_id, n.left_perf, n.right_perf, n.total_perf, s.depth + 1
  FROM chatgpt_placement_nodes n
  INNER JOIN subtree s ON (n.user_id = s.left_child_id OR n.user_id = s.right_child_id)
  WHERE s.depth < ?
)
SELECT user_id, parent_id, left_child_id, right_child_id, left_perf, right_perf, total_perf
FROM subtree
LIMIT ?
`, user.Id, maxDepth, maxNodes).Scan(&allNodes).Error

	if cteErr != nil || len(allNodes) == 0 {
		// 兼容无 CTE / 失败：按左右子 ID 分层批量查（深度次往返，而非节点次）
		allNodes = []placementRow{root}
		seen := map[int]struct{}{root.UserID: {}}
		level := []placementRow{root}
		for depth := 0; depth < maxDepth && len(level) > 0 && len(allNodes) < maxNodes; depth++ {
			childIDs := make([]int, 0, len(level)*2)
			for _, n := range level {
				if n.LeftChildID > 0 {
					if _, ok := seen[n.LeftChildID]; !ok {
						childIDs = append(childIDs, n.LeftChildID)
					}
				}
				if n.RightChildID > 0 {
					if _, ok := seen[n.RightChildID]; !ok {
						childIDs = append(childIDs, n.RightChildID)
					}
				}
			}
			if len(childIDs) == 0 {
				break
			}
			var children []placementRow
			if err := h.DB.Table("chatgpt_placement_nodes").
				Where("user_id IN ?", childIDs).
				Find(&children).Error; err != nil {
				break
			}
			next := make([]placementRow, 0, len(children))
			for _, child := range children {
				if _, ok := seen[child.UserID]; ok {
					continue
				}
				seen[child.UserID] = struct{}{}
				allNodes = append(allNodes, child)
				next = append(next, child)
				if len(allNodes) >= maxNodes {
					break
				}
			}
			level = next
		}
	}

	nodeMap := make(map[int]placementRow, len(allNodes))
	for _, n := range allNodes {
		nodeMap[n.UserID] = n
	}
	if n, ok := nodeMap[root.UserID]; ok {
		root = n
	} else {
		nodeMap[root.UserID] = root
	}

	userIDs := make([]int, 0, len(nodeMap))
	for id := range nodeMap {
		userIDs = append(userIDs, id)
	}

	type userInfo struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Nickname string `json:"nickname"`
	}
	userMap := make(map[int]userInfo, len(userIDs))
	rechargeMap := make(map[int]float64, len(userIDs))
	if len(userIDs) > 0 {
		var users []model.User
		h.DB.Select("id, username, nickname").Where("id IN ?", userIDs).Find(&users)
		for _, u := range users {
			userMap[int(u.Id)] = userInfo{
				ID:       int(u.Id),
				Username: u.Username,
				Nickname: u.Nickname,
			}
		}

		type rechargeResult struct {
			UserId int
			Total  float64
		}
		var rechargeResults []rechargeResult
		_ = h.DB.Table("chatgpt_orders").
			Select("user_id, COALESCE(SUM(sales), 0) as total").
			Where("user_id IN ?", userIDs).
			Group("user_id").
			Scan(&rechargeResults).Error
		for _, r := range rechargeResults {
			rechargeMap[r.UserId] = r.Total
		}
	}

	type nodeWithUser struct {
		placementRow
		UserInfo       userInfo `json:"user_info"`
		RechargeAmount float64  `json:"recharge_amount"`
		Side           string   `json:"side"` // self | left | right
	}

	sideOf := map[int]string{root.UserID: "self"}
	queue := []int{root.UserID}
	for len(queue) > 0 {
		curID := queue[0]
		queue = queue[1:]
		cur, ok := nodeMap[curID]
		if !ok {
			continue
		}
		parentSide := sideOf[curID]
		assign := func(childID int, directSide string) {
			if childID <= 0 {
				return
			}
			if _, exists := sideOf[childID]; exists {
				return
			}
			if parentSide == "self" {
				sideOf[childID] = directSide
			} else {
				sideOf[childID] = parentSide
			}
			queue = append(queue, childID)
		}
		assign(cur.LeftChildID, "left")
		assign(cur.RightChildID, "right")
	}

	result := make([]nodeWithUser, 0, len(nodeMap))
	for _, node := range nodeMap {
		side := sideOf[node.UserID]
		if side == "" {
			side = "left"
		}
		result = append(result, nodeWithUser{
			placementRow:   node,
			UserInfo:       userMap[node.UserID],
			RechargeAmount: rechargeMap[node.UserID],
			Side:           side,
		})
	}

	resp.SUCCESS(c, gin.H{
		"root_user_id": user.Id,
		"left_perf":    root.LeftPerf,
		"right_perf":   root.RightPerf,
		"total_perf":   root.TotalPerf,
		"nodes":        result,
	})
}
