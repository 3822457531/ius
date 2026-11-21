package types

import (
	"fmt"
	"strings"
	"time"
)

type AppConfig struct {
	Path           string `toml:"-"`
	Listen         string
	Session        Session
	AdminSession   Session
	ProxyURL       string
	MysqlDns       string                  // mysql 连接地址
	StaticDir      string                  // 静态资源目录
	StaticUrl      string                  // 静态资源 URL
	Redis          RedisConfig             // redis 连接信息
	ApiConfig      ChatPlusApiConfig       // ChatPlus API authorization configs
	SMS            SMSConfig               // send mobile message config
	OSS            OSSConfig               // OSS config
	MjProxyConfigs []MjProxyConfig         // MJ proxy config
	MjPlusConfigs  []MjPlusConfig          // MJ plus config
	WeChatBot      bool                    // 是否启用微信机器人
	SdConfigs      []StableDiffusionConfig // sd AI draw service pool
	DigiConfig     []DigiConfig            //数字人配置
	XXLConfig      XXLConfig
	AlipayConfig   AlipayConfig
	HuPiPayConfig  HuPiPayConfig
	SmtpConfig     SmtpConfig // 邮件发送配置
	JPayConfig     JPayConfig // payjs 支付配置
	SunoConfig     SunoConfig
	AiSearchConfig AiSearchConfig // 手机端 AI搜（GramJS 侧车）
	Bsc            BscConfig      // BSC 链上转账配置
	FFmpeg FFmpegConfig
}

// FFmpegConfig 文字故事/教学视频合成所需的 ffmpeg、ffprobe 可执行文件路径
type FFmpegConfig struct {
	FFmpegPath  string // 空则使用 PATH 中的 ffmpeg
	FFprobePath string // 空则使用 PATH 中的 ffprobe
}

type SmtpConfig struct {
	Host     string
	Port     int
	AppName  string // 应用名称
	From     string // 发件人邮箱地址
	Password string // 发件人邮箱密码
}

type ChatPlusApiConfig struct {
	ApiURL string
	AppId  string
	Token  string
}

type MjProxyConfig struct {
	Enabled bool
	ApiURL  string // api 地址
	Mode    string // 绘画模式，可选值：fast/turbo/relax
	ApiKey  string
}

type StableDiffusionConfig struct {
	Enabled bool
	Model   string // 模型名称
	ApiURL  string
	ApiKey  string
}

type MjPlusConfig struct {
	Enabled bool   // 如果启用了 MidJourney Plus，将会自动禁用原生的MidJourney服务
	ApiURL  string // api 地址
	Mode    string // 绘画模式，可选值：fast/turbo/relax
	ApiKey  string
}

type DigiConfig struct {
	Enabled bool   // 如果启用了 MidJourney Plus，将会自动禁用原生的MidJourney服务
	ApiURL  string // api 地址
	ApiKey  string
}

type SunoConfig struct {
	Enabled bool   // 如果启用了 MidJourney Plus，将会自动禁用原生的MidJourney服务
	ApiURL  string // api 地址
	ApiKey  string
}

// AiSearchConfig 手机端 AI搜：Go 鉴权后转发到本地 GramJS 侧车（api/service/aisearch）
type AiSearchConfig struct {
	Enabled    bool
	ServiceURL string // 侧车地址，如 http://127.0.0.1:5690
	TimeoutSec int    // JSON 请求超时；媒体流单独更长
}

type AlipayConfig struct {
	Enabled         bool   // 是否启用该支付通道
	SandBox         bool   // 是否沙盒环境
	AppId           string // 应用 ID
	UserId          string // 支付宝用户 ID
	PrivateKey      string // 用户私钥文件路径
	PublicKey       string // 用户公钥文件路径
	AlipayPublicKey string // 支付宝公钥文件路径
	RootCert        string // Root 秘钥路径
	NotifyURL       string // 异步通知回调
	ReturnURL       string // 支付成功返回地址
}

type HuPiPayConfig struct { //虎皮椒第四方支付配置
	Enabled   bool   // 是否启用该支付通道
	Name      string // 支付名称，如：wechat/alipay
	AppId     string // App ID
	AppSecret string // app 密钥
	ApiURL    string // 支付网关
	NotifyURL string // 异步通知回调
	ReturnURL string // 支付成功返回地址
}

// JPayConfig PayJs 支付配置
type JPayConfig struct {
	Enabled    bool
	Name       string // 支付名称，默认 wechat
	AppId      string // 商户 ID
	PrivateKey string // 私钥
	ApiURL     string // API 网关
	NotifyURL  string // 异步回调地址
	ReturnURL  string // 支付成功返回地址
}

type XXLConfig struct { // XXL 任务调度配置
	Enabled      bool
	ServerAddr   string
	ExecutorIp   string
	ExecutorPort string
	AccessToken  string
	RegistryKey  string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

func (c RedisConfig) Url() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type Platform string

const OpenAI = Platform("OpenAI")
const DeepSeek = Platform("DeepSeek")
const Azure = Platform("Azure")
const ChatGLM = Platform("ChatGLM")
const Baidu = Platform("Baidu")
const XunFei = Platform("XunFei")
const QWen = Platform("QWen")

// BscConfig BSC 链上 USDT 转账相关配置
type BscConfig struct {
	Enabled          bool     `toml:"enabled" json:"enabled"`                       // 是否启用链上分账
	RpcURL           string   `toml:"rpc_url" json:"rpc_url"`                       // BSC RPC 节点地址（主节点，向后兼容）
	RpcURLs          []string `toml:"rpc_urls" json:"rpc_urls"`                     // BSC RPC 节点地址列表（多个节点轮询，提高稳定性）
	MoralisApiKey    string   `toml:"moralis_api_key" json:"moralis_api_key"`       // Moralis API Key（优先使用 Moralis RPC，更稳定快速）
	BscScanApiKey    string   `toml:"bscscan_api_key" json:"bscscan_api_key"`       // Etherscan API Key（优先使用，更稳定，V2 API 统一支持多链）
	ChainID          int64    `toml:"chain_id" json:"chain_id"`                     // 链 ID，主网为 56
	UsdtContract     string   `toml:"usdt_contract" json:"usdt_contract"`           // BSC 上 USDT 合约地址
	FromPrivateKey   string   `toml:"from_private_key" json:"from_private_key"`     // 平台收款地址私钥（建议使用环境变量注入）
	PowerPoolAddress string   `toml:"power_pool_address" json:"power_pool_address"` // 算力池地址（70%）
	BuybackAddress   string   `toml:"buyback_address" json:"buyback_address"`       // 回购池地址（25%）
	PlatformAddress  string   `toml:"platform_address" json:"platform_address"`     // 平台抽成地址（5%）
	MainAddress      string   `toml:"main_address" json:"main_address"`             // 平台主地址
}

// TeamRewardWeeklyCapConfig 团队奖励周封顶（算力），key 对应安置层级：vip=10层，d1～d9=1～9钻
type TeamRewardWeeklyCapConfig struct {
	Vip float64 `json:"vip,omitempty"` // 激活/VIP（10 层）
	D1  float64 `json:"d1,omitempty"`  // 1 钻（20 层）
	D2  float64 `json:"d2,omitempty"`
	D3  float64 `json:"d3,omitempty"`
	D4  float64 `json:"d4,omitempty"`
	D5  float64 `json:"d5,omitempty"`
	D6  float64 `json:"d6,omitempty"`
	D7  float64 `json:"d7,omitempty"`
	D8  float64 `json:"d8,omitempty"`
	D9  float64 `json:"d9,omitempty"` // 9 钻（100 层）
}

// DefaultTeamRewardWeeklyCap 后台未配置时的默认周封顶
func DefaultTeamRewardWeeklyCap() TeamRewardWeeklyCapConfig {
	return TeamRewardWeeklyCapConfig{
		Vip: 30000,
		D1:  100000,
		D2:  200000,
		D3:  300000,
		D4:  500000,
		D5:  1000000,
		D6:  2000000,
		D7:  3000000,
		D8:  5000000,
		D9:  10000000,
	}
}

type SystemConfig struct {
	Title         string  `json:"title,omitempty"`
	AdminTitle    string  `json:"admin_title,omitempty"`
	Logo          string  `json:"logo,omitempty"`
	InitPower     float64 `json:"init_power,omitempty"`      // 新用户注册赠送算力值
	DailyPower    float64 `json:"daily_power,omitempty"`     // 每日赠送算力
	InvitePower   float64 `json:"invite_power,omitempty"`    // 邀请新用户赠送算力值

	// 签到福利分段赠送算力（按注册天数），与 daily_power（系统自动补算力）独立
	SignInReward0_30   float64 `json:"sign_in_reward_0_30,omitempty"`    // 0-30 天
	SignInReward31_60  float64 `json:"sign_in_reward_31_60,omitempty"`   // 31-60 天
	SignInReward61_90  float64 `json:"sign_in_reward_61_90,omitempty"`   // 61-90 天
	SignInReward90Plus float64 `json:"sign_in_reward_90_plus,omitempty"` // 90 天以后，0 表示不可领取

	// confirmJihuoEx 自定义套餐：最低单笔投入算力；0 表示不限制（默认 1000）
	JihuoExMinPower float64 `json:"jihuo_ex_min_power,omitempty"`
	// JihuoExTeamRewardMinPower 已废弃，保留 JSON 兼容，不再参与团队奖逻辑
	JihuoExTeamRewardMinPower float64 `json:"jihuo_ex_team_reward_min_power,omitempty"`

	// 会员档位投入区间（普通会员 / VIP）
	MemberNormalMinPower float64 `json:"member_normal_min_power,omitempty"` // 默认 1000
	MemberNormalMaxPower float64 `json:"member_normal_max_power,omitempty"` // 默认 9999
	MemberVipMinPower    float64 `json:"member_vip_min_power,omitempty"`    // 默认 10000

	// TeamRewardNormalCutoff 普通会员小区奖励截止：存在 sales>=legacy_vip_min_power 且 created_at 严格早于该时刻的订单才可收团队奖；之后仅 VIP/历史 VIP
	TeamRewardNormalCutoff string `json:"team_reward_normal_cutoff,omitempty"`
	// LegacyVipMinPower 历史 VIP 最低投入（默认 5000，且首笔 qualifying 订单早于截止）
	LegacyVipMinPower float64 `json:"legacy_vip_min_power,omitempty"`
	// LegacyVipTeamRewardPro 历史 VIP 团队奖励比例（默认 0.1）
	LegacyVipTeamRewardPro float64 `json:"legacy_vip_team_reward_pro,omitempty"`

	VipMonthPower float64 `json:"vip_month_power,omitempty"` // VIP 会员每月赠送的算力值  转账最小额度

	// EnabledTransferFee 是否开启用户间转账手续费；关闭时按 0% 收取，不再额外扣费
	EnabledTransferFee bool `json:"enabled_transfer_fee"`

	// TransferFeePercent 用户间转账手续费比例（0～100，例如 5 表示按转出金额另扣 5%），收款方实收 = 用户填写的转出全额；发送方支出 = 转出额 + 手续费
	// 不使用 omitempty，避免值为 0 时序列化丢字段导致启动时被误判为「未配置」
	TransferFeePercent float64 `json:"transfer_fee_percent"`

	// TransferFeeToPoolPercent 转账/赠送手续费回流认购池比例（0～100，默认 100）；静默注入 chatgpt_power_pool.total_power，不单独记用户可见流水
	TransferFeeToPoolPercent float64 `json:"transfer_fee_to_pool_percent"`

	// WebSearchApiBase DeepSeek 联网搜索/抓取网关，默认 https://api.deepseek.com
	WebSearchApiBase string `json:"web_search_api_base,omitempty"`
	// WebSearchApiKey 可选；留空则使用 API KEY 管理中启用的 DeepSeek 聊天 Key
	WebSearchApiKey string `json:"web_search_api_key,omitempty"`

	// AiSearchSearchPower AI搜每次关键词检索消耗算力（默认 1；0 表示不扣）
	AiSearchSearchPower int `json:"ai_search_search_power"`
	// AiSearchDetailPower AI搜点击资源详情消耗算力（默认 10；0 表示不扣）
	AiSearchDetailPower int `json:"ai_search_detail_power"`

	RegisterWays    []string `json:"register_ways,omitempty"`    // 注册方式：支持手机，邮箱注册，账号密码注册
	EnabledRegister bool     `json:"enabled_register,omitempty"` // 是否开放注册
	// EnabledRegisterIpLimit 是否启用同 IP 注册数量限制
	EnabledRegisterIpLimit bool `json:"enabled_register_ip_limit,omitempty"`
	// RegisterIpLimit 同 IP 最多注册账号数（启用限制时默认 3；0 表示不限制）
	RegisterIpLimit int `json:"register_ip_limit,omitempty"`
	// EnabledActivation 是否开放质押/激活（confirmJihuo / confirmJihuoEx）
	EnabledActivation bool `json:"enabled_activation,omitempty"`
	// EnabledActivationUserLimit 是否启用单用户激活（复投）次数限制
	EnabledActivationUserLimit bool `json:"enabled_activation_user_limit,omitempty"`
	// ActivationUserLimit 单用户 confirmJihuoEx 复投上限（启用限制时默认 3；0 表示不限制）
	ActivationUserLimit int `json:"activation_user_limit,omitempty"`

	// EnabledSohaMarket 全局 SOHA 交易市场开关；关闭时所有用户仅可浏览，不可 Swap / 激活算力机等
	EnabledSohaMarket bool `json:"enabled_soha_market"`
	// EnabledSohaKline 是否向用户展示 K 线；关闭时前端 K 线区域模糊，不影响 Swap、算力机激活
	EnabledSohaKline bool `json:"enabled_soha_kline"`

	// Image Hub 应用开关；关闭时前端展示「维护中」，后端拒绝对应 API
	EnabledAppImage bool `json:"enabled_app_image"`
	EnabledAppMusic bool `json:"enabled_app_music"`
	EnabledAppVideo bool `json:"enabled_app_video"`
	EnabledAppPpt   bool `json:"enabled_app_ppt"`
	EnabledAppLogo  bool `json:"enabled_app_logo"`
	EnabledAppDrama bool `json:"enabled_app_drama"`

	RewardImg     string  `json:"reward_img,omitempty"`     // 众筹收款二维码地址
	EnabledReward bool    `json:"enabled_reward,omitempty"` // 启用众筹功能
	PowerPrice    float64 `json:"power_price,omitempty"`    // 算力单价

	OrderPayTimeout int    `json:"order_pay_timeout,omitempty"` //订单支付超时时间
	VipInfoText     string `json:"vip_info_text,omitempty"`     // 会员页面充值说明
	DefaultModels   []int  `json:"default_models,omitempty"`    // 默认开通的 AI 模型

	// TeamRewardMaxCollision 已废弃，保留 JSON 兼容；请使用下方两项。
	TeamRewardMaxCollision int `json:"team_reward_max_collision,omitempty"`
	// TeamRewardMeasureLayerDepth 量碰测算层深：以收奖用户为主体，测算其弱区向下第 N 逻辑层宽度（默认 4）
	// 逻辑层：单边下行同属一层，仅开出双区才进入下一层；不累加物理跳数，也不计入大区
	TeamRewardMeasureLayerDepth int `json:"team_reward_measure_layer_depth,omitempty"`
	// TeamRewardLayer4WidthLimit 主体弱区下第 N 逻辑层入口节点数阈值：仅当宽度 > 该值时拦截（等于不拦）；默认 8
	TeamRewardLayer4WidthLimit int `json:"team_reward_layer4_width_limit,omitempty"`

	// TeamRewardWeeklyCap 团队奖励周封顶（算力/周），仅统计「团队奖励释放」流水；层数档位与 getUserRewardLayerLimit 一致
	TeamRewardWeeklyCap TeamRewardWeeklyCapConfig `json:"team_reward_weekly_cap,omitempty"`
	// TeamRewardPro 团队奖励释放比例（弱区业绩 × 该比例）；0 表示不发放，未配置时默认 0.2
	// 不用 omitempty：值为 0 时也要落库，否则重启会被当成「未配置」回退成 0.2
	TeamRewardPro float64 `json:"team_reward_pro"`

	ZjPro float64 `json:"zj_pro,omitempty"` // 直接比例
	JjPro float64 `json:"jj_pro,omitempty"` // 间接比例

	// CreditLeverage 授信最大额度杠杆倍数（默认 5：自投 1 万 → 最大额度 5 万）
	CreditLeverage float64 `json:"credit_leverage,omitempty"`
	// CreditDirectRate 直推下级投入转化为上级可用授信的比例（默认 0.5 = 50%）
	CreditDirectRate float64 `json:"credit_direct_rate,omitempty"`

	// CreditLegacyCutoff 授信额度「老会员」分界时刻。留空表示不启用老会员规则（全部按当前机制）。
	// 分界前的投入一律折算成固定上限 credit_legacy_max，不再参与杠杆倍数计算。
	CreditLegacyCutoff string `json:"credit_legacy_cutoff,omitempty"`
	// CreditLegacyMinPower 老会员门槛：分界前存在单笔 sales ≥ 该值的订单才被认定为老会员（默认 10000）
	CreditLegacyMinPower float64 `json:"credit_legacy_min_power,omitempty"`
	// CreditLegacyMax 老会员固定上限基数（默认 50000）。老会员 max = 该值 + 分界后新增投入 × 杠杆
	CreditLegacyMax float64 `json:"credit_legacy_max,omitempty"`

	ZjStaticPro float64 `json:"zj_static_pro,omitempty"` // 推荐奖静态释放比例（直推下级每晚静态释放 × 该比例），默认 0.2
	JjStaticPro float64 `json:"jj_static_pro,omitempty"` // 间接静态收益比例（可选，默认关闭）

	// MgmtStaticPro 管理奖静态释放比例：白天因该投入人获得团队奖的收奖人，晚上跟随其静态释放 × 该比例，默认 0.1
	MgmtStaticPro float64 `json:"mgmt_static_pro,omitempty"`

	// StaticReferralOrderCutoff 已废弃兼容字段：推荐奖静态现按 zj_static_pro 对全部在投订单生效；保留 JSON 以免旧配置反序列化失败。
	StaticReferralOrderCutoff string `json:"static_referral_order_cutoff,omitempty"`

	// InvestMultiplier 自定义投入产出倍数分段（管理员后台可调）
	InvestMultiplier InvestMultiplierConfig `json:"invest_multiplier,omitempty"`

	SanPro   float64 `json:"san_pro,omitempty"`   // 三级比例
	SiPro    float64 `json:"si_pro,omitempty"`    // 四级比例
	OtherPro float64 `json:"other_pro,omitempty"` // 其他比例

	ZjPlus     float64 `json:"zj_plus,omitempty"`      // 0-10 天
	Plus1120   float64 `json:"plus_11_20,omitempty"`   // 11-20 天
	Plus2130   float64 `json:"plus_21_30,omitempty"`   // 21-30 天
	Plus3140   float64 `json:"plus_31_40,omitempty"`   // 31-40 天
	Plus4150   float64 `json:"plus_41_50,omitempty"`   // 41-50 天
	Plus5160   float64 `json:"plus_51_60,omitempty"`   // 51-60 天
	Plus6170   float64 `json:"plus_61_70,omitempty"`   // 61-70 天
	Plus7180   float64 `json:"plus_71_80,omitempty"`   // 71-80 天
	Plus8190   float64 `json:"plus_81_90,omitempty"`   // 81-90 天
	Plus91100  float64 `json:"plus_91_100,omitempty"`  // 91 天及以上（91-120 未单独配置时回退到本值）
	Plus91120  float64 `json:"plus_91_120,omitempty"`  // 91-120 天
	Plus121150 float64 `json:"plus_121_150,omitempty"` // 121-150 天
	Plus151180 float64 `json:"plus_151_180,omitempty"` // 151-180 天（181 天以后沿用本档，封顶）
	JjPlus     float64 `json:"jj_plus,omitempty"`      // 已废弃：11-30 天，未配置新分段时回退
	SanPlus    float64 `json:"san_plus,omitempty"`     // 已废弃：31-60 天，未配置新分段时回退
	SiPlus     float64 `json:"si_plus,omitempty"`      // 已废弃：61-90 天，未配置新分段时回退
	OtherPlus  float64 `json:"other_plus,omitempty"`   // 已废弃：90 天以后，未配置新分段时回退

	// SwapPowerPrice SOHA→算力兑换锚定价（默认 0.01），汇率 = 当前SOHA价 / SwapPowerPrice
	SwapPowerPrice float64 `json:"swap_power_price,omitempty"`
	// SohaGuideBasePrice SOHA 指导价首日基准价（默认 0.1）；之后每日由系统随机涨幅 0.10%~0.50% 复利累加
	SohaGuideBasePrice float64 `json:"soha_guide_base_price,omitempty"`

	// KjZjPro 算力机购买直推推荐奖释放比例（默认 0.3），与套餐 zj_pro 独立
	KjZjPro float64 `json:"kj_zj_pro,omitempty"`
	// KjTeamRewardPro 算力机购买团队弱区奖励比例（默认 0.2），与 team_reward_pro 独立
	KjTeamRewardPro float64 `json:"kj_team_reward_pro,omitempty"`

	// 矿机（算力机）SOHA产出分段释放比例，与静态套餐 zj_plus 等独立配置
	KjZjPlus     float64 `json:"kj_zj_plus,omitempty"`       // 0-10 天
	KjPlus1120   float64 `json:"kj_plus_11_20,omitempty"`    // 11-20 天
	KjPlus2130   float64 `json:"kj_plus_21_30,omitempty"`    // 21-30 天
	KjPlus3140   float64 `json:"kj_plus_31_40,omitempty"`    // 31-40 天
	KjPlus4150   float64 `json:"kj_plus_41_50,omitempty"`    // 41-50 天
	KjPlus5160   float64 `json:"kj_plus_51_60,omitempty"`    // 51-60 天
	KjPlus6170   float64 `json:"kj_plus_61_70,omitempty"`    // 61-70 天
	KjPlus7180   float64 `json:"kj_plus_71_80,omitempty"`    // 71-80 天
	KjPlus8190   float64 `json:"kj_plus_81_90,omitempty"`    // 81-90 天
	KjPlus91100  float64 `json:"kj_plus_91_100,omitempty"`   // 91 天及以上
	KjJjPlus     float64 `json:"kj_jj_plus,omitempty"`       // 已废弃：11-30 天，未配置新分段时回退
	KjSanPlus    float64 `json:"kj_san_plus,omitempty"`      // 已废弃：31-60 天，未配置新分段时回退
	KjSiPlus     float64 `json:"kj_si_plus,omitempty"`       // 已废弃：61-90 天，未配置新分段时回退
	KjOtherPlus  float64 `json:"kj_other_plus,omitempty"`    // 已废弃：90 天以后，未配置新分段时回退

	Usdtimg string `json:"usdtimg,omitempty"`
	Address string `json:"address,omitempty"` //收款地址

	MjPower       float64 `json:"mj_power,omitempty"`        // MJ 绘画消耗算力
	MjActionPower float64 `json:"mj_action_power,omitempty"` // MJ 操作（放大，变换）消耗算力
	SdPower       float64 `json:"sd_power,omitempty"`        // SD 绘画消耗算力
	DallPower     float64 `json:"dall_power,omitempty"`      // DALLE3 绘图消耗算力

	DigiPower  float64 `json:"digi_power,omitempty"`  // 数字人消耗算力
	MusicPower float64 `json:"music_power,omitempty"` // 音乐生成消耗算力

	WechatCardURL string `json:"wechat_card_url,omitempty"` // 微信客服地址

	EnableContext bool  `json:"enable_context,omitempty"`
	ContextDeep   int   `json:"context_deep,omitempty"`
	LimitCancel   int64 `json:"limit_cancel"` // 截止目前已销毁 SOHA 数量；0 为有效值，勿用 omitempty

	PromptPower int `json:"prompt_power,omitempty"` // 教学配图等单次提示词类消耗（按张计费时使用）

	// 短剧（AI 短剧）算力：为 0 时回退到所选 Chat 模型算力 → prompt_power → mj_power → 默认值
	TeachingScriptPower          int `json:"teaching_script_power,omitempty"`            // 脚本 / 分镜提示词 LLM 单次
	TeachingTtsPowerPerSegment   int `json:"teaching_tts_power_per_segment,omitempty"`   // TTS 每段口播

	// 短剧（教学）口播 / TTS / 配图
	TeachingScriptModelId        int    `json:"teaching_script_model_id,omitempty"`
	TtsApiURL                    string `json:"tts_api_url,omitempty"`
	TtsApiKey                    string `json:"tts_api_key,omitempty"`
	TtsAppId                     string `json:"tts_app_id,omitempty"`
	TtsAccessKey                 string `json:"tts_access_key,omitempty"`
	TtsVolcAccessKey             string `json:"tts_volc_access_key,omitempty"` // 火山 IAM AK，用于查询 OpenSpeech 资源包余量
	TtsVolcSecretKey             string `json:"tts_volc_secret_key,omitempty"` // 火山 IAM SK
	TtsResourceId                string `json:"tts_resource_id,omitempty"`
	TtsSpeaker                   string `json:"tts_speaker,omitempty"`
	TtsAudioFormat               string `json:"tts_audio_format,omitempty"`
	TtsSampleRate                int    `json:"tts_sample_rate,omitempty"`
	TeachingImageApiURL          string `json:"teaching_image_api_url,omitempty"`
	TeachingImageApiKey          string `json:"teaching_image_api_key,omitempty"`
	TeachingImageModel           string `json:"teaching_image_model,omitempty"`
	TeachingImageSize            string `json:"teaching_image_size,omitempty"`
	TeachingVisualSparse         bool   `json:"teaching_visual_sparse,omitempty"`
	TeachingVisualShotMaxRunes   int    `json:"teaching_visual_shot_max_runes,omitempty"`
	TeachingVisualMaxTotalShots  int    `json:"teaching_visual_max_total_shots,omitempty"`
	TeachingImagePromptsLlmBatch int    `json:"teaching_image_prompts_llm_batch,omitempty"`
}

// SignInRewardByDays 按注册天数返回签到赠送算力；未单独配置的分段使用默认 10/5/3
func (c SystemConfig) SignInRewardByDays(daysSinceRegistration int) float64 {
	if daysSinceRegistration < 0 {
		daysSinceRegistration = 0
	}
	switch {
	case daysSinceRegistration <= 30:
		if c.SignInReward0_30 > 0 {
			return c.SignInReward0_30
		}
		return 10
	case daysSinceRegistration <= 60:
		if c.SignInReward31_60 > 0 {
			return c.SignInReward31_60
		}
		return 5
	case daysSinceRegistration <= 90:
		if c.SignInReward61_90 > 0 {
			return c.SignInReward61_90
		}
		return 3
	default:
		return c.SignInReward90Plus
	}
}

// SignInRewardByStreak 按「连续签到天数」返回签到赠送算力。
// 第 1 天 2、第 2 天 3 …… 第 9 天 10、第 10 天及以后均为 10；断签后重置回第 1 天。
func SignInRewardByStreak(streak int) float64 {
	if streak < 1 {
		streak = 1
	}
	if streak >= 10 {
		return 10
	}
	return float64(streak + 1)
}

type segmentedReleaseRates struct {
	s0_10   float64
	s11_20  float64
	s21_30  float64
	s31_40  float64
	s41_50  float64
	s51_60  float64
	s61_70  float64
	s71_80  float64
	s81_90  float64
	s91Plus float64
	// 静态释放专用：把「91 天及以上」再细分为 91-120 / 121-150 / 151-180 三档。
	// enableLateTiers=false 时（算力机产出、等级分红）忽略，保持「91 天及以上」单档行为。
	enableLateTiers bool
	s91_120         float64
	s121_150        float64
	s151_180        float64
	l11_30          float64
	l31_60          float64
	l61_90          float64
	l90Plus         float64
}

func releaseRateByDays(segs segmentedReleaseRates, days int) float64 {
	if days < 0 {
		days = 0
	}
	pick := func(primary, legacy float64) float64 {
		if primary > 0 {
			return primary
		}
		return legacy
	}
	// 91 天以后的取值：启用细分档位时按档取；未配置该档则回退原有「91 天及以上」逻辑。
	// 181 天以后沿用 151-180 档（封顶）。
	legacy91 := pick(segs.s91Plus, segs.l90Plus)
	pickLate := func(primary float64) float64 {
		if segs.enableLateTiers {
			return pick(primary, legacy91)
		}
		return legacy91
	}
	switch {
	case days <= 10:
		return segs.s0_10
	case days <= 20:
		return pick(segs.s11_20, segs.l11_30)
	case days <= 30:
		return pick(segs.s21_30, segs.l11_30)
	case days <= 40:
		return pick(segs.s31_40, segs.l31_60)
	case days <= 50:
		return pick(segs.s41_50, segs.l31_60)
	case days <= 60:
		return pick(segs.s51_60, segs.l31_60)
	case days <= 70:
		return pick(segs.s61_70, segs.l61_90)
	case days <= 80:
		return pick(segs.s71_80, segs.l61_90)
	case days <= 90:
		return pick(segs.s81_90, segs.l61_90)
	case days <= 120:
		return pickLate(segs.s91_120)
	case days <= 150:
		return pickLate(segs.s121_150)
	default:
		return pickLate(segs.s151_180)
	}
}

// InvestMultiplierConfig 自定义投入产出倍数分段
// 默认：1000～49999 → 5倍；50000～99999 → 6倍；100000+ → 7倍
type InvestMultiplierConfig struct {
	Tier1Min float64 `json:"tier1_min,omitempty"` // 默认 1000
	Tier1Max float64 `json:"tier1_max,omitempty"` // 默认 49999
	Tier1Mul float64 `json:"tier1_mul,omitempty"` // 默认 5
	Tier2Min float64 `json:"tier2_min,omitempty"` // 默认 50000
	Tier2Max float64 `json:"tier2_max,omitempty"` // 默认 99999
	Tier2Mul float64 `json:"tier2_mul,omitempty"` // 默认 6
	Tier3Min float64 `json:"tier3_min,omitempty"` // 默认 100000
	Tier3Mul float64 `json:"tier3_mul,omitempty"` // 默认 7
}

// DefaultInvestMultiplier 后台未配置时的默认倍数分段
func DefaultInvestMultiplier() InvestMultiplierConfig {
	return InvestMultiplierConfig{
		Tier1Min: 1000,
		Tier1Max: 49999,
		Tier1Mul: 5,
		Tier2Min: 50000,
		Tier2Max: 99999,
		Tier2Mul: 6,
		Tier3Min: 100000,
		Tier3Mul: 7,
	}
}

// Normalize 补齐未配置档位，保证可安全用于倍数计算
func (m InvestMultiplierConfig) Normalize() InvestMultiplierConfig {
	d := DefaultInvestMultiplier()
	if m.Tier1Min <= 0 {
		m.Tier1Min = d.Tier1Min
	}
	if m.Tier1Max <= 0 {
		m.Tier1Max = d.Tier1Max
	}
	if m.Tier1Mul <= 0 {
		m.Tier1Mul = d.Tier1Mul
	}
	if m.Tier2Min <= 0 {
		m.Tier2Min = d.Tier2Min
	}
	if m.Tier2Max <= 0 {
		m.Tier2Max = d.Tier2Max
	}
	if m.Tier2Mul <= 0 {
		m.Tier2Mul = d.Tier2Mul
	}
	if m.Tier3Min <= 0 {
		m.Tier3Min = d.Tier3Min
	}
	if m.Tier3Mul <= 0 {
		m.Tier3Mul = d.Tier3Mul
	}
	return m
}

// InvestOutputMultiplier 按投入算力返回产出倍数（可后台配置）
func (c SystemConfig) InvestOutputMultiplier(sales float64) float64 {
	m := c.InvestMultiplier.Normalize()
	switch {
	case sales >= m.Tier3Min:
		return m.Tier3Mul
	case sales >= m.Tier2Min && sales <= m.Tier2Max:
		return m.Tier2Mul
	case sales >= m.Tier1Min && sales <= m.Tier1Max:
		return m.Tier1Mul
	case sales > m.Tier1Max && sales < m.Tier2Min:
		// 档位缝隙：按较高档
		return m.Tier2Mul
	case sales > m.Tier2Max && sales < m.Tier3Min:
		return m.Tier3Mul
	default:
		// 低于最低档仍给一档倍数，避免产出为 0（投入门槛由业务另行校验）
		return m.Tier1Mul
	}
}

// InvestOutputPower 投入算力 × 分段倍数 = 预产/待产算力
func (c SystemConfig) InvestOutputPower(sales float64) float64 {
	return sales * c.InvestOutputMultiplier(sales)
}

// CreditLeverageOrDefault 授信杠杆，未配置或非法时默认 5
func (c SystemConfig) CreditLeverageOrDefault() float64 {
	if c.CreditLeverage <= 0 {
		return 5
	}
	return c.CreditLeverage
}

// CreditDirectRateOrDefault 直推投入→可用授信比例，未配置或非法时默认 0.5
func (c SystemConfig) CreditDirectRateOrDefault() float64 {
	if c.CreditDirectRate <= 0 {
		return 0.5
	}
	if c.CreditDirectRate > 1 {
		return 1
	}
	return c.CreditDirectRate
}

// CreditLegacyMinPowerOrDefault 老会员门槛单笔投入，未配置时默认 10000
func (c SystemConfig) CreditLegacyMinPowerOrDefault() float64 {
	if c.CreditLegacyMinPower <= 0 {
		return 10000
	}
	return c.CreditLegacyMinPower
}

// CreditLegacyMaxOrDefault 老会员固定上限基数，未配置时默认 50000
func (c SystemConfig) CreditLegacyMaxOrDefault() float64 {
	if c.CreditLegacyMax <= 0 {
		return 50000
	}
	return c.CreditLegacyMax
}

// CreditLegacyCutoffTime 解析老会员分界时刻；未配置或格式非法时 ok=false（不启用老会员规则）
func (c SystemConfig) CreditLegacyCutoffTime() (time.Time, bool) {
	s := strings.TrimSpace(c.CreditLegacyCutoff)
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, time.Local); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// StaticReleaseRateByDays 按订单存续天数返回静态套餐每日释放比例
// 91 天以后细分为 91-120 / 121-150 / 151-180 三档，181 天以后沿用 151-180 档（封顶）
func (c SystemConfig) StaticReleaseRateByDays(days int) float64 {
	return releaseRateByDays(segmentedReleaseRates{
		s0_10:           c.ZjPlus,
		s11_20:          c.Plus1120,
		s21_30:          c.Plus2130,
		s31_40:          c.Plus3140,
		s41_50:          c.Plus4150,
		s51_60:          c.Plus5160,
		s61_70:          c.Plus6170,
		s71_80:          c.Plus7180,
		s81_90:          c.Plus8190,
		s91Plus:         c.Plus91100,
		enableLateTiers: true,
		s91_120:         c.Plus91120,
		s121_150:        c.Plus121150,
		s151_180:        c.Plus151180,
		l11_30:          c.JjPlus,
		l31_60:          c.SanPlus,
		l61_90:          c.SiPlus,
		l90Plus:         c.OtherPlus,
	}, days)
}

// KjReleaseRateByDays 按矿机订单存续天数返回 SOHA 日产出比例
func (c SystemConfig) KjReleaseRateByDays(days int) float64 {
	return releaseRateByDays(segmentedReleaseRates{
		s0_10:   c.KjZjPlus,
		s11_20:  c.KjPlus1120,
		s21_30:  c.KjPlus2130,
		s31_40:  c.KjPlus3140,
		s41_50:  c.KjPlus4150,
		s51_60:  c.KjPlus5160,
		s61_70:  c.KjPlus6170,
		s71_80:  c.KjPlus7180,
		s81_90:  c.KjPlus8190,
		s91Plus: c.KjPlus91100,
		l11_30:  c.KjJjPlus,
		l31_60:  c.KjSanPlus,
		l61_90:  c.KjSiPlus,
		l90Plus: c.KjOtherPlus,
	}, days)
}

// ReleaseRateSegmentLabel 返回天数对应的分段标签（供前端展示）
func ReleaseRateSegmentLabel(days int) string {
	if days < 0 {
		days = 0
	}
	switch {
	case days <= 10:
		return "0-10天"
	case days <= 20:
		return "11-20天"
	case days <= 30:
		return "21-30天"
	case days <= 40:
		return "31-40天"
	case days <= 50:
		return "41-50天"
	case days <= 60:
		return "51-60天"
	case days <= 70:
		return "61-70天"
	case days <= 80:
		return "71-80天"
	case days <= 90:
		return "81-90天"
	case days <= 120:
		return "91-120天"
	case days <= 150:
		return "121-150天"
	case days <= 180:
		return "151-180天"
	default:
		return "181天+"
	}
}

// PptAppSettings PPT 子系统（banana）在平台侧单独存储的模型与密钥，对应管理后台 config key: ppt
type PptAppSettings struct {
	PptAiProviderFormat string `json:"ppt_ai_provider_format,omitempty"` // gemini | openai，与 banana Settings 中 AI 提供商格式一致
	PptTextModel        string `json:"ppt_text_model,omitempty"`
	PptTextApiBaseUrl   string `json:"ppt_text_api_base_url,omitempty"`
	PptTextApiKey       string `json:"ppt_text_api_key,omitempty"`
	PptImageModel       string `json:"ppt_image_model,omitempty"`
	PptImageApiBaseUrl  string `json:"ppt_image_api_base_url,omitempty"`
	PptImageApiKey      string `json:"ppt_image_api_key,omitempty"`
}

// FenHongConfig 等级分红（钻石等级）独立配置，对应管理后台 config key: fenhong
//
// 定时任务 FenHongTask：当天释放量 = BaseParamLvN * TimeParam。
// 分段天数与静态释放一致（0～10、11～20 … 91+），但 TimeParam 使用本结构中的 fh_zj_time 等字段（与静态的 zj_plus 等数值独立）。
// 天数从用户「升级到当前 lev」的时刻 lev_since_at 起算；无该字段时回退为账号注册时间。
// 未配置各等级基础参数时，回退为 (66666 * lev) * TimeParam，TimeParam 在无 fenhong 配置时沿用系统 SysConfig 的 zj_plus 分段系数。
type FenHongConfig struct {
	BaseParamLv1 float64 `json:"fh_base_param_lv1,omitempty"`
	BaseParamLv2 float64 `json:"fh_base_param_lv2,omitempty"`
	BaseParamLv3 float64 `json:"fh_base_param_lv3,omitempty"`
	BaseParamLv4 float64 `json:"fh_base_param_lv4,omitempty"`
	BaseParamLv5 float64 `json:"fh_base_param_lv5,omitempty"`
	BaseParamLv6 float64 `json:"fh_base_param_lv6,omitempty"`
	BaseParamLv7 float64 `json:"fh_base_param_lv7,omitempty"`
	BaseParamLv8 float64 `json:"fh_base_param_lv8,omitempty"`
	BaseParamLv9 float64 `json:"fh_base_param_lv9,omitempty"`

	ZjTime      float64 `json:"fh_zj_time,omitempty"`       // 0-10 天
	Plus1120    float64 `json:"fh_plus_11_20,omitempty"`    // 11-20 天
	Plus2130    float64 `json:"fh_plus_21_30,omitempty"`    // 21-30 天
	Plus3140    float64 `json:"fh_plus_31_40,omitempty"`    // 31-40 天
	Plus4150    float64 `json:"fh_plus_41_50,omitempty"`    // 41-50 天
	Plus5160    float64 `json:"fh_plus_51_60,omitempty"`    // 51-60 天
	Plus6170    float64 `json:"fh_plus_61_70,omitempty"`    // 61-70 天
	Plus7180    float64 `json:"fh_plus_71_80,omitempty"`    // 71-80 天
	Plus8190    float64 `json:"fh_plus_81_90,omitempty"`    // 81-90 天
	Plus91100   float64 `json:"fh_plus_91_100,omitempty"`   // 91 天及以上
	JjTime      float64 `json:"fh_jj_time,omitempty"`       // 已废弃：11-30 天，未配置新分段时回退
	SanTime     float64 `json:"fh_san_time,omitempty"`      // 已废弃：31-60 天，未配置新分段时回退
	SiTime      float64 `json:"fh_si_time,omitempty"`       // 已废弃：61-90 天，未配置新分段时回退
	OtherTime   float64 `json:"fh_other_time,omitempty"`    // 已废弃：90 天以后，未配置新分段时回退
}

// TimeParamByDays 按升级后存续天数返回等级分红时间参数
func (c FenHongConfig) TimeParamByDays(days int) float64 {
	return releaseRateByDays(segmentedReleaseRates{
		s0_10:   c.ZjTime,
		s11_20:  c.Plus1120,
		s21_30:  c.Plus2130,
		s31_40:  c.Plus3140,
		s41_50:  c.Plus4150,
		s51_60:  c.Plus5160,
		s61_70:  c.Plus6170,
		s71_80:  c.Plus7180,
		s81_90:  c.Plus8190,
		s91Plus: c.Plus91100,
		l11_30:  c.JjTime,
		l31_60:  c.SanTime,
		l61_90:  c.SiTime,
		l90Plus: c.OtherTime,
	}, days)
}
