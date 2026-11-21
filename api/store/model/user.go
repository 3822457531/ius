package model

import "time"

type User struct {
	BaseModel
	Username           string
	Nickname           string
	Password           string
	Avatar             string
	Salt               string     // 密码盐
	Power              float64    // 剩余算力
	Coin               float64    // 剩余SOHA币
	DjCoin             float64    // 冻结部分的余额
	ChatConfig         string     `gorm:"column:chat_config_json"`           // 聊天配置 json
	ImageHubCustomApps string     `gorm:"column:image_hub_custom_apps_json"` // 图片导航页自定义应用
	ChatRoles          string     `gorm:"column:chat_roles_json"`            // 聊天角色
	ChatModels         string     `gorm:"column:chat_models_json"`           // AI 模型，不同的用户拥有不同的聊天模型
	ExpiredTime        int64      // 账户到期时间
	Status             bool       `gorm:"default:true"` // 当前状态
	LastLoginAt        int64      // 最后登录时间
	LastLoginIp        string     // 最后登录 IP
	Vip                bool       // 是否 VIP 会员
	Lev                int64      // 用户等级（0-9，对应0钻到9钻）
	LevSinceAt         *time.Time `gorm:"column:lev_since_at"` // 最近一次变更到当前 lev 的时间（等级分红时间参数从此刻起算）
	Floor              int64      // 用户层数（10, 20, 30... 100，对应层级限制）
	Div                string     //自定义名称
	ZjFid              int        //直接上级
	JjFid              int        //间接上级
	SanFid             int        //三代上级
	SiFid              int        //四代上级
	FidsJson           string     `gorm:"column:fids_json"` // 5-10代合伙人ids
	IsPartner          int        //是否已经领取 （0 未领取 1 已领取）
	SignInStreak       int        `gorm:"column:sign_in_streak;default:0"` // 连续签到天数（0 表示今日尚未签到；断签后重置）
	LastSignInDate     *time.Time `gorm:"column:last_sign_in_date"`        // 最近一次签到的自然日（用于判断是否断签）
	AgentNum           int        //直接推荐人人数（有效的，充值了的）
	IsJh                 int //当前用户是否已经激活状态(0 未激活 1 已激活)
	JihuoExReinvestCount int `gorm:"column:jihuo_ex_reinvest_count;default:0"` // confirmJihuoEx 复投次数
	LoginIp              string // 登录IP
	LoginAddress       string     // 登录地址
	Mail               string     // 邮箱
	EthAddress         string     // BSC收款地址
	AlipayAccount      string     `gorm:"column:alipay_account"`       // 支付宝账号
	DefaultPaymentType string     `gorm:"column:default_payment_type"` // 默认收款方式: bsc/alipay
	AllowTrade         bool       // 是否 允许算力交易
	AllowQ2            bool       `gorm:"column:allow_q2;default:true"` // 是否 允许SOHA交易（默认开启）
	AllowMint          bool       `gorm:"column:allow_mint;default:true"` // 是否允许购买算力机
	AllowPlacement     bool       `gorm:"column:allow_placement;default:false"` // 是否允许查看大小区节点
	IsAuditor          bool       `gorm:"default:false"` // 是否 审核用户
	IsCs               bool       `gorm:"column:is_cs;default:false"` // 是否客服（H5 可接入接线台）
	// 授信额度：独立于可用算力，仅可用于授信报单（赠送激活），不可转账/AI 消费。
	// Max = max(新规则自投基数, CreditMaxFloor) − 累计授信报单消耗；Available ≤ Max（还款不回补）。
	CreditMax       float64 `gorm:"column:credit_max;default:0"`        // 当前最大额度（展示/校验）
	CreditMaxFloor  float64 `gorm:"column:credit_max_floor;default:0"`  // 历史保底基数（旧规则已形成的投入侧峰值，只升不降）
	CreditAvailable float64 `gorm:"column:credit_available;default:0"`  // 当前可用额度（只加给直推上级，≤ Max）
	CreditDebt      float64 `gorm:"column:credit_debt;default:0"`       // 已消耗待还额度
}