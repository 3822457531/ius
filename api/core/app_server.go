package core

import (
	"bytes"
	"chatplus/core/types"
	"chatplus/store/model"
	"chatplus/utils"
	"chatplus/utils/resp"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/nfnt/resize"
	"gorm.io/gorm"
)

type AppServer struct {
	Debug        bool
	Config       *types.AppConfig
	Engine       *gin.Engine
	ChatContexts *types.LMap[string, []types.Message] // 聊天上下文 Map [chatId] => []Message

	SysConfig *types.SystemConfig // system config cache

	// 保存 Websocket 会话 UserId, 每个 UserId 只能连接一次
	// 防止第三方直接连接 socket 调用 OpenAI API
	ChatSession   *types.LMap[string, *types.ChatSession] //map[sessionId]UserId
	ChatClients   *types.LMap[string, *types.WsClient]    // map[sessionId]Websocket 连接集合
	ReqCancelFunc *types.LMap[string, context.CancelFunc] // HttpClient 请求取消 handle function
}

func NewServer(appConfig *types.AppConfig) *AppServer {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard
	return &AppServer{
		Debug:         false,
		Config:        appConfig,
		Engine:        gin.Default(),
		ChatContexts:  types.NewLMap[string, []types.Message](),
		ChatSession:   types.NewLMap[string, *types.ChatSession](),
		ChatClients:   types.NewLMap[string, *types.WsClient](),
		ReqCancelFunc: types.NewLMap[string, context.CancelFunc](),
	}
}

func (s *AppServer) Init(debug bool, client *redis.Client) {
	if debug { // 调试模式允许跨域请求 API
		s.Debug = debug
		logger.Info("Enabled debug mode")
	}
	s.Engine.Use(corsMiddleware())
	s.Engine.Use(staticResourceMiddleware())
	s.Engine.Use(authorizeMiddleware(s, client))
	s.Engine.Use(parameterHandlerMiddleware())
	s.Engine.Use(errorHandler)
	// 添加静态资源访问
	s.Engine.Static("/static", s.Config.StaticDir)
}

func (s *AppServer) Run(db *gorm.DB) error {
	// load system configs
	var sysConfig model.Config
	res := db.Where("marker", "system").First(&sysConfig)
	if res.Error != nil {
		return res.Error
	}
	err := utils.JsonDecode(sysConfig.Config, &s.SysConfig)
	if err != nil {
		return err
	}
	// 旧库兼容：无 transfer_fee_percent 时默认 5%；无 enabled_transfer_fee 时默认开启手续费
	var rawSystem map[string]interface{}
	if e := utils.JsonDecode(sysConfig.Config, &rawSystem); e == nil {
		if _, ok := rawSystem["transfer_fee_percent"]; !ok {
			s.SysConfig.TransferFeePercent = 5
		}
		if _, ok := rawSystem["enabled_transfer_fee"]; !ok {
			s.SysConfig.EnabledTransferFee = true
		}
		if _, ok := rawSystem["transfer_fee_to_pool_percent"]; !ok {
			s.SysConfig.TransferFeeToPoolPercent = 100
		} else if s.SysConfig.TransferFeeToPoolPercent == 50 {
			// 产品调整：旧默认 50% 升级为 100% 全部回流认购池
			s.SysConfig.TransferFeeToPoolPercent = 100
		}
		if _, ok := rawSystem["team_reward_measure_layer_depth"]; !ok {
			s.SysConfig.TeamRewardMeasureLayerDepth = 4
		}
		if _, ok := rawSystem["team_reward_layer4_width_limit"]; !ok {
			s.SysConfig.TeamRewardLayer4WidthLimit = 8
		}
		if _, ok := rawSystem["team_reward_weekly_cap"]; !ok {
			s.SysConfig.TeamRewardWeeklyCap = types.DefaultTeamRewardWeeklyCap()
		}
		if _, ok := rawSystem["team_reward_pro"]; !ok {
			s.SysConfig.TeamRewardPro = 0.2
		}
		if _, ok := rawSystem["enabled_soha_market"]; !ok {
			s.SysConfig.EnabledSohaMarket = true
		}
		if _, ok := rawSystem["enabled_soha_kline"]; !ok {
			s.SysConfig.EnabledSohaKline = true
		}
		if _, ok := rawSystem["enabled_register_ip_limit"]; !ok {
			s.SysConfig.EnabledRegisterIpLimit = true
		}
		if _, ok := rawSystem["register_ip_limit"]; !ok {
			s.SysConfig.RegisterIpLimit = 3
		}
		if _, ok := rawSystem["enabled_activation"]; !ok {
			s.SysConfig.EnabledActivation = true
		}
		if _, ok := rawSystem["enabled_activation_user_limit"]; !ok {
			s.SysConfig.EnabledActivationUserLimit = true
		}
		if _, ok := rawSystem["activation_user_limit"]; !ok {
			s.SysConfig.ActivationUserLimit = 3
		}
		if _, ok := rawSystem["jihuo_ex_min_power"]; !ok {
			s.SysConfig.JihuoExMinPower = 1000
		}
		if _, ok := rawSystem["member_normal_min_power"]; !ok {
			s.SysConfig.MemberNormalMinPower = 1000
		}
		if _, ok := rawSystem["member_normal_max_power"]; !ok {
			s.SysConfig.MemberNormalMaxPower = 9999
		}
		if _, ok := rawSystem["member_vip_min_power"]; !ok {
			s.SysConfig.MemberVipMinPower = 10000
		}
		if _, ok := rawSystem["legacy_vip_min_power"]; !ok {
			s.SysConfig.LegacyVipMinPower = 5000
		}
		if _, ok := rawSystem["legacy_vip_team_reward_pro"]; !ok {
			s.SysConfig.LegacyVipTeamRewardPro = 0.1
		}
		if _, ok := rawSystem["swap_power_price"]; !ok {
			s.SysConfig.SwapPowerPrice = 0.01
		}
		if _, ok := rawSystem["soha_guide_base_price"]; !ok {
			s.SysConfig.SohaGuideBasePrice = 0.1
		}
		if _, ok := rawSystem["kj_zj_pro"]; !ok {
			s.SysConfig.KjZjPro = 0.3
		}
		if _, ok := rawSystem["kj_team_reward_pro"]; !ok {
			s.SysConfig.KjTeamRewardPro = 0.2
		}
		if _, ok := rawSystem["zj_static_pro"]; !ok {
			s.SysConfig.ZjStaticPro = 0.2
		}
		if _, ok := rawSystem["mgmt_static_pro"]; !ok {
			s.SysConfig.MgmtStaticPro = 0.1
		}
		if _, ok := rawSystem["invest_multiplier"]; !ok {
			s.SysConfig.InvestMultiplier = types.DefaultInvestMultiplier()
		} else {
			s.SysConfig.InvestMultiplier = s.SysConfig.InvestMultiplier.Normalize()
		}
		if _, ok := rawSystem["zj_static_pro"]; !ok {
			s.SysConfig.ZjStaticPro = 0.2
		}
		if _, ok := rawSystem["mgmt_static_pro"]; !ok {
			s.SysConfig.MgmtStaticPro = 0.1
		}
		if _, ok := rawSystem["credit_leverage"]; !ok {
			s.SysConfig.CreditLeverage = 5
		}
		if _, ok := rawSystem["credit_direct_rate"]; !ok {
			s.SysConfig.CreditDirectRate = 0.5
		}
		if _, ok := rawSystem["credit_legacy_min_power"]; !ok {
			s.SysConfig.CreditLegacyMinPower = 10000
		}
		if _, ok := rawSystem["credit_legacy_max"]; !ok {
			s.SysConfig.CreditLegacyMax = 50000
		}
		// credit_legacy_cutoff 故意不给默认值：留空 = 不启用老会员分段，
		// 避免升级后自动把所有人切成老会员。
		if _, ok := rawSystem["invest_multiplier"]; !ok {
			s.SysConfig.InvestMultiplier = types.DefaultInvestMultiplier()
		} else {
			s.SysConfig.InvestMultiplier = s.SysConfig.InvestMultiplier.Normalize()
		}
		if _, ok := rawSystem["enabled_app_image"]; !ok {
			s.SysConfig.EnabledAppImage = true
		}
		if _, ok := rawSystem["enabled_app_music"]; !ok {
			s.SysConfig.EnabledAppMusic = true
		}
		if _, ok := rawSystem["enabled_app_video"]; !ok {
			s.SysConfig.EnabledAppVideo = false
		}
		if _, ok := rawSystem["enabled_app_ppt"]; !ok {
			s.SysConfig.EnabledAppPpt = true
		}
		if _, ok := rawSystem["enabled_app_logo"]; !ok {
			s.SysConfig.EnabledAppLogo = true
		}
		if _, ok := rawSystem["enabled_app_drama"]; !ok {
			s.SysConfig.EnabledAppDrama = true
		}
		if _, ok := rawSystem["teaching_script_power"]; !ok {
			s.SysConfig.TeachingScriptPower = 10
		}
		if _, ok := rawSystem["teaching_tts_power_per_segment"]; !ok {
			s.SysConfig.TeachingTtsPowerPerSegment = 5
		}
		if _, ok := rawSystem["prompt_power"]; !ok {
			s.SysConfig.PromptPower = 1
		}
		if _, ok := rawSystem["music_power"]; !ok {
			s.SysConfig.MusicPower = 100
		}
		if _, ok := rawSystem["ai_search_search_power"]; !ok {
			s.SysConfig.AiSearchSearchPower = 1
		}
		if _, ok := rawSystem["ai_search_detail_power"]; !ok {
			s.SysConfig.AiSearchDetailPower = 10
		}
	}
	logger.Infof("http://%s", s.Config.Listen)
	return s.Engine.Run(s.Config.Listen)
}

// 全局异常处理
func errorHandler(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("Handler Panic: %v", r)
			debug.PrintStack()
			c.JSON(http.StatusOK, types.BizVo{Code: types.Failed, Message: types.ErrorMsg})
			c.Abort()
		}
	}()
	//加载完 defer recover，继续后续接口调用
	c.Next()
}

// 跨域中间件设置
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			// 设置允许的请求源
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
			//允许跨域设置可以返回其他子段，可以自定义字段
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, Content-Type, Chat-Token, Admin-Authorization")
			// 允许浏览器（客户端）可以解析的头部 （重要）
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers")
			//设置缓存时间
			c.Header("Access-Control-Max-Age", "172800")
			//允许客户端传递校验信息比如 cookie (重要)
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if method == http.MethodOptions {
			c.JSON(http.StatusOK, "ok!")
		}

		defer func() {
			if err := recover(); err != nil {
				logger.Info("Panic info is: %v", err)
			}
		}()

		c.Next()
	}
}

const invalidTokenMsg = "invalid token"

// 用户授权验证
func authorizeMiddleware(s *AppServer, client *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string
		isAdminApi := strings.Contains(c.Request.URL.Path, "/api/admin/")
		if isAdminApi { // 后台管理 API
			tokenString = c.GetHeader(types.AdminAuthHeader)
			// WebSocket 无法自定义 Header，客服管理端走 query token
			if tokenString == "" && c.Request.URL.Path == "/api/admin/cs/ws" {
				tokenString = c.Query("token")
			}
		} else if c.Request.URL.Path == "/api/chat/new" ||
			c.Request.URL.Path == "/api/cs/ws" ||
			c.Request.URL.Path == "/api/cs/agent/ws" {
			tokenString = c.Query("token")
		} else {
			tokenString = c.GetHeader(types.UserAuthHeader)
			// <video>/<img> 无法自定义 Header，AI搜媒体流走 query token
			if tokenString == "" && strings.HasPrefix(c.Request.URL.Path, "/api/ai-search/") {
				tokenString = c.Query("token")
			}
		}

		if tokenString == "" {
			if needLogin(c) {
				resp.NotAuth(c, invalidTokenMsg)
				c.Abort()
			}
			return
		}

		secret := s.Config.Session.SecretKey
		if isAdminApi {
			secret = s.Config.AdminSession.SecretKey
		}
		// 空密钥一律拒绝，杜绝 HS256 空密钥伪造
		if len(strings.TrimSpace(secret)) < utils.MinJWTSecretLen {
			if needLogin(c) {
				resp.NotAuth(c, invalidTokenMsg)
				c.Abort()
			}
			return
		}

		parser := jwt.NewParser(jwt.WithValidMethods([]string{"HS256"}))
		token, err := parser.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil || token == nil || !token.Valid {
			if needLogin(c) {
				resp.NotAuth(c, invalidTokenMsg)
				c.Abort()
			}
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			if needLogin(c) {
				resp.NotAuth(c, invalidTokenMsg)
				c.Abort()
			}
			return
		}

		expr := utils.IntValue(utils.InterfaceToString(claims["expired"]), 0)
		if expr > 0 && int64(expr) < time.Now().Unix() {
			if needLogin(c) {
				resp.NotAuth(c, invalidTokenMsg)
				c.Abort()
			}
			return
		}
		// 后台禁止永久令牌（expired=0）
		if isAdminApi && expr <= 0 {
			if needLogin(c) {
				resp.NotAuth(c, invalidTokenMsg)
				c.Abort()
			}
			return
		}

		if isAdminApi {
			jti := strings.TrimSpace(utils.InterfaceToString(claims["jti"]))
			adminID := utils.IntValue(utils.InterfaceToString(claims["admin_id"]), 0)
			if adminID <= 0 {
				// 兼容旧字段名，但仍必须走 jti 白名单
				adminID = utils.IntValue(utils.InterfaceToString(claims["user_id"]), 0)
			}
			if jti == "" || adminID <= 0 {
				if needLogin(c) {
					resp.NotAuth(c, invalidTokenMsg)
					c.Abort()
				}
				return
			}
			bound, err := client.Get(context.Background(), utils.AdminSessKey(jti)).Result()
			if err != nil || utils.IntValue(bound, 0) != adminID {
				if needLogin(c) {
					resp.NotAuth(c, invalidTokenMsg)
					c.Abort()
				}
				return
			}
			c.Set(types.LoginUserID, adminID)
			c.Set("ADMIN_JTI", jti)
			return
		}

		userID := claims["user_id"]
		key := fmt.Sprintf("users/%v", userID)
		if _, err := client.Get(context.Background(), key).Result(); err != nil {
			if needLogin(c) {
				resp.NotAuth(c, invalidTokenMsg)
				c.Abort()
			}
			return
		}
		c.Set(types.LoginUserID, userID)
	}
}

func needLogin(c *gin.Context) bool {
	if c.Request.URL.Path == "/api/user/login" ||
		c.Request.URL.Path == "/api/user/resetPass" ||
		c.Request.URL.Path == "/api/admin/login" ||
		c.Request.URL.Path == "/api/admin/login/captcha" ||
		c.Request.URL.Path == "/api/user/register" ||
		c.Request.URL.Path == "/api/chat/history" ||
		c.Request.URL.Path == "/api/chat/detail" ||
		c.Request.URL.Path == "/api/chat/list" ||
		c.Request.URL.Path == "/api/role/list" ||
		c.Request.URL.Path == "/api/model/list" ||
		c.Request.URL.Path == "/api/mj/imgWall" ||
		c.Request.URL.Path == "/api/mj/client" ||
		c.Request.URL.Path == "/api/mj/notify" ||
		c.Request.URL.Path == "/api/invite/hits" ||
		c.Request.URL.Path == "/api/sd/imgWall" ||
		c.Request.URL.Path == "/api/sd/client" ||
		c.Request.URL.Path == "/api/config/get" ||
		c.Request.URL.Path == "/api/config/locale" ||
		c.Request.URL.Path == "/api/mint/list" ||
		c.Request.URL.Path == "/api/product/list" ||
		c.Request.URL.Path == "/api/menu/list" ||
		c.Request.URL.Path == "/api/mintOrder/jsCoin" ||
		c.Request.URL.Path == "/api/mintOrder/jsSl" ||
		c.Request.URL.Path == "/api/mintOrder/days" ||
		c.Request.URL.Path == "/api/mintOrder/updateUT" ||
		c.Request.URL.Path == "/api/mintOrder/randomPrise" ||
		c.Request.URL.Path == "/api/mintOrder/tradesPay" ||
		c.Request.URL.Path == "/api/trade/powerTrade/tradesPay" ||
		c.Request.URL.Path == "/api/trade/testBscTransfer" ||
		c.Request.URL.Path == "/api/trade/powerPool/recharge/check" ||
		c.Request.URL.Path == "/api/mintOrder/autoCancel" ||
		c.Request.URL.Path == "/api/mintOrder/addLev" ||
		c.Request.URL.Path == "/api/mintOrder/otherLev" ||
		c.Request.URL.Path == "/api/mintOrder/fenHong" ||
		c.Request.URL.Path == "/api/mintOrder/fenHongTask" ||
		strings.HasPrefix(c.Request.URL.Path, "/api/test") ||
		strings.HasPrefix(c.Request.URL.Path, "/api/function/") ||
		strings.HasPrefix(c.Request.URL.Path, "/api/sms/") ||
		strings.HasPrefix(c.Request.URL.Path, "/api/captcha/") ||
		strings.HasPrefix(c.Request.URL.Path, "/api/payment/") ||
		strings.HasPrefix(c.Request.URL.Path, "/static/") {
		return false
	}
	return true
}

// 统一参数处理
func parameterHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// GET 参数处理
		params := c.Request.URL.Query()
		for key, values := range params {
			for i, value := range values {
				params[key][i] = strings.TrimSpace(value)
			}
		}
		// update get parameters
		c.Request.URL.RawQuery = params.Encode()
		// skip file upload requests
		contentType := c.Request.Header.Get("Content-Type")
		if strings.Contains(contentType, "multipart/form-data") {
			c.Next()
			return
		}

		if strings.Contains(contentType, "application/json") {
			// process POST JSON request body
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.Next()
				return
			}

			// 还原请求体
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			// 将请求体解析为 JSON
			var jsonData map[string]interface{}
			if err := c.ShouldBindJSON(&jsonData); err != nil {
				c.Next()
				return
			}

			// 对 JSON 数据中的字符串值去除两端空格
			trimJSONStrings(jsonData)
			// 更新请求体
			c.Request.Body = io.NopCloser(bytes.NewBufferString(utils.JsonEncode(jsonData)))
		}

		c.Next()
	}
}

// 递归对 JSON 数据中的字符串值去除两端空格
func trimJSONStrings(data interface{}) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			switch valueType := value.(type) {
			case string:
				v[key] = strings.TrimSpace(valueType)
			case map[string]interface{}, []interface{}:
				trimJSONStrings(value)
			}
		}
	case []interface{}:
		for i, value := range v {
			switch valueType := value.(type) {
			case string:
				v[i] = strings.TrimSpace(valueType)
			case map[string]interface{}, []interface{}:
				trimJSONStrings(value)
			}
		}
	}
}

// 静态资源中间件
func staticResourceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		url := c.Request.URL.String()
		// 拦截生成缩略图请求
		if strings.HasPrefix(url, "/static/") && strings.Contains(url, "?imageView2") {
			r := strings.SplitAfter(url, "imageView2")
			size := strings.Split(r[1], "/")
			if len(size) != 8 {
				c.String(http.StatusNotFound, "invalid thumb args")
				return
			}
			with := utils.IntValue(size[3], 0)
			height := utils.IntValue(size[5], 0)
			quality := utils.IntValue(size[7], 75)

			// 打开图片文件
			filePath := strings.TrimLeft(c.Request.URL.Path, "/")
			file, err := os.Open(filePath)
			if err != nil {
				c.String(http.StatusNotFound, "Image not found")
				return
			}
			defer file.Close()

			// 解码图片
			img, _, err := image.Decode(file)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error decoding image")
				return
			}

			var newImg image.Image
			if height == 0 || with == 0 {
				// 固定宽度，高度自适应
				newImg = resize.Resize(uint(with), uint(height), img, resize.Lanczos3)
			} else {
				// 生成缩略图
				newImg = resize.Thumbnail(uint(with), uint(height), img, resize.Lanczos3)
			}
			var buffer bytes.Buffer
			err = jpeg.Encode(&buffer, newImg, &jpeg.Options{Quality: quality})
			if err != nil {
				log.Fatal(err)
			}

			// 设置图片缓存有效期为一年 (365天)
			c.Header("Cache-Control", "max-age=31536000, public")
			// 直接输出图像数据流
			c.Data(http.StatusOK, "image/jpeg", buffer.Bytes())
			c.Abort() // 中断请求
		}
		c.Next()
	}
}
