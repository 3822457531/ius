package model

// PowerPool 算力池表 - 存储平台总算力和总USDT
type PowerPool struct {
	BaseModel
	TotalPower  float64 `gorm:"column:total_power;default:0"`  // 平台总算力
	TotalUsdt   float64 `gorm:"column:total_usdt;default:0"`   // 平台总USDT
	BuybackUsdt float64 `gorm:"column:buyback_usdt;default:0"` // 回购池USDT
}

// TableName 指定表名
func (PowerPool) TableName() string {
	return "chatgpt_power_pool"
}

// PowerPoolRecharge 算力池充值订单表
type PowerPoolRecharge struct {
	BaseModel
	OrderNumber    string  `gorm:"column:order_number;uniqueIndex"` // 订单号
	UserId         uint    `gorm:"column:user_id"`                  // 用户ID
	PowerAmount    float64 `gorm:"column:power_amount"`             // 购买的算力数量
	UsdtAmount     float64 `gorm:"column:usdt_amount"`              // 需要支付的USDT金额
	BscAddress     string  `gorm:"column:bsc_address"`              // BSC收款地址（平台地址）
	UserBscAddress string  `gorm:"column:user_bsc_address"`         // 用户的BSC地址（用于验证from_address）
	Status         int     `gorm:"column:status;default:1"`         // 订单状态: 1-待支付, 2-已支付, 3-已完成, 4-已取消
	TxHash         string  `gorm:"column:tx_hash"`                  // 交易哈希
	PayTime        int64   `gorm:"column:pay_time"`                 // 支付时间
	FinishTime     int64   `gorm:"column:finish_time"`              // 完成时间
}

// TableName 指定表名
func (PowerPoolRecharge) TableName() string {
	return "chatgpt_power_pool_recharge"
}
