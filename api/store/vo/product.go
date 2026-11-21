package vo

type Product struct {
	BaseVo
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Discount float64 `json:"discount"`
	Days     float64 `json:"days"`
	Power    float64 `json:"power"`
	Enabled  bool    `json:"enabled"`
	Sales    float64 `json:"sales"`
	SortNum  int     `json:"sort_num"`
}
