package model

// ApiKey OpenAI API 模型
type ApiKey struct {
	BaseModel
	Platform   string
	Name       string
	Type       string // 用途 chat => 聊天，img/image2 => 绘图，dy => 短剧，sora => 视频
	Value      string // API Key 的值
	ApiURL     string // 当前 KEY 的 API 地址
	Enabled    bool   // 是否启用
	ProxyURL   string // 代理地址
	PublicBase string // 公网域名（图生图相对路径拼接用，如 https://img.example.com）
	Priority   int    // 优先级，数值越大越优先（同用途多 KEY 时按此选用）
	LastUsedAt int64  // 最后使用时间
}
