package types

type OrderStatus int

const (
	OrderNotPaid     = OrderStatus(0)
	OrderScanned     = OrderStatus(1) // 已扫码
	OrderPaidSuccess = OrderStatus(2)
	OrderZyz         = OrderStatus(8)
	OrderYjs         = OrderStatus(9)
)

type OrderRemark struct {
	Days     float64 `json:"days"`  // 有效期
	Power    float64 `json:"power"` // 增加算力点数
	Name     string  `json:"name"`  // 产品名称
	Price    float64 `json:"price"`
	Discount float64 `json:"discount"`
}
