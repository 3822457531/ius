package model

import (
	"chatplus/core/types"
	"time"
)

// PowerLog 算力消费日志
type PowerLog struct {
	Id        uint `gorm:"primarykey;column:id"`
	UserId    uint
	Username  string
	Type      types.PowerType
	Amount    float64
	Balance   float64
	Model     string          // 模型
	Remark    string          // 备注
	Mark      types.PowerMark // 资金类型
	CreatedAt time.Time
	DeletedAt       *time.Time `gorm:"column:deleted_at;index"`
	RollbackBatchId string     `gorm:"column:rollback_batch_id;size:64"`
	OrderNo   string //订单号
}

func (PowerLog) TableName() string {
	return "chatgpt_power_logs"
}
