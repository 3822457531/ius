package model

import "time"

// AppPowerRefundLog PPT / 短剧等同步接口的幂等退款记录（同一 refund_key 仅退一次）
type AppPowerRefundLog struct {
	Id        uint      `gorm:"primarykey;column:id"`
	UserId    uint      `gorm:"column:user_id;uniqueIndex:idx_user_source_refund_key,priority:1"`
	Source    string    `gorm:"column:source;size:32;uniqueIndex:idx_user_source_refund_key,priority:2"`
	RefundKey string    `gorm:"column:refund_key;size:191;uniqueIndex:idx_user_source_refund_key,priority:3"`
	Amount    int       `gorm:"column:amount"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (AppPowerRefundLog) TableName() string {
	return "chatgpt_app_power_refund_logs"
}
