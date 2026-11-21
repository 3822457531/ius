package vo

type User struct {
	BaseVo
	Username    string   `json:"username"`
	Nickname    string   `json:"nickname"`
	Avatar      string   `json:"avatar"`
	Salt        string   `json:"salt"`          // 密码盐
	Power       float64  `json:"power"`         // 剩余算力
	Coin        float64  `json:"coin"`          // 剩余SOHA
	DjCoin      float64  `json:"dj_coin"`       // 剩余SOHA
	ChatRoles   []string `json:"chat_roles"`    // 聊天角色集合
	ChatModels  []int    `json:"chat_models"`   // AI模型集合
	ExpiredTime int64    `json:"expired_time"`  // 账户到期时间
	Status      bool     `json:"status"`        // 当前状态
	LastLoginAt int64    `json:"last_login_at"` // 最后登录时间
	LastLoginIp string   `json:"last_login_ip"` // 最后登录 IP
	Vip         bool     `json:"vip"`
	AllowTrade  bool     `json:"allow_trade"`
	AllowQ2     bool     `json:"allow_q2"`   // 是否 允许SOHA交易
	AllowMint   bool     `json:"allow_mint"` // 是否允许购买算力机
	AllowPlacement bool  `json:"allow_placement"` // 是否允许查看大小区节点
	IsAuditor   bool     `json:"is_auditor"` // 是否 审核用户
	IsCs        bool     `json:"is_cs"`      // 是否客服
	Lev         int64    `json:"lev"`        // 用户等级（0-9，对应0钻到9钻）
	Floor       int64    `json:"floor"`      // 用户层数（10, 20, 30... 100，对应层级限制）
	Div         string   `json:"div"`
	ZjFid       int      //直接上级
	JjFid       int      //间接上级
	SanFid      int      //三代上级
	SiFid       int      //四代上级
	FidsJson    string   `gorm:"column:fids_json"` // 5-10代合伙人ids
	Code        string   `json:"code"`             //邀请码
}
