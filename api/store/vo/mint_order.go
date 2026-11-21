package vo

import (
	"chatplus/core/types"
)

type MintOrder struct {
	BaseVo
	UserId   uint              `json:"user_id"`
	MintId   uint              `json:"mint_id"`
	Username string            `json:"username"`
	OrderNo  string            `json:"order_no"`
	TradeNo  string            `json:"trade_no"`
	Subject  string            `json:"subject"`
	Amount   float64           `json:"amount"`
	Status   types.OrderStatus `json:"status"`
	PayTime  int64             `json:"pay_time"`
	PayWay   string            `json:"pay_way"`
	Remark   types.OrderRemark `json:"remark"`
	Power    float64           `json:"power"`
	JsCoin   float64           `json:"js_coin"`
	Fee      float64           `json:"fee"`
	NowCoin  float64           `json:"now_coin"`
}
