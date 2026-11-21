package model

import (
	"chatplus/core/types"
	"time"
)

// CoinLog 矿机单台 SOHA 产出明细（按 order_no 汇总到 mint_order.js_coin）
type CoinLog struct {
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
	OrderNo   string // 矿机订单号
}

func (CoinLog) TableName() string {
	return "chatgpt_coin_logs"
}
