package vo

type PowerTradeOrder struct {
	Id                 uint    `json:"id"`
	OrderNumber        string  `json:"orderNumber"`
	Role               string  `json:"role"`
	Status             int     `json:"status"`
	ProofUrl           string  `json:"proofUrl"`
	ListedPower        float64 `json:"listedPower"`
	DestroyedPower     float64 `json:"destroyedPower"`
	TotalPower         float64 `json:"totalPower"`
	Price              float64 `json:"price"`
	ValueUsd           float64 `json:"valueUsd"`
	CreatedAt          int64   `json:"createdAt"`
	MatchTime          int64   `json:"matchTime"`
	PayTime            int64   `json:"payTime"`
	FinishTime         int64   `json:"finishTime"`
	CounterpartyName   string  `json:"counterpartyName"`
	CounterpartyAvatar string  `json:"counterpartyAvatar"`
	EthAddress         string  `json:"ethAddress"`         // 兼容旧字段，同 paymentAddress
	PaymentType        string  `json:"paymentType"`        // bsc / alipay
	PaymentAddress     string  `json:"paymentAddress"`     // 卖家收款账号
	SellerName         string  `json:"sellerName"`         // 卖家用户名
	BuyerName          string  `json:"buyerName"`   // 买家用户名
}
