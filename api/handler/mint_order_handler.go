package handler

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/service"
	"chatplus/service/payment"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils"
	"chatplus/utils/resp"
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

//var logger = logger2.GetLogger()

type MintOrderHandler struct {
	BaseHandler
	paymentHandler *PaymentHandler
	creditService  *service.CreditService
	alipayService  *payment.AlipayService
	huPiPayService *payment.HuPiPayService
	js             *payment.PayJS
	snowflake      *service.Snowflake
	fs             embed.FS
	lock           sync.Mutex
}

func NewMintOrderHandler(app *core.AppServer, db *gorm.DB, paymentHandler *PaymentHandler, creditService *service.CreditService) *MintOrderHandler {
	return &MintOrderHandler{
		BaseHandler:    BaseHandler{App: app, DB: db},
		paymentHandler: paymentHandler,
		creditService:  creditService,
	}
}

// pc 界面
func (h *MintOrderHandler) List(c *gin.Context) {
	var data struct {
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	userId := h.GetLoginUserId(c)
	session := h.DB.Session(&gorm.Session{}).Where("user_id = ? AND status = ?", userId, types.OrderZyz)
	var total int64
	session.Model(&model.MintOrder{}).Count(&total)
	var items []model.MintOrder
	var list = make([]vo.MintOrder, 0)
	offset := (data.Page - 1) * data.PageSize
	res := session.Order("id DESC").Offset(offset).Limit(data.PageSize).Find(&items)
	if res.Error == nil {
		for _, item := range items {
			var order vo.MintOrder
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

// 获取正在挖矿的矿机
func (h *MintOrderHandler) GetMintOrderList(c *gin.Context) {

	userId := h.GetLoginUserId(c)
	var items []model.MintOrder
	var list = make([]vo.MintOrder, 0)
	logger.Info("userId--->", userId)
	res := h.DB.Where("user_id = ?  And  status=8 ", userId).Find(&items)
	if res.Error == nil {
		for _, item := range items {
			var order vo.MintOrder
			err := utils.CopyObject(item, &order)
			if err == nil {
				order.Id = item.Id
				order.Subject = item.Subject
				order.Amount = item.Amount
				order.Power = item.Power
				// 已产出以流水合计为准，并回写校正 js_coin（避免累计产出小于明细）
				order.JsCoin = service.SyncMintOrderJsCoin(h.DB, userId, item.OrderNo, item.JsCoin)
				order.Fee = item.Fee

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

// 购买挖矿套餐
func (h *MintOrderHandler) BuyKj(c *gin.Context) {

	var data struct {
		MintId uint `json:"mint_id"`
		UserId int  `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	loginUserId := h.GetLoginUserId(c)
	if loginUserId == 0 {
		resp.NotAuth(c)
		return
	}
	if data.UserId != int(loginUserId) {
		resp.ERROR(c, "无效的用户id")
		return
	}

	var product model.Mint
	res := h.DB.First(&product, data.MintId)
	if res.Error != nil {
		resp.ERROR(c, "算力机不存在")
		return
	}

	//判断用户算力余额
	var user model.User
	res = h.DB.First(&user, data.UserId)
	if res.Error != nil {
		resp.ERROR(c, "无效的用户id")
		return
	}

	if user.Power < product.Power {
		resp.ERROR(c, "你当前可用算力余额不足，请找代理充值后再操作")
		return
	}

	if ok, msg := CanBuyMint(h.DB, h.App.SysConfig, &user); !ok {
		resp.ERROR(c, msg)
		return
	}

	orderNo := utils.GenerateUniqueID()
	logger.Info(orderNo)
	//if err != nil {
	//	resp.ERROR(c, "error with generate trade no: "+err.Error())
	//	return
	//}

	//同类型的产品在质押中状态只能投一个
	//	var items []model.MintOrder

	//前端按钮也要变成灰色
	//h.DB.Where("user_id = ? AND mint_id = ? And  status=8 ", data.UserId, data.MintId).Find(&items)
	//logger.Info("当前用户已经投入的产品为", items)
	//if len(items) != 0 {
	//	resp.ERROR(c, "你已经有投入了该产品，可等释放后再次投入或投入其他类型产品")
	//	return
	//}

	if res.Error != nil {
		resp.ERROR(c, "无效的用户id")
		return
	}

	// 创建订单
	remark := types.OrderRemark{
		Days:  product.Days,
		Power: product.Power,
		Name:  product.Name,
	}

	order := model.MintOrder{
		UserId:   user.Id,
		Username: user.Username,
		MintId:   product.Id,
		OrderNo:  orderNo,
		Subject:  product.Name,
		Status:   types.OrderZyz,
		Power:    product.Power, //消耗算力
		Fee:      product.Fee,
		JsCoin:   0.0,
		NowCoin:  product.Power * product.Fee, //每天不固定产出值
		Remark:   utils.JsonEncode(remark),
	}
	res = h.DB.Create(&order)
	if res.Error != nil || res.RowsAffected == 0 {
		logger.Info(res.Error)
		resp.ERROR(c, "error with create order: "+res.Error.Error())
		return
	}
	//扣减用户可用算力
	h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumn("power", gorm.Expr("power - ?", product.Power))

	//修改log bug
	var orderUser model.User
	res = h.DB.First(&orderUser, order.UserId)

	//更新流水
	h.DB.Create(&model.PowerLog{
		UserId:    user.Id,
		Username:  user.Username,
		Type:      types.PowerWk,
		Model:     "算力机",
		Amount:    product.Power,
		Balance:   orderUser.Power,
		Mark:      types.PowerSub,
		Remark:    fmt.Sprintf("用户购买算力机：%s套餐,花费算力：%f", product.Name, order.Power),
		CreatedAt: time.Now(),
	})

	if h.paymentHandler != nil {
		h.paymentHandler.ProcessMintPurchaseRewards(user, product.Power, product.Name)
	}

	// 累加该算力机已认购台数
	_ = h.DB.Model(&model.Mint{}).Where("id = ?", product.Id).
		UpdateColumn("sales", gorm.Expr("sales + ?", 1)).Error

	resp.SUCCESS(c)
}

// parseStaticReferralOrderDeadline 保留兼容旧配置解析（推荐奖静态已改为按 zj_static_pro 全量生效）
func parseStaticReferralOrderDeadline(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, time.Local); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// releaseStaticFollowRewardFromOrders 从收奖人在投订单中加速释放静态跟随奖励（推荐奖静态 / 管理奖静态）
func (h *MintOrderHandler) releaseStaticFollowRewardFromOrders(tx *gorm.DB, superiorId uint, totalRewardPower float64, fromUsername string, rewardLabel string, logModel string, fromOrderNo string, pro float64) (float64, error) {
	if totalRewardPower <= 0 {
		return 0, nil
	}
	if strings.TrimSpace(logModel) == "" {
		logModel = "推荐奖静态释放"
	}

	var superior model.User
	if err := tx.First(&superior, superiorId).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}

	var superiorOrders []model.Order
	if err := tx.
		Where("user_id = ? AND status = 8 AND now_power > 0", superiorId).
		Order("id ASC").
		Find(&superiorOrders).Error; err != nil {
		return 0, err
	}
	if len(superiorOrders) == 0 {
		return 0, nil
	}

	remaining := totalRewardPower
	releasedTotal := 0.0
	now := time.Now()

	for _, od := range superiorOrders {
		if remaining <= 0 {
			break
		}

		canRelease := od.NowPower
		if left := od.Power - od.JsPower; left < canRelease {
			canRelease = left
		}
		if canRelease <= 0 {
			continue
		}

		release := remaining
		if release > canRelease {
			release = canRelease
		}
		if release <= 0 {
			continue
		}

		if err := tx.Model(&model.Order{}).
			Where("id = ? AND status = 8", od.Id).
			UpdateColumn("js_power", gorm.Expr("js_power + ?", release)).Error; err != nil {
			return releasedTotal, err
		}
		if err := tx.Model(&model.Order{}).
			Where("id = ? AND status = 8", od.Id).
			UpdateColumn("now_power", gorm.Expr("now_power - ?", release)).Error; err != nil {
			return releasedTotal, err
		}
		if od.JsPower+release >= od.Power || od.NowPower-release <= 0 {
			if err := tx.Model(&model.Order{}).
				Where("id = ?", od.Id).
				UpdateColumn("status", 9).Error; err != nil {
				return releasedTotal, err
			}
		}

		if err := tx.Model(&model.User{}).
			Where("id = ?", superiorId).
			UpdateColumn("power", gorm.Expr("power + ?", release)).Error; err != nil {
			return releasedTotal, err
		}

		if err := tx.First(&superior, superiorId).Error; err != nil {
			return releasedTotal, err
		}

		sourceRelease := 0.0
		if pro > 0 {
			sourceRelease = totalRewardPower / pro
		}
		if err := tx.Create(&model.PowerLog{
			UserId:    superiorId,
			Username:  superior.Username,
			Type:      types.PowerJT,
			Amount:    release,
			Balance:   superior.Power,
			Mark:      types.PowerAdd,
			Model:     logModel,
			Remark:    fmt.Sprintf("%s：来自用户 %s（订单号：%s）静态释放 %.4f，比例 %.4f，本次奖励释放 %.4f", rewardLabel, fromUsername, fromOrderNo, sourceRelease, pro, release),
			CreatedAt: now,
			OrderNo:   od.OrderNo,
		}).Error; err != nil {
			return releasedTotal, err
		}
		if h.creditService != nil {
			if _, err := h.creditService.RepayFromIncome(tx, superiorId, superior.Username, release, od.OrderNo); err != nil {
				return releasedTotal, err
			}
		}

		releasedTotal += release
		remaining -= release
	}

	return releasedTotal, nil
}

// releaseStaticReferralRewardFromOrders 兼容旧调用：推荐奖静态释放
func (h *MintOrderHandler) releaseStaticReferralRewardFromOrders(tx *gorm.DB, superiorId uint, totalRewardPower float64, fromUsername string, rewardLabel string, fromOrderNo string, pro float64) (float64, error) {
	return h.releaseStaticFollowRewardFromOrders(tx, superiorId, totalRewardPower, fromUsername, rewardLabel, "推荐奖静态释放", fromOrderNo, pro)
}

// findTeamRewardReceiverIDsOnDay 查找指定自然日内因该投入人获得「团队奖励释放」的收奖人（用于管理奖静态）
func (h *MintOrderHandler) findTeamRewardReceiverIDsOnDay(tx *gorm.DB, investorUserID uint, day time.Time) ([]uint, error) {
	dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	idLike := fmt.Sprintf("%%下级用户ID：%d%%", investorUserID)
	var ids []uint
	err := tx.Model(&model.PowerLog{}).
		Where("model = ? AND mark = ? AND created_at >= ? AND created_at < ? AND deleted_at IS NULL", "团队奖励释放", types.PowerAdd, dayStart, dayEnd).
		Where("remark LIKE ?", idLike).
		Distinct("user_id").
		Pluck("user_id", &ids).Error
	return ids, err
}

// getKjReleaseRate 根据矿机订单存续天数获取 Q2 日产出比例（与静态套餐释放分段独立）
func (h *MintOrderHandler) getKjReleaseRate(daysSinceCreation int) float64 {
	return h.App.SysConfig.KjReleaseRateByDays(daysSinceCreation)
}

// 解锁算力 静态释放
func (h *MintOrderHandler) CalcSuanli(c *gin.Context) {

	key := c.Query("key")

	if key != "2215be7e811d77a5" {
		resp.ERROR(c, "非法访问")
		return
	}

	var order []model.Order

	var timeOut = h.App.SysConfig.OrderPayTimeout

	res := h.DB.Raw(`
		SELECT od.* 
		FROM chatgpt_orders AS od 
		LEFT JOIN chatgpt_users AS us ON us.id = od.user_id 
		WHERE us.status = 1 
		AND TIMESTAMPDIFF(MINUTE, od.created_at, NOW()) > ? 
		AND od.status = 8`, timeOut).Find(&order)

	log.Printf("准备发放的收益订单数量为: %d\n", len(order))

	//添加一个异常判断处理
	if res.Error != nil {
		resp.ERROR(c, res.Error.Error())
		return
	}

	var skippedToday, skippedDone, skippedZero, releasedOK, failedCnt int

	// 计算所有mintorder收益
	for _, orderItem := range order {

		if err := h.DB.Transaction(func(tx *gorm.DB) error {
			// 按主键读取最新订单状态（避免 order_no 为空时 First 命中错误订单）
			var od model.Order
			if err := tx.Where("id = ? AND status = 8", orderItem.Id).First(&od).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return nil
				}
				return err
			}

			// 检查订单是否已经释放完毕（已释放算力 >= 预产值）
			if od.JsPower >= od.Power || od.NowPower <= 0 {
				skippedDone++
				return nil
			}

			// 当天已释放过则跳过（必须 order_no 非空，避免空 order_no 误判把全部订单跳过）
			now := time.Now()
			dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			dayEnd := dayStart.Add(24 * time.Hour)
			if ono := strings.TrimSpace(od.OrderNo); ono != "" {
				var todayReleased int64
				if err := tx.Model(&model.PowerLog{}).
					Where("order_no = ? AND model = ? AND created_at >= ? AND created_at < ? AND deleted_at IS NULL",
						ono, "静态收益释放", dayStart, dayEnd).
					Count(&todayReleased).Error; err != nil {
					return err
				}
				if todayReleased > 0 {
					skippedToday++
					return nil
				}
			}

			// 根据订单创建时间计算已经经过的天数
			orderCreatedAt := od.CreatedAt
			daysSinceCreation := int(now.Sub(orderCreatedAt).Hours() / 24)

			// 根据经过天数选择对应的释放比例（可在后台动态配置）
			releaseRate := h.App.SysConfig.StaticReleaseRateByDays(daysSinceCreation)

			// 计算每天应该释放的算力 = 投入金额 * 动态释放比例
			dailyReleasePower := od.Sales * releaseRate

			// 剩余可释放：预产值 - 已释放，同时不超过 now_power
			remainingPower := od.Power - od.JsPower
			actualReleasePower := dailyReleasePower
			if actualReleasePower > remainingPower {
				actualReleasePower = remainingPower
			}
			if actualReleasePower > od.NowPower {
				actualReleasePower = od.NowPower
			}
			if actualReleasePower <= 0 {
				skippedZero++
				return nil
			}

			// 更新订单（原子递增/递减）
			if err := tx.Model(&model.Order{}).
				Where("id = ? AND status = 8", od.Id).
				UpdateColumn("js_power", gorm.Expr("js_power + ?", actualReleasePower)).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.Order{}).
				Where("id = ? AND status = 8", od.Id).
				UpdateColumn("now_power", gorm.Expr("now_power - ?", actualReleasePower)).Error; err != nil {
				return err
			}
			if od.JsPower+actualReleasePower >= od.Power || od.NowPower-actualReleasePower <= 0 {
				if err := tx.Model(&model.Order{}).
					Where("id = ?", od.Id).
					UpdateColumn("status", 9).Error; err != nil {
					return err
				}
			}

			// 增加用户可用算力
			if err := tx.Model(&model.User{}).
				Where("id = ?", od.UserId).
				UpdateColumn("power", gorm.Expr("power + ?", actualReleasePower)).Error; err != nil {
				return err
			}

			// 获取用户信息（用于日志 + 取上级）
			var user model.User
			if err := tx.First(&user, od.UserId).Error; err != nil {
				return err
			}

			// 记录本人的静态释放流水
			if err := tx.Create(&model.PowerLog{
				UserId:    od.UserId,
				Username:  user.Username,
				Type:      types.PowerSf,
				Amount:    actualReleasePower,
				Balance:   user.Power,
				Mark:      types.PowerAdd,
				Model:     "静态收益释放",
				Remark:    fmt.Sprintf("订单静态收益释放算力：%f，投入金额：%f，预产值：%f，已释放：%f，用户名：%s", actualReleasePower, od.Sales, od.Power, od.JsPower+actualReleasePower, user.Username),
				CreatedAt: now,
				OrderNo:   od.OrderNo,
			}).Error; err != nil {
				return err
			}
			// 有授信欠款时：先入账再立刻扣还款（不回补可用授信）
			if h.creditService != nil {
				if _, err := h.creditService.RepayFromIncome(tx, od.UserId, user.Username, actualReleasePower, od.OrderNo); err != nil {
					return err
				}
			}

			// 推荐奖静态释放：直推上级跟随下级每晚静态释放（默认 20%，后台 zj_static_pro），原有静态/动态不变
			if user.ZjFid > 0 && h.App.SysConfig.ZjStaticPro > 0 {
				rewardPower := actualReleasePower * h.App.SysConfig.ZjStaticPro
				if _, err := h.releaseStaticFollowRewardFromOrders(tx, uint(user.ZjFid), rewardPower, user.Username, "推荐奖静态", "推荐奖静态释放", od.OrderNo, h.App.SysConfig.ZjStaticPro); err != nil {
					return err
				}
			}
			// 间推静态（可选，后台配置 jj_static_pro>0 时生效）
			if user.JjFid > 0 && h.App.SysConfig.JjStaticPro > 0 {
				rewardPower := actualReleasePower * h.App.SysConfig.JjStaticPro
				if _, err := h.releaseStaticFollowRewardFromOrders(tx, uint(user.JjFid), rewardPower, user.Username, "间推静态奖励", "推荐奖静态释放", od.OrderNo, h.App.SysConfig.JjStaticPro); err != nil {
					return err
				}
			}

			// 管理奖静态释放：投入日因该人获得团队奖的收奖人，跟随该投入订单每晚静态释放（默认 10%）
			if h.App.SysConfig.MgmtStaticPro > 0 {
				receiverIDs, err := h.findTeamRewardReceiverIDsOnDay(tx, od.UserId, od.CreatedAt)
				if err != nil {
					return err
				}
				for _, receiverID := range receiverIDs {
					if receiverID == 0 || receiverID == od.UserId {
						continue
					}
					rewardPower := actualReleasePower * h.App.SysConfig.MgmtStaticPro
					if _, err := h.releaseStaticFollowRewardFromOrders(tx, receiverID, rewardPower, user.Username, "管理奖静态", "管理奖静态释放", od.OrderNo, h.App.SysConfig.MgmtStaticPro); err != nil {
						return err
					}
				}
			}

			releasedOK++
			return nil
		}); err != nil {
			failedCnt++
			logger.Error("CalcSuanli: 处理订单失败，订单号:", orderItem.OrderNo, "err:", err)
			continue
		}

		//计算当前应该释放的静态收益比例

	}

	log.Printf("CalcSuanli完成: 候选=%d, 今日已释放跳过=%d, 已完成跳过=%d, 释放额为0跳过=%d, 成功=%d, 失败=%d\n",
		len(order), skippedToday, skippedDone, skippedZero, releasedOK, failedCnt)

	resp.SUCCESS(c)
}

// 挖矿 解锁coin
func (h *MintOrderHandler) CalcJsCoin(c *gin.Context) {

	key := c.Query("key")

	if key != "2215be7e811d77a5" {
		resp.ERROR(c, "非法访问")
		return
	}

	var mintOrders []model.MintOrder
	//获取所有大于后台配置时间的订单
	var timeOut = h.App.SysConfig.OrderPayTimeout

	//res := h.DB.Raw("SELECT *  FROM chatgpt_mint_orders WHERE TIMESTAMPDIFF(MINUTE, created_at, NOW()) > 1440").Find(&mintOrders)
	res := h.DB.Raw("SELECT * FROM chatgpt_mint_orders WHERE status = ? AND TIMESTAMPDIFF(MINUTE, created_at, NOW()) > ?",
		types.OrderZyz, timeOut).Find(&mintOrders)

	//添加一个异常判断处理
	if res.Error != nil {
		resp.ERROR(c, res.Error.Error())
		return
	}

	//计算所有mintorder收益
	for _, order := range mintOrders {

		// 验证当前用户是否被禁用
		var currentUser model.User
		h.DB.Model(&model.User{}).Where("id = ?  ", order.UserId).Find(&currentUser)

		//禁用无coin释放
		if !currentUser.Status {
			logger.Infof("当前用户 %s 已经被禁用，已跳过到下一个用户", order.Username)
			continue
		}

		// 根据矿机订单存续天数获取分段产出比例（后台 kj_* 配置，与静态 zj_plus 独立）
		daysSinceCreation := int(time.Since(order.CreatedAt).Hours() / 24)
		releaseRate := h.getKjReleaseRate(daysSinceCreation)
		if releaseRate <= 0 {
			// 未配置矿机分段比例时，回退到订单保存的费率
			releaseRate = order.Fee
		}
		if releaseRate <= 0 {
			continue
		}
		jsCoin := order.Power * releaseRate

		err := h.DB.Transaction(func(tx *gorm.DB) error {
			return service.RecordMintCoinOutputInTx(tx, order, jsCoin)
		})
		if err != nil {
			logger.Errorf("矿机产出SOHA失败 order=%s user=%s err=%v", order.OrderNo, order.Username, err)
		}

	}

	resp.SUCCESS(c)
}

// 每日免费算力分配  此方法作废
func (h *MintOrderHandler) Days(c *gin.Context) {

	key := c.Query("key")

	if key != "036f482b68b748f9" {
		resp.ERROR(c, "非法访问")
		return
	}

	logger.Info("今日算力派发开始：", time.Now())
	var users []model.User
	res := h.DB.Where("status", 1).Find(&users)

	logger.Info("users列表为", users)
	if res.Error != nil {
		resp.ERROR(c, "No matching users")
		return
	}

	var sysConfig model.Config
	res = h.DB.Where("marker", "system").First(&sysConfig)
	if res.Error != nil {

		resp.ERROR(c, "error with get system config:", res.Error.Error())
		return
	}

	var config types.SystemConfig
	err := utils.JsonDecode(sysConfig.Config, &config)
	if err != nil {
		resp.ERROR(c, "error with decode system config: ", res.Error.Error())
		return
	}

	logger.Info("config.DailyPower为", config.DailyPower)

	if config.DailyPower <= 0 {

		return
		//return "success"
	}

	var counter = 0
	var totalPower = 0.0
	for _, u := range users {
		if u.Power >= config.DailyPower {
			continue
		}
		var power = config.DailyPower - u.Power
		// update user
		tx := h.DB.Model(&model.User{}).Where("id", u.Id).UpdateColumn("power", gorm.Expr("power + ?", power))
		// 记录算力充值日志
		if tx.Error == nil {
			var user model.User
			h.DB.Where("id", u.Id).First(&user)
			h.DB.Create(&model.PowerLog{
				UserId:    u.Id,
				Username:  u.Username,
				Type:      types.PowerGift,
				Amount:    power,
				Mark:      types.PowerAdd,
				Balance:   user.Power,
				Model:     "系统赠送",
				Remark:    fmt.Sprintf("系统每日算力派发，今日额度：%f", config.DailyPower),
				CreatedAt: time.Now(),
			})
		}
		counter++
		totalPower += power
	}
	logger.Infof("今日派发算力结束！累计派发 %d 人，累计派发算力：%f", counter, totalPower)

	resp.SUCCESS(c)
}

// 24小时分配用户的 update time
func (h *MintOrderHandler) UpdateUT(c *gin.Context) {

	key := c.Query("key")

	if key != "036f482b68b748f9" {
		resp.ERROR(c, "非法访问")
		return
	}

	logger.Info("今日算力派发开始：", time.Now())
	var users []model.User
	res := h.DB.Where("status", 1).Find(&users)

	logger.Info("users列表为", users)
	if res.Error != nil {
		resp.ERROR(c, "No matching users")
		return
	}

	for _, u := range users {

		// update user
		h.DB.Model(&model.User{}).Where("id", u.Id).UpdateColumn("is_partner", 0) //0为待领取 1为1领取

	}
	logger.Infof("今日派发算力结束！累计派发 %d 人，累计派发算力：%f")

	resp.SUCCESS(c)

}

// 定时随机当日 SOHA 指导价涨幅（0.10%~0.50% 复利；首日使用基准价）
func (h *MintOrderHandler) RandomPrise(c *gin.Context) {
	key := c.Query("key")
	if key != "036f4821hjsksr4e" {
		resp.ERROR(c, "非法访问")
		return
	}

	logger.Info("随机价格开始：", time.Now())

	todayDate := TodayDateLocal()
	cfg := h.App.SysConfig

	// 当日已有记录：若 price 未随涨幅变化（legacy 0.0001 四舍五入问题），则重新计算并更新
	var todayRecord model.PriceHistory
	if err := h.DB.Where("DATE(`date`) = DATE(?)", todayDate).First(&todayRecord).Error; err == nil {
		var prevRecord model.PriceHistory
		prevErr := h.DB.Where("DATE(`date`) < DATE(?)", todayDate).Order("date DESC, id DESC").First(&prevRecord).Error

		needFix := IsLegacySohaPrice(todayRecord.Price, cfg)
		if !needFix && prevErr == nil && todayRecord.ChangeRate > 0 {
			effectivePrev := EffectiveSohaPreviousPrice(prevRecord.Price, cfg)
			expected := ApplySohaDailyChange(effectivePrev, todayRecord.ChangeRate)
			if todayRecord.Price <= prevRecord.Price || math.Abs(todayRecord.Price-expected) > SohaPriceMinTick() {
				needFix = true
			}
		}

		if !needFix {
			logger.Info("今日价格已存在且有效，跳过：", todayRecord)
			resp.SUCCESS(c, gin.H{
				"skipped":     true,
				"date":        todayDate,
				"price":       todayRecord.Price,
				"change_rate": todayRecord.ChangeRate,
			})
			return
		}

		effectivePrev := SohaGuideBasePrice(cfg)
		if prevErr == nil {
			effectivePrev = EffectiveSohaPreviousPrice(prevRecord.Price, cfg)
		}
		changePercent := todayRecord.ChangeRate
		if changePercent <= 0 {
			changePercent = RandomSohaDailyChangePercent()
		}
		newPrice := ApplySohaDailyChange(effectivePrev, changePercent)
		changePercent = SohaChangeRateFromPrices(effectivePrev, newPrice)

		if err := h.DB.Model(&todayRecord).Updates(map[string]interface{}{
			"price":       newPrice,
			"change_rate": changePercent,
			"updated_at":  time.Now(),
		}).Error; err != nil {
			logger.Error("修正今日价格失败：", err)
			resp.ERROR(c, "修正价格记录失败")
			return
		}

		logger.Info("修正今日价格：", map[string]interface{}{
			"id":            todayRecord.Id,
			"effectivePrev": effectivePrev,
			"new_price":     newPrice,
			"change_rate":   changePercent,
		})
		resp.SUCCESS(c, gin.H{
			"fixed":       true,
			"date":        todayDate,
			"new_price":   newPrice,
			"change_rate": changePercent,
		})
		return
	}

	var lastPrice model.PriceHistory
	err := h.DB.Order("date DESC, id DESC").First(&lastPrice).Error

	var newPrice, changePercent float64
	var effectivePrev float64
	if err == gorm.ErrRecordNotFound {
		newPrice = SohaGuideBasePrice(cfg)
		changePercent = 0
		logger.Info("首日基准价：", newPrice)
	} else if err != nil {
		logger.Error("获取最新价格记录失败：", err)
		resp.ERROR(c, "获取价格记录失败")
		return
	} else {
		targetRate := RandomSohaDailyChangePercent()
		effectivePrev = EffectiveSohaPreviousPrice(lastPrice.Price, cfg)
		newPrice = ApplySohaDailyChange(effectivePrev, targetRate)
		changePercent = SohaChangeRateFromPrices(effectivePrev, newPrice)
		logger.Info("最新价格", lastPrice.Price, "有效前价", effectivePrev, "涨幅%", changePercent, "新价", newPrice)
	}

	newPriceHistory := model.PriceHistory{
		Date:       todayDate,
		Price:      newPrice,
		ChangeRate: changePercent,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := h.DB.Create(&newPriceHistory).Error; err != nil {
		logger.Error("保存新价格记录失败：", err)
		resp.ERROR(c, "保存价格记录失败")
		return
	}

	logger.Info("随机价格生成成功：", map[string]interface{}{
		"date":        todayDate,
		"new_price":   newPrice,
		"change_rate": changePercent,
	})

	resp.SUCCESS(c, gin.H{
		"date":        todayDate,
		"new_price":   newPrice,
		"change_rate": changePercent,
	})
}

// 获取最近30分钟内的状态为 status 为3的 chatgpt_trades 表中的数据
// 示例数据为 id	order_number	user_id	user_name	number	price	create_time	target_user_id	match_time	image	pay_time	finish_time	status	charge_number	is_comment	types	total_price	total_price_china	created_at	updated_at	eth_address
func (h *MintOrderHandler) TradesPay(c *gin.Context) {
	key := c.Query("key")
	if key != "2215be7e811d77a5" {
		resp.ERROR(c, "非法访问")
		return
	}

	logger.Info("开始检查交易支付状态：", time.Now())

	// 获取最近30分钟内状态为3的交易数据
	thirtyMinutesAgo := time.Now().Add(-30 * time.Minute)
	var trades []model.Trade

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
		isPaid := h.verifyBSCTransactionOptimized(trade.EthAddress, trade.Number*trade.Price)

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

// 优化的BSC交易验证方法11
func (h *MintOrderHandler) verifyBSCTransactionOptimized(address string, expectedAmount float64) bool {
	apiKey := "XT44Q8NA1P4FDIEWPS2HCU5EYT2MWXCIWM"
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

	// 2. 估算30分钟前的区块号（BSC平均3秒一个块，30分钟约600个块）
	startBlock := latestBlock - 600
	if startBlock < 0 {
		startBlock = 0
	}

	logger.Info("区块范围：", startBlock, "到", latestBlock)

	// 3. 使用优化的区块范围查询代币交易
	url := fmt.Sprintf("https://api.bscscan.com/api?module=account&action=tokentx&address=%s&startblock=%d&endblock=%d&sort=desc&apikey=%s",
		address, startBlock, latestBlock, apiKey)

	resp, err := http.Get(url)
	if err != nil {
		logger.Error("BSC API请求失败：", err)
		return false
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("读取响应失败：", err)
		return false
	}

	// 打印BSC返回的原始JSON数据
	logger.Info("BSC API返回的原始数据：", string(body))

	// 解析响应 - 适配tokentx API的返回格式
	var bscResponse struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  []struct {
			BlockNumber       string `json:"blockNumber"`
			TimeStamp         string `json:"timeStamp"`
			Hash              string `json:"hash"`
			Nonce             string `json:"nonce"`
			BlockHash         string `json:"blockHash"`
			From              string `json:"from"`
			ContractAddress   string `json:"contractAddress"`
			To                string `json:"to"`
			Value             string `json:"value"`
			TokenName         string `json:"tokenName"`
			TokenSymbol       string `json:"tokenSymbol"`
			TokenDecimal      string `json:"tokenDecimal"`
			TransactionIndex  string `json:"transactionIndex"`
			Gas               string `json:"gas"`
			GasPrice          string `json:"gasPrice"`
			GasUsed           string `json:"gasUsed"`
			CumulativeGasUsed string `json:"cumulativeGasUsed"`
			Input             string `json:"input"`
			MethodId          string `json:"methodId"`
			FunctionName      string `json:"functionName"`
			Confirmations     string `json:"confirmations"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &bscResponse); err != nil {
		logger.Error("解析BSC响应失败：", err)
		return false
	}

	if bscResponse.Status != "1" {
		logger.Error("BSC API错误：", bscResponse.Message)
		return false
	}

	logger.Info("找到代币交易数量：", len(bscResponse.Result))

	// 4. 在本地进一步按时间戳筛选（确保是30分钟内的交易）
	thirtyMinutesAgo := time.Now().Add(-60 * time.Minute).Unix()

	// BSC USDT 合约地址
	usdtContractAddress := "0x55d398326f99059ff775485246999027b3197955"

	for i, tx := range bscResponse.Result {
		logger.Info(fmt.Sprintf("=== 处理第 %d 个交易 ===", i+1))
		logger.Info("交易哈希：", tx.Hash)
		logger.Info("发送方：", tx.From)
		logger.Info("接收方：", tx.To)
		logger.Info("合约地址：", tx.ContractAddress)
		logger.Info("代币名称：", tx.TokenName)
		logger.Info("代币符号：", tx.TokenSymbol)
		logger.Info("代币小数位：", tx.TokenDecimal)
		logger.Info("转账金额（原始值）：", tx.Value)
		logger.Info("时间戳：", tx.TimeStamp)
		logger.Info("方法ID：", tx.MethodId)
		logger.Info("函数名：", tx.FunctionName)

		// 检查是否是USDT转账
		if strings.ToLower(tx.ContractAddress) != strings.ToLower(usdtContractAddress) {
			logger.Info("不是USDT转账，跳过")
			continue
		}

		// 检查是否是转入交易
		if strings.ToLower(tx.To) != strings.ToLower(address) {
			logger.Info("不是转入目标地址的交易，跳过")
			continue
		}

		// 检查时间戳（确保是30分钟内）
		timestamp, err := strconv.ParseInt(tx.TimeStamp, 10, 64)
		if err != nil {
			logger.Error("时间戳解析失败：", err)
			continue
		}

		logger.Info("交易时间戳：", timestamp, "，60分钟前时间戳：", thirtyMinutesAgo)
		if timestamp < thirtyMinutesAgo {
			logger.Info("交易时间超过60分钟，跳过")
			continue
		}

		// 检查金额 - 使用tokentx API的value字段
		if h.checkTokenTransactionAmount(tx.Value, tx.TokenDecimal, expectedAmount) {
			logger.Info("✅ 找到匹配的USDT交易：", tx.Hash)
			return true
		} else {
			logger.Info("❌ 交易金额不匹配")
		}
	}

	logger.Info("未找到匹配的交易")
	return false
}

// 检查代币交易金额是否匹配
func (h *MintOrderHandler) checkTokenTransactionAmount(valueStr, decimalStr string, expectedAmount float64) bool {
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

// 获取最新区块号
func (h *MintOrderHandler) getLatestBlockNumber(apiKey string) (int64, error) {
	url := fmt.Sprintf("https://api.bscscan.com/api?module=proxy&action=eth_blockNumber&apikey=%s", apiKey)

	resp, err := http.Get(url)
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
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return 0, err
	}

	// 将十六进制转换为十进制
	blockNumber, err := strconv.ParseInt(strings.TrimPrefix(response.Result, "0x"), 16, 64)
	if err != nil {
		return 0, err
	}

	return blockNumber, nil
}

// 更新交易状态为已付款
func (h *MintOrderHandler) updateTradeStatusToPaid(trade model.Trade) error {
	// 更新交易状态为4（已确认）
	err := h.DB.Model(&trade).Where("id = ?", trade.Id).Updates(model.Trade{
		Status:     4,
		FinishTime: time.Now().Unix(),
	}).Error

	if err != nil {
		return err
	}

	// 获取用户信息
	var buyUser model.User
	var sellUser model.User

	// 处理SOHA币转账
	if trade.Types == 1 {

		// 卖方解冻并扣除SOHA币，买方增加SOHA币
		h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin - ?", trade.Number))
		h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("coin", gorm.Expr("coin - ?", trade.Number))
		h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("coin", gorm.Expr("coin + ?", trade.Number))

		h.DB.First(&buyUser, trade.UserId)
		h.DB.First(&sellUser, trade.TargetUserId)

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

		// 买方解冻并扣除SOHA币，卖方增加SOHA币
		h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin - ?", trade.Number))
		h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("coin", gorm.Expr("coin - ?", trade.Number))
		h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("coin", gorm.Expr("coin + ?", trade.Number))

		h.DB.First(&buyUser, trade.TargetUserId)
		h.DB.First(&sellUser, trade.UserId)

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

	return nil
}

// 检查交易是否超时
func (h *MintOrderHandler) isTradeTimeout(trade model.Trade) bool {
	// 假设交易超时时间为2小时
	timeout := 2 * time.Hour
	return time.Since(trade.CreatedAt) > timeout
}

// 更新交易状态为已取消
func (h *MintOrderHandler) updateTradeStatusToCancelled(trade model.Trade) error {
	// 更新交易状态为5（已取消）
	err := h.DB.Model(&trade).Where("id = ?", trade.Id).Updates(model.Trade{
		Status: 5,
	}).Error

	if err != nil {
		return err
	}

	// 解冻相关用户的SOHA币
	if trade.Types == 1 {
		// 解冻卖方的SOHA币
		h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin - ?", trade.Number))
		h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("coin", gorm.Expr("coin + ?", trade.Number))
	} else if trade.Types == 2 {
		// 解冻买方的SOHA币
		h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin - ?", trade.Number))
		h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("coin", gorm.Expr("coin + ?", trade.Number))
	}

	return nil
}

// 匹配前3位数的方法
func (h *MintOrderHandler) matchFirst3Digits(actualAmount, expectedAmount float64) bool {
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
func (h *MintOrderHandler) getBSCAPIKey() string {
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

// 超过当前时间的两个小时内的订单（即状态未更改为3的情况，超过两个小时还是2 ）则自动取消订单
func (h *MintOrderHandler) AutoCancel(c *gin.Context) {

	key := c.Query("key")
	if key != "2215be7e811d77a5" {
		resp.ERROR(c, "非法访问")
		return
	}

	logger.Info("开始检查超时订单自动取消：", time.Now())

	// 获取自创建时间以来，超过两个小时，但是状态还是为2的订单记录
	twoHoursAgo := time.Now().Add(-2 * time.Hour)
	var trades []model.Trade

	res := h.DB.Where("status = ? AND updated_at <= ?", 2, twoHoursAgo).Find(&trades)
	if res.Error != nil {
		logger.Error("获取超时交易数据失败：", res.Error)
		resp.ERROR(c, "获取超时交易数据失败")
		return
	}

	logger.Info("找到超时未确认交易数量：", len(trades))

	var cancelledCount int
	var errorCount int

	// 循环处理每个超时订单
	for _, trade := range trades {
		logger.Info("处理超时交易ID：", trade.Id, "订单号：", trade.OrderNumber, "类型：", trade.Types)

		// 将状态更改为 5 已取消状态
		err := h.DB.Model(&trade).Where("id = ?", trade.Id).Update("status", 5).Error
		if err != nil {
			logger.Error("更新交易状态失败：", err)
			errorCount++
			continue
		}

		if trade.Types == 1 {

			//解冻
			h.DB.Model(&model.User{}).Where("id = ?", trade.TargetUserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin -?", trade.Number))

		} else if trade.Types == 2 {

			//解冻
			h.DB.Model(&model.User{}).Where("id = ?", trade.UserId).UpdateColumn("dj_coin", gorm.Expr("dj_coin -?", trade.Number))

		} else {
			logger.Error("未知的交易类型：", trade.Types)
			errorCount++
			continue
		}

		cancelledCount++
		logger.Info("订单已超时系統自动取消", trade.OrderNumber, "退还用户ID：", "金额：", trade.Number)
	}

	logger.Info("超时订单处理完成，成功取消：", cancelledCount, "个，失败：", errorCount, "个")

	resp.SUCCESS(c, gin.H{
		"total_checked":   len(trades),
		"cancelled_count": cancelledCount,
		"error_count":     errorCount,
		"message":         "超时订单自动取消处理完成",
	})
}

//增加一个用户级别提升的方法,

func (h *MintOrderHandler) AddLev(c *gin.Context) {
	// 1. 获取所有用户
	var allUsers []model.User
	h.DB.Table("chatgpt_users").Find(&allUsers)
	if len(allUsers) == 0 {
		resp.ERROR(c, "无用户数据")
		return
	}

	var upgraded []uint // 记录升级成功的用户ID

	for _, user := range allUsers {
		userId := user.Id

		// 统计userId在zj_fid、jj_fid、san_fid、si_fid、fids_json中出现的次数
		var count int64
		likeQuery := fmt.Sprintf("[%%,%d,%%]", userId)   // 匹配逗号分隔的 ID
		likeQueryStart := fmt.Sprintf("[%%,%d]", userId) // 匹配开头的 ID
		likeQueryEnd := fmt.Sprintf("[%d,%%]", userId)   // 匹配结尾的 ID
		exactMatch := fmt.Sprintf("[%d]", userId)        // 精确匹配唯一 ID

		h.DB.Table("chatgpt_users").
			Where("zj_fid = ? OR jj_fid = ? OR san_fid = ? OR si_fid = ? OR fids_json LIKE ? OR fids_json LIKE ? OR fids_json LIKE ? OR fids_json = ?",
				userId, userId, userId, userId, likeQuery, likeQueryStart, likeQueryEnd, exactMatch).
			Count(&count)

		if count < 15 {
			continue // 出现次数不足，直接跳过
		}

		// 1. 获取所有直推用户
		var directUsers []model.User
		h.DB.Where("zj_fid = ?", userId).Find(&directUsers)
		if len(directUsers) == 0 {
			continue // 没有直推用户，跳过
		}

		type UserBusiness struct {
			UserId uint
			Total  float64
		}
		var userBizList []UserBusiness

		// 2. 统计每个直推用户的自身套餐业务和10代以内下级业务量
		for _, du := range directUsers {
			// 直推用户自己的套餐一和套餐二业务
			var selfSales float64
			h.DB.Table("chatgpt_orders").Where("user_id = ?", du.Id).Select("IFNULL(SUM(sales),0)").Scan(&selfSales)
			var selfPower float64
			h.DB.Table("chatgpt_mint_orders").Where("user_id = ?", du.Id).Select("IFNULL(SUM(power),0)").Scan(&selfPower)

			// 10代以内下级
			subUserIds := getTenLevelUserIds(h.DB, du.Id)
			var subSalesSum float64
			var subPowerSum float64
			if len(subUserIds) > 0 {
				h.DB.Table("chatgpt_orders").Where("user_id IN ?", subUserIds).Select("IFNULL(SUM(sales),0)").Scan(&subSalesSum)
				h.DB.Table("chatgpt_mint_orders").Where("user_id IN ?", subUserIds).Select("IFNULL(SUM(power),0)").Scan(&subPowerSum)
			}

			total := selfSales + selfPower + subSalesSum + subPowerSum
			userBizList = append(userBizList, UserBusiness{du.Id, total})
		}

		// 3. 去掉最大业务量的直推用户
		if len(userBizList) == 0 {
			continue
		}
		sort.Slice(userBizList, func(i, j int) bool { return userBizList[i].Total > userBizList[j].Total })
		userBizList = userBizList[1:] // 去掉最大

		// 4. 剩余业务量求和
		var total float64
		for _, ub := range userBizList {
			total += ub.Total
		}

		// 5. 判断并更新lev
		if total >= 1000000 && user.Lev < 1 {
			ts := time.Now()
			h.DB.Table("chatgpt_users").Where("id = ?", userId).Updates(map[string]interface{}{
				"lev":          1,
				"lev_since_at": ts,
			})
			upgraded = append(upgraded, userId)
		}
	}

	resp.SUCCESS(c, gin.H{
		"upgraded_count": len(upgraded),
		"upgraded_users": upgraded,
	})
}

//
//// 获取10代以内下级user_id集合
//func getTenLevelUserIds(db *gorm.DB, userId uint) []uint {
//	var ids []uint
//	// 2~4代
//	var users []struct {
//		Id       uint
//		ZjFid    uint
//		JjFid    uint
//		SanFid   uint
//		SiFid    uint
//		FidsJson string
//	}
//	db.Table("chatgpt_users").Where("zj_fid = ? OR jj_fid = ? OR san_fid = ? OR si_fid = ?", userId, userId, userId, userId).Find(&users)
//	for _, u := range users {
//		ids = append(ids, u.Id)
//	}
//	// 5~10代
//	var usersWithFids []struct {
//		Id       uint
//		FidsJson string
//	}
//	db.Table("chatgpt_users").Where("JSON_CONTAINS(fids_json, ?)", fmt.Sprintf("\"%d\"", userId)).Find(&usersWithFids)
//	for _, u := range usersWithFids {
//		var fids []uint
//		_ = json.Unmarshal([]byte(u.FidsJson), &fids)
//		for _, fid := range fids {
//			if fid == userId {
//				ids = append(ids, u.Id)
//				break
//			}
//		}
//	}
//	return ids
//}
//

// 获取10代以内下级user_id集合（纯递归版本，通过zj_fid递归查找10层）
func getTenLevelUserIds(db *gorm.DB, userId uint) []uint {
	return getTenLevelUserIdsRecursive(db, userId, 0)
}

// 递归辅助函数，通过zj_fid递归查找下级，最多查找10层
func getTenLevelUserIdsRecursive(db *gorm.DB, userId uint, depth int) []uint {
	// 如果超过10层，停止递归
	if depth >= 10 {
		return nil
	}

	var ids []uint
	// 查找直接下级（zj_fid = userId的所有用户）
	var subUsers []struct {
		Id uint
	}
	db.Table("chatgpt_users").Where("zj_fid = ?", userId).Select("id").Find(&subUsers)

	// 遍历每个直接下级
	for _, u := range subUsers {
		// 将下级ID加入结果
		ids = append(ids, u.Id)
		// 递归查找下级的下级
		subIds := getTenLevelUserIdsRecursive(db, u.Id, depth+1)
		if len(subIds) > 0 {
			ids = append(ids, subIds...)
		}
	}

	return ids
}

// 获取20代以内下级user_id集
func getTenLevelUserIds2(db *gorm.DB, userId uint) []uint {
	return getTenLevelUserIdsRecursive2(db, userId, 0)
}

// 递归辅助函数，通过zj_fid递归查找下级，最多查找10层
func getTenLevelUserIdsRecursive2(db *gorm.DB, userId uint, depth int) []uint {
	// 如果超过10层，停止递归
	if depth >= 100 {
		return nil
	}

	var ids []uint
	// 查找直接下级（zj_fid = userId的所有用户）
	var subUsers []struct {
		Id uint
	}
	db.Table("chatgpt_users").Where("zj_fid = ?", userId).Select("id").Find(&subUsers)

	// 遍历每个直接下级
	for _, u := range subUsers {
		// 将下级ID加入结果
		ids = append(ids, u.Id)
		// 递归查找下级的下级
		subIds := getTenLevelUserIdsRecursive2(db, u.Id, depth+1)
		if len(subIds) > 0 {
			ids = append(ids, subIds...)
		}
	}

	return ids
}

// 获取10代以内上级 user_id 集合（递归向上，去重防循环）
func getUpIds(db *gorm.DB, startZjFid uint) []uint {
	visited := make(map[uint]bool)
	return getAllFatherIds(db, startZjFid, 0, visited)
}

// 递归辅助函数：沿着 zj_fid 向上查找，最多 10 层；若出现重复则停止
func getAllFatherIds(db *gorm.DB, currentId uint, depth int, visited map[uint]bool) []uint {
	if depth >= 100 || currentId == 0 {
		return nil
	}
	if visited[currentId] {
		// 出现重复（环或数据异常），直接停止向上
		return nil
	}
	visited[currentId] = true

	ids := []uint{currentId}

	// 读取当前用户的直接上级 zj_fid
	var row struct{ ZjFid uint }
	db.Table("chatgpt_users").Select("zj_fid").Where("id = ?", currentId).First(&row)

	if row.ZjFid == 0 {
		return ids
	}

	up := getAllFatherIds(db, row.ZjFid, depth+1, visited)
	if len(up) > 0 {
		ids = append(ids, up...)
	}
	return ids
}

// 这里再增加一个方法，
// 那就是获取用户直推有中两个lev为1的，则当前用户的lev升级为2,
// 用户直推有中两个lev为2的，则当前用户的lev升级为3,
// 用户直推有中两个lev为3的，且其中有一个间推为3的则当前用户的lev升级为4
// 更新用户的等级
func (h *MintOrderHandler) OtherLev(c *gin.Context) {
	var allUsers []model.User
	h.DB.Table("chatgpt_users").Find(&allUsers)
	if len(allUsers) == 0 {
		resp.ERROR(c, "无用户数据")
		return
	}

	var updated []uint

	for _, user := range allUsers {
		userId := user.Id

		// 1. 获取所有直推
		var directUsers []model.User
		h.DB.Table("chatgpt_users").Where("zj_fid = ?", userId).Find(&directUsers)

		// 统计lev
		var lev1, lev2, lev3 []model.User
		for _, du := range directUsers {
			switch du.Lev {
			case 1:
				lev1 = append(lev1, du)
			case 2:
				lev2 = append(lev2, du)
			case 3:
				lev3 = append(lev3, du)
			}
		}

		// 规则1：2个lev=1，升2
		if len(lev1) >= 2 && user.Lev < 2 {
			ts := time.Now()
			h.DB.Table("chatgpt_users").Where("id = ?", userId).Updates(map[string]interface{}{"lev": 2, "lev_since_at": ts})
			updated = append(updated, userId)
			continue
		}
		// 规则2：2个lev=2，升3
		if len(lev2) >= 2 && user.Lev < 3 {
			ts := time.Now()
			h.DB.Table("chatgpt_users").Where("id = ?", userId).Updates(map[string]interface{}{"lev": 3, "lev_since_at": ts})
			updated = append(updated, userId)
			continue
		}
		// 规则3：2个lev=3，且其中有1个的间推有lev=3，升4  此处有bug
		if len(lev3) >= 2 && user.Lev < 4 {
			found := false
			for _, du := range lev3 {
				var indirects []model.User
				h.DB.Table("chatgpt_users").Where("jj_fid = ?", du.Id).Find(&indirects)
				for _, ind := range indirects {
					if ind.Lev == 3 {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if found {
				ts := time.Now()
				h.DB.Table("chatgpt_users").Where("id = ?", userId).Updates(map[string]interface{}{"lev": 4, "lev_since_at": ts})
				updated = append(updated, userId)
			}
		}
	}

	resp.SUCCESS(c, gin.H{
		"updated_count": len(updated),
		"updated_users": updated,
	})
}

// 这里使用动态升级的方式来计算用户等级
// 根据用户的钻石等级动态计算并更新用户的 lev 字段
// lev 映射关系：
// 0: 未激活或激活会员（10层）
// 1: 1钻（20层）
// 2: 2钻（30层）
// 3: 3钻（40层）
// 4: 4钻（50层）
// 5: 5钻（60层）
// 6: 6钻（70层）
// 7: 7钻（80层）
// 8: 8钻（90层）
// 9: 9钻（100层）
func (h *MintOrderHandler) FenHong(c *gin.Context) {
	logger.Info("=== 开始批量升级用户等级 ===")

	// 1. 查询所有激活的用户
	var users []model.User
	if err := h.DB.Where("is_jh = ?", 1).Find(&users).Error; err != nil {
		logger.Error("查询激活用户失败:", err)
		resp.ERROR(c, "查询用户失败")
		return
	}

	logger.Info("查询到激活用户数量:", len(users))

	// 2. 统计信息
	var updatedCount int
	var updatedUsers []map[string]interface{}

	// 3. 遍历所有用户，计算并更新等级和层数
	for _, user := range users {
		// 获取用户的团队奖励层级限制（使用工具函数，返回的是层数 floor）
		floor := GetUserRewardLayerLimit(h.DB, int(user.Id))

		// 将层数映射到等级 lev
		var newLev int64
		switch floor {
		case -1: // 未激活（理论上不应该出现，因为已经过滤了）
			newLev = 0
		case 10: // 激活会员
			newLev = 0
		case 20: // 1钻
			newLev = 1
		case 30: // 2钻
			newLev = 2
		case 40: // 3钻
			newLev = 3
		case 50: // 4钻
			newLev = 4
		case 60: // 5钻
			newLev = 5
		case 70: // 6钻
			newLev = 6
		case 80: // 7钻
			newLev = 7
		case 90: // 8钻
			newLev = 8
		case 100: // 9钻
			newLev = 9
		default:
			newLev = 0
		}

		// 如果等级或层数有变化，更新数据库
		newFloor := int64(floor)
		if user.Lev != newLev || user.Floor != newFloor {
			updates := map[string]interface{}{
				"lev":   newLev,
				"floor": newFloor,
			}
			if user.Lev != newLev {
				updates["lev_since_at"] = time.Now()
			}
			if err := h.DB.Model(&model.User{}).Where("id = ?", user.Id).Updates(updates).Error; err != nil {
				logger.Error("更新用户等级和层数失败，用户ID:", user.Id, "错误:", err)
				continue
			}

			updatedCount++
			updatedUsers = append(updatedUsers, map[string]interface{}{
				"user_id":   user.Id,
				"username":  user.Username,
				"old_lev":   user.Lev,
				"new_lev":   newLev,
				"old_floor": user.Floor,
				"new_floor": newFloor,
			})

			logger.Info(fmt.Sprintf("用户等级和层数已更新 - ID: %d, 用户名: %s, 旧等级: %d, 新等级: %d, 旧层数: %d, 新层数: %d",
				user.Id, user.Username, user.Lev, newLev, user.Floor, newFloor))
		}
	}

	logger.Info(fmt.Sprintf("=== 批量升级完成，共更新 %d 个用户 ===", updatedCount))

	resp.SUCCESS(c, gin.H{
		"total_users":   len(users),
		"updated_count": updatedCount,
		"updated_users": updatedUsers,
	})
}

// 定时任务执行 用户对应等级分红
func (h *MintOrderHandler) FenHongTask(c *gin.Context) {
	logger.Info("=== 开始执行用户对应等级分红 ===")
	// 业务调整：1钻～9钻等级分红已关闭（保留接口与定时任务路由，直接跳过发放）
	logger.Info("等级分红已停用（1钻～9钻），跳过发放")
	resp.SUCCESS(c, gin.H{"fenhong_count": 0, "disabled": true})
}

// // 这里再增加一个方法，那就是对于 lev 字段不为空或者0的用户进行奖励
// func (h *MintOrderHandler) FenHong(c *gin.Context) {
// 	// 1. 查询所有lev>0的用户
// 	var users []model.User
// 	h.DB.Table("chatgpt_users").Where("lev > 0").Find(&users)

// 	logger.Info("FenHong: 查询到lev>0的用户数量:", len(users))

// 	var count int
// 	for _, user := range users {
// 		logger.Info("FenHong: 处理用户:", user.Username, "lev:", user.Lev)
// 		var power int
// 		switch user.Lev {
// 		case 1:
// 			power = 1000
// 		case 2:
// 			power = 2000
// 		case 3:
// 			power = 3000
// 		case 4:
// 			power = 4000
// 		default:
// 			continue
// 		}

// 		//获取用户chatgpt_orders表中，status为8的订单数量 按照now_power排序
// 		var orders []model.Order
// 		var ordersObj model.Order
// 		h.DB.Table("chatgpt_orders").Where("user_id = ? AND status = 8", user.Id).Order("now_power DESC").Find(&orders)
// 		logger.Info("FenHong: 用户", user.Username, "的status=8订单数量:", len(orders))
// 		if len(orders) == 0 {
// 			logger.Info("FenHong: 用户", user.Username, "没有status=8的订单，跳过")
// 			continue
// 		}

// 		//获取now_power最大的那笔订单
// 		nowPowerMax := orders[0].NowPower
// 		ordersObj = orders[0]

// 		//判断nowPowerMax是否大于power
// 		if nowPowerMax < float64(power) {
// 			logger.Info("FenHong: 用户", user.Username, "剩余算力不足，调整power从", power, "到", nowPowerMax)
// 			//则直接将power设置为nowPowerMax
// 			power = int(nowPowerMax)
// 			//continue
// 		}

// 		//将最大那笔订单的now_power减少对应的power值
// 		h.DB.Table("chatgpt_orders").Where("id = ?", orders[0].Id).UpdateColumn("now_power", gorm.Expr("now_power - ?", power))

// 		//将最大那笔订单的js_power增加对应的power值
// 		h.DB.Table("chatgpt_orders").Where("id = ?", orders[0].Id).UpdateColumn("js_power", gorm.Expr("js_power + ?", power))

// 		//更新用户的余额
// 		h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumn("power", gorm.Expr("power + ?", power))

// 		// 3. 写入算力流水日志 - 只有当power大于0时才记录
// 		if power > 0 {
// 			h.DB.Create(&model.PowerLog{
// 				UserId:    user.Id,
// 				Username:  user.Username,
// 				Type:      types.PowerSf, //释放
// 				Amount:    float64(power),
// 				Balance:   user.Power + float64(power), // 这里是旧值+新加的
// 				Mark:      types.PowerAdd,
// 				OrderNo:   ordersObj.OrderNo,
// 				Model:     "团队分红奖励",
// 				Remark:    fmt.Sprintf("团队分红奖励释放算力：%d，用户名：%s", power, user.Username),
// 				CreatedAt: time.Now(),
// 			})
// 		}
// 		count++
// 		logger.Info("FenHong: 成功处理用户", user.Username, "，当前count:", count)
// 	}

// 	resp.SUCCESS(c, gin.H{
// 		"fenhong_count": count,
// 	})
// }

// updateLimitCancelConfig 更新系统配置中的limit_cancel字段
func (h *MintOrderHandler) updateLimitCancelConfig() error {
	// 1. 计算所有状态为4（已确认）的交易的数量总和
	var totalNumber float64
	err := h.DB.Model(&model.Trade{}).Where("status = ?", 4).Select("COALESCE(SUM(number), 0)").Scan(&totalNumber).Error
	if err != nil {
		return fmt.Errorf("计算交易数量总和失败: %v", err)
	}

	// 2. 计算新的limit_cancel值：总和 + 564519
	newLimitCancel := int64(totalNumber) + 564519

	// 3. 获取当前系统配置
	var config model.Config
	err = h.DB.Where("marker = ?", "system").First(&config).Error
	if err != nil {
		return fmt.Errorf("获取系统配置失败: %v", err)
	}

	// 4. 解析当前配置JSON
	var currentConfig types.SystemConfig
	err = utils.JsonDecode(config.Config, &currentConfig)
	if err != nil {
		return fmt.Errorf("解析系统配置失败: %v", err)
	}

	// 保存原值用于日志
	oldLimitCancel := currentConfig.LimitCancel

	// 5. 更新limit_cancel字段
	currentConfig.LimitCancel = newLimitCancel

	// 6. 将更新后的配置编码为JSON
	updatedConfigJSON := utils.JsonEncode(&currentConfig)

	// 7. 更新数据库中的配置
	err = h.DB.Model(&config).Update("config_json", updatedConfigJSON).Error
	if err != nil {
		return fmt.Errorf("更新数据库配置失败: %v", err)
	}

	// 8. 更新AppServer缓存
	err = utils.JsonDecode(updatedConfigJSON, &h.App.SysConfig)
	if err != nil {
		return fmt.Errorf("更新AppServer缓存失败: %v", err)
	}

	logger.Infof("成功更新limit_cancel配置: %d (原值: %d, 新增交易数量: %f)",
		newLimitCancel, oldLimitCancel, totalNumber)

	return nil
}
