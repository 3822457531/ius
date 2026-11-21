package vo

import "chatplus/core/types"

type CoinLog struct {
	Id        uint            `json:"id"`
	UserId    uint            `json:"user_id"`
	Username  string          `json:"username"`
	Type      types.PowerType `json:"type"`
	TypeStr   string          `json:"type_str"`
	Amount    float64         `json:"amount"`
	Mark      types.PowerMark `json:"mark"`
	Balance   float64         `json:"balance"`
	Model     string          `json:"model"`
	Remark    string          `json:"remark"`
	OrderNo   string          `json:"order_no"`
	CreatedAt int64           `json:"created_at"`
}
