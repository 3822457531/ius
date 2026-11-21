package model

import (
	"chatplus/core/types"
	"time"
)

// Q2Log 用户 SOHA 钱包总账（user.coin 变动流水，对标 chatgpt_power_logs）
type Q2Log struct {
	Id        uint `gorm:"primarykey;column:id"`
	UserId    uint
	Username  string
	Type      types.PowerType
	Amount    float64
	Balance   float64
	Model     string          // 业务类型
	Remark    string          // 备注
	Mark      types.PowerMark // 1增加 0减少
	CreatedAt time.Time
	OrderNo   string // 关联订单号（矿机产出等）
}

func (Q2Log) TableName() string {
	return "chatgpt_q2_logs"
}
