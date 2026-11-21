package vo

type Mint struct {
	BaseVo
	Name     string  `json:"name"`
	Fee      float64 `json:"fee"`
	Discount float64 `json:"discount"`
	Days     int     `json:"days"`
	Power    float64 `json:"power"`
	Enabled  bool    `json:"enabled"`
	Sales    int     `json:"sales"`
	SortNum  int     `json:"sort_num"`
}
