package main

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/handler"
	"chatplus/handler/admin"
	"chatplus/handler/chatimpl"
	logger2 "chatplus/logger"
	"chatplus/service"
	"chatplus/service/cs"
	"chatplus/service/mj"
	"chatplus/service/oss"
	"chatplus/service/payment"
	"chatplus/service/sd"
	"chatplus/service/sms"
	"chatplus/service/sora2"
	"chatplus/service/suno"
	"chatplus/service/wx"
	"chatplus/store"
	"chatplus/store/model"
	"context"
	"embed"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

var logger = logger2.GetLogger()

//go:embed res
var xdbFS embed.FS

// AppLifecycle 应用程序生命周期
type AppLifecycle struct {
}

// OnStart 应用程序启动时执行
func (l *AppLifecycle) OnStart(context.Context) error {
	log.Println("AppLifecycle OnStart")
	return nil
}

// OnStop 应用程序停止时执行
func (l *AppLifecycle) OnStop(context.Context) error {
	log.Println("AppLifecycle OnStop")
	return nil
}

func NewAppLifeCycle() *AppLifecycle {
	return &AppLifecycle{}
}

func main() {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "config.toml"
	}
	debug, _ := strconv.ParseBool(os.Getenv("APP_DEBUG"))
	logger.Info("Loading config file: ", configFile)
	if !debug {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("Panic Error:", err)
			}
		}()
	}

	app := fx.New(
		// 初始化配置应用配置
		fx.Provide(func() *types.AppConfig {
			config, err := core.LoadConfig(configFile)
			if err != nil {
				log.Fatal(err)
			}
			config.Path = configFile
			if debug {
				_ = core.SaveConfig(config)
			}
			return config
		}),
		// 创建应用服务
		fx.Provide(core.NewServer),
		// 初始化
		fx.Invoke(func(s *core.AppServer, client *redis.Client) {
			s.Init(debug, client)
		}),

		// 初始化数据库
		fx.Provide(store.NewGormConfig),
		fx.Provide(store.NewMysql),
		fx.Provide(store.NewRedisClient),
		fx.Provide(store.NewLevelDB),

		fx.Invoke(func(db *gorm.DB) {
			_ = db.AutoMigrate(&model.TeachingRecord{}, &model.AppPowerRefundLog{}, &model.CsSession{}, &model.CsMessage{})
			// 客服身份字段：缺列时自动补齐，避免 Profile 读库失败导致 H5 无法分流
			if !db.Migrator().HasColumn(&model.User{}, "is_cs") {
				_ = db.Migrator().AddColumn(&model.User{}, "IsCs")
			}
			// 连续签到的两个字段：缺列时必须补齐。
			// 否则 GetFuli 里「is_partner + sign_in_streak + last_sign_in_date」的批量 Updates 会整条 SQL 报错，
			// 连 is_partner 一起写不进去，两个防重复守卫全部失效 → 可无限重复签到。
			if !db.Migrator().HasColumn(&model.User{}, "sign_in_streak") {
				_ = db.Migrator().AddColumn(&model.User{}, "SignInStreak")
			}
			if !db.Migrator().HasColumn(&model.User{}, "last_sign_in_date") {
				_ = db.Migrator().AddColumn(&model.User{}, "LastSignInDate")
			}
			// 授信额度三字段
			if !db.Migrator().HasColumn(&model.User{}, "credit_max") {
				_ = db.Migrator().AddColumn(&model.User{}, "CreditMax")
			}
			if !db.Migrator().HasColumn(&model.User{}, "credit_available") {
				_ = db.Migrator().AddColumn(&model.User{}, "CreditAvailable")
			}
			if !db.Migrator().HasColumn(&model.User{}, "credit_debt") {
				_ = db.Migrator().AddColumn(&model.User{}, "CreditDebt")
			}
		}),

		// 在线客服 Hub
		fx.Provide(cs.NewHub),

		// BSC USDT 分账服务
		fx.Provide(service.NewBscUsdtService),

		fx.Provide(func() embed.FS {
			return xdbFS
		}),

		// 创建 Ip2Region 查询对象
		fx.Provide(func() (*xdb.Searcher, error) {
			file, err := xdbFS.Open("res/ip2region.xdb")
			if err != nil {
				return nil, err
			}
			cBuff, err := io.ReadAll(file)
			if err != nil {
				return nil, err
			}

			return xdb.NewWithBuffer(cBuff)
		}),

		// 创建控制器
		fx.Provide(handler.NewChatRoleHandler),
		fx.Provide(handler.NewUserHandler),
		fx.Provide(chatimpl.NewChatHandler),
		fx.Provide(handler.NewUploadHandler),
		fx.Provide(handler.NewSmsHandler),
		fx.Provide(handler.NewRewardHandler),
		fx.Provide(handler.NewCaptchaHandler),
		fx.Provide(handler.NewMidJourneyHandler),
		fx.Provide(handler.NewChatModelHandler),
		fx.Provide(handler.NewSdJobHandler),
		fx.Provide(handler.NewPaymentHandler),
		fx.Provide(handler.NewOrderHandler),
		fx.Provide(handler.NewTradeHandler),

		fx.Provide(handler.NewProductHandler),
		fx.Provide(handler.NewMintHandler),
		fx.Provide(handler.NewConfigHandler),
		fx.Provide(handler.NewPowerLogHandler),
		fx.Provide(handler.NewMintOrderHandler),
		fx.Provide(handler.NewDigitalHandler),
		fx.Provide(handler.NewSunoHandler),
		fx.Provide(handler.NewAiSearchHandler),
		fx.Provide(handler.NewVideoHandler),
		fx.Provide(handler.NewSora2Handler),
		fx.Provide(handler.NewTeachingHandler),
		fx.Provide(handler.NewQ2LogHandler),
		fx.Provide(handler.NewCoinLogHandler),
		fx.Provide(handler.NewCsHandler),

		fx.Provide(admin.NewConfigHandler),
		fx.Provide(admin.NewAdminHandler),
		fx.Provide(admin.NewCsHandler),
		fx.Provide(admin.NewApiKeyHandler),
		fx.Provide(admin.NewUserHandler),
		fx.Provide(admin.NewChatRoleHandler),
		fx.Provide(admin.NewRewardHandler),
		fx.Provide(admin.NewDashboardHandler),
		fx.Provide(admin.NewChatModelHandler),
		fx.Provide(admin.NewProductHandler),
		fx.Provide(admin.NewMintHandler),
		fx.Provide(admin.NewMintOrderHandler),
		fx.Provide(admin.NewOrderHandler),
		fx.Provide(admin.NewTradeHandler),
		fx.Provide(admin.NewChatHandler),
		fx.Provide(admin.NewPowerLogHandler),
		fx.Provide(admin.NewQ2LogHandler),
		fx.Provide(admin.NewCoinLogHandler),
		fx.Provide(admin.NewMusicHandler),

		// 创建服务
		fx.Provide(sms.NewSendServiceManager),
		fx.Provide(func(config *types.AppConfig) *service.CaptchaService {
			return service.NewCaptchaService(config.ApiConfig)
		}),
		fx.Provide(oss.NewUploaderManager),
		fx.Provide(mj.NewService),
		fx.Provide(service.NewWebsocketService),
		fx.Provide(service.NewUserService),
		fx.Provide(service.NewCreditService),

		// 邮件服务
		fx.Provide(service.NewSmtpService),

		// 微信机器人服务
		fx.Provide(wx.NewWeChatBot),
		fx.Invoke(func(config *types.AppConfig, bot *wx.Bot) {
			if config.WeChatBot {
				err := bot.Run()
				if err != nil {
					logger.Error("微信登录失败：", err)
				}
			}
		}),

		// MidJourney service pool
		fx.Provide(mj.NewServicePool),
		fx.Invoke(func(pool *mj.ServicePool) {
			if pool.HasAvailableService() {
				logger.Info("有服务")
				pool.DownloadImages()
				pool.CheckTaskNotify()
				pool.SyncTaskProgress()
			}
		}),

		//数字人
		fx.Provide(mj.NewDigiServicePool),
		fx.Invoke(func(pool *mj.DigiServicePool) {
			if pool.HasAvailableService() {
				logger.Info("有服务")
				pool.DownloadVideo()
				pool.CheckTaskNotify()
				pool.SyncTaskProgress()
			}
		}),

		fx.Provide(suno.NewSunoServicePool),
		fx.Invoke(func(pool *suno.SunoServicePool) {

			// 移除条件判断，直接启动
			logger.Info("启动音乐任务监控服务...")
			pool.SyncTaskProgress()

			//if pool.HasAvailableService() {
			//	logger.Info("有服务")
			//	pool.SyncTaskProgress()
			//}
		}),

		// Stable Diffusion 机器人
		fx.Provide(sd.NewServicePool),
		fx.Invoke(func(pool *sd.ServicePool) {
			if pool.HasAvailableService() {
				pool.CheckTaskNotify()
				pool.CheckTaskStatus()
			}
		}),

		// Sora2 服务
		fx.Provide(sora2.NewService),
		fx.Invoke(func(s *sora2.Service) {
			s.Run()
			s.SyncTaskProgress()
			s.CheckTaskNotify()
			s.DownloadFiles()
		}),

		fx.Provide(payment.NewAlipayService),
		fx.Provide(payment.NewHuPiPay),
		fx.Provide(payment.NewPayJS),
		fx.Provide(service.NewSnowflake),
		fx.Provide(service.NewXXLJobExecutor),
		fx.Invoke(func(exec *service.XXLJobExecutor, config *types.AppConfig) {
			if config.XXLConfig.Enabled {
				go func() {
					log.Fatal(exec.Run())
				}()
			}
		}),

		// 注册路由
		fx.Invoke(func(s *core.AppServer, h *handler.ChatRoleHandler) {
			group := s.Engine.Group("/api/role/")
			group.GET("list", h.List)
			group.POST("update", h.UpdateRole)
		}),
		fx.Invoke(func(s *core.AppServer, h *handler.UserHandler) {
			group := s.Engine.Group("/api/user/")
			group.POST("register", h.Register)
			group.POST("login", h.Login)
			group.GET("logout", h.Logout)
			group.GET("session", h.Session)
			group.GET("profile", h.Profile)
			group.POST("profile/update", h.ProfileUpdate)
			group.POST("password", h.UpdatePass)
			group.POST("transfer", h.Transfer)
			group.POST("bind/username", h.BindUsername)
			group.POST("updateMail", h.BindMail)
			group.POST("saveDiv", h.SaveDiv)
			group.GET("imageHubCustomApps", h.ImageHubCustomApps)
			group.POST("imageHubCustomApps/update", h.UpdateImageHubCustomApps)
			group.POST("bindEthAddress", h.BindEthAddress)
			group.POST("updatePaymentMethods", h.UpdatePaymentMethods)
			group.POST("resetPass", h.ResetPass)
			group.POST("buyback", h.Buyback) // 算力回购
			group.POST("getFuli", h.GetFuli)
			group.POST("signInCalendar", h.SignInCalendar) // 当月签到日历
			group.POST("swapQ2ToPower", h.SwapQ2ToPower)  //将SOHA兑换成算力
			group.POST("ppt/consume", h.PptConsume)       // PPT 子系统扣减算力
			group.POST("ppt/refund", h.PptRefund)         // PPT 子系统退还算力
			group.GET("ppt/ai-settings", h.PptAiSettings) // PPT 子系统拉取主站后台 PPT 大模型配置（需登录）
			group.GET("placement/tree", h.PlacementTree)  // 当前用户安置树（大小区节点）

		}),
		fx.Invoke(func(s *core.AppServer, h *chatimpl.ChatHandler) {
			group := s.Engine.Group("/api/chat/")
			group.Any("new", h.ChatHandle)
			group.GET("list", h.List)
			group.GET("detail", h.Detail)
			group.POST("update", h.Update)
			group.GET("remove", h.Remove)
			group.GET("history", h.History)
			group.GET("clear", h.Clear)
			group.POST("tokens", h.Tokens)
			group.GET("stop", h.StopGenerate)
		}),
		fx.Invoke(func(s *core.AppServer, h *handler.UploadHandler) {
			s.Engine.POST("/api/upload", h.Upload)
			s.Engine.GET("/api/upload/list", h.List)
			s.Engine.GET("/api/upload/remove", h.Remove)
		}),
		fx.Invoke(func(s *core.AppServer, h *handler.SmsHandler) {
			group := s.Engine.Group("/api/sms/")
			group.POST("code", h.SendCode)
		}),
		fx.Invoke(func(s *core.AppServer, h *handler.CaptchaHandler) {
			group := s.Engine.Group("/api/captcha/")
			group.GET("get", h.Get)
			group.POST("check", h.Check)
			group.GET("slide/get", h.SlideGet)
			group.POST("slide/check", h.SlideCheck)
		}),
		fx.Invoke(func(s *core.AppServer, h *handler.RewardHandler) {
			group := s.Engine.Group("/api/reward/")
			group.POST("verify", h.Verify)
		}),
		fx.Invoke(func(s *core.AppServer, h *handler.MidJourneyHandler) {
			group := s.Engine.Group("/api/mj/")
			group.Any("client", h.Client)
			group.POST("image", h.Image)
			group.POST("upscale", h.Upscale)
			group.POST("variation", h.Variation)
			group.GET("jobs", h.JobList)
			group.GET("imgWall", h.ImgWall)
			group.POST("remove", h.Remove)
			group.POST("publish", h.Publish)
		}),
		fx.Invoke(func(s *core.AppServer, h *handler.SdJobHandler) {
			group := s.Engine.Group("/api/sd")
			group.Any("client", h.Client)
			group.POST("image", h.Image)
			group.GET("jobs", h.JobList)
			group.GET("imgWall", h.ImgWall)
			group.POST("remove", h.Remove)
			group.POST("publish", h.Publish)
		}),
		//数字人
		fx.Invoke(func(s *core.AppServer, h *handler.DigitalHandler) {
			group := s.Engine.Group("/api/digi")
			//	group.GET("getParamter", h.GetParamter)
			//	group.POST("startTask", h.StartTask)
			group.Any("client", h.Client)
			group.POST("createDigi", h.Digi)
			group.GET("jobs", h.JobList)
			//group.GET("imgWall", h.ImgWall)
			//group.POST("remove", h.Remove)
			//group.POST("publish", h.Publish)
		}),
		fx.Invoke(func(s *core.AppServer, h *handler.ConfigHandler) {
			group := s.Engine.Group("/api/config/")
			group.GET("get", h.Get)
			group.GET("locale", h.Locale) // 根据 IP 推荐语言
			//group.GET("test", h.Test)
		}),

		// 管理后台控制器
		fx.Invoke(func(s *core.AppServer, h *admin.ConfigHandler) {
			group := s.Engine.Group("/api/admin/config/")
			group.POST("update", h.Update)
			group.POST("updateAddress", h.UpdateAddress) // 更新 Moralis Streams 监听地址
			group.PATCH("updateAddress", h.UpdateAddress)
			group.GET("get", h.Get)
			group.GET("secret", h.GetSecretField)
		}),
		fx.Invoke(func(s *core.AppServer, h *admin.ManagerHandler) {
			group := s.Engine.Group("/api/admin/")
			group.POST("login", h.Login)
			group.GET("logout", h.Logout)
			group.GET("session", h.Session)
			group.GET("list", h.List)
			group.POST("save", h.Save)
			group.POST("enable", h.Enable)
			group.GET("remove", h.Remove)
			group.POST("resetPass", h.ResetPass)
			group.GET("placement/tree", h.PlacementTree)
		}),
		fx.Invoke(func(s *core.AppServer, h *admin.ApiKeyHandler) {
			group := s.Engine.Group("/api/admin/apikey/")
			group.POST("save", h.Save)
			group.GET("list", h.List)
			group.POST("set", h.Set)
			group.GET("remove", h.Remove)
		}),
		fx.Invoke(func(s *core.AppServer, h *admin.UserHandler) {
			group := s.Engine.Group("/api/admin/user/")
			group.GET("list", h.List)
			group.POST("save", h.Save)
			group.GET("remove", h.Remove)
			group.GET("loginLog", h.LoginLog)
			group.POST("resetPass", h.ResetPass)
			group.POST("setAuditor", h.SetAuditor) // 批量设置审核用户
			group.POST("setCs", h.SetCs)           // 批量设置/取消客服
			group.POST("setAllowMint", h.SetAllowMint)
			group.POST("setAllowPlacement", h.SetAllowPlacement) // 批量开通/关闭大小区显示
			group.GET("previewMusicRefundRollback", h.PreviewMusicRefundRollback)
			group.POST("rollbackMusicRefundExploit", h.RollbackMusicRefundExploit)
			group.GET("previewPowerPointRollback", h.PreviewPowerPointRollback)
			group.POST("executePowerPointRollback", h.ExecutePowerPointRollback)
		}),
		fx.Invoke(func(s *core.AppServer, h *admin.ChatRoleHandler) {
			group := s.Engine.Group("/api/admin/role/")
			group.GET("list", h.List)
			group.POST("save", h.Save)
			group.POST("sort", h.Sort)
			group.POST("set", h.Set)
			group.POST("remove", h.Remove)
		}),
		fx.Invoke(func(s *core.AppServer, h *admin.RewardHandler) {
			group := s.Engine.Group("/api/admin/reward/")
			group.GET("list", h.List)
			group.POST("remove", h.Remove)
		}),
		fx.Invoke(func(s *core.AppServer, h *admin.DashboardHandler) {
			group := s.Engine.Group("/api/admin/dashboard/")
			group.GET("stats", h.Stats)
		}),
		fx.Invoke(func(s *core.AppServer, h *handler.ChatModelHandler) {
			group := s.Engine.Group("/api/model/")
			group.GET("list", h.List)
		}),
		fx.Invoke(func(s *core.AppServer, h *admin.ChatModelHandler) {
			group := s.Engine.Group("/api/admin/model/")
			group.POST("save", h.Save)
			group.GET("list", h.List)
			group.POST("set", h.Set)
			group.POST("sort", h.Sort)
			group.GET("remove", h.Remove)
		}),
		fx.Invoke(func(s *core.AppServer, h *handler.PaymentHandler) {
			group := s.Engine.Group("/api/payment/")
			//group.GET("doPay", h.DoPay)
			//group.GET("payWays", h.GetPayWays)
			//group.POST("query", h.OrderQuery)
			//group.POST("qrcode", h.PayQrcode)
			//group.POST("mobile", h.Mobile)
			group.POST("confirmJihuo", h.ConfirmJihuo)
			group.POST("confirmJihuoEx", h.ConfirmJihuoEx)
			group.POST("confirmJihuoGift", h.ConfirmJihuoGift)
			group.POST("confirmJihuoCreditGift", h.ConfirmJihuoCreditGift)
			//group.POST("paySuccess", h.PaySuccess)
			//group.POST("alipay/notify", h.AlipayNotify)
			//group.POST("hupipay/notify", h.HuPiPayNotify)
			//group.POST("payjs/notify", h.PayJsNotify)
		}),
		fx.Invoke(func(h *handler.PaymentHandler) {
			service.RegisterAppConsumeRewardHandler(h)
		}),
		fx.Invoke(func(s *core.AppServer, h *admin.MintHandler) {
			group := s.Engine.Group("/api/admin/mint/")
			group.POST("save", h.Save)
			group.GET("list", h.List)
			group.POST("enable", h.Enable)
			group.POST("sort", h.Sort)
			group.GET("remove", h.Remove)
		}),

		fx.Invoke(func(s *core.AppServer, h *admin.MintOrderHandler) {
			group := s.Engine.Group("/api/admin/mintOrder/")
			group.POST("list", h.List)
			group.GET("close", h.Close)
		}),

		fx.Invoke(func(s *core.AppServer, h *handler.MintOrderHandler) {
			group := s.Engine.Group("/api/mintOrder/")
			group.POST("list", h.List)
			group.POST("buyKj", h.BuyKj)
			group.GET("jsCoin", h.CalcJsCoin)
			group.GET("jsSl", h.CalcSuanli)
			group.GET("days", h.Days)
			group.GET("updateUT", h.UpdateUT)
			group.GET("randomPrise", h.RandomPrise)
			group.GET("tradesPay", h.TradesPay)
			group.GET("autoCancel", h.AutoCancel)
			group.GET("addLev", h.AddLev)
			group.GET("otherLev", h.OtherLev)
			//等级升级
			group.GET("fenHong", h.FenHong)
			//定时任务执行 用户对应等级分红
			group.GET("fenHongTask", h.FenHongTask)

			group.GET("getMintOrderList", h.GetMintOrderList)

			//http://localhost:5678/api/mintOrder/BuyKj
		}),

		fx.Invoke(func(s *core.AppServer, h *admin.ProductHandler) {
			group := s.Engine.Group("/api/admin/product/")
			group.POST("save", h.Save)
			group.GET("list", h.List)
			group.POST("enable", h.Enable)
			group.POST("sort", h.Sort)
			group.GET("remove", h.Remove)
		}),

		fx.Invoke(func(s *core.AppServer, h *admin.OrderHandler) {
			group := s.Engine.Group("/api/admin/order/")
			group.POST("list", h.List)

			group.GET("remove", h.Remove)
			group.GET("paycomplete", h.PayComplete)
		}),

		fx.Invoke(func(s *core.AppServer, h *admin.TradeHandler) {
			group := s.Engine.Group("/api/admin/trade/")
			group.POST("listBuy", h.ListBuy)
			group.GET("cancel", h.CancelOrder)    //管理员取消订单
			group.GET("complete", h.ComfireOrder) //管理员确认订单
			group.POST("listSell", h.ListSell)
			group.GET("remove", h.Remove)
			group.POST("listPowerTrade", h.ListPowerTrade)                     //算力撮合订单列表
			group.POST("listPowerPoolRecharge", h.ListPowerPoolRecharge)       //算力池充值订单列表
			group.POST("confirmPowerTradePayment", h.ConfirmPowerTradePayment) //管理员确认付款到账
			group.POST("listBuyback", h.ListBuyback)                           //算力回购订单列表
			group.POST("confirmBuybackPayment", h.ConfirmBuybackPayment)       //管理员确认回购订单已打款
		}),

		fx.Invoke(func(s *core.AppServer, h *handler.OrderHandler) {
			group := s.Engine.Group("/api/order/")
			group.POST("list", h.List)
			group.GET("getOrderList", h.GetOrderList)
			group.GET("getEndOrderList", h.GetEndOrderList)
			//用户产品的算力收益详情
			group.GET("getOrderItems", h.GetOrderItems)
		}),

		fx.Invoke(func(s *core.AppServer, h *handler.TradeHandler) {
			group := s.Engine.Group("/api/trade/")

			group.POST("list", h.List)
			group.POST("audit/buyback/list", h.AuditBuybackList)       // 审核用户查看算力提现订单列表
			group.POST("audit/buyback/confirm", h.AuditBuybackConfirm) // 审核用户确认回购订单已打款
			group.POST("submitOrder", h.SubmitOrder)                   //提交订单
			group.GET("buyAndSellList", h.BuyAndSellList)              //获取订单数据

			group.GET("powerPool/orders", h.PowerPoolOrderList)  //算力池订单
			group.GET("powerPool/mine", h.MyPowerPoolOrderList)  //我的订单
			group.POST("powerPool/match", h.MatchPowerPoolOrder) //撮合算力订单
			group.GET("powerPool/info", h.GetPowerPoolInfo)      //获取算力池信息

			//平台充值
			group.POST("powerPool/recharge", h.CreatePowerPoolRecharge)        //创建充值订单
			group.GET("powerPool/recharge/order", h.GetPowerPoolRechargeOrder) //获取充值订单信息

			//group.GET("powerPool/recharge/check", h.CheckPowerPoolRechargePayment)  //检查充值订单支付状态（管理员，需要key）
			group.POST("powerPool/recharge/check", h.CheckPowerPoolRechargePayment) //Moralis Webhook 主动推送

			group.GET("powerPool/recharge/checkMy", h.CheckMyRechargePayment) //用户检查自己的充值订单支付状态
			group.GET("powerPool/recharge/mine", h.ListMyPowerPoolRecharges)  //用户最近5笔充值记录
			// 算力提现（5折），历史路径 powerPool/buyback/withdraw 保留兼容

			group.POST("powerPool/withdraw", h.PowerPoolWithdraw)
			group.POST("powerPool/buyback/withdraw", h.PowerPoolWithdraw) // 兼容旧前端

			//c2c撮合
			group.GET("powerTrade/c2c/mine", h.MyPowerTradeOrders) //我的撮合订单
			group.POST("powerTrade/c2c/pay", h.MarkPowerTradePaid) //买家已付款
			group.POST("powerTrade/c2c/confirm", h.ConfirmPowerTradeReceipt)
			group.GET("powerTrade/c2c/tradesPay", h.TradesPay)          //算力撮合验证自动付款
			group.POST("powerTrade/c2c/submit", h.SubmitPowerPoolOrder) //提交算力订单

			// BSC USDT 测试转账接口（测试用）
			group.GET("testBscTransfer", h.TestBscTransfer)

			//区块链自动验证付款

			//取消订单

			//@main.go (489) 完成CancelPowerTradeOrder方法，且需要将对应的算力给增加回去，包括预扣除的50%
			//@main.go (489) 完成CancelPowerTradeOrder方法，且需要将对应的算力给增加回去，包括预扣除的50%
			group.POST("powerTrade/cancel", h.CancelPowerTradeOrder)

			//q2相关订单
			group.POST("waitComfireOrder", h.WaitComfireOrder) //已付款待确认
			group.POST("comfireOrder", h.ComfireOrder)         //已确认
			group.POST("cancel", h.CancelOrder)                //取消订单
			group.POST("appeal", h.AppealOrder)                //虚假付款，申述

			group.GET("destroyedCoin", h.DestroyedCoin) //我的购买订单
			group.GET("buyOrderList", h.BuyOrderList)   //我的购买订单

			group.GET("sellOrderList", h.SellOrderList)           //我的卖出订单
			group.GET("completedOrderList", h.CompletedOrderList) //我的完成订单
			group.GET("ingOrderList", h.IngOrderList)             // 匹配中订单
			group.POST("matchOrder", h.MatchOrder)                //匹配订单

			//k线价格
			group.GET("priceHistory", h.PriceHistory)
			group.GET("sumCoin", h.SumCoin)
			group.GET("swapInfo", h.SwapInfo)
		}),

		fx.Invoke(func(s *core.AppServer, h *handler.MintHandler) {
			group := s.Engine.Group("/api/mint/")
			group.GET("getCoinItems", h.GetCoinItems)
			group.GET("list", h.List)
			group.GET("releaseConfig", h.ReleaseConfig)
		}),
		fx.Invoke(func(s *core.AppServer, h *handler.ProductHandler) {
			group := s.Engine.Group("/api/product/")
			group.GET("list", h.List)
		}),

		fx.Provide(handler.NewInviteHandler),
		fx.Invoke(func(s *core.AppServer, h *handler.InviteHandler) {
			group := s.Engine.Group("/api/invite/")
			group.GET("code", h.Code)
			group.POST("list", h.List)
			group.GET("hits", h.Hits)
		}),

		fx.Provide(admin.NewFunctionHandler),
		fx.Invoke(func(s *core.AppServer, h *admin.FunctionHandler) {
			group := s.Engine.Group("/api/admin/function/")
			group.POST("save", h.Save)
			group.POST("set", h.Set)
			group.GET("list", h.List)
			group.GET("remove", h.Remove)
			group.GET("token", h.GenToken)
		}),

		// 验证码
		fx.Provide(admin.NewCaptchaHandler),
		fx.Invoke(func(s *core.AppServer, h *admin.CaptchaHandler) {
			group := s.Engine.Group("/api/admin/login/")
			group.GET("captcha", h.GetCaptcha)
		}),

		fx.Provide(admin.NewUploadHandler),
		fx.Invoke(func(s *core.AppServer, h *admin.UploadHandler) {
			s.Engine.POST("/api/admin/upload", h.Upload)
		}),

		fx.Provide(handler.NewFunctionHandler),
		fx.Invoke(func(s *core.AppServer, h *handler.FunctionHandler) {
			group := s.Engine.Group("/api/function/")
			group.POST("weibo", h.WeiBo)
			group.POST("zaobao", h.ZaoBao)
			group.POST("dalle3", h.Dall3)
		}),
		fx.Invoke(func(s *core.AppServer, h *admin.ChatHandler) {
			group := s.Engine.Group("/api/admin/chat/")
			group.POST("list", h.List)
			group.POST("message", h.Messages)
			group.GET("history", h.History)
			group.GET("remove", h.RemoveChat)
			group.GET("message/remove", h.RemoveMessage)
		}),
		// 在线客服（用户端）
		fx.Invoke(func(s *core.AppServer, h *handler.CsHandler) {
			group := s.Engine.Group("/api/cs/")
			group.Any("ws", h.Client)
			group.GET("status", h.Status)
			group.GET("unread", h.Unread)
			group.GET("history", h.History)
			group.POST("send", h.Send)
			// 前台指定客服接入（用户 JWT + is_cs）
			group.Any("agent/ws", h.AgentClient)
			group.POST("agent/sessions", h.AgentSessions)
			group.GET("agent/messages", h.AgentMessages)
			group.GET("agent/unread", h.AgentUnread)
			group.POST("agent/reply", h.AgentReply)
			group.GET("agent/close", h.AgentClose)
		}),
		// 在线客服（管理端）
		fx.Invoke(func(s *core.AppServer, h *admin.CsHandler) {
			group := s.Engine.Group("/api/admin/cs/")
			group.Any("ws", h.Client)
			group.POST("sessions", h.Sessions)
			group.GET("messages", h.Messages)
			group.GET("unread", h.Unread)
			group.POST("reply", h.Reply)
			group.GET("close", h.Close)
		}),
		fx.Invoke(func(s *core.AppServer, h *handler.PowerLogHandler) {
			group := s.Engine.Group("/api/powerLog/")
			group.POST("list", h.List)
		}),

		//SOHA明细
		fx.Invoke(func(s *core.AppServer, h *handler.Q2LogHandler) {
			group := s.Engine.Group("/api/q2Log/")
			group.POST("list", h.List)
		}),
		//SOHA钱包总账明细（user.coin，对标 powerLog）
		fx.Invoke(func(s *core.AppServer, h *handler.CoinLogHandler) {
			group := s.Engine.Group("/api/coinLog/")
			group.POST("list", h.List)
		}),
		fx.Invoke(func(s *core.AppServer, h *admin.PowerLogHandler) {
			group := s.Engine.Group("/api/admin/powerLog/")
			group.POST("list", h.List)
		}),

		fx.Invoke(func(s *core.AppServer, h *admin.Q2LogHandler) {
			group := s.Engine.Group("/api/admin/q2Log/")
			group.POST("list", h.List)
		}),
		fx.Invoke(func(s *core.AppServer, h *admin.CoinLogHandler) {
			group := s.Engine.Group("/api/admin/coinLog/")
			group.POST("list", h.List)
		}),
		fx.Invoke(func(s *core.AppServer, h *admin.MusicHandler) {
			group := s.Engine.Group("/api/admin/music/")
			group.GET("failed-stats", h.FailedStats)
			group.POST("repair-history", h.RepairHistory)
		}),
		fx.Provide(admin.NewMenuHandler),
		fx.Invoke(func(s *core.AppServer, h *admin.MenuHandler) {
			group := s.Engine.Group("/api/admin/menu/")
			group.POST("save", h.Save)
			group.GET("list", h.List)
			group.POST("enable", h.Enable)
			group.POST("sort", h.Sort)
			group.GET("remove", h.Remove)
		}),
		fx.Provide(handler.NewMenuHandler),
		fx.Invoke(func(s *core.AppServer, h *handler.MenuHandler) {
			group := s.Engine.Group("/api/menu/")
			group.GET("list", h.List)
		}),
		//音乐生成器
		fx.Invoke(func(s *core.AppServer, h *handler.SunoHandler) {
			group := s.Engine.Group("/api/suno")
			group.POST("/generate", h.Generate)
			group.GET("/status/:id", h.GetStatus)
			group.GET("/list", h.GetList) // 添加获取列表接口
			group.GET("/download/:id", h.Download)
		}),

		// 手机端 AI搜（DeepSeek Search）
		fx.Invoke(func(s *core.AppServer, h *handler.AiSearchHandler) {
			group := s.Engine.Group("/api/ai-search")
			group.POST("/search", h.Search)
			group.POST("/fetch", h.Fetch)
		}),

		//视频生成器
		fx.Invoke(func(s *core.AppServer, h *handler.VideoHandler) {
			group := s.Engine.Group("/api/video/")
			group.POST("generate", h.Generate)
			group.GET("status/:id", h.GetStatus)
		}),

		// Sora2 视频生成器
		fx.Invoke(func(s *core.AppServer, h *handler.Sora2Handler) {
			group := s.Engine.Group("/api/sora2")
			group.POST("generate", h.Generate)            // 生成视频
			group.GET("list", h.List)                     // 获取视频列表
			group.POST("favorite", h.ToggleFavorite)      // 切换收藏状态
			group.GET("remove", h.Remove)                 // 删除视频
			group.GET("progress/:task_id", h.GetProgress) // 查询任务进度
		}),

		// 短剧（教学）创作
		fx.Invoke(func(s *core.AppServer, h *handler.TeachingHandler) {
			group := s.Engine.Group("/api/teaching")
			group.POST("/script", h.Script)
			group.POST("/tts", h.TTS)
			group.POST("/image-prompts", h.ImagePrompts)
			group.POST("/images", h.RenderImages)
			group.GET("/record/list", h.RecordList)
			group.POST("/record/render-video", h.RecordRenderVideo)
			group.GET("/record/:id", h.RecordGet)
			group.POST("/record/save", h.RecordSave)

			adminGroup := s.Engine.Group("/api/admin/teaching")
			adminGroup.GET("/tts-quota", h.AdminTTSQuota)
		}),

		// 添加通用WebSocket路由
		fx.Invoke(func(s *core.AppServer, h *chatimpl.ChatHandler) {
			s.Engine.GET("/api/ws", h.ChatHandle)
		}),

		fx.Invoke(func(s *core.AppServer, db *gorm.DB) {
			go func() {
				err := s.Run(db)
				if err != nil {
					log.Fatal(err)
				}
			}()
		}),
		fx.Provide(NewAppLifeCycle),
		// 注册生命周期回调函数
		fx.Invoke(func(lifecycle fx.Lifecycle, lc *AppLifecycle) {
			lifecycle.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					return lc.OnStart(ctx)
				},
				OnStop: func(ctx context.Context) error {
					return lc.OnStop(ctx)
				},
			})
		}),
	)
	// 启动应用程序
	go func() {
		if err := app.Start(context.Background()); err != nil {
			log.Fatal(err)
		}
	}()

	// 监听退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 关闭应用程序
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Stop(ctx); err != nil {
		log.Fatal(err)
	}

}
