package vo

type Trade struct {
	BaseVo
	OrderNumber     string
	EthAddress      string
	UserId          int
	Number          float64
	Price           float64
	CreateTime      int64
	TargetUserId    int
	MatchTime       int64
	Image           string //图片上传路径
	PayTime         int64  //支付确认时间
	FinishTime      int64  //交易订单结束时间
	Status          int    //订单状态 状态 1 初始化 2 匹配中 3 已完成  4其它(买家取消，卖家未收款，买家超时未支付)
	ChargeNumber    float64
	IsComment       bool    `gorm:"default:false"` // 是否评论
	Types           int     //订单类型 0 买单 1卖单
	TotalPrice      float64 //总价值
	TotalPriceChina float64 //总价值
	UserName        string  //用户名
	TargetUserName  string  //用户名
	Avatar          string  //头像
	Id              uint
	CreatedAt       int64
	UpdatedAt       int64
}
