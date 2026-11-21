package handler

import (
	"bytes"
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/service"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils"
	"chatplus/utils/resp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
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
	BscService *service.BscUsdtService
}

const (
	powerPoolPriceUSD  = 0.01
	powerPoolBurnRatio = 0.5
	powerPoolMaxList   = 200
)

func NewTradeHandler(app *core.AppServer, db *gorm.DB, bscService *service.BscUsdtService) *TradeHandler {
	return &TradeHandler{
		BaseHandler: BaseHandler{App: app, DB: db},
		BscService:  bscService,
	}
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

// 系統销毁量
func (h *TradeHandler) DestroyedCoin(c *gin.Context) {

	//获取系统配置的最大取消限制
	var limitCancel int64 = h.App.SysConfig.LimitCancel

	logger.Info("系统的销毁量为", limitCancel)

	resp.SUCCESS(c, limitCancel)

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

	if ok, msg := CanOperateSohaMarket(h.App.SysConfig, &user); !ok {
		resp.ERROR(c, msg)
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

		if data.Num > user.Coin {
			resp.ERROR(c, "可用冻结余额不足")
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

// SubmitPowerPoolOrder 提交算力撮合池挂单
func (h *TradeHandler) SubmitPowerPoolOrder(c *gin.Context) {
	var data struct {
		Power float64 `json:"power"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	if data.Power <= 0 {
		resp.ERROR(c, "请输入有效的算力数量")
		return
	}

	//

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	//检查是否允许算力交易
	if !user.AllowTrade {
		resp.ERROR(c, "您没有权限进行算力交易")
		return
	}

	if user.EthAddress == "" && user.AlipayAccount == "" {
		// 提示用户设置收款地址
		resp.ERROR(c, "请至少设置一个收款地址")
		return
	}

	paymentType, paymentAddr := resolveUserPaymentInfo(user)
	if paymentAddr == "" {
		if paymentType == "alipay" {
			resp.ERROR(c, "请先在收款方式管理中设置支付宝账号")
		} else {
			resp.ERROR(c, "请先在收款方式管理中设置BSC收款地址")
		}
		return
	}

	if user.Power < data.Power {
		resp.ERROR(c, "可用算力不足")
		return
	}

	//一个用户只能有3个待匹配的算力撮合池订单
	count := int64(0)
	res := h.DB.Model(&model.PowerTrade{}).Where("user_id = ? AND status = 1", user.Id).Count(&count)
	if res.Error == nil && count >= 3 {
		resp.ERROR(c, "一个用户只能有3个待匹配的算力撮合池订单")
		return
	}

	destroyedPower := data.Power * powerPoolBurnRatio
	listedPower := data.Power - destroyedPower

	if listedPower <= 0 {
		resp.ERROR(c, "挂卖数量无效")
		return
	}

	orderNumber := utils.GenerateUniqueID()
	var createdTrade model.PowerTrade
	var updatedUser model.User

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.User{}).
			Where("id = ? AND power >= ?", user.Id, data.Power).
			Updates(map[string]interface{}{
				"power": gorm.Expr("power - ?", data.Power),
				//"coin":    gorm.Expr("coin + ?", listedPower),
				//"dj_coin": gorm.Expr("dj_coin + ?", listedPower),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("可用算力不足")
		}

		if err := tx.First(&updatedUser, user.Id).Error; err != nil {
			return err
		}

		if err := tx.Create(&model.PowerLog{
			UserId:    user.Id,
			Username:  user.Username,
			Type:      types.PowerDh,
			Amount:    data.Power,
			Balance:   updatedUser.Power,
			Mark:      types.PowerSub,
			Model:     "算力撮合池",
			Remark:    fmt.Sprintf("挂卖算力%.2f，预销毁%.2f", data.Power, destroyedPower),
			CreatedAt: time.Now(),
			OrderNo:   orderNumber,
		}).Error; err != nil {
			return err
		}

		//if err := tx.Create(&model.Q2Log{
		//	UserId:    user.Id,
		//	Username:  user.Username,
		//	Type:      types.CoinDh,
		//	Amount:    listedPower,
		//	Balance:   updatedUser.Coin,
		//	Mark:      types.PowerAdd,
		//	Model:     "算力撮合池",
		//	Remark:    fmt.Sprintf("转换算力%.2f为Q2 %.2f", data.Power, listedPower),
		//	CreatedAt: time.Now(),
		//	OrderNo:   orderNumber,
		//}).Error; err != nil {
		//	return err
		//}

		createdTrade = model.PowerTrade{
			UserId:          user.Id,
			OrderNumber:     orderNumber,
			Number:          listedPower,
			Price:           powerPoolPriceUSD,
			Types:           2,
			Status:          1,
			EthAddress:      paymentAddr,
			TotalPrice:      data.Power,
			TotalPriceChina: destroyedPower,
		}

		if err := tx.Create(&createdTrade).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	resp.SUCCESS(c, gin.H{
		"orderId":        createdTrade.Id,
		"orderNumber":    createdTrade.OrderNumber,
		"listedPower":    createdTrade.Number,
		"destroyedPower": destroyedPower,
		"price":          createdTrade.Price,
	})
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

	if ok, msg := CanOperateSohaMarket(h.App.SysConfig, &user); !ok {
		resp.ERROR(c, msg)
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

// MatchPowerPoolOrder 匹配算力撮合池订单
func (h *TradeHandler) MatchPowerPoolOrder(c *gin.Context) {
	var data struct {
		Id uint `json:"id"`
	}

	if err := c.ShouldBindJSON(&data); err != nil || data.Id == 0 {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	var order model.PowerTrade
	if err := h.DB.First(&order, data.Id).Error; err != nil {
		resp.ERROR(c, "订单不存在或已被删除")
		return
	}

	if order.Status != 1 {
		resp.ERROR(c, "当前订单无法撮合")
		return
	}

	if order.UserId == user.Id {
		resp.ERROR(c, "不能撮合自己的订单")
		return
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.PowerTrade{}).
			Where("id = ? AND status = ?", data.Id, 1).
			Updates(map[string]interface{}{
				"target_user_id": user.Id,
				"status":         2,
				"match_time":     time.Now().Unix(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("订单已被抢先撮合")
		}
		return nil
	})
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	resp.SUCCESS(c)
}

// MarkPowerTradePaid 买家上传付款凭证，标记已付款
func (h *TradeHandler) MarkPowerTradePaid(c *gin.Context) {
	var data struct {
		OrderId  uint   `json:"orderId"`
		ProofUrl string `json:"proofUrl"`
	}

	if err := c.ShouldBindJSON(&data); err != nil || data.OrderId == 0 {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	proofUrl := strings.TrimSpace(data.ProofUrl)
	if proofUrl == "" {
		resp.ERROR(c, "请上传付款凭证")
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	var order model.PowerTrade
	if err := h.DB.First(&order, data.OrderId).Error; err != nil {
		resp.ERROR(c, "订单不存在或已被删除")
		return
	}

	if order.TargetUserId <= 0 || uint(order.TargetUserId) != user.Id {
		resp.ERROR(c, "无权操作该订单")
		return
	}

	if order.Status != 2 {
		resp.ERROR(c, "当前状态不可标记为已付款")
		return
	}

	res := h.DB.Model(&model.PowerTrade{}).
		Where("id = ? AND status = ? AND target_user_id = ?", data.OrderId, 2, user.Id).
		Updates(map[string]interface{}{
			"status":   3,
			"image":    proofUrl,
			"pay_time": time.Now().Unix(),
		})
	if res.Error != nil {
		resp.ERROR(c, res.Error.Error())
		return
	}
	if res.RowsAffected == 0 {
		resp.ERROR(c, "订单状态已变更，请刷新重试")
		return
	}

	resp.SUCCESS(c)
}

// ConfirmPowerTradeReceipt 卖家确认收款并转移算力
func (h *TradeHandler) ConfirmPowerTradeReceipt(c *gin.Context) {
	var data struct {
		OrderId uint `json:"orderId"`
	}

	if err := c.ShouldBindJSON(&data); err != nil || data.OrderId == 0 {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	var order model.PowerTrade
	if err := h.DB.First(&order, data.OrderId).Error; err != nil {
		resp.ERROR(c, "订单不存在或已被删除")
		return
	}

	if order.UserId != user.Id {
		resp.ERROR(c, "无权操作该订单")
		return
	}

	if order.Status != 3 {
		resp.ERROR(c, "当前状态不可确认收款")
		return
	}

	if order.TargetUserId <= 0 {
		resp.ERROR(c, "订单买家信息异常")
		return
	}

	buyerId := uint(order.TargetUserId)
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.PowerTrade{}).
			Where("id = ? AND status = ? AND user_id = ?", data.OrderId, 3, user.Id).
			Updates(map[string]interface{}{
				"status":      4,
				"finish_time": time.Now().Unix(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("订单状态已变更，无法确认")
		}

		// 1. 给买家增加算力（订单撮合得到的算力）
		if err := tx.Model(&model.User{}).
			Where("id = ?", buyerId).
			Update("power", gorm.Expr("power + ?", order.Number)).Error; err != nil {
			return err
		}
		//
		//resSeller := tx.Model(&model.User{}).
		//	Where("id = ? AND dj_coin >= ?", order.UserId, order.Number).
		//	Update("dj_coin", gorm.Expr("dj_coin - ?", order.Number))
		//if resSeller.Error != nil {
		//	return resSeller.Error
		//}
		//if resSeller.RowsAffected == 0 {
		//	return errors.New("卖家冻结Q2不足，请联系管理员")
		//}

		// 2. 将成交算力的 10% 注入算力池（认购池）
		// 这里使用订单挂单的初始算力总量 TotalPrice 的 10%

		//直接注释掉，不注入算力池
		// poolIncrement := order.TotalPrice * 0.1
		// if poolIncrement > 0 {
		// 	if err := tx.Model(&model.PowerPool{}).
		// 		Where("id = ?", 1).
		// 		Update("total_power", gorm.Expr("total_power + ?", poolIncrement)).Error; err != nil {
		// 		return err
		// 	}
		// }

		var buyer model.User
		if err := tx.First(&buyer, buyerId).Error; err != nil {
			return err
		}

		return tx.Create(&model.PowerLog{
			UserId:    buyer.Id,
			Username:  buyer.Username,
			Type:      types.PowerSk,
			Amount:    order.Number,
			Balance:   buyer.Power,
			Mark:      types.PowerAdd,
			Model:     "算力撮合池",
			Remark:    fmt.Sprintf("订单%s成交，获得算力%.2f", order.OrderNumber, order.Number),
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

// PowerPoolOrderList 获取算力撮合池挂单
func (h *TradeHandler) PowerPoolOrderList(c *gin.Context) {
	var items []model.PowerTrade
	res := h.DB.Where("status = ? AND total_price > 0", 1).
		Order("id DESC").
		Limit(powerPoolMaxList).
		Find(&items)
	if res.Error != nil {
		resp.ERROR(c, res.Error.Error())
		return
	}

	resp.SUCCESS(c, h.buildPowerPoolOrderVo(items))
}

// MyPowerPoolOrderList 获取我的算力撮合池挂单
func (h *TradeHandler) MyPowerPoolOrderList(c *gin.Context) {
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	var items []model.PowerTrade
	res := h.DB.Where("user_id = ? AND total_price > 0", user.Id).
		Order("id DESC").
		Find(&items)
	if res.Error != nil {
		resp.ERROR(c, res.Error.Error())
		return
	}

	resp.SUCCESS(c, h.buildPowerPoolOrderVo(items))
}

// MyPowerTradeOrders 获取用户参与的算力撮合订单
func (h *TradeHandler) MyPowerTradeOrders(c *gin.Context) {
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	var items []model.PowerTrade
	res := h.DB.Where("(user_id = ? OR target_user_id = ?) AND total_price > 0", user.Id, user.Id).
		Order("id DESC").
		Find(&items)
	if res.Error != nil {
		resp.ERROR(c, res.Error.Error())
		return
	}

	resp.SUCCESS(c, h.buildPowerTradeOrderVo(items, user.Id))
}

// 已付款待确认
func (h *TradeHandler) WaitComfireOrder(c *gin.Context) {
	var data struct {
		OrderId  int    `json:"orderId"`  //订单id
		ProofUrl string `json:"proofUrl"` //凭证
	}

	logger.Info(data)

	if err := c.ShouldBindJSON(&data); err != nil {
		fmt.Printf("Bind JSON error: %v\n", err)
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	fmt.Printf("Received data: %+v\n", data) // 调试信息
	//自己订单不能自己买卖

	// 获取订单信息
	var trade model.Trade
	res := h.DB.First(&trade, data.OrderId)
	if res.Error != nil {
		resp.ERROR(c, "No trade found")
		return
	}

	resCheck := h.DB.Model(&trade).Where("id = ?", data.OrderId).Updates(model.Trade{
		Status: 3,             //已付款待确认
		Image:  data.ProofUrl, //已付款待确认
	})

	fmt.Printf("Database response: %+v\n", resCheck) // 调试信息

	if resCheck.Error != nil {
		resp.ERROR(c, resCheck.Error.Error())
		return
	}

	resp.SUCCESS(c)
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

	//用户SOHA币转账转移操作
	if trade.Types == 1 {

		logger.Info("111111111111")
		//解冻
		//h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin -?", trade.Number))
		//
		////更新 - 160 -8
		//h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("coin", gorm.Expr("coin - ?", trade.Number))
		//
		////更新 +
		//h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("coin", gorm.Expr("coin + ?", trade.Number))
		//
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

// 取消订单
func (h *TradeHandler) CancelOrder(c *gin.Context) {
	var data struct {
		Id int `json:"orderId"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		fmt.Printf("Bind JSON error: %v\n", err)
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	logger.Info("收到取消订单请求，解析后的数据：", data)
	fmt.Printf("Received data: %+v\n", data) // 调试信息

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
	}

	//else if trade.Types == 1 {
	//	h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin -?", trade.Number))
	//}

	resp.SUCCESS(c)
}

// 订单申述
func (h *TradeHandler) AppealOrder(c *gin.Context) {

	var data struct {
		Id int `json:"orderId"` //订单id
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		fmt.Printf("Bind JSON error: %v\n", err)
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	logger.Info("收到取消订单请求，解析后的数据：", data)
	fmt.Printf("Received data: %+v\n", data) // 调试信息

	// 获取订单信息
	var trade model.Trade

	resCheck := h.DB.Model(&trade).Where("id = ?", data.Id).Updates(model.Trade{
		Status: 6, //订单申述中
	})

	if resCheck.Error != nil {
		resp.ERROR(c, "申述失败")
		return
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

	//	logger.Info(items)

	// 如果没有数据,生成默认数据（从基准价复利模拟，便于 K 线展示）
	if len(list) == 0 {
		list = h.getDefaultPriceHistory()
	}

	list = EnrichSohaPriceHistoryChangeRates(list)
	resp.SUCCESS(c, list)
}

// getDefaultPriceHistory 无历史数据时，从基准价按 0.25%/日复利模拟近 7 日 K 线
func (h *TradeHandler) getDefaultPriceHistory() []vo.PriceHistory {
	var list []vo.PriceHistory
	now := time.Now()
	basePrice := SohaGuideBasePrice(h.App.SysConfig)
	baseVolume := 1000.0

	for i := 6; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		daysFromStart := 6 - i
		price := basePrice
		for d := 0; d < daysFromStart; d++ {
			price = ApplySohaDailyChange(price, 0.25)
		}

		list = append(list, vo.PriceHistory{
			Date:       date.Format("1/2"),
			Price:      price,
			Volume:     baseVolume + float64(daysFromStart)*500,
			ChangeRate: 0,
		})
	}

	return EnrichSohaPriceHistoryChangeRates(list)
}

// getLatestQ2Price 获取最新 SOHA 指导价
func (h *TradeHandler) getLatestQ2Price() float64 {
	var item model.PriceHistory
	if err := h.DB.Order("date DESC").First(&item).Error; err == nil && item.Price > 0 {
		return item.Price
	}
	defaultList := h.getDefaultPriceHistory()
	if len(defaultList) > 0 {
		return defaultList[len(defaultList)-1].Price
	}
	return SohaGuideBasePrice(h.App.SysConfig)
}

// SwapInfo 获取 SOHA→算力 动态兑换信息
func (h *TradeHandler) SwapInfo(c *gin.Context) {
	q2Price := h.getLatestQ2Price()
	swapPowerPrice := h.App.SysConfig.SwapPowerPrice
	if swapPowerPrice <= 0 {
		swapPowerPrice = 0.01
	}
	exchangeRate := q2Price / swapPowerPrice
	resp.SUCCESS(c, gin.H{
		"q2_price":          q2Price,
		"swap_power_price":  swapPowerPrice,
		"exchange_rate":     exchangeRate,
		"max_swap_amount":   500000,
		"destroyed_volume":  h.App.SysConfig.LimitCancel,
	})
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

// resolveUserPaymentInfo 按用户 default_payment_type 返回收款方式与账号
func resolveUserPaymentInfo(user model.User) (paymentType, paymentAddress string) {
	paymentType = strings.ToLower(strings.TrimSpace(user.DefaultPaymentType))
	if paymentType == "" {
		paymentType = "bsc"
	}
	switch paymentType {
	case "alipay":
		return "alipay", strings.TrimSpace(user.AlipayAccount)
	default:
		return "bsc", strings.TrimSpace(user.EthAddress)
	}
}

func (h *TradeHandler) buildPowerPoolOrderVo(items []model.PowerTrade) []vo.PowerPoolOrder {
	list := make([]vo.PowerPoolOrder, 0, len(items))
	if len(items) == 0 {
		return list
	}

	userIds := make([]uint, 0)
	userSet := make(map[uint]struct{})
	for _, item := range items {
		if _, ok := userSet[item.UserId]; !ok {
			userSet[item.UserId] = struct{}{}
			userIds = append(userIds, item.UserId)
		}
	}

	userMap := make(map[uint]model.User)
	if len(userIds) > 0 {
		var users []model.User
		h.DB.Where("id IN ?", userIds).Find(&users)
		for _, u := range users {
			userMap[u.Id] = u
		}
	}

	for _, item := range items {
		userInfo := userMap[item.UserId]
		paymentType, paymentAddr := resolveUserPaymentInfo(userInfo)
		if paymentAddr == "" {
			paymentAddr = strings.TrimSpace(item.EthAddress)
			if paymentType != "alipay" {
				paymentType = "bsc"
			}
		}
		list = append(list, vo.PowerPoolOrder{
			Id:             item.Id,
			OrderNumber:    item.OrderNumber,
			Username:       userInfo.Username,
			Avatar:         userInfo.Avatar,
			ListedPower:    item.Number,
			DestroyedPower: item.TotalPriceChina,
			TotalPower:     item.TotalPrice,
			Price:          item.Price,
			ValueUsd:       item.Number * item.Price,
			Status:         item.Status,
			CreatedAt:      item.CreatedAt.Unix(),
			EthAddress:     paymentAddr,
			PaymentType:    paymentType,
			PaymentAddress: paymentAddr,
		})
	}

	return list
}

func (h *TradeHandler) buildPowerTradeOrderVo(items []model.PowerTrade, currentUserId uint) []vo.PowerTradeOrder {
	list := make([]vo.PowerTradeOrder, 0, len(items))
	if len(items) == 0 {
		return list
	}

	userIds := make(map[uint]struct{})
	for _, item := range items {
		if item.UserId > 0 {
			userIds[item.UserId] = struct{}{}
		}
		if item.TargetUserId > 0 {
			userIds[uint(item.TargetUserId)] = struct{}{}
		}
	}

	userMap := make(map[uint]model.User)
	if len(userIds) > 0 {
		idList := make([]uint, 0, len(userIds))
		for id := range userIds {
			idList = append(idList, id)
		}
		var users []model.User
		h.DB.Where("id IN ?", idList).Find(&users)
		for _, u := range users {
			userMap[u.Id] = u
		}
	}

	for _, item := range items {
		role := "seller"
		counterpartyId := uint(item.TargetUserId)

		if currentUserId == uint(item.TargetUserId) {
			role = "buyer"
			counterpartyId = item.UserId
		} else if currentUserId != item.UserId {
			// fallback when current user only appears as target but target id might be zero
			role = "buyer"
		}

		var counterparty model.User
		if counterpartyId > 0 {
			counterparty = userMap[counterpartyId]
		}

		seller := userMap[item.UserId]
		paymentType, paymentAddr := resolveUserPaymentInfo(seller)
		if paymentAddr == "" {
			paymentAddr = strings.TrimSpace(item.EthAddress)
			if paymentType != "alipay" {
				paymentType = "bsc"
			}
		}

		list = append(list, vo.PowerTradeOrder{
			Id:                 item.Id,
			OrderNumber:        item.OrderNumber,
			Role:               role,
			Status:             item.Status,
			ProofUrl:           item.Image,
			ListedPower:        item.Number,
			DestroyedPower:     item.TotalPriceChina,
			TotalPower:         item.TotalPrice,
			Price:              item.Price,
			ValueUsd:           item.Number * item.Price,
			CreatedAt:          item.CreatedAt.Unix(),
			MatchTime:          item.MatchTime,
			PayTime:            item.PayTime,
			FinishTime:         item.FinishTime,
			CounterpartyName:   counterparty.Username,
			CounterpartyAvatar: counterparty.Avatar,
			EthAddress:         paymentAddr,
			PaymentType:        paymentType,
			PaymentAddress:     paymentAddr,
		})
	}

	return list
}

// 算力取消订单
// 本方法会将对应的算力返还给用户，包括预扣除的50%算力也要归还。
func (h *TradeHandler) CancelPowerTradeOrder(c *gin.Context) {
	var data struct {
		Id int `json:"orderId"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		fmt.Printf("Bind JSON error: %v\n", err)
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	logger.Info("收到取消订单请求，解析后的数据：", data)
	fmt.Printf("Received data: %+v\n", data) // 调试信息

	//获取系统配置的最大取消限制
	//var limitCancel int64 = h.App.SysConfig.LimitCancel
	//	var limitCancel int64 = 3 //最大3次
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}
	// 获取当天用户取消订单次数
	//var currentTrade model.PowerTrade
	//var currentTradeCount int64 = 0
	//
	//currentTradeRes := h.DB.Model(&currentTrade).
	//	Where("target_user_id = ? AND types = ? AND status=5  AND DATE(updated_at) = CURDATE()", user.Id, 2).
	//	Count(&currentTradeCount)
	//if currentTradeRes.Error != nil {
	//	logger.Error("Error fetching trade count:", currentTradeRes.Error)
	//} else {
	//	logger.Info("Current user trade count with types=2:", currentTradeCount)
	//}
	//fmt.Printf("当天取消的数量为 data: %+v\n", currentTradeCount) // 调试信息
	//
	//if currentTradeCount == limitCancel {
	//	logger.Info("已达到当天取消最大限制")
	//}

	// 获取订单信息
	var trade model.PowerTrade
	res := h.DB.First(&trade, data.Id)
	if res.Error != nil {
		resp.ERROR(c, "No trade found")
		return
	}
	// 只允许未完成的订单取消
	if trade.Status == 5 || trade.Status == 4 {
		resp.ERROR(c, "订单已完成或已取消，无法进行操作")
		return
	}

	// 回滚算力逻辑，Types区分买单(1)和卖单(2)
	// 先标记订单为已取消
	updateRes := h.DB.Model(&trade).Where("id = ?", data.Id).Updates(model.PowerTrade{
		Status: 5, //订单已取消
	})
	fmt.Printf("Database response: %+v\n", updateRes) // 调试信息

	if updateRes.Error != nil {
		resp.ERROR(c, updateRes.Error.Error())
		return
	}

	//原始值
	totalToReturn := trade.TotalPrice
	h.DB.Model(&model.User{}).
		Where("id = ?", trade.UserId).
		UpdateColumn("power", gorm.Expr("power + ?", totalToReturn))

	//原始用户
	var userModel model.User
	h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).First(&userModel)

	h.DB.Create(&model.PowerLog{
		UserId:    userModel.Id,
		Username:  userModel.Username,
		Type:      types.PowerFk,
		Amount:    totalToReturn,
		Mark:      types.PowerAdd,
		Balance:   userModel.Power,
		Model:     fmt.Sprintf("订单取消 取消人 %s ,下单人 %s", user.Username, userModel.Username), //session.Model.Value,
		Remark:    fmt.Sprintf("算力回滚：%f, 回滚后用户余额为 %f ", totalToReturn, userModel.Power),
		CreatedAt: time.Now(),
	})

	resp.SUCCESS(c)
}

// 获取最近30分钟内的状态为 status 为3的 chatgpt_power_trades 表中的数据
// 示例数据为 id	order_number	user_id	user_name	number	price	create_time	target_user_id	match_time	image	pay_time	finish_time	status	charge_number	is_comment	types	total_price	total_price_china	created_at	updated_at	eth_address
func (h *TradeHandler) TradesPay(c *gin.Context) {
	key := c.Query("key")
	if key != "test8899" {
		resp.ERROR(c, "非法访问")
		return
	}

	logger.Info("开始检查交易支付状态：", time.Now())

	// 获取最近30分钟内状态为3的交易数据
	thirtyMinutesAgo := time.Now().Add(-30 * time.Minute)
	var trades []model.PowerTrade

	res := h.DB.Where("status = ? AND created_at >= ?", 3, thirtyMinutesAgo).Find(&trades)
	if res.Error != nil {
		logger.Error("获取交易数据失败：", res.Error)
		resp.ERROR(c, "获取交易数据失败")
		return
	}

	logger.Info("找到待确认交易数量：", len(trades))

	// 循环处理每个交易
	for _, trade := range trades {
		logger.Info("处理交易ID：", trade.Id, "订单号：", trade.OrderNumber)

		// 校验是否已付款（通过BSC链验证）
		// 注意：交易验证暂时不检查 from_address，保持向后兼容
		// TODO: 后续如果需要，可以添加交易方的BSC地址检查
		isPaid := h.verifyBSCTransactionOptimized(trade.EthAddress, "", trade.Number*trade.Price)

		if isPaid {
			// 如果已付款，则更新状态为4（已确认）
			err := h.updateTradeStatusToPaid(trade)
			if err != nil {
				logger.Error("更新交易状态失败：", err)
				continue
			}

			//买方和卖方都添加交易Q2变更的日志
			logger.Info("交易已确认并完成：", trade.OrderNumber)

			//更新 config_json字段中"limit_cancel"
			//买卖不销毁

			//err = h.updateLimitCancelConfig()
			//if err != nil {
			//	logger.Error("更新limit_cancel配置失败：", err)
			//}

		} else {
			// 如果未付款，检查是否超时，超时则更改状态为5（已取消）
			if h.isTradeTimeout(trade) {
				err := h.updateTradeStatusToCancelled(trade)
				if err != nil {
					logger.Error("取消交易失败：", err)
					continue
				}
				logger.Info("交易已超时取消：", trade.OrderNumber)
			}
		}
	}

	resp.SUCCESS(c, gin.H{
		"processed_count": len(trades),
		"message":         "交易支付状态检查完成",
	})
}

// verifyBSCTransactionWithBscScan 使用 Etherscan API V2 验证交易（推荐方案，最稳定）
// 注意：
//  1. BSCScan API V1 已弃用（将在 2025年8月15日停止服务），现在统一使用 Etherscan API V2
//  2. Etherscan 统一了多链 API，BSC 链通过 chainid=56 访问
//  3. 重要：Etherscan API V2 的免费计划不支持 BSC 链（chainid=56），需要付费计划
//     如果遇到"Free API access is not supported"错误，代码会自动降级到 RPC 方式
//  4. 如需使用 Etherscan API，请访问 https://etherscan.io/apis 升级到付费计划
func (h *TradeHandler) verifyBSCTransactionWithBscScan(address string, expectedAmount float64) bool {
	apiKey := h.App.Config.Bsc.BscScanApiKey
	if apiKey == "" {
		// 如果没有配置 Etherscan API Key，降级到 RPC 方式
		logger.Info("Etherscan API Key 未配置，降级到 RPC 方式")
		return h.verifyBSCTransactionWithMultiRPC(address, expectedAmount)
	}

	// Etherscan API V2 地址（统一支持多链，BSC chainid=56）
	apiURL := "https://api.etherscan.io/v2/api"
	chainID := 56 // BSC 主网 chainid

	// 获取最新区块号
	latestBlock, err := h.getLatestBlockNumber(apiKey)
	if err != nil {
		logger.Error("获取最新区块号失败：", err)
		return false
	}

	// 计算30分钟前的区块号（BSC平均3秒一个块，30分钟约600个块）
	thirtyMinutesBlocks := int64(600)
	startBlock := latestBlock - thirtyMinutesBlocks
	if startBlock < 0 {
		startBlock = 0
	}

	// BSC USDT 合约地址
	usdtContractAddress := "0x55d398326f99059ff775485246999027b3197955"

	// 构建 Etherscan API V2 请求
	// 使用 getLogs 接口查询 Transfer 事件
	// 注意：Etherscan API V2 统一支持多链，需要指定 chainid
	client := &http.Client{Timeout: 60 * time.Second}
	thirtyMinutesAgo := time.Now().Add(-60 * time.Minute).Unix()
	usdtDecimal := int64(18)

	// Etherscan API V2 支持批量查询，但建议分批查询以避免超时
	batchSize := int64(1000) // 单次最多返回 10000 条记录，我们分批查询
	currentEndBlock := latestBlock

	for currentEndBlock > startBlock {
		currentStartBlock := currentEndBlock - batchSize + 1
		if currentStartBlock < startBlock {
			currentStartBlock = startBlock
		}

		logger.Info(fmt.Sprintf("Etherscan API V2 查询区块范围：%d 到 %d", currentStartBlock, currentEndBlock))

		// 构建 Etherscan API V2 请求 URL
		// 查询 Transfer 事件，过滤 to 地址
		// topic0 = Transfer 事件签名
		// topic2 = to 地址（目标地址）
		toAddressTopic := strings.TrimPrefix(strings.ToLower(address), "0x")
		transferEventSignature := "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

		url := fmt.Sprintf("%s?chainid=%d&module=logs&action=getLogs&fromBlock=%d&toBlock=%d&address=%s&topic0=%s&topic2=0x000000000000000000000000%s&apikey=%s",
			apiURL,
			chainID,
			currentStartBlock,
			currentEndBlock,
			usdtContractAddress,
			transferEventSignature,
			toAddressTopic,
			apiKey,
		)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			logger.Error("创建 BSCScan API 请求失败：", err)
			currentEndBlock = currentStartBlock - 1
			continue
		}

		// 打印请求信息
		logger.Info(fmt.Sprintf("=== Etherscan API V2 请求（区块 %d-%d）===", currentStartBlock, currentEndBlock))
		logger.Info(fmt.Sprintf("请求 URL：%s", url))

		resp, err := client.Do(req)
		if err != nil {
			logger.Error("Etherscan API V2 请求失败：", err)
			currentEndBlock = currentStartBlock - 1
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			logger.Error("读取 Etherscan API V2 响应失败：", err)
			currentEndBlock = currentStartBlock - 1
			continue
		}

		// 打印响应信息
		logger.Info(fmt.Sprintf("=== Etherscan API V2 响应（区块 %d-%d）===", currentStartBlock, currentEndBlock))
		logger.Info(fmt.Sprintf("HTTP 状态码：%d", resp.StatusCode))
		logger.Info(fmt.Sprintf("响应体长度：%d 字节", len(body)))
		logger.Info(fmt.Sprintf("完整响应内容：%s", string(body)))

		// 解析 Etherscan API V2 响应
		var bscScanResponse struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Result  []struct {
				Address         string   `json:"address"`
				Topics          []string `json:"topics"`
				Data            string   `json:"data"`
				BlockNumber     string   `json:"blockNumber"`
				TransactionHash string   `json:"transactionHash"`
				BlockHash       string   `json:"blockHash"`
				TimeStamp       string   `json:"timeStamp"`
				GasPrice        string   `json:"gasPrice"`
				GasUsed         string   `json:"gasUsed"`
				LogIndex        string   `json:"logIndex"`
			} `json:"result"`
		}

		if err := json.Unmarshal(body, &bscScanResponse); err != nil {
			logger.Error("解析 Etherscan API V2 响应失败：", err)
			bodyStr := string(body)
			// 检查是否是 V1 弃用错误
			if strings.Contains(bodyStr, "deprecated") || strings.Contains(bodyStr, "V1 endpoint") {
				logger.Error("检测到 API V1 弃用错误，请确认已迁移到 V2 API")
			}
			// 检查是否是免费计划不支持的错误
			if strings.Contains(strings.ToLower(bodyStr), "free api access is not supported") ||
				strings.Contains(strings.ToLower(bodyStr), "upgrade your api plan") {
				logger.Warn("Etherscan API V2 免费计划不支持 BSC 链（chainid=56），需要付费计划。降级到 RPC 方式")
				logger.Warn("提示：如需使用 Etherscan API，请访问 https://etherscan.io/apis 升级到付费计划")
				return h.verifyBSCTransactionWithMultiRPC(address, expectedAmount)
			}
			currentEndBlock = currentStartBlock - 1
			continue
		}

		// 检查 API 响应状态
		if bscScanResponse.Status != "1" {
			// 检查是否是免费计划不支持的错误
			errorMsg := strings.ToLower(bscScanResponse.Message)
			if strings.Contains(errorMsg, "free api access is not supported") ||
				strings.Contains(errorMsg, "upgrade your api plan") ||
				strings.Contains(errorMsg, "not supported for this chain") {
				logger.Warn("Etherscan API V2 免费计划不支持 BSC 链（chainid=56），需要付费计划。降级到 RPC 方式")
				logger.Warn("提示：如需使用 Etherscan API，请访问 https://etherscan.io/apis 升级到付费计划")
				// 降级到 RPC 方式
				return h.verifyBSCTransactionWithMultiRPC(address, expectedAmount)
			}
			// 检查是否是速率限制错误
			if strings.Contains(errorMsg, "rate limit") ||
				strings.Contains(errorMsg, "max rate limit") {
				logger.Warn(fmt.Sprintf("Etherscan API V2 达到速率限制，降级到 RPC 方式：%s", bscScanResponse.Message))
				// 降级到 RPC 方式
				return h.verifyBSCTransactionWithMultiRPC(address, expectedAmount)
			}
			logger.Warn(fmt.Sprintf("Etherscan API V2 返回错误：%s - %s", bscScanResponse.Status, bscScanResponse.Message))
			currentEndBlock = currentStartBlock - 1
			continue
		}

		logger.Info(fmt.Sprintf("Etherscan API V2 找到 %d 个 Transfer 事件（区块 %d-%d）", len(bscScanResponse.Result), currentStartBlock, currentEndBlock))

		// 处理查询结果
		for i, log := range bscScanResponse.Result {
			logger.Info(fmt.Sprintf("=== 处理第 %d 个 Transfer 事件 ===", i+1))
			logger.Info("交易哈希：", log.TransactionHash)
			logger.Info("区块号：", log.BlockNumber)

			// 解析时间戳
			timestamp, err := strconv.ParseInt(log.TimeStamp, 10, 64)
			if err != nil {
				logger.Error("解析时间戳失败：", err)
				continue
			}

			logger.Info("交易时间戳：", timestamp, "，60分钟前时间戳：", thirtyMinutesAgo)
			if timestamp < thirtyMinutesAgo {
				logger.Info("交易时间超过60分钟，跳过")
				continue
			}

			// 解析转账金额（data 字段是 uint256，32字节）
			valueHex := strings.TrimPrefix(log.Data, "0x")
			if len(valueHex) < 64 {
				logger.Error("数据格式错误，跳过")
				continue
			}

			// 解析 value（big.Int）
			valueBig, ok := new(big.Int).SetString(valueHex, 16)
			if !ok {
				logger.Error("解析转账金额失败，跳过")
				continue
			}

			// 转换为带小数的浮点数
			valueFloat := new(big.Float).SetInt(valueBig)
			divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(usdtDecimal), nil))
			valueFloat.Quo(valueFloat, divisor)
			actualAmount, _ := valueFloat.Float64()

			logger.Info("转账金额（原始值）：", valueBig.String())
			logger.Info("转账金额（USDT）：", actualAmount)
			logger.Info("期望金额：", expectedAmount)

			// 检查金额是否匹配
			if h.checkTokenTransactionAmountByFloat(actualAmount, expectedAmount) {
				// 验证交易收据状态和确认数
				blockNum, _ := strconv.ParseInt(log.BlockNumber, 10, 64)
				if h.verifyTransactionReceiptByBscScan(apiKey, log.TransactionHash, latestBlock, blockNum) {
					logger.Info("✅ 找到匹配的USDT交易（已确认）：", log.TransactionHash)
					return true
				} else {
					logger.Info("⚠️ 交易金额匹配，但交易未充分确认或状态异常：", log.TransactionHash)
				}
			} else {
				logger.Info("❌ 交易金额不匹配")
			}
		}

		// 移动到下一批
		currentEndBlock = currentStartBlock - 1

		// Etherscan API V2 有速率限制，添加延迟
		time.Sleep(200 * time.Millisecond)
	}

	logger.Info("Etherscan API V2 未找到匹配的交易")
	return false
}

// verifyBSCTransactionWithMultiRPC 使用多个 RPC 节点轮询验证交易（备选方案）
func (h *TradeHandler) verifyBSCTransactionWithMultiRPC(address string, expectedAmount float64) bool {
	// 获取 RPC URL 列表
	rpcURLs := h.getRpcURLList()
	if len(rpcURLs) == 0 {
		logger.Error("没有可用的 RPC 节点")
		return false
	}

	logger.Info(fmt.Sprintf("使用 %d 个 RPC 节点进行轮询查询", len(rpcURLs)))

	// 尝试每个 RPC 节点
	for i, rpcURL := range rpcURLs {
		logger.Info(fmt.Sprintf("尝试 RPC 节点 %d/%d: %s", i+1, len(rpcURLs), rpcURL))

		// 使用单个 RPC 节点查询
		if h.verifyBSCTransactionWithSingleRPC(rpcURL, address, expectedAmount) {
			return true
		}

		// 如果当前节点失败，等待后尝试下一个节点
		if i < len(rpcURLs)-1 {
			logger.Info("当前 RPC 节点查询失败，切换到下一个节点")
			time.Sleep(1 * time.Second)
		}
	}

	logger.Info("所有 RPC 节点查询均失败")
	return false
}

// verifyBSCTransactionWithSingleRPC 使用单个 RPC 节点验证交易
func (h *TradeHandler) verifyBSCTransactionWithSingleRPC(rpcURL string, address string, expectedAmount float64) bool {
	apiKey := "XT44Q8NA1P4FDIEWPS2HCU5EYT2MWXCIWM" // 用于获取最新区块号
	if apiKey == "" {
		logger.Error("BSC API Key未配置")
		return false
	}

	// 1. 先获取最新区块号
	latestBlock, err := h.getLatestBlockNumber(apiKey)
	if err != nil {
		logger.Error("获取最新区块号失败：", err)
		return false
	}

	// 2. 估算30分钟前的区块号
	thirtyMinutesBlocks := int64(600)
	startBlock := latestBlock - thirtyMinutesBlocks
	if startBlock < 0 {
		startBlock = 0
	}

	// 3. 使用 JSON-RPC eth_getLogs 查询 ERC20 Transfer 事件
	usdtContractAddress := "0x55d398326f99059ff775485246999027b3197955"
	transferEventSignature := "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	toAddressTopic := "0x000000000000000000000000" + strings.TrimPrefix(strings.ToLower(address), "0x")

	// 确保URL格式正确
	if !strings.HasSuffix(rpcURL, "/") {
		rpcURL += "/"
	}

	// 4. 分批查询，每次查询较小范围的区块
	batchSize := int64(5)            // 更小的批次大小，避免 limit exceeded
	minBatchSize := int64(2)         // 最小批次大小
	absoluteMinBatchSize := int64(1) // 绝对最小批次大小
	thirtyMinutesAgo := time.Now().Add(-30 * time.Minute).Unix()
	usdtDecimal := int64(18)

	// 指数退避策略
	consecutiveFailures := 0
	baseDelay := 2 * time.Second // 增加基础延迟
	maxDelay := 60 * time.Second
	limitExceededDelay := 30 * time.Second

	// 从最新区块开始，向前分批查询
	currentEndBlock := latestBlock
	for currentEndBlock > startBlock {
		currentStartBlock := currentEndBlock - batchSize + 1
		if currentStartBlock < startBlock {
			currentStartBlock = startBlock
		}

		logger.Info(fmt.Sprintf("RPC 查询区块范围：%d 到 %d（批次大小：%d，连续失败次数：%d）", currentStartBlock, currentEndBlock, batchSize, consecutiveFailures))

		// 在请求前添加延迟（指数退避）
		if consecutiveFailures > 0 {
			delay := baseDelay * time.Duration(1<<uint(consecutiveFailures))
			if delay > maxDelay {
				delay = maxDelay
			}
			logger.Info(fmt.Sprintf("等待 %v 后重试（指数退避策略）", delay))
			time.Sleep(delay)
		} else {
			time.Sleep(baseDelay)
		}

		// 构建请求
		requestBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "eth_getLogs",
			"params": []interface{}{
				map[string]interface{}{
					"fromBlock": fmt.Sprintf("0x%x", currentStartBlock),
					"toBlock":   fmt.Sprintf("0x%x", currentEndBlock),
					"address":   usdtContractAddress,
					"topics": []interface{}{
						transferEventSignature,
						nil,
						toAddressTopic,
					},
				},
			},
			"id": 1,
		}

		jsonData, err := json.Marshal(requestBody)
		if err != nil {
			logger.Error("构建 JSON-RPC 请求失败：", err)
			currentEndBlock = currentStartBlock - 1
			continue
		}

		req, err := http.NewRequest("POST", rpcURL, bytes.NewBuffer(jsonData))
		if err != nil {
			logger.Error("创建 HTTP 请求失败：", err)
			currentEndBlock = currentStartBlock - 1
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		// 如果是 Moralis RPC，添加 API Key header
		h.addMoralisApiKeyHeader(req, rpcURL)

		// 打印完整的请求日志
		logger.Info(fmt.Sprintf("=== JSON-RPC 完整请求（区块 %d-%d，RPC: %s）===", currentStartBlock, currentEndBlock, rpcURL))
		logger.Info(fmt.Sprintf("请求 URL：%s", rpcURL))
		logger.Info(fmt.Sprintf("请求方法：%s", req.Method))
		logger.Info(fmt.Sprintf("请求头：%v", req.Header))
		logger.Info(fmt.Sprintf("完整请求内容：%s", string(jsonData)))

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			logger.Error("JSON-RPC 请求失败：", err)
			consecutiveFailures++
			currentEndBlock = currentStartBlock - 1
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			logger.Error("读取响应失败：", err)
			consecutiveFailures++
			currentEndBlock = currentStartBlock - 1
			continue
		}

		// 打印完整的返回日志
		logger.Info(fmt.Sprintf("=== JSON-RPC 完整响应（区块 %d-%d）===", currentStartBlock, currentEndBlock))
		logger.Info(fmt.Sprintf("HTTP 状态码：%d", resp.StatusCode))
		logger.Info(fmt.Sprintf("响应体长度：%d 字节", len(body)))
		logger.Info(fmt.Sprintf("完整响应内容：%s", string(body)))

		// 解析 JSON-RPC 响应
		var rpcResponse struct {
			Result []struct {
				Address          string   `json:"address"`
				Topics           []string `json:"topics"`
				Data             string   `json:"data"`
				BlockNumber      string   `json:"blockNumber"`
				TransactionHash  string   `json:"transactionHash"`
				TransactionIndex string   `json:"transactionIndex"`
				BlockHash        string   `json:"blockHash"`
				LogIndex         string   `json:"logIndex"`
				Removed          bool     `json:"removed"`
			} `json:"result"`
			Error *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}

		if err := json.Unmarshal(body, &rpcResponse); err != nil {
			logger.Error("解析 JSON-RPC 响应失败：", err)
			consecutiveFailures++
			currentEndBlock = currentStartBlock - 1
			continue
		}

		// 如果返回错误，记录并处理
		if rpcResponse.Error != nil {
			logger.Warn(fmt.Sprintf("=== JSON-RPC 错误详情（区块 %d-%d）===", currentStartBlock, currentEndBlock))
			logger.Warn(fmt.Sprintf("错误代码：%d", rpcResponse.Error.Code))
			logger.Warn(fmt.Sprintf("错误消息：%s", rpcResponse.Error.Message))

			isLimitExceeded := rpcResponse.Error.Code == -32005 ||
				strings.Contains(strings.ToLower(rpcResponse.Error.Message), "limit") ||
				strings.Contains(strings.ToLower(rpcResponse.Error.Message), "exceeded") ||
				strings.Contains(strings.ToLower(rpcResponse.Error.Message), "query returned more than")

			if isLimitExceeded {
				consecutiveFailures++
				logger.Warn(fmt.Sprintf("检测到 limit exceeded 错误（连续失败 %d 次），等待 %v 后重试", consecutiveFailures, limitExceededDelay))
				time.Sleep(limitExceededDelay)

				if batchSize > minBatchSize {
					batchSize = batchSize / 2
					if batchSize < minBatchSize {
						batchSize = minBatchSize
					}
					logger.Info(fmt.Sprintf("减小批次大小到 %d，重新尝试当前批次", batchSize))
					continue
				} else if batchSize > absoluteMinBatchSize {
					batchSize = absoluteMinBatchSize
					logger.Info(fmt.Sprintf("批次大小已是最小值 %d，尝试使用绝对最小批次大小 %d", minBatchSize, absoluteMinBatchSize))
					continue
				} else {
					// 批次已经是最小值，返回 false，让调用者尝试下一个 RPC 节点
					logger.Warn(fmt.Sprintf("批次大小已是绝对最小值 %d，仍然失败，当前 RPC 节点不可用", absoluteMinBatchSize))
					return false
				}
			} else {
				consecutiveFailures++
			}
			currentEndBlock = currentStartBlock - 1
			continue
		}

		// 成功请求后重置连续失败计数和批次大小
		consecutiveFailures = 0
		batchSize = int64(5)

		logger.Info(fmt.Sprintf("找到 %d 个 Transfer 事件（区块 %d-%d）", len(rpcResponse.Result), currentStartBlock, currentEndBlock))

		// 处理查询结果
		for i, log := range rpcResponse.Result {
			logger.Info(fmt.Sprintf("=== 处理第 %d 个 Transfer 事件 ===", i+1))
			logger.Info("交易哈希：", log.TransactionHash)
			logger.Info("区块号：", log.BlockNumber)

			// 解析 from 地址（topic[1]）
			fromAddress := ""
			if len(log.Topics) > 1 {
				fromTopic := log.Topics[1]
				if len(fromTopic) >= 42 {
					fromAddress = "0x" + fromTopic[len(fromTopic)-40:]
				}
			}
			logger.Info("From 地址：", fromAddress)

			// 解析 to 地址（topic[2]）
			toAddress := ""
			if len(log.Topics) > 2 {
				toTopic := log.Topics[2]
				if len(toTopic) >= 42 {
					toAddress = "0x" + toTopic[len(toTopic)-40:]
				}
			}

			// 验证 to 地址是否匹配
			if !strings.EqualFold(toAddress, address) {
				logger.Info("接收方地址不匹配，跳过")
				continue
			}

			// 获取区块时间戳
			timestamp, err := h.getBlockTimestamp(rpcURL, log.BlockNumber)
			if err != nil {
				logger.Error("获取区块时间戳失败：", err)
				continue
			}

			logger.Info("交易时间戳：", timestamp, "，30分钟前时间戳：", thirtyMinutesAgo)
			if timestamp < thirtyMinutesAgo {
				logger.Info("交易时间超过60分钟，跳过")
				continue
			}

			// 解析转账金额（data 字段是 uint256，32字节）
			valueHex := strings.TrimPrefix(log.Data, "0x")
			if len(valueHex) < 64 {
				logger.Error("数据格式错误，跳过")
				continue
			}

			// 解析 value（big.Int）
			valueBig, ok := new(big.Int).SetString(valueHex, 16)
			if !ok {
				logger.Error("解析转账金额失败，跳过")
				continue
			}

			// 转换为带小数的浮点数
			valueFloat := new(big.Float).SetInt(valueBig)
			divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(usdtDecimal), nil))
			valueFloat.Quo(valueFloat, divisor)
			actualAmount, _ := valueFloat.Float64()

			logger.Info("转账金额（原始值）：", valueBig.String())
			logger.Info("转账金额（USDT）：", actualAmount)
			logger.Info("期望金额：", expectedAmount)

			// 检查金额是否匹配
			if h.checkTokenTransactionAmountByFloat(actualAmount, expectedAmount) {
				// 验证交易收据状态和确认数
				if h.verifyTransactionReceipt(rpcURL, log.TransactionHash, latestBlock) {
					logger.Info("✅ 找到匹配的USDT交易（已确认）：", log.TransactionHash)
					return true
				} else {
					logger.Info("⚠️ 交易金额匹配，但交易未充分确认或状态异常：", log.TransactionHash)
				}
			} else {
				logger.Info("❌ 交易金额不匹配")
			}
		}

		// 移动到下一批
		currentEndBlock = currentStartBlock - 1
		time.Sleep(100 * time.Millisecond)
	}

	logger.Info("RPC 查询未找到匹配的交易")
	return false
}

// addMoralisApiKeyHeader 如果是 Moralis RPC，添加 API Key header
func (h *TradeHandler) addMoralisApiKeyHeader(req *http.Request, rpcURL string) {
	// 检查是否是 Moralis RPC 节点
	if h.App.Config.Bsc.MoralisApiKey != "" &&
		(strings.Contains(rpcURL, "moralis-nodes.com") || strings.Contains(rpcURL, "moralis.io")) {
		req.Header.Set("X-Api-Key", h.App.Config.Bsc.MoralisApiKey)
	}
}

// getRpcURLList 获取 RPC URL 列表（支持多个节点）
func (h *TradeHandler) getRpcURLList() []string {
	var urls []string

	// 优先使用 Moralis RPC（如果配置了 API Key）
	// 注意：Moralis RPC 需要在请求头中传递 API Key，URL 本身不包含 API Key
	if h.App.Config.Bsc.MoralisApiKey != "" {
		// Moralis 提供两个节点，优先使用 site1，失败时自动切换到 site2
		moralisRPC1 := "https://site1.moralis-nodes.com/bsc/"
		moralisRPC2 := "https://site2.moralis-nodes.com/bsc/"
		urls = append(urls, moralisRPC1, moralisRPC2)
		logger.Info("优先使用 Moralis RPC 节点 (site1 和 site2)")
	}

	// 然后使用配置的多个 RPC URL
	if len(h.App.Config.Bsc.RpcURLs) > 0 {
		urls = append(urls, h.App.Config.Bsc.RpcURLs...)
	}

	// 如果配置了单个 RPC URL，也添加到列表
	if h.App.Config.Bsc.RpcURL != "" {
		// 避免重复
		found := false
		for _, url := range urls {
			if url == h.App.Config.Bsc.RpcURL {
				found = true
				break
			}
		}
		if !found {
			urls = append(urls, h.App.Config.Bsc.RpcURL)
		}
	}

	// 如果没有配置任何 RPC URL，使用默认的公共节点列表
	if len(urls) == 0 {
		urls = []string{
			"https://bsc-dataseed.binance.org/",
			"https://bsc-dataseed1.defibit.io/",
			"https://bsc-dataseed1.ninicoin.io/",
			"https://bsc-dataseed2.defibit.io/",
			"https://bsc-dataseed3.defibit.io/",
		}
		logger.Info("使用默认的公共 RPC 节点列表")
	}

	return urls
}

// verifyTransactionReceiptByBscScan 使用 Etherscan API V2 验证交易收据
func (h *TradeHandler) verifyTransactionReceiptByBscScan(apiKey string, txHash string, latestBlock int64, txBlock int64) bool {
	// Etherscan API V2 获取交易状态（BSC chainid=56）
	apiURL := "https://api.etherscan.io/v2/api"
	chainID := 56 // BSC 主网 chainid
	url := fmt.Sprintf("%s?chainid=%d&module=proxy&action=eth_getTransactionReceipt&txhash=%s&apikey=%s",
		apiURL, chainID, txHash, apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		logger.Error("Etherscan API V2 获取交易收据失败：", err)
		return false
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取 Etherscan API V2 响应失败：", err)
		return false
	}

	var response struct {
		Status string `json:"status"`
		Result struct {
			Status        string `json:"status"`
			BlockNumber   string `json:"blockNumber"`
			Confirmations string `json:"-"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		logger.Error("解析 Etherscan API V2 响应失败：", err)
		// 检查是否是 V1 弃用错误
		if strings.Contains(string(body), "deprecated") || strings.Contains(string(body), "V1 endpoint") {
			logger.Error("检测到 API V1 弃用错误，请确认已迁移到 V2 API")
		}
		return false
	}

	if response.Status != "1" {
		return false
	}

	// 检查交易状态（0x1 表示成功）
	if response.Result.Status != "0x1" {
		logger.Info("交易状态异常：", response.Result.Status)
		return false
	}

	// 检查确认数（至少需要 1 个确认）
	confirmations := latestBlock - txBlock + 1
	if confirmations < 1 {
		logger.Info("交易确认数不足：", confirmations)
		return false
	}

	return true
}

// 优化的BSC交易验证方法11（分批查询，避免 limit exceeded 错误）
// verifyBSCTransactionWithMoralisWalletHistory 使用 Moralis Wallet History API 验证交易（推荐方案，高效且无限制）
// 此方法通过查询钱包历史记录来验证 USDT 转账，比 RPC 方式更高效且不会限流
// address: 平台收款地址（to_address）
// userBscAddress: 用户的BSC地址（from_address，必须严格匹配）
// expectedAmount: 期望的订单金额
// excludeHashes: 当前批次已使用的hash集合（用于避免同一批次内重复使用）
// 返回值: (交易hash, 是否匹配)
func (h *TradeHandler) verifyBSCTransactionWithMoralisWalletHistory(address string, userBscAddress string, expectedAmount float64, excludeHashes map[string]bool) (string, bool) {
	apiKey := h.App.Config.Bsc.MoralisApiKey
	if apiKey == "" {
		logger.Info("Moralis API Key 未配置，降级到其他方式")
		return "", false
	}

	// Moralis Wallet History API 地址
	apiURL := fmt.Sprintf("https://deep-index.moralis.io/api/v2.2/wallets/%s/history", address)

	// 构建请求参数
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		logger.Error("创建 Moralis API 请求失败：", err)
		return "", false
	}

	// 设置请求头
	req.Header.Set("accept", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	// 设置查询参数
	q := req.URL.Query()
	q.Set("chain", "bsc")
	q.Set("order", "DESC")
	q.Set("limit", "5") // 查询最近25笔交易
	req.URL.RawQuery = q.Encode()

	logger.Info(fmt.Sprintf("使用 Moralis Wallet History API 查询地址：%s，用户BSC地址：%s，期望金额：%.2f USDT", address, userBscAddress, expectedAmount))

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Moralis API 请求失败：", err)
		return "", false
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		logger.Error(fmt.Sprintf("Moralis API 返回错误状态码：%d，响应：%s", resp.StatusCode, string(body)))
		return "", false
	}

	// 解析响应
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取 Moralis API 响应失败：", err)
		return "", false
	}

	// 定义响应结构
	var moralisResponse struct {
		PageSize int `json:"page_size"`
		Page     int `json:"page"`
		Limit    int `json:"limit"`
		Result   []struct {
			Hash           string `json:"hash"`
			BlockNumber    string `json:"block_number"`
			BlockTimestamp string `json:"block_timestamp"`
			ERC20Transfers []struct {
				TokenSymbol    string `json:"token_symbol"`
				Address        string `json:"address"`
				FromAddress    string `json:"from_address"`    // 转账发起地址
				ToAddress      string `json:"to_address"`      // 转账接收地址
				ValueFormatted string `json:"value_formatted"` // 转账金额（格式化后的字符串）
				Direction      string `json:"direction"`       // 方向：send/receive
			} `json:"erc20_transfers"`
		} `json:"result"`
		Cursor interface{} `json:"cursor"`
	}

	if err := json.Unmarshal(body, &moralisResponse); err != nil {
		logger.Error("解析 Moralis API 响应失败：", err)
		return "", false
	}

	logger.Info(fmt.Sprintf("Moralis API 返回 %d 笔交易记录", len(moralisResponse.Result)))

	// BSC USDT 合约地址
	usdtContractAddress := "0x55d398326f99059ff775485246999027b3197955"

	// 平台 BSC 收款地址（必须严格匹配）
	platformAddress := h.App.SysConfig.Address // h.App.Config.Bsc.MainAddress
	if platformAddress == "" {
		// 如果配置中没有，使用配置的默认地址
		platformAddress = h.App.Config.Bsc.MainAddress //"0x354611cb70246b69db0c5b5d847a6d903115731d"
		logger.Warn("配置中未找到平台地址，使用默认地址")
	}
	logger.Info(fmt.Sprintf("平台收款地址：%s，用户BSC地址：%s，期望金额：%.2f USDT", platformAddress, userBscAddress, expectedAmount))

	// 计算30分钟前的时间戳
	thirtyMinutesAgo := time.Now().Add(-60 * time.Minute)

	// 将期望金额格式化为字符串（保留2位小数，用于精确比较）
	expectedAmountStr := fmt.Sprintf("%.2f", expectedAmount)

	// 标准化用户BSC地址（统一转为小写，去除0x前缀后重新添加）
	normalizedUserAddress := strings.ToLower(strings.TrimPrefix(userBscAddress, "0x"))
	if normalizedUserAddress != "" {
		normalizedUserAddress = "0x" + normalizedUserAddress
	}

	// 查询已使用的交易hash，避免重复使用
	// 先从数据库查询已保存的hash
	var usedTxHashes []string
	h.DB.Model(&model.PowerPoolRecharge{}).
		Where("tx_hash != '' AND tx_hash IS NOT NULL").
		Pluck("tx_hash", &usedTxHashes)

	usedTxHashMap := make(map[string]bool)
	for _, hash := range usedTxHashes {
		usedTxHashMap[strings.ToLower(hash)] = true
	}
	logger.Info(fmt.Sprintf("数据库中已使用的交易hash数量：%d", len(usedTxHashMap)))

	// 合并当前批次排除的hash（如果提供了）
	if excludeHashes != nil {
		for hash := range excludeHashes {
			usedTxHashMap[strings.ToLower(hash)] = true
		}
		logger.Info(fmt.Sprintf("合并后已使用的交易hash数量：%d", len(usedTxHashMap)))
	}

	// 遍历交易记录，查找匹配的 USDT 转账
	for _, tx := range moralisResponse.Result {
		// 检查交易hash是否已被使用
		txHashLower := strings.ToLower(tx.Hash)
		if usedTxHashMap[txHashLower] {
			logger.Info(fmt.Sprintf("交易hash %s 已被使用，跳过", tx.Hash))
			continue
		}
		// 解析区块时间戳
		blockTime, err := time.Parse(time.RFC3339, tx.BlockTimestamp)
		if err != nil {
			logger.Warn(fmt.Sprintf("解析区块时间戳失败：%s", tx.BlockTimestamp))
			continue
		}

		// 只检查最近30分钟内的交易
		if blockTime.Before(thirtyMinutesAgo) {
			logger.Info(fmt.Sprintf("交易 %s 时间超过60分钟，跳过", tx.Hash))
			continue
		}

		// 遍历 ERC20 转账记录
		for _, transfer := range tx.ERC20Transfers {
			// 第一步：检查是否为 USDT
			if !strings.EqualFold(transfer.TokenSymbol, "usdt") {
				continue
			}

			// 第二步：检查 USDT 合约地址是否匹配
			if !strings.EqualFold(transfer.Address, usdtContractAddress) {
				continue
			}

			// 第三步：检查方向是否为 receive（平台接收到的转账）
			if transfer.Direction != "receive" {
				continue
			}

			// 第四步：严格检查 from_address 必须等于用户的BSC地址（最关键的安全校验）
			if transfer.FromAddress == "" {
				logger.Warn(fmt.Sprintf("交易 %s 的 from_address 为空，跳过", tx.Hash))
				continue
			}
			normalizedFromAddress := strings.ToLower(strings.TrimPrefix(transfer.FromAddress, "0x"))
			if normalizedFromAddress != "" {
				normalizedFromAddress = "0x" + normalizedFromAddress
			}
			if normalizedFromAddress != normalizedUserAddress {
				logger.Info(fmt.Sprintf("交易 %s 的 from_address (%s) 不匹配用户BSC地址 (%s)，跳过",
					tx.Hash, transfer.FromAddress, userBscAddress))
				continue
			}

			// 第五步：严格检查 to_address 必须等于平台地址
			if !strings.EqualFold(transfer.ToAddress, platformAddress) {
				logger.Info(fmt.Sprintf("交易 %s 的 to_address (%s) 不匹配平台地址 (%s)，跳过",
					tx.Hash, transfer.ToAddress, platformAddress))
				continue
			}

			// 第六步：严格检查金额必须完全匹配（不允许误差）
			// 将 value_formatted 格式化为标准格式进行比较
			transferAmount, err := strconv.ParseFloat(transfer.ValueFormatted, 64)
			if err != nil {
				logger.Warn(fmt.Sprintf("解析金额失败：%s", transfer.ValueFormatted))
				continue
			}
			transferAmountStr := fmt.Sprintf("%.2f", transferAmount)

			logger.Info(fmt.Sprintf("找到 USDT 转账：交易哈希=%s，from_address=%s，to_address=%s，金额=%s USDT，期望金额=%s USDT",
				tx.Hash, transfer.FromAddress, transfer.ToAddress, transferAmountStr, expectedAmountStr))

			// 严格比较金额（字符串比较，确保完全匹配）
			if transferAmountStr == expectedAmountStr {
				logger.Info(fmt.Sprintf("✅ 严格匹配成功！交易哈希：%s，from_address：%s，to_address：%s，金额：%s USDT",
					tx.Hash, transfer.FromAddress, transfer.ToAddress, transferAmountStr))
				return tx.Hash, true
			} else {
				logger.Info(fmt.Sprintf("金额不匹配：实际金额=%s，期望金额=%s", transferAmountStr, expectedAmountStr))
			}
		}
	}

	logger.Info("未找到严格匹配的 USDT 转账记录（from_address、to_address 和金额都必须完全匹配）")
	return "", false
}

// 注意：此方法已被新的 verifyBSCTransactionWithMoralisWalletHistory、verifyBSCTransactionWithBscScan 和 verifyBSCTransactionWithMultiRPC 替代
// 保留此方法以保持向后兼容性
// address: 平台收款地址（to_address）
// userBscAddress: 用户的BSC地址（from_address，必须严格匹配）
// expectedAmount: 期望的订单金额
func (h *TradeHandler) verifyBSCTransactionOptimized(address string, userBscAddress string, expectedAmount float64) bool {
	// 优先使用 Moralis Wallet History API（如果配置了 API Key，推荐方案）
	if h.App.Config.Bsc.MoralisApiKey != "" {
		// 单个订单检查时，传入nil，函数内部会查询数据库中的已使用hash
		if _, result := h.verifyBSCTransactionWithMoralisWalletHistory(address, userBscAddress, expectedAmount, nil); result {
			return true
		}
		logger.Info("Moralis Wallet History API 未找到匹配交易，尝试其他方式")
	}

	// 其次使用 Etherscan API V2（如果配置了 API Key）
	// 注意：BscScan 和 MultiRPC 方法也需要添加 from_address 检查，但为了保持兼容性，暂时保留原参数
	// TODO: 后续需要修改这两个方法也支持 from_address 检查
	if h.App.Config.Bsc.BscScanApiKey != "" {
		return h.verifyBSCTransactionWithBscScan(address, expectedAmount)
	}

	// 最后使用多 RPC 节点轮询
	return h.verifyBSCTransactionWithMultiRPC(address, expectedAmount)
}

// 检查交易是否超时
func (h *TradeHandler) isTradeTimeout(trade model.PowerTrade) bool {
	// 假设交易超时时间为2小时
	timeout := 2 * time.Hour
	return time.Since(trade.CreatedAt) > timeout
}

// 更新交易状态为已取消
func (h *TradeHandler) updateTradeStatusToCancelled(trade model.PowerTrade) error {
	// 更新交易状态为5（已取消）
	err := h.DB.Model(&trade).Where("id = ?", trade.Id).Updates(model.PowerTrade{
		Status: 5,
	}).Error

	if err != nil {
		return err
	}

	////买家算力池放到原来的地方
	//if trade.Types == 1 {
	//	// 解冻卖方的SOHA币
	//	h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin - ?", trade.Number))
	//	h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("coin", gorm.Expr("coin + ?", trade.Number))
	//} else if trade.Types == 2 {
	//	// 解冻买方的SOHA币
	//	h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin - ?", trade.Number))
	//	h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("coin", gorm.Expr("coin + ?", trade.Number))
	//}

	return nil
}

// 获取最新区块号
func (h *TradeHandler) getLatestBlockNumber(apiKey string) (int64, error) {
	// 优先使用 Moralis RPC（如果配置了 API Key）
	var url string
	if h.App.Config.Bsc.MoralisApiKey != "" {
		// Moralis 新格式：使用 site1.moralis-nodes.com，API Key 通过 header 传递
		url = "https://site1.moralis-nodes.com/bsc/"
		logger.Info("使用 Moralis RPC 获取最新区块号")
	} else if h.App.Config.Bsc.RpcURL != "" {
		// 使用配置文件中的RPC URL
		url = h.App.Config.Bsc.RpcURL
	} else {
		// 使用默认值
		url = "https://bsc-dataseed.binance.org/"
	}
	// 确保URL以/结尾
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}

	// 构建 JSON-RPC 请求
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_blockNumber",
		"params":  []interface{}{},
		"id":      1,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	// 如果是 Moralis RPC，添加 API Key header
	h.addMoralisApiKeyHeader(req, url)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var response struct {
		Result string `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return 0, err
	}

	if response.Error != nil {
		return 0, fmt.Errorf("JSON-RPC error: %s (code: %d)", response.Error.Message, response.Error.Code)
	}

	// 将十六进制转换为十进制
	blockNumber, err := strconv.ParseInt(strings.TrimPrefix(response.Result, "0x"), 16, 64)
	if err != nil {
		return 0, err
	}

	return blockNumber, nil
}

// 更新交易状态为已付款
func (h *TradeHandler) updateTradeStatusToPaid(trade model.PowerTrade) error {
	// 更新交易状态为4（已确认）
	err := h.DB.Model(&trade).Where("id = ?", trade.Id).Updates(model.PowerTrade{
		Status:     4,
		FinishTime: time.Now().Unix(),
	}).Error

	if err != nil {
		return err
	}

	// 获取用户信息
	var buyUser model.User
	//var sellUser model.User

	// 处理算力转账

	//买方增加算力
	h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("power", gorm.Expr("power + ?", trade.Number))

	h.DB.First(&buyUser, trade.TargetUserId)

	// 添加买家用户的算力变化日志
	h.DB.Create(&model.PowerLog{
		UserId:    buyUser.Id,
		Username:  buyUser.Username,
		Type:      types.PowerSk,
		Amount:    trade.Number,
		Mark:      types.PowerAdd,
		Balance:   buyUser.Power,
		Model:     "算力撮合池",
		Remark:    fmt.Sprintf("订单%s成交，获得算力%.2f", trade.OrderNumber, trade.Number),
		CreatedAt: time.Now(),
	})

	return nil
}

// 检查代币交易金额是否匹配
func (h *TradeHandler) checkTokenTransactionAmount(valueStr, decimalStr string, expectedAmount float64) bool {
	logger.Info("=== 检查代币交易金额 ===")
	logger.Info("原始金额字符串：", valueStr)
	logger.Info("代币小数位：", decimalStr)
	logger.Info("期望金额：", expectedAmount)

	// 解析代币小数位
	decimal, err := strconv.ParseInt(decimalStr, 10, 64)
	if err != nil {
		logger.Error("解析代币小数位失败：", err)
		return false
	}

	// 解析原始金额
	value := new(big.Int)
	value.SetString(valueStr, 10)

	// 转换为实际金额
	divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(decimal), nil))
	amountFloat := new(big.Float).SetInt(value)
	amountFloat.Quo(amountFloat, divisor)

	actualAmount, _ := amountFloat.Float64()

	logger.Info("解析后的实际金额：", actualAmount)
	logger.Info("期望金额：", expectedAmount)

	// 使用前3位数匹配逻辑
	result := h.matchFirst3Digits(actualAmount, expectedAmount)
	logger.Info("金额匹配结果：", result)

	return result
}

// 检查代币交易金额是否匹配（使用浮点数）
func (h *TradeHandler) checkTokenTransactionAmountByFloat(actualAmount, expectedAmount float64) bool {
	logger.Info("=== 检查代币交易金额（浮点数） ===")
	logger.Info("实际金额：", actualAmount)
	logger.Info("期望金额：", expectedAmount)

	// 使用前3位数匹配逻辑
	result := h.matchFirst3Digits(actualAmount, expectedAmount)
	logger.Info("金额匹配结果：", result)

	return result
}

// 获取区块时间戳
func (h *TradeHandler) getBlockTimestamp(rpcURL, blockNumberHex string) (int64, error) {
	// 构建 eth_getBlockByNumber 请求
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getBlockByNumber",
		"params": []interface{}{
			blockNumberHex,
			false, // 不返回完整交易信息
		},
		"id": 1,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest("POST", rpcURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var response struct {
		Result *struct {
			Timestamp string `json:"timestamp"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return 0, err
	}

	if response.Error != nil {
		return 0, fmt.Errorf("JSON-RPC error: %s (code: %d)", response.Error.Message, response.Error.Code)
	}

	if response.Result == nil {
		return 0, fmt.Errorf("block not found")
	}

	// 将十六进制时间戳转换为十进制
	timestamp, err := strconv.ParseInt(strings.TrimPrefix(response.Result.Timestamp, "0x"), 16, 64)
	if err != nil {
		return 0, err
	}

	return timestamp, nil
}

// verifyTransactionReceipt 验证交易收据状态和确认数
// 要求至少 12 个确认数（BSC 平均 3 秒一个块，12 个确认约 36 秒，足够安全）
func (h *TradeHandler) verifyTransactionReceipt(rpcURL, txHash string, latestBlock int64) bool {
	// 构建 eth_getTransactionReceipt 请求
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getTransactionReceipt",
		"params":  []interface{}{txHash},
		"id":      1,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		logger.Error("构建交易收据请求失败：", err)
		return false
	}

	req, err := http.NewRequest("POST", rpcURL, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("创建交易收据请求失败：", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("交易收据请求失败：", err)
		return false
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取交易收据响应失败：", err)
		return false
	}

	var response struct {
		Result *struct {
			Status           string `json:"status"`           // "0x1" 表示成功，"0x0" 表示失败
			BlockNumber      string `json:"blockNumber"`      // 交易所在区块号
			TransactionIndex string `json:"transactionIndex"` // 交易在区块中的索引
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		logger.Error("解析交易收据响应失败：", err)
		return false
	}

	if response.Error != nil {
		logger.Error("交易收据查询错误：", response.Error.Message)
		return false
	}

	if response.Result == nil {
		logger.Warn("交易收据不存在（可能还在 pending）：", txHash)
		return false
	}

	// 检查交易状态：status 为 "0x1" 表示成功
	if response.Result.Status != "0x1" {
		logger.Warn("交易执行失败（status != 0x1）：", txHash, "status:", response.Result.Status)
		return false
	}

	// 解析交易所在区块号
	txBlockNumber, err := strconv.ParseInt(strings.TrimPrefix(response.Result.BlockNumber, "0x"), 16, 64)
	if err != nil {
		logger.Error("解析交易区块号失败：", err)
		return false
	}

	// 计算确认数：当前最新区块号 - 交易所在区块号
	confirmations := latestBlock - txBlockNumber
	logger.Info(fmt.Sprintf("交易 %s 的确认数：%d（最新区块：%d，交易区块：%d）", txHash, confirmations, latestBlock, txBlockNumber))

	// 要求至少 12 个确认数（BSC 平均 3 秒一个块，12 个确认约 36 秒）
	minConfirmations := int64(12)
	if confirmations < minConfirmations {
		logger.Warn(fmt.Sprintf("交易确认数不足（当前：%d，要求：%d）：%s", confirmations, minConfirmations, txHash))
		return false
	}

	logger.Info(fmt.Sprintf("✅ 交易已充分确认（确认数：%d）：%s", confirmations, txHash))
	return true
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

// GetPowerPoolInfo 获取算力池信息
func (h *TradeHandler) GetPowerPoolInfo(c *gin.Context) {
	var pool model.PowerPool
	// 获取或创建算力池记录
	result := h.DB.Where("id = ?", 1).FirstOrCreate(&pool, model.PowerPool{
		BaseModel:   model.BaseModel{Id: 1},
		TotalPower:  0,
		TotalUsdt:   0,
		BuybackUsdt: 0,
	})
	if result.Error != nil {
		resp.ERROR(c, "获取算力池信息失败: "+result.Error.Error())
		return
	}

	resp.SUCCESS(c, gin.H{
		"total_power":  pool.TotalPower,
		"total_usdt":   pool.TotalUsdt,
		"buyback_usdt": pool.BuybackUsdt,
	})
}

// CreatePowerPoolRecharge 创建算力池充值订单
func (h *TradeHandler) CreatePowerPoolRecharge(c *gin.Context) {
	var data struct {
		PowerAmount float64 `json:"power_amount"` // 购买的算力数量
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	if data.PowerAmount <= 0 {
		resp.ERROR(c, "请输入有效的算力数量")
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	// 检查用户是否设置了BSC地址
	if user.EthAddress == "" {
		resp.ERROR(c, "请先设置您的BSC地址")
		return
	}

	// 获取算力池信息
	var pool model.PowerPool
	result := h.DB.Where("id = ?", 1).FirstOrCreate(&pool, model.PowerPool{
		BaseModel:   model.BaseModel{Id: 1},
		TotalPower:  0,
		TotalUsdt:   0,
		BuybackUsdt: 0,
	})
	if result.Error != nil {
		resp.ERROR(c, "获取算力池信息失败: "+result.Error.Error())
		return
	}

	// 检查算力池是否有足够的算力
	if pool.TotalPower < data.PowerAmount {
		resp.ERROR(c, fmt.Sprintf("算力池算力不足，当前可用算力：%.2f", pool.TotalPower))
		return
	}

	// 计算需要支付的USDT金额 (100 USDT = 10000 算力，即 1 USDT = 100 算力)
	usdtAmount := data.PowerAmount / 100.0

	// 生成订单号
	orderNumber := fmt.Sprintf("RECHARGE%d%d", time.Now().Unix(), user.Id)

	// 获取BSC收款地址（从系统配置中获取，这里先使用固定地址，后续可以从配置中读取）
	bscAddress := h.getBscRechargeAddress()
	if bscAddress == "" {
		resp.ERROR(c, "BSC收款地址未配置")
		return
	}

	// 创建充值订单
	recharge := model.PowerPoolRecharge{
		OrderNumber:    orderNumber,
		UserId:         user.Id,
		PowerAmount:    data.PowerAmount,
		UsdtAmount:     usdtAmount,
		BscAddress:     bscAddress,
		UserBscAddress: user.EthAddress, // 保存用户的BSC地址，用于验证from_address
		Status:         1,               // 待支付
	}

	if err := h.DB.Create(&recharge).Error; err != nil {
		resp.ERROR(c, "创建充值订单失败: "+err.Error())
		return
	}

	resp.SUCCESS(c, gin.H{
		"order_id":     recharge.Id,
		"order_number": recharge.OrderNumber,
		"power_amount": recharge.PowerAmount,
		"usdt_amount":  recharge.UsdtAmount,
		"bsc_address":  recharge.BscAddress,
		"status":       recharge.Status,
	})
}

// GetBscRechargeAddress 获取BSC充值地址（从系统配置中获取）
func (h *TradeHandler) getBscRechargeAddress() string {
	// 这里可以从系统配置中读取BSC地址
	// 暂时返回一个固定地址，实际应该从配置表读取
	// var config model.Config
	// if err := h.DB.Where("marker = ?", "power_pool_bsc_address").First(&config).Error; err == nil {
	// 	// 解析配置JSON获取地址
	// 	var configData map[string]interface{}
	// 	if err := json.Unmarshal([]byte(config.Config), &configData); err == nil {
	// 		if address, ok := configData["address"].(string); ok {
	// 			return address
	// 		}
	// 	}
	// }
	// 默认地址

	return h.App.SysConfig.Address
	//return "0x7C156Eafa7686e7F4ad88aAF51BA6329FC9eEeFE"
}

// GetPowerPoolRechargeOrder 获取充值订单信息
func (h *TradeHandler) GetPowerPoolRechargeOrder(c *gin.Context) {
	orderId := c.Query("order_id")
	if orderId == "" {
		resp.ERROR(c, "订单ID不能为空")
		return
	}

	var recharge model.PowerPoolRecharge
	if err := h.DB.First(&recharge, orderId).Error; err != nil {
		resp.ERROR(c, "订单不存在")
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	if recharge.UserId != user.Id {
		resp.ERROR(c, "无权查看该订单")
		return
	}

	resp.SUCCESS(c, gin.H{
		"order_id":     recharge.Id,
		"order_number": recharge.OrderNumber,
		"power_amount": recharge.PowerAmount,
		"usdt_amount":  recharge.UsdtAmount,
		"bsc_address":  recharge.BscAddress,
		"status":       recharge.Status,
		"tx_hash":      recharge.TxHash,
		"pay_time":     recharge.PayTime,
		"finish_time":  recharge.FinishTime,
		"created_at":   recharge.CreatedAt.Unix(),
	})
}

// ListMyPowerPoolRecharges 列出当前用户最近 5 笔算力池充值订单
func (h *TradeHandler) ListMyPowerPoolRecharges(c *gin.Context) {
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	var recharges []model.PowerPoolRecharge
	if err := h.DB.
		Where("user_id = ?", user.Id).
		Order("id DESC").
		Limit(5).
		Find(&recharges).Error; err != nil {
		resp.ERROR(c, "查询充值记录失败："+err.Error())
		return
	}

	items := make([]gin.H, 0, len(recharges))
	for _, r := range recharges {
		items = append(items, gin.H{
			"order_id":     r.Id,
			"order_number": r.OrderNumber,
			"power_amount": r.PowerAmount,
			"usdt_amount":  r.UsdtAmount,
			"status":       r.Status,
			"status_text":  h.getRechargeStatusText(r.Status),
			"tx_hash":      r.TxHash,
			"pay_time":     r.PayTime,
			"finish_time":  r.FinishTime,
			"created_at":   r.CreatedAt.Unix(),
		})
	}

	resp.SUCCESS(c, gin.H{
		"items": items,
		"total": len(items),
	})
}

// CheckMyRechargePayment 用户主动检查自己的充值订单支付状态
func (h *TradeHandler) CheckMyRechargePayment(c *gin.Context) {
	orderId := c.Query("order_id")
	if orderId == "" {
		resp.ERROR(c, "订单ID不能为空")
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	var recharge model.PowerPoolRecharge
	if err := h.DB.First(&recharge, orderId).Error; err != nil {
		resp.ERROR(c, "订单不存在")
		return
	}

	if recharge.UserId != user.Id {
		resp.ERROR(c, "无权查看该订单")
		return
	}

	// 如果订单已经是已完成或已取消状态，直接返回
	if recharge.Status == 3 || recharge.Status == 4 {
		resp.SUCCESS(c, gin.H{
			"order_id":     recharge.Id,
			"order_number": recharge.OrderNumber,
			"status":       recharge.Status,
			"status_text":  h.getRechargeStatusText(recharge.Status),
			"message":      "订单状态已确定，无需检查",
		})
		return
	}

	// 如果订单是待支付状态，检查是否已超时
	if recharge.Status == 1 {
		if h.isRechargeTimeout(recharge) {
			err := h.cancelPowerPoolRecharge(recharge)
			if err != nil {
				logger.Error("取消超时订单失败：", err)
			} else {
				resp.SUCCESS(c, gin.H{
					"order_id":     recharge.Id,
					"order_number": recharge.OrderNumber,
					"status":       4,
					"status_text":  "已取消（超时）",
					"message":      "订单已超时，已自动取消",
				})
				return
			}
		}
	}

	// 如果订单状态是待支付或已支付，检查BSC链上的交易
	if recharge.Status == 1 || recharge.Status == 2 {
		logger.Info("检查充值订单支付状态，订单号：", recharge.OrderNumber)

		// 校验是否已付款（通过BSC链验证）
		// 严格检查：from_address 必须等于用户的BSC地址，to_address 必须等于平台地址，金额必须等于订单金额
		if recharge.UserBscAddress == "" {
			logger.Error("订单缺少用户BSC地址，无法验证")
			resp.ERROR(c, "订单信息不完整，无法验证支付状态")
			return
		}
		// 优先使用 Moralis Wallet History API 获取交易hash
		var txHash string
		var isPaid bool
		if h.App.Config.Bsc.MoralisApiKey != "" {
			// 单个订单检查时，传入nil，函数内部会查询数据库中的已使用hash
			txHash, isPaid = h.verifyBSCTransactionWithMoralisWalletHistory(recharge.BscAddress, recharge.UserBscAddress, recharge.UsdtAmount, nil)
		} else {
			// 降级到其他方式（不返回hash）
			isPaid = h.verifyBSCTransactionOptimized(recharge.BscAddress, recharge.UserBscAddress, recharge.UsdtAmount)
		}

		if isPaid {
			// 如果已付款，则更新状态并处理算力分配
			err := h.confirmPowerPoolRecharge(recharge, txHash)
			if err != nil {
				logger.Error("确认充值订单失败：", err)
				resp.ERROR(c, "确认充值订单失败："+err.Error())
				return
			}

			// 重新查询订单状态
			h.DB.First(&recharge, orderId)

			resp.SUCCESS(c, gin.H{
				"order_id":     recharge.Id,
				"order_number": recharge.OrderNumber,
				"status":       recharge.Status,
				"status_text":  h.getRechargeStatusText(recharge.Status),
				"message":      "支付已确认，算力已到账",
				"power_amount": recharge.PowerAmount,
			})
			return
		} else {
			// 未找到支付记录
			resp.SUCCESS(c, gin.H{
				"order_id":     recharge.Id,
				"order_number": recharge.OrderNumber,
				"status":       recharge.Status,
				"status_text":  h.getRechargeStatusText(recharge.Status),
				"message":      "未检测到支付记录，请确认是否已完成转账",
			})
			return
		}
	}

	resp.SUCCESS(c, gin.H{
		"order_id":     recharge.Id,
		"order_number": recharge.OrderNumber,
		"status":       recharge.Status,
		"status_text":  h.getRechargeStatusText(recharge.Status),
	})
}

// getRechargeStatusText 获取充值订单状态文本
func (h *TradeHandler) getRechargeStatusText(status int) string {
	switch status {
	case 1:
		return "待支付"
	case 2:
		return "已支付"
	case 3:
		return "已完成"
	case 4:
		return "已取消"
	default:
		return "未知状态"
	}
}

// 新版本更加稳定
// CheckPowerPoolRechargePayment 检查充值订单支付状态（BSC链监控）
func (h *TradeHandler) CheckPowerPoolRechargePayment(c *gin.Context) {
	key := c.Query("key")
	if key != "test8899" {
		resp.ERROR(c, "非法访问")
		return
	}

	// Moralis Stream 主动推送模式：接收 body 后直接按 ERC20 transfer 入账
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error("读取body失败：", err)
		resp.ERROR(c, "读取body失败")
		return
	}
	logger.Info("Moralis 推送 body：", string(body))

	var payload struct {
		Confirmed      bool `json:"confirmed"`
		ERC20Transfers []struct {
			Contract          string `json:"contract"`
			From              string `json:"from"`
			To                string `json:"to"`
			TransactionHash   string `json:"transactionHash"`
			LogIndex          string `json:"logIndex"`
			TokenSymbol       string `json:"tokenSymbol"`
			ValueWithDecimals string `json:"valueWithDecimals"`
		} `json:"erc20Transfers"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		logger.Error("解析 Moralis body 失败：", err)
		resp.ERROR(c, "解析推送数据失败")
		return
	}

	if len(payload.ERC20Transfers) == 0 {
		resp.SUCCESS(c, gin.H{"message": "无 ERC20 转账数据"})
		return
	}

	processedCount := 0
	matchedOrderIds := make([]uint, 0)

	// 默认平台地址（配置优先）
	platformAddress := strings.ToLower(strings.TrimSpace(h.App.SysConfig.Address))
	if platformAddress != "" && !strings.HasPrefix(platformAddress, "0x") {
		platformAddress = "0x" + strings.TrimPrefix(platformAddress, "0x")
	}

	thirtyMinutesAgo := time.Now().Add(-60 * time.Minute)
	usdtContractAddress := "0x55d398326f99059ff775485246999027b3197955"

	for _, transfer := range payload.ERC20Transfers {
		txHash := strings.ToLower(strings.TrimSpace(transfer.TransactionHash))
		fromAddress := strings.ToLower(strings.TrimSpace(transfer.From))
		toAddress := strings.ToLower(strings.TrimSpace(transfer.To))
		contractAddress := strings.ToLower(strings.TrimSpace(transfer.Contract))

		// 只处理 USDT 合约转账
		if contractAddress != usdtContractAddress {
			continue
		}
		if txHash == "" || fromAddress == "" || toAddress == "" {
			continue
		}

		// 如果配置了平台地址，to 必须匹配平台地址
		if platformAddress != "" && toAddress != platformAddress {
			logger.Info(fmt.Sprintf("忽略非平台入账转账：tx=%s, to=%s", txHash, toAddress))
			continue
		}

		// tx hash 只能使用一次：数据库里已存在则跳过
		var existed int64
		if err := h.DB.Model(&model.PowerPoolRecharge{}).Where("tx_hash = ?", txHash).Count(&existed).Error; err == nil && existed > 0 {
			logger.Info(fmt.Sprintf("交易hash已使用，跳过：%s", txHash))
			continue
		}

		transferAmount, err := strconv.ParseFloat(strings.TrimSpace(transfer.ValueWithDecimals), 64)
		if err != nil {
			logger.Warn(fmt.Sprintf("解析 valueWithDecimals 失败：%s, tx=%s", transfer.ValueWithDecimals, txHash))
			continue
		}

		// 匹配最近30分钟内待支付订单：from 对应 user_bsc_address，to 对应 bsc_address（平台地址）
		// 同一用户可能有多笔待支付（金额不同）；不能只取 id 最早的一笔，否则会出现「0.5U 付款却先撞上 0.8U 订单」后整条 continue、永远不到账
		var pendingList []model.PowerPoolRecharge
		query := h.DB.Where("status = ? AND created_at >= ? AND LOWER(user_bsc_address) = ? AND LOWER(bsc_address) = ?",
			1, thirtyMinutesAgo, fromAddress, toAddress).
			Order("id ASC")
		if err := query.Find(&pendingList).Error; err != nil || len(pendingList) == 0 {
			logger.Info(fmt.Sprintf("未找到匹配订单：from=%s,to=%s,tx=%s", fromAddress, toAddress, txHash))
			continue
		}

		var recharge *model.PowerPoolRecharge
		for i := range pendingList {
			if math.Abs(pendingList[i].UsdtAmount-transferAmount) <= 0.00000001 {
				recharge = &pendingList[i]
				break
			}
		}
		if recharge == nil {
			logger.Info(fmt.Sprintf("无金额匹配的待支付订单：from=%s,to=%s,transfer=%.8f,tx=%s,候选数=%d",
				fromAddress, toAddress, transferAmount, txHash, len(pendingList)))
			continue
		}

		if err := h.confirmPowerPoolRecharge(*recharge, txHash); err != nil {
			logger.Error("确认充值订单失败：", err)
			continue
		}

		processedCount++
		matchedOrderIds = append(matchedOrderIds, recharge.Id)
		logger.Info(fmt.Sprintf("Moralis webhook 入账成功：order=%s, user=%s, amount=%.8f, tx=%s",
			recharge.OrderNumber, fromAddress, transferAmount, txHash))
	}

	resp.SUCCESS(c, gin.H{
		"message":           "Moralis 推送处理完成",
		"processed_count":   processedCount,
		"matched_order_ids": matchedOrderIds,
		"confirmed":         payload.Confirmed,
	})
}

// 原始版本
// CheckPowerPoolRechargePayment 检查充值订单支付状态（BSC链监控）
func (h *TradeHandler) CheckPowerPoolRechargePayment2(c *gin.Context) {
	key := c.Query("key")
	if key != "test8899" {
		resp.ERROR(c, "非法访问")
		return
	}

	logger.Info("开始检查算力池充值订单支付状态：", time.Now())

	// 获取最近60分钟内状态为1（待支付）的充值订单
	sixMinutesAgo := time.Now().Add(-60 * time.Minute)
	var recharges []model.PowerPoolRecharge

	res := h.DB.Where("status = ? AND created_at >= ?", 1, sixMinutesAgo).Find(&recharges)
	if res.Error != nil {
		logger.Error("获取充值订单失败：", res.Error)
		resp.ERROR(c, "获取充值订单失败")
		return
	}

	logger.Info("找到待支付充值订单数量：", len(recharges))

	// 查询数据库中已使用的交易hash，避免重复使用
	var usedTxHashes []string
	h.DB.Model(&model.PowerPoolRecharge{}).
		Where("tx_hash != '' AND tx_hash IS NOT NULL").
		Pluck("tx_hash", &usedTxHashes)

	// 维护当前批次已使用的hash集合（内存中），用于避免同一批次内重复使用
	currentBatchUsedHashes := make(map[string]bool)
	for _, hash := range usedTxHashes {
		currentBatchUsedHashes[strings.ToLower(hash)] = true
	}
	logger.Info(fmt.Sprintf("初始化已使用的交易hash数量：%d", len(currentBatchUsedHashes)))

	// 循环处理每个充值订单
	for _, recharge := range recharges {
		logger.Info("处理充值订单ID：", recharge.Id, "订单号：", recharge.OrderNumber)

		// 校验是否已付款（使用 Moralis Wallet History API，高效且无限制）
		// 严格检查：from_address 必须等于用户的BSC地址，to_address 必须等于平台地址，金额必须等于订单金额
		if recharge.UserBscAddress == "" {
			logger.Error("订单缺少用户BSC地址，无法验证，订单号：", recharge.OrderNumber)
			continue
		}
		// 传入当前批次已使用的hash集合，避免同一批次内重复使用
		txHash, isPaid := h.verifyBSCTransactionWithMoralisWalletHistory(recharge.BscAddress, recharge.UserBscAddress, recharge.UsdtAmount, currentBatchUsedHashes)

		if isPaid {
			// 如果已付款，则更新状态并处理算力分配
			err := h.confirmPowerPoolRecharge(recharge, txHash)
			if err != nil {
				logger.Error("确认充值订单失败：", err)
				continue
			}
			// 将本次使用的hash添加到当前批次集合中，避免后续订单重复使用
			if txHash != "" {
				currentBatchUsedHashes[strings.ToLower(txHash)] = true
				logger.Info(fmt.Sprintf("已将交易hash %s 添加到当前批次已使用集合", txHash))
			}
			logger.Info(fmt.Sprintf("充值订单已确认并完成：%s，交易hash：%s", recharge.OrderNumber, txHash))
		} else {
			// 如果未付款，检查是否超时，超时则更改状态为已取消
			if h.isRechargeTimeout(recharge) {
				err := h.cancelPowerPoolRecharge(recharge)
				if err != nil {
					logger.Error("取消充值订单失败：", err)
					continue
				}
				logger.Info("充值订单已超时取消：", recharge.OrderNumber)
			}
		}
	}

	resp.SUCCESS(c, gin.H{
		"processed_count": len(recharges),
		"message":         "充值订单支付状态检查完成",
	})
}

func (h *TradeHandler) confirmPowerPoolRecharge(recharge model.PowerPoolRecharge, txHash string) error {
	// 先在数据库事务中完成内部账务处理
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// 更新充值订单状态，并保存交易hash
		res := tx.Model(&recharge).Updates(map[string]interface{}{
			"status":      2,      // 已支付
			"tx_hash":     txHash, // 保存交易hash
			"pay_time":    time.Now().Unix(),
			"finish_time": time.Now().Unix(),
		})
		if res.Error != nil {
			return res.Error
		}

		// 获取算力池信息
		var pool model.PowerPool
		if err := tx.First(&pool, 1).Error; err != nil {
			return err
		}

		// 计算分配金额
		// 70% 进入算力池USDT
		poolUsdtIncrease := recharge.UsdtAmount * 0.70
		// 25% 进入回购池
		buybackUsdtIncrease := recharge.UsdtAmount * 0.25
		// 5% 平台抽取（不需要单独存储，直接扣除）

		// 更新算力池：算力减少，USDT增加
		updateRes := tx.Model(&pool).Updates(map[string]interface{}{
			"total_power":  gorm.Expr("total_power - ?", recharge.PowerAmount),
			"total_usdt":   gorm.Expr("total_usdt + ?", poolUsdtIncrease),
			"buyback_usdt": gorm.Expr("buyback_usdt + ?", buybackUsdtIncrease),
		})
		if updateRes.Error != nil {
			return updateRes.Error
		}

		// 给用户增加算力
		var user model.User
		if err := tx.First(&user, recharge.UserId).Error; err != nil {
			return err
		}

		updateUserRes := tx.Model(&user).UpdateColumn("power", gorm.Expr("power + ?", recharge.PowerAmount))
		if updateUserRes.Error != nil {
			return updateUserRes.Error
		}

		// 重新获取用户信息以获取最新余额
		if err := tx.First(&user, recharge.UserId).Error; err != nil {
			return err
		}

		// 记录算力日志
		if err := tx.Create(&model.PowerLog{
			UserId:    user.Id,
			Username:  user.Username,
			Type:      types.PowerDh, // 使用算力兑换类型
			Amount:    recharge.PowerAmount,
			Balance:   user.Power,
			Mark:      types.PowerAdd,
			Model:     "算力",
			Remark:    fmt.Sprintf("算力池充值，购买算力：%.2f，支付USDT：%.2f，订单号：%s", recharge.PowerAmount, recharge.UsdtAmount, recharge.OrderNumber),
			CreatedAt: time.Now(),
			OrderNo:   recharge.OrderNumber,
		}).Error; err != nil {
			return err
		}

		// 更新订单状态为已完成
		return tx.Model(&recharge).Update("status", 3).Error
	})
	if err != nil {
		return err
	}

	// 事务成功后再做链上分账，避免因为链上失败影响主业务
	//注释掉分账功能
	// if h.BscService != nil && h.BscService.Enabled() {
	// 	go func(recharge model.PowerPoolRecharge) {
	// 		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	// 		defer cancel()
	// 		if err := h.BscService.DistributeUSDT(ctx, recharge.UsdtAmount, recharge.OrderNumber); err != nil {
	// 			logger.Error("BSC USDT 分账失败：", err)
	// 		}
	// 	}(recharge)
	// }

	return nil
}

// isRechargeTimeout 检查充值订单是否超时
func (h *TradeHandler) isRechargeTimeout(recharge model.PowerPoolRecharge) bool {
	// 假设充值超时时间为2小时
	timeout := 2 * time.Hour
	return time.Since(recharge.CreatedAt) > timeout
}

// cancelPowerPoolRecharge 取消充值订单
func (h *TradeHandler) cancelPowerPoolRecharge(recharge model.PowerPoolRecharge) error {
	return h.DB.Model(&recharge).Update("status", 4).Error
}

// PowerPoolWithdraw 算力提现（从回购池出 USDT，用户扣减算力；5折：0.0005U/算力）
// 说明：历史上前端/路由使用了 buyback/withdraw，但这里本质是“提现卖算力”，不是“用USDT买回算力”的回购。
func (h *TradeHandler) PowerPoolWithdraw(c *gin.Context) {
	var data struct {
		UsdtAmount float64 `json:"usdt_amount"` // 申请提现的USDT数量
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	if data.UsdtAmount <= 0 {
		resp.ERROR(c, "请输入有效的提现金额")
		return
	}

	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	// 检查用户是否设置了BSC地址
	if user.EthAddress == "" {
		resp.ERROR(c, "请先设置BSC付款地址")
		return
	}

	// 获取算力池信息
	var pool model.PowerPool
	result := h.DB.Where("id = ?", 1).FirstOrCreate(&pool, model.PowerPool{
		BaseModel:   model.BaseModel{Id: 1},
		TotalPower:  0,
		TotalUsdt:   0,
		BuybackUsdt: 0,
	})
	if result.Error != nil {
		resp.ERROR(c, "获取回购池信息失败: "+result.Error.Error())
		return
	}

	// 检查回购池是否有足够的USDT（但不立即扣除，等审核通过后再扣除）
	if pool.BuybackUsdt < data.UsdtAmount {
		resp.ERROR(c, fmt.Sprintf("回购池USDT不足，当前可用：%.2f", pool.BuybackUsdt))
		return
	}

	// 提现价格：5折（0.005U/算力）
	// 算力 = USDT / 0.005
	withdrawPrice := 0.005
	requiredPower := data.UsdtAmount / withdrawPrice

	// 检查用户算力余额是否足够
	if user.Power < requiredPower {
		resp.ERROR(c, fmt.Sprintf("算力余额不足，需要提现算力：%.2f，当前可用算力：%.2f", requiredPower, user.Power))
		return
	}

	// 生成提现订单号：WITHDRAW + 时间戳 + 用户ID
	orderNumber := fmt.Sprintf("WITHDRAW%d%d", time.Now().Unix(), user.Id)

	var updatedUser model.User

	// 使用事务处理提现申请（扣减算力 + 生成订单，等待管理员审核打款）
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		// 在事务中再次查询用户算力余额，防止并发问题
		var currentUser model.User
		if err := tx.First(&currentUser, user.Id).Error; err != nil {
			return fmt.Errorf("获取用户信息失败: %v", err)
		}

		// 再次校验算力余额是否足够
		if currentUser.Power < requiredPower {
			return fmt.Errorf("算力余额不足，当前可用算力：%.2f", currentUser.Power)
		}

		// 扣除用户算力（使用乐观锁，确保余额足够才扣减）
		updateResult := tx.Model(&model.User{}).
			Where("id = ? AND power >= ?", user.Id, requiredPower).
			UpdateColumn("power", gorm.Expr("power - ?", requiredPower))

		if updateResult.Error != nil {
			return updateResult.Error
		}

		// 检查是否成功扣减（如果受影响行数为0，说明余额不足）
		if updateResult.RowsAffected == 0 {
			return fmt.Errorf("算力余额不足，无法完成提现")
		}

		// 获取更新后的用户信息
		err = tx.First(&updatedUser, user.Id).Error
		if err != nil {
			return err
		}

		// 创建提现订单（沿用 Buyback 表，管理员审核后从回购池出 USDT）
		buybackOrder := model.Buyback{
			OrderNumber: orderNumber,
			UserId:      user.Id,
			UserName:    user.Username,
			Number:      requiredPower, // 提现扣减的算力数量
			Price:       withdrawPrice, // 提现单价（5折）
			CreateTime:  time.Now().Unix(),
			Status:      1,               // 待处理（等待管理员审核）
			Types:       1,               // 卖单（用户卖出算力换USDT）
			TotalPrice:  data.UsdtAmount, // 提现总金额（USDT）
			EthAddress:  user.EthAddress, // BSC收款地址
		}

		if err := tx.Create(&buybackOrder).Error; err != nil {
			return err
		}

		// 记录算力日志（关联订单号）
		err = tx.Create(&model.PowerLog{
			UserId:    user.Id,
			Username:  user.Username,
			Type:      types.PowerBuyback,
			Amount:    requiredPower,
			Mark:      types.PowerSub,
			Balance:   updatedUser.Power,
			OrderNo:   orderNumber,
			Model:     "算力提现",
			Remark:    fmt.Sprintf("算力提现，扣减算力：%.2f，提现金额：%.2f美金，BSC地址：%s，订单号：%s", requiredPower, data.UsdtAmount, user.EthAddress, orderNumber),
			CreatedAt: time.Now(),
		}).Error
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		// 如果是算力余额不足的错误，返回具体错误信息
		if strings.Contains(err.Error(), "算力余额不足") {
			resp.ERROR(c, err.Error())
		} else {
			resp.ERROR(c, "提现申请失败："+err.Error())
		}
		return
	}

	resp.SUCCESS(c, gin.H{
		"order_number": orderNumber,
		"usdt_amount":  data.UsdtAmount,
		"power":        requiredPower,
		"new_power":    updatedUser.Power,
		"message":      "提现申请提交成功，等待管理员审核",
	})
}

// AuditBuybackList 审核用户查看算力提现订单列表
func (h *TradeHandler) AuditBuybackList(c *gin.Context) {
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	// 检查用户是否为审核用户
	if !user.IsAuditor {
		resp.ERROR(c, "您没有审核权限")
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

// AuditBuybackConfirm 审核用户确认回购订单已打款
func (h *TradeHandler) AuditBuybackConfirm(c *gin.Context) {
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	// 检查用户是否为审核用户
	if !user.IsAuditor {
		resp.ERROR(c, "您没有审核权限")
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
	err = h.DB.Transaction(func(tx *gorm.DB) error {
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

// TestBscTransfer 测试 BSC USDT 分账功能
// 访问方式：GET /api/trade/testBscTransfer
// 功能：从平台收款地址分别转 0.7u、0.25u、0.05u 到三个配置的地址
func (h *TradeHandler) TestBscTransfer(c *gin.Context) {
	if h.BscService == nil || !h.BscService.Enabled() {
		resp.ERROR(c, "BSC USDT 分账服务未启用或配置不完整")
		return
	}

	// 测试金额：1 USDT（会自动分成 0.7u + 0.25u + 0.05u）
	testAmount := 1.0
	testOrderNo := fmt.Sprintf("TEST_%d", time.Now().Unix())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	err := h.BscService.DistributeUSDT(ctx, testAmount, testOrderNo)
	if err != nil {
		logger.Error("BSC USDT 测试转账失败：", err)
		resp.ERROR(c, fmt.Sprintf("转账失败：%v", err))
		return
	}

	resp.SUCCESS(c, gin.H{
		"message":       "BSC USDT 测试转账已提交",
		"test_order_no": testOrderNo,
		"total_amount":  testAmount,
		"distribution": gin.H{
			"power_pool_70": 0.7,
			"buyback_25":    0.25,
			"platform_5":    0.05,
		},
		"note": "请查看日志获取交易哈希，或在区块链浏览器中查询平台收款地址的交易记录",
	})
}
