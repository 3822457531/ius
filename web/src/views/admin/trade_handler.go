package handler

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils"
	"chatplus/utils/resp"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TradeHandler struct {
	BaseHandler
}

func NewTradeHandler(app *core.AppServer, db *gorm.DB) *TradeHandler {
	return &TradeHandler{BaseHandler: BaseHandler{App: app, DB: db}}
}

func (h *TradeHandler) List(c *gin.Context) {
	var data struct {
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	userId := h.GetLoginUserId(c)
	session := h.DB.Session(&gorm.Session{}).Where("user_id = ? AND status = ?", userId, types.OrderPaidSuccess)
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

// 获取用户的套餐列表
func (h *TradeHandler) ListByUid(c *gin.Context) {
	var data struct {
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	userId := h.GetLoginUserId(c)
	session := h.DB.Session(&gorm.Session{}).Where("user_id = ? AND status = ?", userId, types.OrderPaidSuccess)
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

// 前端列表
// 匹配中
func (h *TradeHandler) IngOrderList(c *gin.Context) {

	userId := h.GetLoginUserId(c)
	var items []model.Trade
	var list = make([]vo.Trade, 0)
	logger.Info("userId--->", userId)
	//res := h.DB.Where("user_id", userId, "status", 8).Find(&items)
	//res := h.DB.Where("user_id = ?  or   target_user_id =?  And  status=2  ", userId, userId).Find(&items)
	res := h.DB.Where("(user_id = ? OR target_user_id = ?) AND  ( status = 2 or  status = 3 )   ", userId, userId).Find(&items)

	if res.Error == nil {
		for _, item := range items {
			var order vo.Trade
			err := utils.CopyObject(item, &order)
			if err == nil {
				order.Id = item.Id
				order.Price = item.Price
				order.Number = item.Number
				order.OrderNumber = item.OrderNumber
				order.Status = item.Status
				order.EthAddress = item.EthAddress

				if item.TargetUserId == int(userId) && item.Types == 1 {
					order.IsComment = false //区分是买单还是卖单 true 是买 false是卖
				} else if item.TargetUserId == int(userId) && item.Types == 2 {
					order.IsComment = true //区分是买单还是卖单 true 是买 false是卖
				} else if item.TargetUserId != int(userId) && item.Types == 2 {
					order.IsComment = false //区分是买单还是卖单 true 是买 false是卖
				} else if item.TargetUserId != int(userId) && item.Types == 1 {
					order.IsComment = true //区分是买单还是卖单 true 是买 false是卖
				}

				order.CreatedAt = item.CreatedAt.Unix()
				order.UpdatedAt = item.UpdatedAt.Unix()
				list = append(list, order)
			} else {
				logger.Error(err)
			}
		}
	}
	resp.SUCCESS(c, list)
}

// 我的买单
func (h *TradeHandler) BuyOrderList(c *gin.Context) {

	userId := h.GetLoginUserId(c)
	var items []model.Trade
	var list = make([]vo.Trade, 0)
	logger.Info("userId--->", userId)
	//res := h.DB.Where("user_id", userId, "status", 8).Find(&items)
	res := h.DB.Where("user_id = ?  And  status=1  And  types=1  ", userId).Find(&items)
	if res.Error == nil {
		for _, item := range items {
			var order vo.Trade
			err := utils.CopyObject(item, &order)
			if err == nil {
				order.Id = item.Id
				order.Price = item.Price
				order.Number = item.Number
				order.OrderNumber = item.OrderNumber
				order.Status = item.Status

				order.CreatedAt = item.CreatedAt.Unix()
				order.UpdatedAt = item.UpdatedAt.Unix()
				list = append(list, order)
			} else {
				logger.Error(err)
			}
		}
	}
	resp.SUCCESS(c, list)
}

// 我的卖单
func (h *TradeHandler) SellOrderList(c *gin.Context) {

	userId := h.GetLoginUserId(c)
	var items []model.Trade
	var list = make([]vo.Trade, 0)
	logger.Info("userId--->", userId)
	//res := h.DB.Where("user_id", userId, "status", 8).Find(&items)
	res := h.DB.Where("user_id = ?  And  status=1  And  types=2  ", userId).Find(&items)
	if res.Error == nil {
		for _, item := range items {
			var order vo.Trade
			err := utils.CopyObject(item, &order)
			if err == nil {
				order.Id = item.Id
				order.Price = item.Price
				order.Number = item.Number
				order.OrderNumber = item.OrderNumber
				order.Status = item.Status

				order.CreatedAt = item.CreatedAt.Unix()
				order.UpdatedAt = item.UpdatedAt.Unix()
				list = append(list, order)
			} else {
				logger.Error(err)
			}
		}
	}
	resp.SUCCESS(c, list)
}

// 我的已完成单
func (h *TradeHandler) CompletedOrderList(c *gin.Context) {

	userId := h.GetLoginUserId(c)
	var items []model.Trade
	var list = make([]vo.Trade, 0)
	logger.Info("userId--->", userId)
	//res := h.DB.Where("user_id", userId, "status", 8).Find(&items)
	res := h.DB.Where("(user_id = ? OR target_user_id = ?) And  status=4    ", userId, userId).Find(&items)
	if res.Error == nil {
		for _, item := range items {
			var order vo.Trade
			err := utils.CopyObject(item, &order)
			if err == nil {
				order.Id = item.Id
				order.Price = item.Price
				order.Number = item.Number
				order.OrderNumber = item.OrderNumber
				order.Status = item.Status

				if item.TargetUserId == int(userId) && item.Types == 1 {
					order.IsComment = false //区分是买单还是卖单 true 是买 false是卖
				} else if item.TargetUserId == int(userId) && item.Types == 2 {
					order.IsComment = true //区分是买单还是卖单 true 是买 false是卖
				} else if item.TargetUserId != int(userId) && item.Types == 2 {
					order.IsComment = false //区分是买单还是卖单 true 是买 false是卖
				} else if item.TargetUserId != int(userId) && item.Types == 1 {
					order.IsComment = true //区分是买单还是卖单 true 是买 false是卖
				}

				order.CreatedAt = item.CreatedAt.Unix()
				order.UpdatedAt = item.UpdatedAt.Unix()
				list = append(list, order)
			} else {
				logger.Error(err)
			}
		}
	}
	resp.SUCCESS(c, list)
}

// List 获取进行中的order
func (h *TradeHandler) GetEndOrderList(c *gin.Context) {

	userId := h.GetLoginUserId(c)
	var items []model.Order
	var list = make([]vo.Order, 0)
	//res := h.DB.Where("user_id", userId).Find(&items)
	res := h.DB.Where("user_id = ?  And  status=9 ", userId).Find(&items)
	if res.Error == nil {
		for _, item := range items {
			var order vo.Order
			err := utils.CopyObject(item, &order)
			if err == nil {
				order.Id = item.Id
				order.Subject = item.Subject
				order.Amount = item.Amount
				order.Power = item.Power
				order.JsPower = item.JsPower

				order.CreatedAt = item.CreatedAt.Unix()
				order.UpdatedAt = item.UpdatedAt.Unix()
				list = append(list, order)
			} else {
				logger.Error(err)
			}
		}
	}
	resp.SUCCESS(c, list)
}

//交易市场

// 下单
func (h *TradeHandler) SubmitOrder(c *gin.Context) {
	var data struct {
		Price float64 `json:"price"`
		Num   float64 `json:"num"`
		Types int     `json:"types"`
	}

	logger.Info(data)

	if err := c.ShouldBindJSON(&data); err != nil {
		fmt.Printf("Bind JSON error: %v\n", err)
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	fmt.Printf("Received data: %+v\n", data) // 调试信息

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	//获取用户可用余额
	var availableBalance float64
	availableBalance = user.Coin - user.DjCoin

	////如果用户没有设置收款地址提示设置
	//if data.Types == 2 {
	//	if user.EthAddress == "" {
	//		// 提示用户设置收款地址
	//		resp.ERROR(c, "请设置收款地址")
	//		return
	//	}
	//}

	//卖出不能超过自己的最大可用余额
	if data.Types == 2 {
		if data.Num > availableBalance {
			resp.ERROR(c, "可用余额不足")
			return
		}

		if user.EthAddress == "" {
			// 提示用户设置收款地址
			resp.ERROR(c, "请设置收款地址")
			return
		}
	}
	//卖单
	if data.Types == 2 {

		resCheck := h.DB.Create(&model.Trade{
			UserId:      user.Id,
			OrderNumber: utils.GenerateUniqueID(),
			Number:      data.Num,
			Price:       data.Price,
			Types:       data.Types,
			Status:      1,
			EthAddress:  user.EthAddress,
		})

		fmt.Printf("Database response: %+v\n", resCheck) // 调试信息
		if resCheck.Error != nil {
			resp.ERROR(c, resCheck.Error.Error())
			return
		}
		//买单
	} else if data.Types == 1 {

		resCheck := h.DB.Create(&model.Trade{
			UserId:      user.Id,
			OrderNumber: utils.GenerateUniqueID(),
			Number:      data.Num,
			Price:       data.Price,
			Types:       data.Types,
			Status:      1,
		})
		fmt.Printf("Database response: %+v\n", resCheck) // 调试信息
		if resCheck.Error != nil {
			resp.ERROR(c, resCheck.Error.Error())
			return
		}
	}

	//fmt.Printf("Database response: %+v\n", resCheck) // 调试信息

	//if resCheck.Error != nil {
	//	resp.ERROR(c, resCheck.Error.Error())
	//	return
	//}

	//如果是卖单的话冻结用户当前的币种金额
	if data.Types == 2 {
		//冻结 +
		res := h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumn("dj_coin", gorm.Expr("dj_coin + ?", data.Num))
		if res.Error == nil {
			logger.Info("冻结成功 ")

			// 记录算力消费日志
			//var u model.User
			//h.DB.Where("id", user.Id).First(&u)
			//
			//h.DB.Create(&model.PowerLog{
			//	UserId:    user.Id,
			//	Username:  user.Username,
			//	Type:      types.PowerFk,
			//	Amount:    dataPower,
			//	Mark:      types.PowerSub,
			//	Balance:   u.Power,
			//	Model:     fmt.Sprintf("转账:收款人(%s) ", u2.Username), //session.Model.Value,
			//	Remark:    fmt.Sprintf("用户转出算力：%s ", data.Power),
			//	CreatedAt: time.Now(),
			//})

		} else {
			logger.Error("更新数据库失败: ", res.Error)
			resp.ERROR(c, "更新数据库失败")
			return

		}
	}

	resp.SUCCESS(c)
}

// 匹配
func (h *TradeHandler) MatchOrder(c *gin.Context) {
	var data struct {
		Price float64 `json:"price"`
		Num   float64 `json:"num"`
		Types int     `json:"types"`
		Id    int     `json:"id"` //订单id
	}

	logger.Info(data)

	if err := c.ShouldBindJSON(&data); err != nil {
		fmt.Printf("Bind JSON error: %v\n", err)
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	fmt.Printf("Received data: %+v\n", data) // 调试信息

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}
	//校验用户余额
	//获取用户可用余额
	var availableBalance float64
	availableBalance = user.Coin - user.DjCoin

	//卖出不能超过自己的最大可用余额
	if data.Types == 1 {
		if data.Num > availableBalance {
			resp.ERROR(c, "可用余额不足")
			return
		}

		if user.EthAddress == "" {
			// 提示用户设置收款地址
			resp.ERROR(c, "请设置收款地址")
			return
		}
	}

	// 获取订单信息
	var trade model.Trade
	res := h.DB.First(&trade, data.Id)
	if res.Error != nil {
		resp.ERROR(c, "No trade found")
		return
	}

	//自己订单不能自己买卖

	if trade.UserId == user.Id {

		resp.ERROR(c, "自己的订单不能够自己交易哦")
		return
	}

	if data.Types == 1 {
		resCheck := h.DB.Model(&trade).Where("id = ?", data.Id).Updates(model.Trade{
			TargetUserId: int(user.Id),
			Status:       2, //匹配中
			EthAddress:   user.EthAddress,
		})
		fmt.Printf("Database response: %+v\n", resCheck) // 调试信息
		if resCheck.Error != nil {
			resp.ERROR(c, resCheck.Error.Error())
			return
		}
	} else if data.Types == 2 {

		resCheck := h.DB.Model(&trade).Where("id = ?", data.Id).Updates(model.Trade{
			TargetUserId: int(user.Id),
			Status:       2, //匹配中
		})
		fmt.Printf("Database response: %+v\n", resCheck) // 调试信息
		if resCheck.Error != nil {
			resp.ERROR(c, resCheck.Error.Error())
			return
		}
	}

	//如果是卖单的话冻结用户当前的币种金额

	if data.Types == 1 {
		//冻结 +
		res := h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumn("dj_coin", gorm.Expr("dj_coin + ?", data.Num))
		if res.Error == nil {
			logger.Info("冻结成功 ")

		} else {
			logger.Error("更新数据库失败: ", res.Error)
			resp.ERROR(c, "更新数据库失败")
			return
		}
	}

	resp.SUCCESS(c)
}

// 市场加载买单列表和卖单列表数据
func (h *TradeHandler) BuyAndSellList(c *gin.Context) {
	// userId := h.GetLoginUserId(c)
	var items []model.Trade
	var list = make([]vo.Trade, 0)

	// 查询所有状态为1的交易，买单按价格升序，卖单按价格降序
	res := h.DB.Where("status = 1").Order("CASE WHEN types = 2 THEN price END ASC, CASE WHEN types = 1 THEN price END DESC").Find(&items)
	if res.Error == nil {
		for _, item := range items {
			// 直接构造 vo.Trade 对象，不使用 CopyObject
			order := vo.Trade{
				Id:        item.Id,
				Price:     item.Price,
				Number:    item.Number,
				Types:     item.Types,
				CreatedAt: item.CreatedAt.Unix(),
				UpdatedAt: item.UpdatedAt.Unix(),
			}

			// 获取用户信息
			var user model.User
			if err := h.DB.Where("id = ?", item.UserId).First(&user).Error; err == nil {
				order.UserName = user.Username
				order.Avatar = user.Avatar
			}

			list = append(list, order)
		}
	}

	resp.SUCCESS(c, list)
}

// 已付款待确认
func (h *TradeHandler) WaitComfireOrder(c *gin.Context) {
	var data struct {
		OrderId  int    `json:"orderId"`  //订单id
		ProofUrl string `json:"proofUrl"` //凭证
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		fmt.Printf("Bind JSON error: %v\n", err)
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	fmt.Printf("Received data: %+v\n", data) // 调试信息

	logger.Info(data)

	// 获取订单信息
	var trade model.Trade
	res := h.DB.First(&trade, data.OrderId)
	if res.Error != nil {
		resp.ERROR(c, "No trade found")
		return
	}

	// 验证 BSC 交易
	isPay := h.verifyBSCTransaction(trade.EthAddress, trade.Number*trade.Price)

	//如果已经付款了
	if isPay {
		//则直接将订单状态更改为已完成
		resCheck := h.DB.Model(&trade).Where("id = ?", data.OrderId).Updates(model.Trade{
			Status: 4, //已确认
			//FinishTime: time.Now(), //已付款待确认
		})

		//用户Q2币转账转移操作
		if trade.Types == 1 {
			logger.Info("111111111111")
			//解冻
			h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin -?", trade.Number))

			//更新 - 160 -8
			h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("coin", gorm.Expr("coin - ?", trade.Number))
			//更新 +
			h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("coin", gorm.Expr("coin + ?", trade.Number))

		} else if trade.Types == 2 {
			//解冻
			h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin -?", trade.Number))

			//更新 -
			h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("coin", gorm.Expr("coin - ?", trade.Number))

			//更新 +
			h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("coin", gorm.Expr("coin + ?", trade.Number))
		}

		if resCheck.Error != nil {
			resp.ERROR(c, resCheck.Error.Error())
			return
		}

		fmt.Printf("BSC verification passed, order completed automatically\n")
	} else {
		//自己订单不能自己买卖
		resCheck := h.DB.Model(&trade).Where("id = ?", data.OrderId).Updates(model.Trade{
			Status: 3,             //已付款待确认
			Image:  data.ProofUrl, //已付款待确认
		})

		fmt.Printf("Database response: %+v\n", resCheck) // 调试信息

		if resCheck.Error != nil {
			resp.ERROR(c, resCheck.Error.Error())
			return
		}

		fmt.Printf("BSC verification failed, order status updated to pending confirmation\n")
	}

	resp.SUCCESS(c)
}

// 验证 BSC 交易

// 验证 BSC 交易
// 验证 BSC 交易
func (h *TradeHandler) verifyBSCTransaction(address string, expectedAmount float64) bool {
	// 获取 BSC API Key
	apiKey := "XT44Q8NA1P4FDIEWPS2HCU5EYT2MWXCIWM" //h.getBSCAPIKey()
	if apiKey == "" {
		fmt.Printf("BSC API Key not configured\n")
		return false
	}

	// 计算30分钟前的时间戳
	thirtyMinutesAgo := time.Now().Add(-500 * time.Minute).Unix()
	fmt.Printf("=== BSC Transaction Verification ===\n")
	fmt.Printf("Address: %s\n", address)
	fmt.Printf("Expected Amount: %f\n", expectedAmount)
	fmt.Printf("Time Range: After %d (30 minutes ago)\n", thirtyMinutesAgo)

	// 构建 BSCScan API URL
	url := fmt.Sprintf("https://api.bscscan.com/api?module=account&action=txlist&address=%s&startblock=0&endblock=99999999&sort=desc&apikey=%s", address, apiKey)
	fmt.Printf("API URL: %s\n", url)

	// 发送 HTTP 请求
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Failed to fetch BSC transactions: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read response body: %v\n", err)
		return false
	}

	fmt.Printf("BSC API Response: %s\n", string(body))

	// 解析响应
	var bscResponse struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  []struct {
			Hash            string `json:"hash"`
			From            string `json:"from"`
			To              string `json:"to"`
			Value           string `json:"value"`
			TimeStamp       string `json:"timeStamp"`
			IsError         string `json:"isError"`
			ContractAddress string `json:"contractAddress"`
			Input           string `json:"input"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &bscResponse); err != nil {
		fmt.Printf("Failed to parse BSC response: %v\n", err)
		return false
	}

	if bscResponse.Status != "1" {
		fmt.Printf("BSC API error: %s\n", bscResponse.Message)
		return false
	}

	fmt.Printf("Found %d transactions\n", len(bscResponse.Result))

	// 检查交易记录
	for i, tx := range bscResponse.Result {
		fmt.Printf("\n--- Transaction %d ---\n", i+1)
		fmt.Printf("Hash: %s\n", tx.Hash)
		fmt.Printf("From: %s\n", tx.From)
		fmt.Printf("To: %s\n", tx.To)
		fmt.Printf("Value: %s\n", tx.Value)
		fmt.Printf("TimeStamp: %s\n", tx.TimeStamp)
		fmt.Printf("IsError: %s\n", tx.IsError)
		fmt.Printf("ContractAddress: %s\n", tx.ContractAddress)
		fmt.Printf("Input: %s\n", tx.Input)

		// 检查交易是否成功
		if tx.IsError != "0" {
			fmt.Printf("Transaction failed, skipping\n")
			continue
		}

		// 检查是否是转入交易
		if strings.ToLower(tx.To) != strings.ToLower(address) {
			fmt.Printf("Not a transfer to target address, skipping\n")
			continue
		}

		// 检查时间戳
		timestamp, err := strconv.ParseInt(tx.TimeStamp, 10, 64)
		if err != nil {
			fmt.Printf("Invalid timestamp, skipping\n")
			continue
		}

		fmt.Printf("Transaction timestamp: %d, cutoff: %d\n", timestamp, thirtyMinutesAgo)
		if timestamp < thirtyMinutesAgo {
			fmt.Printf("Transaction too old, skipping\n")
			continue
		}

		// 检查金额
		fmt.Printf("Checking transaction amount...\n")
		if h.checkTransactionAmount(map[string]interface{}{
			"value":           tx.Value,
			"contractAddress": tx.ContractAddress,
			"input":           tx.Input,
		}, expectedAmount) {
			fmt.Printf("✅ Found matching transaction: %s, amount matches expected: %f\n", tx.Hash, expectedAmount)
			return true
		} else {
			fmt.Printf("❌ Transaction amount does not match\n")
		}
	}

	fmt.Printf("No matching transaction found for address: %s, amount: %f\n", address, expectedAmount)
	return false
}

// 检查交易金额是否匹配
func (h *TradeHandler) checkTransactionAmount(tx interface{}, expectedAmount float64) bool {
	txMap, ok := tx.(map[string]interface{})
	if !ok {
		return false
	}

	value, _ := txMap["value"].(string)
	contractAddress, _ := txMap["contractAddress"].(string)
	input, _ := txMap["input"].(string)

	logger.Info(value)

	// 只处理USDT代币转账（BSC链上的USDT合约地址）
	// BSC USDT 合约地址: 0x55d398326f99059ff775485246999027b3197955
	usdtContractAddress := "0x55d398326f99059ff775485246999027b3197955"

	if contractAddress != "" && strings.ToLower(contractAddress) == strings.ToLower(usdtContractAddress) {
		return h.parseTokenTransfer(input, expectedAmount)
	}

	// 如果不是USDT转账，直接返回false
	return false
}

// 解析代币转账
func (h *TradeHandler) parseTokenTransfer(input string, expectedAmount float64) bool {
	fmt.Printf("=== Token Transfer Parsing ===\n")
	fmt.Printf("Input length: %d\n", len(input))
	fmt.Printf("Input: %s\n", input)
	fmt.Printf("Expected amount: %f USDT\n", expectedAmount)

	// 检查input长度，transfer方法的input应该是138个字符（包含0x）
	if len(input) < 138 {
		fmt.Printf("❌ Input too short (expected 138+ chars, got %d)\n", len(input))
		return false
	}

	// 检查是否是transfer方法调用 (0xa9059cbb)
	if !strings.HasPrefix(strings.ToLower(input), "0xa9059cbb") {
		fmt.Printf("❌ Not a transfer method call (expected 0xa9059cbb prefix)\n")
		return false
	}

	fmt.Printf("✅ Valid transfer method call detected\n")

	// 提取金额部分（最后64个字符）
	amountHex := input[len(input)-64:]
	fmt.Printf("Amount hex: %s\n", amountHex)

	// 将十六进制转换为大整数
	amount := new(big.Int)
	amount.SetString(amountHex, 16)

	// USDT有18位小数，转换为实际金额
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	amountFloat := new(big.Float).SetInt(amount)
	amountFloat.Quo(amountFloat, divisor)

	actualAmount, _ := amountFloat.Float64()

	fmt.Printf("Parsed USDT amount: %.18f\n", actualAmount)
	fmt.Printf("Expected USDT amount: %.18f\n", expectedAmount)

	// 只匹配前3位数的逻辑
	return h.matchFirst3Digits(actualAmount, expectedAmount)
}

// 匹配前3位数的方法
func (h *TradeHandler) matchFirst3Digits(actualAmount, expectedAmount float64) bool {
	fmt.Printf("=== Matching First 3 Digits ===\n")

	// 将数字转换为字符串，提取前3位有效数字
	actualStr := fmt.Sprintf("%.18f", actualAmount)
	expectedStr := fmt.Sprintf("%.18f", expectedAmount)

	fmt.Printf("Actual amount string: %s\n", actualStr)
	fmt.Printf("Expected amount string: %s\n", expectedStr)

	// 移除小数点，只保留数字
	actualDigits := strings.Replace(actualStr, ".", "", 1)
	expectedDigits := strings.Replace(expectedStr, ".", "", 1)

	fmt.Printf("After removing decimal point:\n")
	fmt.Printf("  Actual: %s\n", actualDigits)
	fmt.Printf("  Expected: %s\n", expectedDigits)

	// 移除前导零
	actualDigits = strings.TrimLeft(actualDigits, "0")
	expectedDigits = strings.TrimLeft(expectedDigits, "0")

	fmt.Printf("After removing leading zeros:\n")
	fmt.Printf("  Actual: %s\n", actualDigits)
	fmt.Printf("  Expected: %s\n", expectedDigits)

	// 如果长度不足3位，直接比较
	if len(actualDigits) < 3 || len(expectedDigits) < 3 {
		result := actualDigits == expectedDigits
		fmt.Printf("Short digits comparison: %t\n", result)
		return result
	}

	// 比较前3位数字
	actualFirst3 := actualDigits[:3]
	expectedFirst3 := expectedDigits[:3]

	fmt.Printf("First 3 digits comparison:\n")
	fmt.Printf("  Actual: %s\n", actualFirst3)
	fmt.Printf("  Expected: %s\n", expectedFirst3)

	result := actualFirst3 == expectedFirst3
	fmt.Printf("Match result: %t\n", result)

	return result
}

// 获取 BSC API Key
func (h *TradeHandler) getBSCAPIKey() string {
	// 从系统配置中获取 BSC API Key
	var config struct {
		Value string `json:"value"`
	}

	result := h.DB.Table("chatgpt_configs").
		Select("value").
		Where("marker = ?", "bsc_api_key").
		First(&config)

	if result.Error != nil {
		fmt.Printf("Failed to get BSC API key: %v\n", result.Error)
		return ""
	}

	return config.Value
}

// 已确认
func (h *TradeHandler) ComfireOrder(c *gin.Context) {
	var data struct {
		OrderId int `json:"orderId"` //订单id
	}

	logger.Info(data)

	if err := c.ShouldBindJSON(&data); err != nil {
		fmt.Printf("Bind JSON error: %v\n", err)
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	fmt.Printf("Received data: %+v\n", data) // 调试信息

	// 获取订单信息
	var trade model.Trade
	res := h.DB.First(&trade, data.OrderId)
	if res.Error != nil {
		resp.ERROR(c, "No trade found")
		return
	}

	resCheck := h.DB.Model(&trade).Where("id = ?", data.OrderId).Updates(model.Trade{
		Status: 4, //已付款待确认
		//FinishTime: time.Now(), //已付款待确认
	})

	fmt.Printf("Database 6666 response: %+v\n", resCheck) // 调试信息

	if resCheck.Error != nil {
		resp.ERROR(c, resCheck.Error.Error())
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	//用户Q2币转账转移操作
	if trade.Types == 1 {

		logger.Info("111111111111")
		//解冻
		h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin -?", trade.Number))

		//更新 - 160 -8
		h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("coin", gorm.Expr("coin - ?", trade.Number))
		//更新 +
		h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("coin", gorm.Expr("coin + ?", trade.Number))

	} else if trade.Types == 2 {

		//解冻
		h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumn("dj_coin", gorm.Expr("dj_coin -?", trade.Number))

		//更新 -
		h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("coin", gorm.Expr("coin - ?", trade.Number))

		//更新 +
		h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("coin", gorm.Expr("coin + ?", trade.Number))

	}

	resp.SUCCESS(c)
}

// 取消订单
func (h *TradeHandler) CancelOrder(c *gin.Context) {
	var data struct {
		Id int `json:"orderId"` //订单id
	}

	logger.Info(data)

	//获取系统配置的最大取消限制
	var limitCancel int64 = h.App.SysConfig.LimitCancel

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	// 获取当前用户的订单信息
	var currentTrade model.Trade
	var currentTradeCount int64 = 0

	// 获取订单信息
	//区分买单和卖单

	//当前取消先不限制
	currentTradeRes := h.DB.Model(&currentTrade).
		Where("target_user_id = ? AND types = ? ANd status=5  AND DATE(updated_at) = CURDATE()", user.Id, 2).
		Count(&currentTradeCount)

	if currentTradeRes.Error != nil {
		logger.Error("Error fetching trade count:", currentTradeRes.Error)
	} else {
		logger.Info("Current user trade count with types=2:", currentTradeCount)
	}
	fmt.Printf("当天取消的数量为 data: %+v\n", currentTradeCount) // 调试信息

	if err := c.ShouldBindJSON(&data); err != nil {
		fmt.Printf("Bind JSON error: %v\n", err)
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	fmt.Printf("Received data: %+v\n", data) // 调试信息

	if currentTradeCount == limitCancel {
		logger.Info("11111111111111111111")
		//resp.ERROR(c, "已达到当天取消最大限制")
		//return
	}

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
		h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin -?", trade.Number))
	} else if trade.Types == 1 {
		h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin -?", trade.Number))
	}

	resp.SUCCESS(c)
}

//返回价格k线

// handler/trade.go

// PriceHistory 获取最近7天的价格历史数据
func (h *TradeHandler) PriceHistory(c *gin.Context) {
	var items []model.PriceHistory
	var list = make([]vo.PriceHistory, 0)

	// 查询最近7天的价格数据,按日期升序排序
	res := h.DB.Where("date >= DATE_SUB(CURDATE(), INTERVAL 60 DAY)").
		Order("date ASC").
		Limit(100).
		Find(&items)

	logger.Info(res)

	if res.Error == nil {
		for _, item := range items {
			// 构造返回数据对象
			priceData := vo.PriceHistory{
				Date:       item.Date.Format("1/2"), // 格式化为 "月/日"
				Price:      item.Price,
				Volume:     item.Volume,
				ChangeRate: item.ChangeRate,
			}
			list = append(list, priceData)
		}
	}

	logger.Info(items)

	// 如果没有数据,生成默认数据
	if len(list) == 0 {
		list = getDefaultPriceHistory()
	}

	resp.SUCCESS(c, list)
}

// 生成默认数据的辅助函数
func getDefaultPriceHistory() []vo.PriceHistory {
	var list []vo.PriceHistory
	now := time.Now()

	basePrice := 0.001
	baseVolume := 1000.0

	for i := 6; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		price := basePrice + float64(i)*0.0005
		volume := baseVolume + float64(i)*500

		// 计算变化率
		var changeRate float64
		if i > 0 {
			prevPrice := basePrice + float64(i-1)*0.0005
			changeRate = (price - prevPrice) / prevPrice * 100
		}

		list = append(list, vo.PriceHistory{
			Date:       date.Format("1/2"),
			Price:      price,
			Volume:     volume,
			ChangeRate: changeRate,
		})
	}

	return list
}

// 获取产量
func (h *TradeHandler) SumCoin(c *gin.Context) {
	// 从数据库查询总和
	var sumCoin float64
	err := h.DB.Table("chatgpt_users").Select("COALESCE(SUM(coin), 0)").Scan(&sumCoin).Error
	if err != nil {
		resp.ERROR(c, "获取产量失败: "+err.Error())
		return
	}

	// 返回成功响应
	resp.SUCCESS(c, gin.H{
		"total": sumCoin,
	})
}
