package model

// Product 充值产品
type Product struct {
	BaseModel
	Name     string
	Price    float64
	Discount float64
	Days     float64
	Power    float64
	Enabled  bool
	Sales    float64
	SortNum  int
}
