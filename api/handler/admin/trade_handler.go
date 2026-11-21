package admin

import (
	"bytes"
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/handler"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils"
	"chatplus/utils/resp"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

//var logger = logger2.GetLogger()

type TradeHandler struct {
	handler.BaseHandler
}

func NewTradeHandler(app *core.AppServer, db *gorm.DB) *TradeHandler {
	return &TradeHandler{BaseHandler: handler.BaseHandler{App: app, DB: db}}
}

// 获取交易订单情况
func (h *TradeHandler) ListBuy(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		fmt.Printf("Permission check failed: %v\n", err)
		resp.NotPermission(c)
		return
	}

	var data struct {
		OrderNo  string   `json:"order_no"`
		Status   *int     `json:"status"`
		PayTime  []string `json:"pay_time"`
		Page     int      `json:"page"`
		PageSize int      `json:"page_size"`
	}

	// 添加请求数据日志
	rawData, _ := c.GetRawData()
	fmt.Printf("ListBuy - Raw request data: %s\n", string(rawData))

	// 重新设置请求体
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(rawData))

	if err := c.ShouldBindJSON(&data); err != nil {
		fmt.Printf("ListBuy - Bind JSON error: %v\n", err)
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	fmt.Printf("ListBuy - Received data: %+v\n", data)

	// 设置默认分页参数
	if data.Page <= 0 {
		data.Page = 1
	}
	if data.PageSize <= 0 {
		data.PageSize = 15
	}

	session := h.DB.Session(&gorm.Session{})

	// 修复查询条件
	if data.OrderNo != "" {
		session = session.Where("trades.order_no = ?", data.OrderNo)
		fmt.Printf("ListBuy - Added order_no filter: %s\n", data.OrderNo)
	}

	if data.Status != nil {
		session = session.Where("trades.status = ?", *data.Status)
		fmt.Printf("ListBuy - Added status filter: %d\n", *data.Status)
	}

	if len(data.PayTime) == 2 {
		start := utils.Str2stamp(data.PayTime[0] + " 00:00:00")
		end := utils.Str2stamp(data.PayTime[1] + " 23:59:59")
		session = session.Where("trades.pay_time BETWEEN ? AND ?", start, end)
		fmt.Printf("ListBuy - Added time filter: %s to %s\n", data.PayTime[0], data.PayTime[1])
	}

	// 统计总数
	var total int64
	countSession := session
	countResult := countSession.Model(&model.Trade{}).Count(&total)
	if countResult.Error != nil {
		fmt.Printf("ListBuy - Count error: %v\n", countResult.Error)
		resp.ERROR(c, "统计数据失败")
		return
	}
	fmt.Printf("ListBuy - Total count: %d\n", total)

	// 查询数据 - 添加关联查询用户表
	var items []struct {
		model.Trade
		Username       string `gorm:"column:username"`
		TargetUserName string `gorm:"column:target_username"`
	}
	var list = make([]vo.Trade, 0)
	offset := (data.Page - 1) * data.PageSize

	fmt.Printf("ListBuy - Query params: offset=%d, limit=%d\n", offset, data.PageSize)

	// 使用两次 Joins 关联查询 users 表获取两个用户名
	res := session.Table("chatgpt_trades").
		Select("chatgpt_trades.*, u1.username, u2.username as target_username").
		Joins("LEFT JOIN chatgpt_users u1 ON chatgpt_trades.user_id = u1.id").
		Joins("LEFT JOIN chatgpt_users u2 ON chatgpt_trades.target_user_id = u2.id").
		Order("chatgpt_trades.id DESC").
		Offset(offset).
		Limit(data.PageSize).
		Find(&items)

	if res.Error != nil {
		fmt.Printf("ListBuy - Query error: %v\n", res.Error)
		resp.ERROR(c, "查询数据失败")
		return
	}

	fmt.Printf("ListBuy - Found %d items\n", len(items))

	// 处理查询结果
	for i, item := range items {
		var order vo.Trade
		if err := utils.CopyObject(item.Trade, &order); err == nil {
			order.Id = item.Trade.Id
			order.CreatedAt = item.Trade.CreatedAt.Unix()
			order.UpdatedAt = item.Trade.UpdatedAt.Unix()
			order.UserName = item.Username             // 添加用户名字段
			order.TargetUserName = item.TargetUserName // 添加目标用户名字段
			list = append(list, order)
			fmt.Printf("ListBuy - Processed item %d: ID=%d, Username=%s, TargetUserName=%s\n", i+1, item.Trade.Id, item.Username, item.TargetUserName)
		} else {
			fmt.Printf("ListBuy - Copy object error for item %d: %v\n", i+1, err)
		}
	}

	result := vo.NewPage(total, data.Page, data.PageSize, list)
	fmt.Printf("ListBuy - Final result: total=%d, page=%d, pageSize=%d, items=%d\n",
		total, data.Page, data.PageSize, len(list))

	resp.SUCCESS(c, result)
}

// 获取交易订单情况
func (h *TradeHandler) ListSell(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		fmt.Printf("Permission check failed: %v\n", err)
		resp.NotPermission(c)
		return
	}

	var data struct {
		OrderNo  string   `json:"order_no"`
		Status   *int     `json:"status"`
		PayTime  []string `json:"pay_time"`
		Page     int      `json:"page"`
		PageSize int      `json:"page_size"`
	}

	// 添加请求数据日志
	rawData, _ := c.GetRawData()
	fmt.Printf("ListBuy - Raw request data: %s\n", string(rawData))

	// 重新设置请求体
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(rawData))

	if err := c.ShouldBindJSON(&data); err != nil {
		fmt.Printf("ListBuy - Bind JSON error: %v\n", err)
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	fmt.Printf("ListBuy - Received data: %+v\n", data)

	// 设置默认分页参数
	if data.Page <= 0 {
		data.Page = 1
	}
	if data.PageSize <= 0 {
		data.PageSize = 15
	}

	session := h.DB.Session(&gorm.Session{})

	// 修复查询条件
	if data.OrderNo != "" {
		session = session.Where("trades.order_no = ?", data.OrderNo)
		fmt.Printf("ListBuy - Added order_no filter: %s\n", data.OrderNo)
	}

	if data.Status != nil {
		session = session.Where("trades.status = ?", *data.Status)
		fmt.Printf("ListBuy - Added status filter: %d\n", *data.Status)
	}

	if len(data.PayTime) == 2 {
		start := utils.Str2stamp(data.PayTime[0] + " 00:00:00")
		end := utils.Str2stamp(data.PayTime[1] + " 23:59:59")
		session = session.Where("trades.pay_time BETWEEN ? AND ?", start, end)
		fmt.Printf("ListBuy - Added time filter: %s to %s\n", data.PayTime[0], data.PayTime[1])
	}

	// 统计总数
	var total int64
	countSession := session
	countResult := countSession.Model(&model.Trade{}).Count(&total)
	if countResult.Error != nil {
		fmt.Printf("ListBuy - Count error: %v\n", countResult.Error)
		resp.ERROR(c, "统计数据失败")
		return
	}
	fmt.Printf("ListBuy - Total count: %d\n", total)

	// 查询数据 - 添加关联查询用户表
	var items []struct {
		model.Trade
		Username       string `gorm:"column:username"`
		TargetUserName string `gorm:"column:target_username"`
	}
	var list = make([]vo.Trade, 0)
	offset := (data.Page - 1) * data.PageSize

	fmt.Printf("ListBuy - Query params: offset=%d, limit=%d\n", offset, data.PageSize)

	// 使用两次 Joins 关联查询 users 表获取两个用户名
	res := session.Table("chatgpt_trades").
		Select("chatgpt_trades.*, u1.username, u2.username as target_username").
		Joins("LEFT JOIN chatgpt_users u1 ON chatgpt_trades.user_id = u1.id").
		Joins("LEFT JOIN chatgpt_users u2 ON chatgpt_trades.target_user_id = u2.id").
		Order("chatgpt_trades.id DESC").
		Offset(offset).
		Limit(data.PageSize).
		Find(&items)

	if res.Error != nil {
		fmt.Printf("ListBuy - Query error: %v\n", res.Error)
		resp.ERROR(c, "查询数据失败")
		return
	}

	fmt.Printf("ListBuy - Found %d items\n", len(items))

	// 处理查询结果
	for i, item := range items {
		var order vo.Trade
		if err := utils.CopyObject(item.Trade, &order); err == nil {
			order.Id = item.Trade.Id
			order.CreatedAt = item.Trade.CreatedAt.Unix()
			order.UpdatedAt = item.Trade.UpdatedAt.Unix()
			order.UserName = item.Username             // 添加用户名字段
			order.TargetUserName = item.TargetUserName // 添加目标用户名字段
			list = append(list, order)
			fmt.Printf("ListBuy - Processed item %d: ID=%d, Username=%s, TargetUserName=%s\n", i+1, item.Trade.Id, item.Username, item.TargetUserName)
		} else {
			fmt.Printf("ListBuy - Copy object error for item %d: %v\n", i+1, err)
		}
	}

	result := vo.NewPage(total, data.Page, data.PageSize, list)
	fmt.Printf("ListBuy - Final result: total=%d, page=%d, pageSize=%d, items=%d\n",
		total, data.Page, data.PageSize, len(list))

	resp.SUCCESS(c, result)
}

func (h *TradeHandler) Remove(c *gin.Context) {
	id := h.GetInt(c, "id", 0)

	if id > 0 {
		var item model.Trade
		res := h.DB.First(&item, id)
		if res.Error != nil {
			resp.ERROR(c, "记录不存在！")
			return
		}

		if item.Status == 2 {
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

// 管理员取消订单
func (h *TradeHandler) CancelOrder(c *gin.Context) {
	var data struct {
		Id int `json:"orderId" form:"orderId"` //订单id
	}

	// 对于GET请求使用Query参数绑定，对于POST请求使用JSON绑定
	if c.Request.Method == "GET" {
		if err := c.ShouldBindQuery(&data); err != nil {
			resp.ERROR(c, types.InvalidArgs)
			return
		}
	} else {
		if err := c.ShouldBindJSON(&data); err != nil {
			resp.ERROR(c, types.InvalidArgs)
			return
		}
	}

	logger.Info("收到取消订单请求，订单ID：", data.Id)
	// 获取订单信息
	var trade model.Trade
	res := h.DB.First(&trade, data.Id)
	if res.Error != nil {
		resp.ERROR(c, "No trade found")
		return
	}

	resCheck := h.DB.Model(&trade).Where("id = ?", data.Id).Updates(model.Trade{
		Status: 5, //订单已取消
	})

	fmt.Printf("Database response: %+v\n", resCheck) // 调试信息

	if resCheck.Error != nil {
		resp.ERROR(c, resCheck.Error.Error())
		return
	}

	//取消之后释放卖家用户的额度
	//更改匹配用户的冻结额度

	if trade.Types == 2 {
		logger.Info("返回卖方的冻结金额，订单ID：", data.Id)

		h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin -?", trade.Number))
	}

	resp.SUCCESS(c)

}

// 确认支付
func (h *TradeHandler) ComfireOrder(c *gin.Context) {

	var data struct {
		Id int `json:"orderId" form:"orderId"` //订单id
	}

	// 对于GET请求使用Query参数绑定，对于POST请求使用JSON绑定
	if c.Request.Method == "GET" {
		if err := c.ShouldBindQuery(&data); err != nil {
			resp.ERROR(c, types.InvalidArgs)
			return
		}
	} else {
		if err := c.ShouldBindJSON(&data); err != nil {
			resp.ERROR(c, types.InvalidArgs)
			return
		}
	}

	fmt.Printf("Received data: %+v\n", data) // 调试信息

	// 获取订单信息
	var trade model.Trade
	res := h.DB.First(&trade, data.Id)
	if res.Error != nil {
		resp.ERROR(c, "No trade found")
		return
	}

	resCheck := h.DB.Model(&trade).Where("id = ?", data.Id).Updates(model.Trade{
		Status: 4, //已付款待确认
		//FinishTime: time.Now(), //已付款待确认
	})

	fmt.Printf("Database 6666 response: %+v\n", resCheck) // 调试信息

	if resCheck.Error != nil {
		resp.ERROR(c, resCheck.Error.Error())
		return
	}

	//用户SOHA币转账转移操作
	if trade.Types == 1 {

		logger.Info("111111111111")

		// 获取用户信息
		var buyUser model.User
		var sellUser model.User

		//解冻
		h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin -?", trade.Number))

		//更新 -   卖家
		h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("coin", gorm.Expr("coin - ?", trade.Number))

		//更新 + 买家
		h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("coin", gorm.Expr("coin + ?", trade.Number))

		res := h.DB.First(&buyUser, trade.UserId)
		if res.Error != nil {
			resp.ERROR(c, "No trade found")
			return
		}

		res2 := h.DB.First(&sellUser, trade.TargetUserId)
		if res2.Error != nil {
			resp.ERROR(c, "No trade found")
			return
		}

		//添加用户SOHA日志

		// 添加买家用户的Q2变化日志
		h.DB.Create(&model.Q2Log{
			UserId:   buyUser.Id,
			Username: buyUser.Username,
			Type:     types.CoinBuy,
			Amount:   trade.Number,
			//Mark:     types.PowerAdd,
			Mark:      types.PowerAdd,
			Balance:   buyUser.Coin,
			Model:     "自助下单买入SOHA",
			Remark:    fmt.Sprintf("用户自助下单买入SOHA，订单金额：%f，获得SOHA：%f", trade.Number*trade.Price, trade.Number),
			CreatedAt: time.Now(),
		})

		// 添加卖家用户的Q2变化日志
		h.DB.Create(&model.Q2Log{
			UserId:    sellUser.Id,
			Username:  sellUser.Username,
			Type:      types.CoinSell,
			Amount:    trade.Number,
			Mark:      types.PowerSub,
			Balance:   sellUser.Coin,
			Model:     "自助下单卖出SOHA",
			Remark:    fmt.Sprintf("用户自助下单卖出SOHA，订单金额：%f，消耗SOHA：%f", trade.Number*trade.Price, trade.Number),
			CreatedAt: time.Now(),
		})

	} else if trade.Types == 2 {

		// 获取用户信息
		var buyUser model.User
		var sellUser model.User

		//解冻
		h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin -?", trade.Number))

		//更新 - 卖家
		h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("coin", gorm.Expr("coin - ?", trade.Number))

		//更新 + 买家
		h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("coin", gorm.Expr("coin + ?", trade.Number))

		res := h.DB.First(&buyUser, trade.TargetUserId)
		if res.Error != nil {
			resp.ERROR(c, "No trade found")
			return
		}

		res2 := h.DB.First(&sellUser, trade.UserId)
		if res2.Error != nil {
			resp.ERROR(c, "No trade found")
			return
		}

		//添加用户SOHA日志

		// 添加买家用户的Q2变化日志
		h.DB.Create(&model.Q2Log{
			UserId:   buyUser.Id,
			Username: buyUser.Username,
			Type:     types.CoinBuy,
			Amount:   trade.Number,
			//Mark:     types.PowerAdd,
			Mark:      types.PowerAdd,
			Balance:   buyUser.Coin,
			Model:     "自助下单买入SOHA",
			Remark:    fmt.Sprintf("用户自助下单买入SOHA，订单金额：%f，获得SOHA：%f", trade.Number*trade.Price, trade.Number),
			CreatedAt: time.Now(),
		})

		// 添加卖家用户的Q2变化日志
		h.DB.Create(&model.Q2Log{
			UserId:    sellUser.Id,
			Username:  sellUser.Username,
			Type:      types.CoinSell,
			Amount:    trade.Number,
			Mark:      types.PowerSub,
			Balance:   sellUser.Coin,
			Model:     "自助下单卖出SOHA",
			Remark:    fmt.Sprintf("用户自助下单卖出SOHA，订单金额：%f，消耗SOHA：%f", trade.Number*trade.Price, trade.Number),
			CreatedAt: time.Now(),
		})

	}

	resp.SUCCESS(c)
}

// 获取算力撮合订单列表
func (h *TradeHandler) ListPowerTrade(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		fmt.Printf("Permission check failed: %v\n", err)
		resp.NotPermission(c)
		return
	}

	var data struct {
		OrderNo  string   `json:"order_no"`
		Status   *int     `json:"status"`
		PayTime  []string `json:"pay_time"`
		Page     int      `json:"page"`
		PageSize int      `json:"page_size"`
	}

	// 添加请求数据日志
	rawData, _ := c.GetRawData()
	fmt.Printf("ListPowerTrade - Raw request data: %s\n", string(rawData))

	// 重新设置请求体
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(rawData))

	if err := c.ShouldBindJSON(&data); err != nil {
		fmt.Printf("ListPowerTrade - Bind JSON error: %v\n", err)
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	fmt.Printf("ListPowerTrade - Received data: %+v\n", data)

	// 设置默认分页参数
	if data.Page <= 0 {
		data.Page = 1
	}
	if data.PageSize <= 0 {
		data.PageSize = 15
	}

	session := h.DB.Session(&gorm.Session{})

	// 修复查询条件
	if data.OrderNo != "" {
		session = session.Where("chatgpt_power_trades.order_number = ?", data.OrderNo)
		fmt.Printf("ListPowerTrade - Added order_no filter: %s\n", data.OrderNo)
	}

	if data.Status != nil {
		session = session.Where("chatgpt_power_trades.status = ?", *data.Status)
		fmt.Printf("ListPowerTrade - Added status filter: %d\n", *data.Status)
	}

	if len(data.PayTime) == 2 {
		start := utils.Str2stamp(data.PayTime[0] + " 00:00:00")
		end := utils.Str2stamp(data.PayTime[1] + " 23:59:59")
		session = session.Where("chatgpt_power_trades.created_at BETWEEN ? AND ?", start, end)
		fmt.Printf("ListPowerTrade - Added time filter: %s to %s\n", data.PayTime[0], data.PayTime[1])
	}

	// 统计总数
	var total int64
	countSession := session
	countResult := countSession.Table("chatgpt_power_trades").Count(&total)
	if countResult.Error != nil {
		fmt.Printf("ListPowerTrade - Count error: %v\n", countResult.Error)
		resp.ERROR(c, "统计数据失败")
		return
	}
	fmt.Printf("ListPowerTrade - Total count: %d\n", total)

	// 查询数据 - 添加关联查询用户表
	var items []struct {
		model.PowerTrade
		Username       string `gorm:"column:username"`
		TargetUserName string `gorm:"column:target_username"`
	}
	var list = make([]vo.PowerTradeOrder, 0)
	offset := (data.Page - 1) * data.PageSize

	fmt.Printf("ListPowerTrade - Query params: offset=%d, limit=%d\n", offset, data.PageSize)

	// 使用两次 Joins 关联查询 users 表获取两个用户名
	res := session.Table("chatgpt_power_trades").
		Select("chatgpt_power_trades.*, u1.username, u2.username as target_username").
		Joins("LEFT JOIN chatgpt_users u1 ON chatgpt_power_trades.user_id = u1.id").
		Joins("LEFT JOIN chatgpt_users u2 ON chatgpt_power_trades.target_user_id = u2.id").
		Order("chatgpt_power_trades.id DESC").
		Offset(offset).
		Limit(data.PageSize).
		Find(&items)

	if res.Error != nil {
		fmt.Printf("ListPowerTrade - Query error: %v\n", res.Error)
		resp.ERROR(c, "查询数据失败")
		return
	}

	fmt.Printf("ListPowerTrade - Found %d items\n", len(items))

	// 处理查询结果
	for i, item := range items {
		var order vo.PowerTradeOrder
		order.Id = item.PowerTrade.Id
		order.OrderNumber = item.PowerTrade.OrderNumber
		order.Status = item.PowerTrade.Status
		order.ProofUrl = item.PowerTrade.Image
		order.ListedPower = item.PowerTrade.Number
		order.DestroyedPower = item.PowerTrade.TotalPriceChina
		order.TotalPower = item.PowerTrade.TotalPrice
		order.Price = item.PowerTrade.Price
		order.ValueUsd = item.PowerTrade.Number * item.PowerTrade.Price
		order.CreatedAt = item.PowerTrade.CreatedAt.Unix()
		order.MatchTime = item.PowerTrade.MatchTime
		order.PayTime = item.PowerTrade.PayTime
		order.FinishTime = item.PowerTrade.FinishTime
		order.EthAddress = item.PowerTrade.EthAddress

		// user_id 是卖家, target_user_id 是买家
		order.SellerName = item.Username
		order.BuyerName = item.TargetUserName
		order.CounterpartyName = item.TargetUserName

		// 判断角色：如果 target_user_id 存在，说明已匹配
		if item.PowerTrade.TargetUserId > 0 {
			order.Role = "seller"
		} else {
			order.Role = "seller"
		}

		list = append(list, order)
		fmt.Printf("ListPowerTrade - Processed item %d: ID=%d, SellerName=%s, BuyerName=%s\n", i+1, item.PowerTrade.Id, item.Username, item.TargetUserName)
	}

	result := vo.NewPage(total, data.Page, data.PageSize, list)
	fmt.Printf("ListPowerTrade - Final result: total=%d, page=%d, pageSize=%d, items=%d\n",
		total, data.Page, data.PageSize, len(list))

	resp.SUCCESS(c, result)
}

// ConfirmPowerTradePayment 管理员确认付款到账
func (h *TradeHandler) ConfirmPowerTradePayment(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		fmt.Printf("Permission check failed: %v\n", err)
		resp.NotPermission(c)
		return
	}

	var data struct {
		OrderId uint `json:"orderId"`
	}

	if err := c.ShouldBindJSON(&data); err != nil || data.OrderId == 0 {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	var order model.PowerPoolRecharge
	if err := h.DB.First(&order, data.OrderId).Error; err != nil {
		resp.ERROR(c, "订单不存在或已被删除")
		return
	}

	if order.Status != 1 && order.Status != 2 {
		resp.ERROR(c, "当前状态不可确认到账，只有待支付或已支付状态的充值订单才能确认")
		return
	}
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 更新充值订单状态：待支付/已支付 -> 已完成
		res := tx.Model(&model.PowerPoolRecharge{}).
			Where("id = ? AND status IN ?", data.OrderId, []int{1, 2}).
			Updates(map[string]interface{}{
				"status":      3,
				"pay_time":    time.Now().Unix(),
				"finish_time": time.Now().Unix(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("订单状态已变更，无法确认")
		}

		// 获取算力池信息
		var pool model.PowerPool
		if err := tx.First(&pool, 1).Error; err != nil {
			return err
		}

		// 分配金额：70% 进算力池USDT，25% 进回购池，5% 平台留存
		poolUsdtIncrease := order.UsdtAmount * 0.70
		buybackUsdtIncrease := order.UsdtAmount * 0.25

		// 更新算力池：扣减算力库存，增加USDT
		if err := tx.Model(&pool).Updates(map[string]interface{}{
			"total_power":  gorm.Expr("total_power - ?", order.PowerAmount),
			"total_usdt":   gorm.Expr("total_usdt + ?", poolUsdtIncrease),
			"buyback_usdt": gorm.Expr("buyback_usdt + ?", buybackUsdtIncrease),
		}).Error; err != nil {
			return err
		}

		// 给用户增加算力
		if err := tx.Model(&model.User{}).
			Where("id = ?", order.UserId).
			Update("power", gorm.Expr("power + ?", order.PowerAmount)).Error; err != nil {
			return err
		}

		// 获取用户信息用于记录日志
		var user model.User
		if err := tx.First(&user, order.UserId).Error; err != nil {
			return err
		}

		// 记录算力变动日志
		return tx.Create(&model.PowerLog{
			UserId:    user.Id,
			Username:  user.Username,
			Type:      types.PowerDh,
			Amount:    order.PowerAmount,
			Balance:   user.Power,
			Mark:      types.PowerAdd,
			Model:     "算力",
			Remark:    fmt.Sprintf("算力池充值（管理员确认到账），购买算力：%.2f，支付USDT：%.2f，订单号：%s", order.PowerAmount, order.UsdtAmount, order.OrderNumber),
			CreatedAt: time.Now(),
			OrderNo:   order.OrderNumber,
		}).Error
	})

	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	resp.SUCCESS(c)
}

// chatgpt_power_pool_recharge 模型确认充值到账
// ConfirmPayment 管理员确认充值到账（兼容入口）
func (h *TradeHandler) ConfirmPayment(c *gin.Context) {
	h.ConfirmPowerTradePayment(c)
}

// ConfirmBuybackPayment 管理员确认回购订单已打款
func (h *TradeHandler) ConfirmBuybackPayment(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		fmt.Printf("Permission check failed: %v\n", err)
		resp.NotPermission(c)
		return
	}

	var data struct {
		OrderId uint `json:"orderId"`
	}

	if err := c.ShouldBindJSON(&data); err != nil || data.OrderId == 0 {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	var order model.Buyback
	if err := h.DB.First(&order, data.OrderId).Error; err != nil {
		resp.ERROR(c, "订单不存在或已被删除")
		return
	}

	if order.Status != 1 {
		resp.ERROR(c, "当前状态不可确认打款，只有状态为'待管理员打款'的订单才能确认")
		return
	}

	// 获取回购池信息
	var pool model.PowerPool
	if err := h.DB.Where("id = ?", 1).First(&pool).Error; err != nil {
		resp.ERROR(c, "获取回购池信息失败："+err.Error())
		return
	}

	// 检查回购池是否有足够的USDT
	if pool.BuybackUsdt < order.TotalPrice {
		resp.ERROR(c, fmt.Sprintf("回购池USDT不足，当前可用：%.2f，需要：%.2f", pool.BuybackUsdt, order.TotalPrice))
		return
	}

	// 使用事务处理订单状态更新和回购池扣除
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 更新订单状态为已完成
		if err := tx.Model(&model.Buyback{}).
			Where("id = ? AND status = ?", data.OrderId, 1).
			Updates(map[string]interface{}{
				"status":      2,
				"finish_time": time.Now().Unix(),
			}).Error; err != nil {
			return fmt.Errorf("更新订单状态失败：%v", err)
		}

		// 从回购池中扣除对应的USDT，并将用户卖出的算力回流到算力池
		if err := tx.Model(&model.PowerPool{}).
			Where("id = ?", 1).
			Updates(map[string]interface{}{
				"buyback_usdt": gorm.Expr("buyback_usdt - ?", order.TotalPrice),
				"total_power":  gorm.Expr("total_power + ?", order.Number),
			}).Error; err != nil {
			return fmt.Errorf("更新算力池失败：%v", err)
		}

		return nil
	})

	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	resp.SUCCESS(c)
}

// ListBuyback 获取回购订单列表
func (h *TradeHandler) ListBuyback(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		fmt.Printf("Permission check failed: %v\n", err)
		resp.NotPermission(c)
		return
	}

	var data struct {
		OrderNo  string   `json:"order_no"`
		Status   *int     `json:"status"`
		PayTime  []string `json:"pay_time"`
		Page     int      `json:"page"`
		PageSize int      `json:"page_size"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	if data.Page <= 0 {
		data.Page = 1
	}
	if data.PageSize <= 0 {
		data.PageSize = 15
	}

	// 构建查询会话
	session := h.DB.Table("chatgpt_buybacks")

	// 添加查询条件
	if data.OrderNo != "" {
		session = session.Where("chatgpt_buybacks.order_number = ?", data.OrderNo)
	}

	if data.Status != nil {
		session = session.Where("chatgpt_buybacks.status = ?", *data.Status)
	}

	if len(data.PayTime) == 2 {
		start := utils.Str2stamp(data.PayTime[0] + " 00:00:00")
		end := utils.Str2stamp(data.PayTime[1] + " 23:59:59")
		session = session.Where("chatgpt_buybacks.created_at BETWEEN ? AND ?", start, end)
	}

	// 统计总数
	var total int64
	countSession := session
	countResult := countSession.Count(&total)
	if countResult.Error != nil {
		resp.ERROR(c, "统计数据失败")
		return
	}

	// 查询数据 - 添加关联查询用户表
	var items []struct {
		model.Buyback
		Username string `gorm:"column:username"`
	}
	var list = make([]vo.PowerTradeOrder, 0)
	offset := (data.Page - 1) * data.PageSize

	// 使用 Joins 关联查询 users 表获取用户名
	res := session.Table("chatgpt_buybacks").
		Select("chatgpt_buybacks.*, u.username").
		Joins("LEFT JOIN chatgpt_users u ON chatgpt_buybacks.user_id = u.id").
		Order("chatgpt_buybacks.id DESC").
		Offset(offset).
		Limit(data.PageSize).
		Find(&items)

	if res.Error != nil {
		resp.ERROR(c, "查询数据失败")
		return
	}

	// 处理查询结果
	for _, item := range items {
		var order vo.PowerTradeOrder
		order.Id = item.Buyback.Id
		order.OrderNumber = item.Buyback.OrderNumber
		order.Status = item.Buyback.Status
		order.ProofUrl = item.Buyback.Image
		order.ListedPower = item.Buyback.Number
		order.DestroyedPower = item.Buyback.TotalPriceChina
		order.TotalPower = item.Buyback.TotalPrice
		order.Price = item.Buyback.Price
		// 优先使用 TotalPrice，如果为0则使用 Number * Price
		if item.Buyback.TotalPrice > 0 {
			order.ValueUsd = item.Buyback.TotalPrice
		} else {
			order.ValueUsd = item.Buyback.Number * item.Buyback.Price
		}
		order.CreatedAt = item.Buyback.CreatedAt.Unix()
		order.MatchTime = item.Buyback.MatchTime
		order.PayTime = item.Buyback.PayTime
		order.FinishTime = item.Buyback.FinishTime
		order.EthAddress = item.Buyback.EthAddress

		// 回购订单只有卖家（用户自己）
		order.SellerName = item.Username
		order.BuyerName = "系统回购"
		order.CounterpartyName = "系统回购"
		order.Role = "seller"

		list = append(list, order)
	}

	result := vo.NewPage(total, data.Page, data.PageSize, list)
	resp.SUCCESS(c, result)
}

// ListPowerPoolRecharge 获取算力池充值订单列表（chatgpt_power_pool_recharge）
func (h *TradeHandler) ListPowerPoolRecharge(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		fmt.Printf("Permission check failed: %v\n", err)
		resp.NotPermission(c)
		return
	}

	var data struct {
		OrderNo  string   `json:"order_no"`
		Status   *int     `json:"status"`
		PayTime  []string `json:"pay_time"`
		Page     int      `json:"page"`
		PageSize int      `json:"page_size"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	if data.Page <= 0 {
		data.Page = 1
	}
	if data.PageSize <= 0 {
		data.PageSize = 15
	}

	// 构建查询会话
	session := h.DB.Table("chatgpt_power_pool_recharge")

	// 添加查询条件
	if data.OrderNo != "" {
		session = session.Where("chatgpt_power_pool_recharge.order_number = ?", data.OrderNo)
	}

	if data.Status != nil {
		session = session.Where("chatgpt_power_pool_recharge.status = ?", *data.Status)
	}

	if len(data.PayTime) == 2 {
		start := utils.Str2stamp(data.PayTime[0] + " 00:00:00")
		end := utils.Str2stamp(data.PayTime[1] + " 23:59:59")
		session = session.Where("chatgpt_power_pool_recharge.created_at BETWEEN ? AND ?", start, end)
	}

	// 统计总数
	var total int64
	countSession := session
	countResult := countSession.Count(&total)
	if countResult.Error != nil {
		resp.ERROR(c, "统计数据失败")
		return
	}

	// 查询数据 - 关联用户表
	var items []struct {
		model.PowerPoolRecharge
		Username string `gorm:"column:username"`
	}
	var list = make([]vo.PowerTradeOrder, 0)
	offset := (data.Page - 1) * data.PageSize

	res := session.Table("chatgpt_power_pool_recharge").
		Select("chatgpt_power_pool_recharge.*, u.username").
		Joins("LEFT JOIN chatgpt_users u ON chatgpt_power_pool_recharge.user_id = u.id").
		Order("chatgpt_power_pool_recharge.id DESC").
		Offset(offset).
		Limit(data.PageSize).
		Find(&items)

	if res.Error != nil {
		resp.ERROR(c, "查询数据失败")
		return
	}

	// 组装返回数据，复用 PowerTradeOrder 结构
	for _, item := range items {
		var order vo.PowerTradeOrder
		order.Id = item.PowerPoolRecharge.Id
		order.OrderNumber = item.PowerPoolRecharge.OrderNumber
		order.Status = item.PowerPoolRecharge.Status

		// 充值没有图片凭证，这里留空
		order.ProofUrl = ""

		// 充值算力和金额
		order.ListedPower = item.PowerPoolRecharge.PowerAmount
		order.DestroyedPower = 0
		order.TotalPower = item.PowerPoolRecharge.PowerAmount

		// 单价 = USDT / 算力
		if item.PowerPoolRecharge.PowerAmount > 0 {
			order.Price = item.PowerPoolRecharge.UsdtAmount / item.PowerPoolRecharge.PowerAmount
		} else {
			order.Price = 0
		}
		order.ValueUsd = item.PowerPoolRecharge.UsdtAmount

		order.CreatedAt = item.PowerPoolRecharge.CreatedAt.Unix()
		order.MatchTime = 0
		order.PayTime = item.PowerPoolRecharge.PayTime
		order.FinishTime = item.PowerPoolRecharge.FinishTime

		// 收款地址：平台BSC地址
		order.EthAddress = item.PowerPoolRecharge.BscAddress

		// 用户信息：买家为用户，卖家为系统
		order.SellerName = "系统"
		order.BuyerName = item.Username
		order.CounterpartyName = item.Username
		order.Role = "buyer"

		list = append(list, order)
	}

	result := vo.NewPage(total, data.Page, data.PageSize, list)
	resp.SUCCESS(c, result)
}
