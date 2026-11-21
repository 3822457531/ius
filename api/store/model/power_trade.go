package model

// PowerTrade keeps power pool orders in a dedicated table.
type PowerTrade struct {
	BaseModel
	OrderNumber     string
	UserId          uint
	Number          float64
	Price           float64
	CreateTime      int64
	TargetUserId    int
	MatchTime       int64
	Image           string
	PayTime         int64
	FinishTime      int64
	Status          int
	ChargeNumber    float64
	IsComment       bool
	Types           int
	TotalPrice      float64
	TotalPriceChina float64
	UserName        string
	avatar          string
	EthAddress      string
}
