package model

import (
	"chatplus/core/types"
	"gorm.io/gorm"
)

// MintOrder 挖矿列表
type MintOrder struct {
	BaseModel
	UserId    uint
	MintId    uint
	Username  string
	OrderNo   string
	TradeNo   string
	Subject   string
	Amount    float64
	Status    types.OrderStatus
	Remark    string
	PayTime   int64
	PayWay    string // 支付方式
	DeletedAt gorm.DeletedAt
	Power     float64
	JsCoin    float64
	Fee       float64
	NowCoin   float64
}
