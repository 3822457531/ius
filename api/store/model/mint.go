package model

// Mint 挖矿产品列表
type Mint struct {
	BaseModel
	Name string  //名称
	Fee  float64 //每天产出费率
	//Discount float64
	Days    float64
	Power   float64 //消耗算力
	Enabled bool    //是否启用
	Sales   int
	SortNum int
}
