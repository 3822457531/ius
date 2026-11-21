package vo

// ApiKey OpenAI API 模型
type ApiKey struct {
	BaseVo
	Platform   string `json:"platform"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Value      string `json:"value"` // API Key 的值
	ApiURL     string `json:"api_url"`
	Enabled    bool   `json:"enabled"`
	ProxyURL   string `json:"proxy_url"`
	PublicBase string `json:"public_base"`  // 公网域名，图生图相对路径拼接
	Priority   int    `json:"priority"`     // 优先级，越大越优先
	LastUsedAt int64  `json:"last_used_at"` // 最后使用时间
}
