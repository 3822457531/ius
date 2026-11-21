package model

// CsSession 人工客服会话
type CsSession struct {
	BaseModel
	UserId      uint   `gorm:"index;not null"`
	Username    string `gorm:"size:128"`
	Nickname    string `gorm:"size:128"`
	Avatar      string `gorm:"size:512"`
	Status      int    `gorm:"default:0;index"` // 0=进行中 1=已关闭
	AdminId     uint   `gorm:"default:0"`       // 最近回复的管理员
	LastMsg     string `gorm:"size:500"`
	LastMsgAt   int64  `gorm:"index"`
	UserUnread  int    `gorm:"default:0"`
	AdminUnread int    `gorm:"default:0"`
}

func (CsSession) TableName() string {
	return "chatgpt_cs_sessions"
}

// CsMessage 客服消息
type CsMessage struct {
	BaseModel
	SessionId uint   `gorm:"index;not null"`
	UserId    uint   `gorm:"index"`
	AdminId   uint   `gorm:"default:0"`
	Role      string `gorm:"size:16;not null"` // user | admin | system
	Content   string `gorm:"type:text"`
	MsgType   string `gorm:"size:16;default:text"`
}

func (CsMessage) TableName() string {
	return "chatgpt_cs_messages"
}
