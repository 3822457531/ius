package types

// BizVo 业务返回 VO
type BizVo struct {
	Code     BizCode     `json:"code"`
	Page     int         `json:"page,omitempty"`
	PageSize int         `json:"page_size,omitempty"`
	Total    int         `json:"total,omitempty"`
	Message  string      `json:"message,omitempty"`
	Data     interface{} `json:"data,omitempty"`
}

// WsMessage Websocket message
type WsMessage struct {
	Type       WsMsgType   `json:"type"` // 消息类别，start, end, img；客户端上行可用 chat
	Content    interface{} `json:"content"`
	WebSearch  bool        `json:"web_search,omitempty"` // 是否开启本轮联网搜索
}
type WsMsgType string

const (
	WsStart  = WsMsgType("start")
	WsMiddle = WsMsgType("middle")
	WsEnd    = WsMsgType("end")
	WsMjImg  = WsMsgType("mj")
)

// ReplyMessage 对话回复消息结构
type ReplyMessage struct {
	Channel  WsChannel   `json:"channel"`  // 消息频道
	ClientId string      `json:"clientId"` // 客户端ID
	Type     WsMsgType   `json:"type"`     // 消息类别
	Body     interface{} `json:"body"`
}

type WsChannel string

const (
	ChPing  = WsChannel("ping")
	ChChat  = WsChannel("chat")
	ChMj    = WsChannel("mj")
	ChSd    = WsChannel("sd")
	ChDall  = WsChannel("dall")
	ChSuno  = WsChannel("suno")
	ChLuma  = WsChannel("luma")
	ChSora2 = WsChannel("sora2")
)

// InputMessage 对话输入消息结构
type InputMessage struct {
	Channel WsChannel   `json:"channel"` // 消息频道
	Type    WsMsgType   `json:"type"`    // 消息类别
	Body    interface{} `json:"body"`
}

type BizCode int

const (
	Success       = BizCode(0)
	Failed        = BizCode(1)
	NotAuthorized = BizCode(400) // 未授权
	NotPermission = BizCode(403) // 没有权限

	OkMsg       = "Success"
	ErrorMsg    = "系统开小差了"
	InvalidArgs = "非法参数或参数解析失败"
	NoData      = "No Data"
)
