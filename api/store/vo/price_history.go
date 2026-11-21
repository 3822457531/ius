package vo

// vo/price_history.go
type PriceHistory struct {
	Date       string  `json:"Date"`       // 日期,格式: "M/D"
	Price      float64 `json:"Price"`      // 价格
	Volume     float64 `json:"Volume"`     // 交易量
	ChangeRate float64 `json:"ChangeRate"` // 价格变化率
}
