package admin

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/handler"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils"
	"chatplus/utils/resp"
	"fmt"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderHandler struct {
	handler.BaseHandler
}

func NewOrderHandler(app *core.AppServer, db *gorm.DB) *OrderHandler {
	return &OrderHandler{BaseHandler: handler.BaseHandler{App: app, DB: db}}
}

func (h *OrderHandler) List(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		resp.NotPermission(c)
		return
	}

	var data struct {
		OrderNo  string   `json:"order_no"`
		Status   int      `json:"status"`
		PayTime  []string `json:"pay_time"`
		Page     int      `json:"page"`
		PageSize int      `json:"page_size"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	session := h.DB.Session(&gorm.Session{})
	if data.OrderNo != "" {
		session = session.Where("order_no", data.OrderNo)
	}
	if len(data.PayTime) == 2 {
		start := utils.Str2stamp(data.PayTime[0] + " 00:00:00")
		end := utils.Str2stamp(data.PayTime[1] + " 00:00:00")
		session = session.Where("pay_time >= ? AND pay_time <= ?", start, end)
	}
	if data.Status >= 0 {
		session = session.Where("status", data.Status)
	}
	var total int64
	session.Model(&model.Order{}).Count(&total)
	var items []model.Order
	var list = make([]vo.Order, 0)
	offset := (data.Page - 1) * data.PageSize
	res := session.Order("id DESC").Offset(offset).Limit(data.PageSize).Find(&items)
	if res.Error == nil {
		for _, item := range items {
			var order vo.Order
			err := utils.CopyObject(item, &order)
			if err == nil {
				order.Id = item.Id
				order.CreatedAt = item.CreatedAt.Unix()
				order.UpdatedAt = item.UpdatedAt.Unix()
				list = append(list, order)
			} else {
				logger.Error(err)
			}
		}
	}
	resp.SUCCESS(c, vo.NewPage(total, data.Page, data.PageSize, list))
}

func (h *OrderHandler) Remove(c *gin.Context) {
	id := h.GetInt(c, "id", 0)

	if id > 0 {
		var item model.Order
		res := h.DB.First(&item, id)
		if res.Error != nil {
			resp.ERROR(c, "记录不存在！")
			return
		}

		if item.Status == types.OrderPaidSuccess {
			resp.ERROR(c, "已支付订单不允许删除！")
			return
		}

		res = h.DB.Unscoped().Where("id = ?", id).Delete(&model.Order{})
		if res.Error != nil {
			resp.ERROR(c, "更新数据库失败！")
			return
		}
	}
	resp.SUCCESS(c)
}

// 模拟订单支付成功
func (h *OrderHandler) PayComplete(c *gin.Context) {

	id := h.GetInt(c, "id", 0)

	if id > 0 {
		var item model.Order
		res := h.DB.First(&item, id)
		if res.Error != nil {
			resp.ERROR(c, "记录不存在！")
			return
		}

		if item.Status == types.OrderPaidSuccess {
			resp.ERROR(c, "已支付订单不允许再次支付！")
			return
		}

		res = h.DB.Model(&model.Order{}).Where("id = ?", id).UpdateColumns(model.Order{Status: types.OrderPaidSuccess, PayTime: time.Now().Unix()})

		//获取用户id
		userId := item.UserId
		//logger.Info("userId--->", userId)

		//判断用户算力余额
		var user model.User
		res = h.DB.First(&user, userId)
		if res.Error != nil {
			resp.ERROR(c, "无效的用户id")
			return
		}

		var product model.Product
		productRes := h.DB.First(&product, item.ProductId)
		if productRes.Error != nil {
			resp.ERROR(c, "产品不存在")
			return
		}

		//更新流水
		resCheck := h.DB.Create(&model.PowerLog{
			UserId:    user.Id,
			Username:  user.Username,
			Type:      types.PowerRecharge,
			Amount:    0.0,
			Balance:   user.Power,
			Mark:      types.PowerSub,
			Remark:    fmt.Sprintf("用户usdt付款审核通过：%s套餐,花费usdt：%f，订单号：%s", product.Name, item.Price, item.OrderNo),
			CreatedAt: time.Now(),
		})

		if resCheck.Error != nil {
			resp.ERROR(c, resCheck.Error.Error())
			return
		}

		//更新用户的订单状态
		//更改为质押中
		res = h.DB.Model(&model.Order{}).Where("id = ?", id).UpdateColumns(model.Order{Status: types.OrderZyz, PayTime: time.Now().Unix()})

		//获取上下级用户的质押中的算力（根据相应的等级来解锁比例）
		//获取直接推荐人用户的质押中算力
		var zjOrder []model.Order
		logger.Info("用户直接上级为", user.ZjFid)

		//用户json 解析
		var models []int
		utils.JsonDecode(user.FidsJson, &models)

		var ids [10]int
		ids[0] = user.ZjFid
		ids[1] = user.JjFid
		ids[2] = user.SanFid
		ids[3] = user.SiFid

		if len(models) != 0 {
			for i := 0; i < len(models); i++ {
				ids[i+4] = models[i]
			}
		}

		//获取用户的所有上级关系（已取消间推 i==1 的加速释放）
		for i := 0; i < len(ids); i++ {
			if i == 1 {
				continue
			}

			if ids[i] > 0 {
				//添加一个判断
				res = h.DB.Model(&model.Order{}).Where("user_id = ? and status=8 ", ids[i]).Find(&zjOrder)
				logger.Info(zjOrder)

				if len(zjOrder) == 0 {
					continue
				}

				var amountArray []int

				// 分配内存空间
				amountArray = make([]int, len(zjOrder))

				//先获取数组
				for i, order := range zjOrder {
					amountArray[i] = int(order.Amount)
				}
				//排序
				sort.Ints(amountArray)

				//查询
				//found := sort.SearchInts(amountArray, product.SortNum) != len(amountArray)

				index := sort.SearchInts(amountArray, product.SortNum)
				found := index < len(amountArray) && amountArray[index] == product.SortNum

				logger.Info("found值为", found)

				//如果存在则级
				if found {

					logger.Info("在上级中存在")

					res = h.DB.Model(&model.Order{}).Where("user_id = ? and status=8 ", ids[i]).Find(&zjOrder)

					for _, item := range zjOrder {

						//判读级别相等为止
						if int(item.Amount) == product.SortNum {

							var order vo.Order
							err := utils.CopyObject(item, &order)
							if err == nil {
								order.Id = item.Id
								order.CreatedAt = item.CreatedAt.Unix()
								order.UpdatedAt = item.UpdatedAt.Unix()

								//计算值
								var scale = 1.0

								//解锁比例
								switch i {
								case 0:
									scale = h.App.SysConfig.ZjPro
									break
								case 1:
									scale = h.App.SysConfig.JjPro
									break
								case 2:
									scale = h.App.SysConfig.SanPro
									break
								case 3:
									scale = h.App.SysConfig.SiPro
									break
								default:
									scale = h.App.SysConfig.OtherPro
									break
								}
								logger.Info(scale)

								//解锁算力计算
								//precentPower := float64(product.Power) / float64(scale)

								var turesubPower float64

								//实际计算值来计算
								turesubPower = item.Sales * scale //大于最大按照最大计算

								//判断当前算力是否够释放 后期处理

								//解锁到最大为止
								var nowPower = order.NowPower
								logger.Info("order.NowPower为", order.NowPower)

								//没有解锁过

								if order.NowPower != 0 && nowPower >= turesubPower {
									logger.Info("正常解锁")
									nowPower = nowPower - turesubPower

								} else if order.NowPower != 0 && nowPower < turesubPower {
									logger.Info("非正常解锁")

									turesubPower = nowPower
									nowPower = 0.0

								}

								logger.Info("turesubPower值为", turesubPower)
								logger.Info("nowPower值为", nowPower)

								//测试阶段
								res = h.DB.Model(&model.Order{}).Where("order_no = ?", item.OrderNo).UpdateColumn("js_power", gorm.Expr("js_power + ?", turesubPower))
								res = h.DB.Model(&model.Order{}).Where("order_no = ?", item.OrderNo).UpdateColumn("now_power", nowPower)

								if nowPower == 0.0 {
									res = h.DB.Model(&model.Order{}).Where("order_no = ?", item.OrderNo).UpdateColumn("status", "9")
								}

								//增加用户可用算力
								//判断
								h.DB.Model(&model.User{}).Where("id = ?", item.UserId).UpdateColumn("power", gorm.Expr("power + ?", turesubPower))

								//修改log bug
								var orderUser model.User
								res = h.DB.First(&orderUser, item.UserId)

								//上下级用户流水
								h.DB.Create(&model.PowerLog{
									UserId:   item.UserId,
									Username: item.Username,
									Type:     types.PowerSf, //释放
									Amount:   turesubPower,
									Balance:  orderUser.Power,
									Mark:     types.PowerAdd,
									Model:    order.Subject,
									//Remark:    fmt.Sprintf("%s，算力：%f，订单号：%s", "", user.Power, item.OrderNo),
									//Remark:    fmt.Sprintf("下级用户充值加速释放算力：%f，用户名：%s", turesubPower, user.Username),
									Remark:    fmt.Sprintf("下级用户 用户名：%s 充值:%s 产品  加速释放算力：%f，", user.Username, product.Name, turesubPower),
									CreatedAt: time.Now(),
								})

							} else {
								logger.Error(err)
							}
						}
					}

				} else {
					logger.Info("在上级中不存在")

					//小于最小
					//	amountArraylen := len(amountArray)
					var amountArrayleni = len(amountArray)
					i2 := amountArrayleni - 1
					if amountArray[0] > product.SortNum {

						logger.Info("小于最小", i2)

						var order model.Order
						h.DB.Where("user_id = ? And  amount = ?   and status=8 ", ids[i], amountArray[0]).First(&order)

						//计算值
						var scale = 1.0

						//解锁比例
						switch i {
						case 0:
							scale = h.App.SysConfig.ZjPro
							break
						case 1:
							scale = h.App.SysConfig.JjPro
							break
						case 2:
							scale = h.App.SysConfig.SanPro
							break
						case 3:
							scale = h.App.SysConfig.SiPro
							break
						default:
							scale = h.App.SysConfig.OtherPro
							break
						}
						logger.Info(scale)

						//解锁算力计算
						//precentPower := float64(product.Power) / float64(scale)
						//precentPower := product.Sales * scale

						var turesubPower float64

						//实际计算值来计算
						turesubPower = product.Sales * scale //大于最大按照最大计算

						//判断当前算力是否够释放 后期处理

						//解锁到最大为止
						var nowPower = order.NowPower
						logger.Info("order.NowPower为", nowPower)

						//没有解锁过
						if order.NowPower != 0 && nowPower >= turesubPower {
							logger.Info("正常解锁")
							nowPower = nowPower - turesubPower

						} else if order.NowPower != 0 && nowPower < turesubPower {
							logger.Info("非正常解锁")
							turesubPower = order.NowPower
							nowPower = 0.0

						}

						logger.Info("turesubPower值为", turesubPower)
						logger.Info("nowPower值为", nowPower)

						//测试阶段
						res = h.DB.Model(&model.Order{}).Where("order_no = ?", order.OrderNo).UpdateColumn("js_power", gorm.Expr("js_power + ?", turesubPower))
						res = h.DB.Model(&model.Order{}).Where("order_no = ?", order.OrderNo).UpdateColumn("now_power", nowPower)

						//res = h.DB.Model(&model.Order{}).Where("order_no = ?", order.OrderNo).UpdateColumn("now_power", gorm.Expr("now_power - ?", nowPower))

						if nowPower == 0.0 {
							res = h.DB.Model(&model.Order{}).Where("order_no = ?", order.OrderNo).UpdateColumn("status", "9")
						}

						//增加用户可用算力
						//判断
						h.DB.Model(&model.User{}).Where("id = ?", order.UserId).UpdateColumn("power", gorm.Expr("power + ?", turesubPower))

						//修改log bug
						var orderUser model.User
						res = h.DB.First(&orderUser, order.UserId)

						//上下级用户流水
						h.DB.Create(&model.PowerLog{
							UserId:   order.UserId,
							Username: order.Username,
							Type:     types.PowerSf, //释放
							Amount:   turesubPower,
							Balance:  orderUser.Power,
							Mark:     types.PowerAdd,
							Model:    order.Subject,
							//Remark:    fmt.Sprintf("%s，算力：%f，订单号：%s", "", user.Power, item.OrderNo),
							//Remark:    fmt.Sprintf("下级用户充值加速释放算力：%f，用户名：%s", turesubPower, user.Username),
							Remark:    fmt.Sprintf("下级用户 用户名：%s 充值:%s 产品  加速释放算力：%f，", user.Username, product.Name, turesubPower),
							CreatedAt: time.Now(),
						})

						//在其中
					} else if product.SortNum > amountArray[0] && product.SortNum < amountArray[i2] {

						logger.Info("在其中 解锁最大产品套餐", i2)
						var order model.Order
						h.DB.Where("user_id = ? And  amount = ?  and status=8 ", ids[i], amountArray[i2]).First(&order)

						//计算值
						var scale = 1.0

						//解锁比例
						switch i {
						case 0:
							scale = h.App.SysConfig.ZjPro
							break
						case 1:
							scale = h.App.SysConfig.JjPro
							break
						case 2:
							scale = h.App.SysConfig.SanPro
							break
						case 3:
							scale = h.App.SysConfig.SiPro
							break
						default:
							scale = h.App.SysConfig.OtherPro
							break
						}
						logger.Info(scale)

						//解锁算力计算

						var turesubPower float64

						//实际计算值来计算
						//	turesubPower = order.Sales * scale //大于最大按照最大计算
						turesubPower = product.Sales * scale //大于最大按照最大计算 解锁最大产品订单的下一级值

						//判断当前算力是否够释放 后期处理

						//解锁到最大为止
						var nowPower = order.NowPower
						//var sfEnd = false
						logger.Info("order.NowPower为", order.NowPower)

						//没有解锁过

						//没有解锁过
						if nowPower != 0 && nowPower >= turesubPower {
							logger.Info("正常解锁")
							nowPower = nowPower - turesubPower

						} else if order.NowPower != 0 && nowPower < turesubPower {

							turesubPower = order.NowPower
							nowPower = 0.0
							//	sfEnd = true

						}

						logger.Info("turesubPower值为", turesubPower)
						logger.Info("nowPower值为", nowPower)

						//测试阶段
						res = h.DB.Model(&model.Order{}).Where("order_no = ?", order.OrderNo).UpdateColumn("js_power", gorm.Expr("js_power + ?", turesubPower))
						res = h.DB.Model(&model.Order{}).Where("order_no = ?", order.OrderNo).UpdateColumn("now_power", nowPower)

						if nowPower == 0.0 {
							res = h.DB.Model(&model.Order{}).Where("order_no = ?", order.OrderNo).UpdateColumn("status", "9")
						}

						//增加用户可用算力
						//判断
						h.DB.Model(&model.User{}).Where("id = ?", order.UserId).UpdateColumn("power", gorm.Expr("power + ?", turesubPower))

						//修改log bug
						var orderUser model.User
						res = h.DB.First(&orderUser, order.UserId)

						//上下级用户流水
						h.DB.Create(&model.PowerLog{
							UserId:   order.UserId,
							Username: order.Username,
							Type:     types.PowerSf, //释放
							Amount:   turesubPower,
							Balance:  orderUser.Power,
							Mark:     types.PowerAdd,
							Model:    order.Subject,
							//Remark:    fmt.Sprintf("%s，算力：%f，订单号：%s", "", user.Power, item.OrderNo),
							//	Remark:    fmt.Sprintf("下级用户充值加速释放算力：%f，用户名：%s", turesubPower, user.Username),
							Remark:    fmt.Sprintf("下级用户 用户名：%s 充值:%s 产品  加速释放算力：%f，", user.Username, product.Name, turesubPower),
							CreatedAt: time.Now(),
						})

						//按照上级用户系统最大值计算解锁
						//大于最大
					} else {

						logger.Info("大于最大", i2)
						var order model.Order
						h.DB.Where("user_id = ? And  amount = ?  and status=8", ids[i], amountArray[i2]).First(&order)

						//计算值
						var scale = 1.0

						//解锁比例
						switch i {
						case 0:
							scale = h.App.SysConfig.ZjPro
							break
						case 1:
							scale = h.App.SysConfig.JjPro
							break
						case 2:
							scale = h.App.SysConfig.SanPro
							break
						case 3:
							scale = h.App.SysConfig.SiPro
							break
						default:
							scale = h.App.SysConfig.OtherPro
							break
						}
						logger.Info(scale)

						var turesubPower float64

						//实际计算值来计算
						turesubPower = order.Sales * scale //大于最大按照最大计算

						//解锁到最大为止
						var nowPower = order.NowPower
						logger.Info("order.NowPower为", nowPower)

						//没有解锁过
						if order.NowPower != 0 && nowPower >= turesubPower {
							logger.Info("正常解锁")
							nowPower = nowPower - turesubPower

						} else if order.NowPower != 0 && nowPower < turesubPower {

							turesubPower = order.NowPower
							nowPower = 0.0
							//	sfEnd = true

						}

						logger.Info("turesubPower值为", turesubPower)
						logger.Info("nowPower值为", nowPower)

						//测试阶段
						res = h.DB.Model(&model.Order{}).Where("order_no = ?", order.OrderNo).UpdateColumn("js_power", gorm.Expr("js_power + ?", turesubPower))

						res = h.DB.Model(&model.Order{}).Where("order_no = ?", order.OrderNo).UpdateColumn("now_power", nowPower)

						if nowPower == 0.0 {
							res = h.DB.Model(&model.Order{}).Where("order_no = ?", order.OrderNo).UpdateColumn("status", "9")
						}

						//增加用户可用算力
						//判断
						h.DB.Model(&model.User{}).Where("id = ?", order.UserId).UpdateColumn("power", gorm.Expr("power + ?", turesubPower))

						//修改log bug
						var orderUser model.User
						res = h.DB.First(&orderUser, order.UserId)

						//上下级用户流水
						h.DB.Create(&model.PowerLog{
							UserId:   order.UserId,
							Username: order.Username,
							Type:     types.PowerSf, //释放
							Amount:   turesubPower,
							Balance:  orderUser.Power,
							Mark:     types.PowerAdd,
							Model:    order.Subject,
							//Remark:    fmt.Sprintf("%s，算力：%f，订单号：%s", "", user.Power, item.OrderNo),
							Remark:    fmt.Sprintf("下级用户 用户名：%s 充值:%s 产品  加速释放算力：%f，", user.Username, product.Name, turesubPower),
							CreatedAt: time.Now(),
						})
						//按照上级用户系统最大值计算解锁

					}

				}

			}

		}

		//var orderPower = h.DB.Model(&model.Order{}).Where("user_id = ?", user.JjFid).Where("product_id = ?", data.ProductId).UpdateColumn("js_power", gorm.Expr("js_power + ?", product.Power))
		resp.SUCCESS(c)

	}
}

// 合伙人每月一次分红
func (h *UserHandler) Dividend(c *gin.Context) {

	//获取所有合伙人
	//var h = UserHandler{}
	var users []model.User
	h.DB.Model(UserHandler{}).Where("is_partner = ?", 1).Find(&users)

	//合伙人数量
	var userCount int64
	h.DB.Model(&model.User{}).Where("is_partner = ?", 1).Count(&userCount)

	//获取当月充值积分
	stats := statsVo{}
	var orders []model.Order

	now := time.Now()
	zeroTime := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	h.DB.Where("status = ?", types.OrderPaidSuccess).Where("created_at > ?", zeroTime).Find(&orders)
	for _, item := range orders {
		var remark types.OrderRemark
		utils.JsonDecode(item.Remark, &remark)
		power := remark.Power
		stats.Tokens += power //当月充值的积分总额
	}

	//计算每人获得积分数量
	result := stats.Tokens / 10
	//fmt.Println("当月积分总额的10分之一为", result)
	everyPort := int64(result) / userCount
	//fmt.Println("当月积分总额的10分之一为", everyPort)

	for _, user := range users {
		//增加相应的积分
		h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumn("power", gorm.Expr("power + ?", everyPort))

		//增加流水记录
		h.DB.Create(&model.PowerLog{
			UserId:    user.Id,
			Username:  user.Username,
			Type:      types.PowerRecharge,
			Amount:    float64(everyPort),
			Balance:   user.Power,
			Mark:      types.PowerAdd,
			Model:     "",
			Remark:    fmt.Sprintf("%s，金额：%f，订单号：%s", "合伙人每月分红", "", ""),
			CreatedAt: time.Now(),
		})
	}
	resp.SUCCESS(c, "每月合伙人分红成功")
}
