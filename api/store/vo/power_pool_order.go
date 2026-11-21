package vo

type PowerPoolOrder struct {
	Id             uint    `json:"id"`
	OrderNumber    string  `json:"orderNumber"`
	Username       string  `json:"username"`
	Avatar         string  `json:"avatar"`
	ListedPower    float64 `json:"listedPower"`
	DestroyedPower float64 `json:"destroyedPower"`
	TotalPower     float64 `json:"totalPower"`
	Price          float64 `json:"price"`
	ValueUsd       float64 `json:"valueUsd"`
	Status         int     `json:"status"`
	CreatedAt      int64   `json:"createdAt"`
	EthAddress     string  `json:"ethAddress"`
	PaymentType    string  `json:"paymentType"`
	PaymentAddress string  `json:"paymentAddress"`
}
