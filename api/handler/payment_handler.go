package handler

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/service"
	"chatplus/service/payment"
	"chatplus/store/model"
	"chatplus/utils"
	"chatplus/utils/resp"
	"embed"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	PayWayAlipay = "支付宝"
	PayWayXunHu  = "虎皮椒"
	PayWayJs     = "PayJS"
)

// PaymentHandler 支付服务回调 handler
type PaymentHandler struct {
	BaseHandler
	alipayService  *payment.AlipayService
	huPiPayService *payment.HuPiPayService
	js             *payment.PayJS
	snowflake      *service.Snowflake
	creditService  *service.CreditService
	fs             embed.FS
	lock           sync.Mutex
}

// GetUserRewardLayerLimit 是一个工具函数，用于在其他 handler 中复用用户等级计算逻辑
// 返回：-1表示未激活，2表示激活会员（2层），3表示1钻（3层），4表示2钻（4层），5表示3钻（5层），6表示4钻（6层）
func GetUserRewardLayerLimit(db *gorm.DB, userID int) int {
	h := &PaymentHandler{
		BaseHandler: BaseHandler{
			DB: db,
		},
	}
	return h.getUserRewardLayerLimit(userID)
}

// PlacementNode 用于维护双轨安置关系（严格二叉树）
type PlacementNode struct {
	UserID       int       `gorm:"primaryKey;column:user_id"`
	ParentID     int       `gorm:"column:parent_id"`
	LeftChildID  int       `gorm:"column:left_child_id"`
	RightChildID int       `gorm:"column:right_child_id"`
	LeftPerf     float64   `gorm:"column:left_perf"`
	RightPerf    float64   `gorm:"column:right_perf"`
	TotalPerf    float64   `gorm:"column:total_perf"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (PlacementNode) TableName() string { return "chatgpt_placement_nodes" }

func NewPaymentHandler(
	server *core.AppServer,
	alipayService *payment.AlipayService,
	huPiPayService *payment.HuPiPayService,
	js *payment.PayJS,
	db *gorm.DB,
	snowflake *service.Snowflake,
	creditService *service.CreditService,
	fs embed.FS) *PaymentHandler {
	return &PaymentHandler{
		alipayService:  alipayService,
		huPiPayService: huPiPayService,
		js:             js,
		snowflake:      snowflake,
		creditService:  creditService,
		fs:             fs,
		lock:           sync.Mutex{},
		BaseHandler: BaseHandler{
			App: server,
			DB:  db,
		},
	}
}

// 激活产品
func (h *PaymentHandler) ConfirmJihuo(c *gin.Context) {
	var data struct {
		ProductId uint `json:"product_id"`
		UserId    int  `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	if !service.ActivationEnabled(h.App.SysConfig) {
		resp.ERROR(c, "系统暂未开放激活")
		return
	}

	var product model.Product
	res := h.DB.First(&product, data.ProductId)
	if res.Error != nil {
		resp.ERROR(c, "产品不存在")
		return
	}

	//判断用户算力余额
	var user model.User
	res = h.DB.First(&user, data.UserId)
	if res.Error != nil {
		resp.ERROR(c, "无效的用户id")
		return
	}

	if user.Power < product.Sales {
		resp.ERROR(c, "你当前可用算力余额不足，请充值后再操作")
		return
	}

	orderNo, err := h.snowflake.Next(false)
	if err != nil {
		resp.ERROR(c, "error with generate trade no: "+err.Error())
		return
	}

	//同类型的产品在质押中状态只能投一个

	//先注释掉
	var items []model.Order
	h.DB.Where("user_id = ? AND product_id = ? And  status=8 ", data.UserId, data.ProductId).Find(&items)
	logger.Info("当前用户已经投入的产品为", items)
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
		Days:     product.Days,
		Power:    product.Power,
		Name:     product.Name,
		Price:    product.Price,
		Discount: product.Discount,
	}

	order := model.Order{
		UserId:    user.Id,
		Username:  user.Username,
		ProductId: product.Id,
		OrderNo:   orderNo,
		Subject:   product.Name,
		Amount:    float64(product.SortNum), //等级
		Status:    types.OrderZyz,
		NowPower:  product.Power,
		PayWay:    "",
		Power:     product.Power, //当前产品产出值
		JsPower:   0,
		Remark:    utils.JsonEncode(remark),
		Sales:     product.Sales, //系统实际计算值
	}
	res = h.DB.Create(&order)
	if res.Error != nil || res.RowsAffected == 0 {
		resp.ERROR(c, "error with create order: "+res.Error.Error())
		return
	}
	//扣减用户可用算力
	h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumn("power", gorm.Expr("power - ?", product.Sales))

	//先查询上级用的激活人数
	//获取激活总数
	var jhTotal int64
	h.DB.Raw("SELECT COUNT(*) FROM chatgpt_users WHERE zj_fid = ? AND is_jh = 1", user.ZjFid).Scan(&jhTotal)

	//判断当前用户是否是第一次激活
	if user.IsJh != 1 {

		//更新用户状态为激活
		h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumn("is_jh", 1)

		//如果上级用户的激活人数为0，则不执行以下代码
		if jhTotal != 0 {
			//追加上级agent-num
			h.DB.Model(&model.User{}).Where("id = ?", user.ZjFid).UpdateColumn("agent_num", gorm.Expr("agent_num + ?", 1))
		}

	}

	//更新流水
	h.DB.Create(&model.PowerLog{
		UserId:   user.Id,
		Username: user.Username,
		Type:     types.PowerRecharge,
		Amount:   product.Sales,
		Balance:  user.Power - product.Sales,
		Mark:     types.PowerSub,
		Model:    fmt.Sprintf("购买算力产品：%s", product.Name),
		//Model:    fmt.Sprintf("团队用户  充值:%s 产品  加速释放算力：%f，", product.Name, turesubPower),
		//OrderNo:   order.OrderNo,
		Remark:    fmt.Sprintf("用户质押：%s套餐,花费算力：%f，订单号：%s", product.Name, user.Power, orderNo),
		CreatedAt: time.Now(),
	})

	// 记录安置关系并累加团队业绩（双轨弱区滑落）
	if err := h.recordBinaryPlacement(user, user.ZjFid, product.Sales); err != nil {
		logger.Error("处理安置树失败:", err)
	}

	// 授信：刷新本人最大额度，并给直推上级增加可用额度
	if h.creditService != nil {
		if err := h.creditService.AfterInvest(h.DB, user, product.Sales, h.App.SysConfig); err != nil {
			logger.Error("激活后授信额度更新失败:", err)
		}
	}

	//激活后直接释放上下级用户的算力
	//获取直接推荐人用户的质押中算力
	logger.Info("=== 开始处理上级奖励释放 ===")
	logger.Info("当前用户ID:", user.Id, "用户名:", user.Username)
	logger.Info("用户直接上级为", user.ZjFid)
	logger.Info("系统配置 - 直推加速比例 ZjPro:", h.App.SysConfig.ZjPro, "（间推奖励已关闭）原 jj_pro:", h.App.SysConfig.JjPro, "SanPro:", h.App.SysConfig.SanPro, "OtherPro:", h.App.SysConfig.OtherPro)
	logger.Info("产品信息 - 名称:", product.Name, "等级:", product.SortNum, "投入值:", product.Sales, "产出值:", product.Power)

	// 用户上级关系：只处理直接上级和间接上级
	var ids [1]int
	ids[0] = user.ZjFid // 直接上级
	//ids[1] = user.JjFid // 间接上级

	logger.Info("上级ID数组:", ids)

	// 处理每个上级用户
	for i := 0; i < len(ids); i++ {
		logger.Info("=== 处理第", i+1, "代上级 ===")
		logger.Info("当前处理的上级ID:", ids[i])

		// 跳过无效的上级ID
		if ids[i] <= 0 {
			logger.Info("跳过无效上级ID:", ids[i])
			continue
		}

		// 验证上级用户是否存在
		var superiorUser model.User
		res = h.DB.Model(&model.User{}).Where("id = ?", ids[i]).First(&superiorUser)
		if res.Error != nil {
			logger.Info("上级用户不存在，用户ID:", ids[i], "错误:", res.Error)
			continue
		}
		logger.Info("处理上级用户:", superiorUser.Username, "代数为:", i+1, "用户ID:", superiorUser.Id)

		// 获取上级用户的质押中订单
		var superiorOrders []model.Order
		res = h.DB.Model(&model.Order{}).Where("user_id = ? AND status = 8", ids[i]).Find(&superiorOrders)
		if res.Error != nil {
			logger.Info("查询上级订单失败，用户ID:", ids[i], "错误:", res.Error)
			continue
		}
		if len(superiorOrders) == 0 {
			logger.Info("上级用户没有质押中的订单，用户ID:", ids[i])
			continue
		}

		logger.Info("上级用户质押中订单数量:", len(superiorOrders))

		// 提取上级用户的套餐等级并排序
		var superiorLevels []int
		for _, order := range superiorOrders {
			superiorLevels = append(superiorLevels, int(order.Amount))
			logger.Info("上级订单详情 - 订单号:", order.OrderNo, "等级:", order.Amount, "剩余算力:", order.NowPower, "已释放:", order.JsPower)
		}
		sort.Ints(superiorLevels)
		logger.Info("上级用户套餐等级列表:", superiorLevels)
		logger.Info("当前用户购买套餐等级:", product.SortNum)

		// 确定要释放的套餐等级
		targetLevel := h.determineTargetLevel(superiorLevels, product.SortNum)
		logger.Info("确定的目标套餐等级:", targetLevel)
		if targetLevel == 0 {
			logger.Info("无法确定目标套餐等级")
			continue
		}

		// 找到对应的订单
		var targetOrder model.Order
		found := false
		for _, order := range superiorOrders {
			if int(order.Amount) == targetLevel {
				targetOrder = order
				found = true
				logger.Info("找到目标订单 - 订单号:", order.OrderNo, "等级:", order.Amount, "剩余算力:", order.NowPower)
				break
			}
		}

		if !found {
			logger.Info("未找到目标套餐订单，等级:", targetLevel)
			continue
		}

		// 仅直推上级获得推荐加速释放（间推已取消）
		scale := h.App.SysConfig.ZjPro
		logger.Info("直推上级释放比例:", scale)

		// 计算应释放的算力
		releasePower := product.Sales * scale
		logger.Info("计算释放算力:", releasePower, "比例:", scale, "产品投入值:", product.Sales)

		// 执行算力释放 - 实现0烧伤
		logger.Info("=== 开始执行算力释放（推荐奖励，支持0烧伤）===")

		// 优先从目标等级的订单释放，如果不足则从其他订单补充
		var ordersToRelease []model.Order

		// 先添加目标订单（优先）
		ordersToRelease = append(ordersToRelease, targetOrder)
		logger.Info("优先从目标订单释放，订单号:", targetOrder.OrderNo, "等级:", targetOrder.Amount, "剩余算力:", targetOrder.NowPower)

		// 如果目标订单剩余算力不足，添加其他订单作为补充
		if targetOrder.NowPower < releasePower {
			remainingNeeded := releasePower - targetOrder.NowPower
			logger.Info("目标订单剩余算力不足，需要从其他订单补充:", remainingNeeded)

			// 添加其他订单（排除目标订单）
			for _, order := range superiorOrders {
				if order.OrderNo != targetOrder.OrderNo {
					ordersToRelease = append(ordersToRelease, order)
				}
			}
		}

		// 使用多订单释放函数，实现0烧伤
		totalReleased := h.releaseSuperiorPowerFromMultipleOrders(ordersToRelease, releasePower, user.Username, product.Name, "推荐奖励", user.Id)
		logger.Info("=== 算力释放完成，实际释放:", totalReleased, "需要释放:", releasePower, "烧伤:", releasePower-totalReleased)
	}

	logger.Info("=== 上级奖励释放处理完成 ===")

	// 团队奖励：按照弱区规则派发（比例见系统配置 team_reward_pro）
	teamScale := h.teamRewardScale()
	if err := h.distributeTeamRewardByWeakArea(int(user.Id), product.Sales, teamScale, product.Name); err != nil {
		logger.Error("派发团队奖励失败:", err)
	}

	resp.SUCCESS(c)
}

// 激活产品 扩展方法
func (h *PaymentHandler) ConfirmJihuoEx(c *gin.Context) {
	var data struct {
		Power  float64 `json:"power"`
		UserId int     `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	if !service.ActivationEnabled(h.App.SysConfig) {
		resp.ERROR(c, "系统暂未开放激活")
		return
	}

	//判断用户算力余额
	var user model.User
	res := h.DB.First(&user, data.UserId)
	if res.Error != nil {
		resp.ERROR(c, "无效的用户id")
		return
	}

	if user.Power < data.Power {
		resp.ERROR(c, "你当前可用算力余额不足，请充值后再操作")
		return
	}

	minPower := h.App.SysConfig.JihuoExMinPower
	if minPower > 0 && data.Power < minPower {
		resp.ERROR(c, fmt.Sprintf("最低投入算力为%g", minPower))
		return
	}
	if err := service.ValidateInvestPower(data.Power, h.App.SysConfig); err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	// 按投入档位硬编码复投次数（1000～9.9999万各10次；10万+不限）
	if err := service.CheckReinvestTierLimit(h.DB, user.Id, data.Power); err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	orderNo, err := h.snowflake.Next(false)
	if err != nil {
		resp.ERROR(c, "error with generate trade no: "+err.Error())
		return
	}

	if res.Error != nil {
		resp.ERROR(c, "无效的用户id")
		return
	}

	remark := types.OrderRemark{
		Days:     1,
		Power:    data.Power,
		Name:     "自定义套餐",
		Price:    3,
		Discount: 2,
	}

	outputPower := h.App.SysConfig.InvestOutputPower(data.Power)
	order := model.Order{
		UserId:    user.Id,
		Username:  user.Username,
		ProductId: 0,
		OrderNo:   orderNo,
		Subject:   fmt.Sprintf("自定义套餐：%f", data.Power),
		Amount:    float64(1), //等级
		Status:    types.OrderZyz,
		NowPower:  outputPower,
		PayWay:    "",
		Power:     outputPower, //产出值
		JsPower:   0,
		Remark:    utils.JsonEncode(remark),
		Sales:     data.Power, //投入值
	}
	res = h.DB.Create(&order)
	if res.Error != nil || res.RowsAffected == 0 {
		resp.ERROR(c, "error with create order: "+res.Error.Error())
		return
	}
	//扣减用户可用算力
	h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumn("power", gorm.Expr("power - ?", data.Power))

	//先查询上级用的激活人数
	//获取激活总数
	var jhTotal int64
	h.DB.Raw("SELECT COUNT(*) FROM chatgpt_users WHERE zj_fid = ? AND is_jh = 1", user.ZjFid).Scan(&jhTotal)

	alreadyActivated := user.IsJh == 1

	//判断当前用户是否是第一次激活
	if !alreadyActivated {

		//更新用户状态为激活
		h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumn("is_jh", 1)

		//如果上级用户的激活人数为0，则不执行以下代码
		if jhTotal != 0 {
			//追加上级agent-num
			h.DB.Model(&model.User{}).Where("id = ?", user.ZjFid).UpdateColumn("agent_num", gorm.Expr("agent_num + ?", 1))
		}

	}

	//更新流水
	h.DB.Create(&model.PowerLog{
		UserId:   user.Id,
		Username: user.Username,
		Type:     types.PowerRecharge,
		Amount:   data.Power,
		Balance:  user.Power - data.Power,
		Mark:     types.PowerSub,
		Model:    fmt.Sprintf("购买算力产品： 算力 %f ", data.Power),
		//Model:    fmt.Sprintf("团队用户  充值:%s 产品  加速释放算力：%f，", product.Name, turesubPower),
		//OrderNo:   order.OrderNo,
		Remark:    fmt.Sprintf("用户质押：%s套餐,花费算力：%f，订单号：%s", "自定义套餐", user.Power, orderNo),
		CreatedAt: time.Now(),
	})

	// 记录安置关系并累加团队业绩（双轨弱区滑落）
	if err := h.recordBinaryPlacement(user, user.ZjFid, data.Power); err != nil {
		logger.Error("处理安置树失败:", err)
	}

	if alreadyActivated {
		h.DB.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumn("jihuo_ex_reinvest_count", gorm.Expr("jihuo_ex_reinvest_count + ?", 1))
	}

	// 授信：刷新本人最大额度，并给直推上级增加可用额度
	if h.creditService != nil {
		if err := h.creditService.AfterInvest(h.DB, user, data.Power, h.App.SysConfig); err != nil {
			logger.Error("复投后授信额度更新失败:", err)
		}
	}

	// 推荐/团队奖励计算较重，异步处理以尽快响应前端（避免复投长时间转圈）
	rewardUser := user
	rewardPower := data.Power
	go h.processConsumeRewardsAsync(rewardUser, rewardPower, "自定义套餐")

	resp.SUCCESS(c)
}

// ConfirmJihuoGift 赠送激活：付款人扣算力，为收款人（转账码/邀请码）创建激活/复投订单
func (h *PaymentHandler) ConfirmJihuoGift(c *gin.Context) {
	var data struct {
		Code     string  `json:"code"`
		Power    float64 `json:"power"`
		Password string  `json:"password"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	data.Code = strings.TrimSpace(data.Code)
	if data.Code == "" || data.Power <= 0 || strings.TrimSpace(data.Password) == "" {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	if !service.ActivationEnabled(h.App.SysConfig) {
		resp.ERROR(c, "系统暂未开放激活")
		return
	}

	giver, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	// 校验密码
	if utils.GenPassword(data.Password, giver.Salt) != giver.Password {
		resp.ERROR(c, "密码输入错误")
		return
	}

	// 付款人需已有激活订单（与转账门槛一致）
	var giverOrders []model.Order
	h.DB.Where("user_id = ?", giver.Id).Find(&giverOrders)
	if len(giverOrders) == 0 {
		resp.ERROR(c, "请先激活产品才能赠送")
		return
	}

	inviteCode := model.InviteCode{}
	if err := h.DB.Where("code = ?", data.Code).First(&inviteCode).Error; err != nil {
		resp.ERROR(c, "赠送码不存在，请核对后再操作")
		return
	}

	var receiver model.User
	if err := h.DB.First(&receiver, inviteCode.UserId).Error; err != nil {
		resp.ERROR(c, "收款用户不存在")
		return
	}
	if receiver.Id == giver.Id {
		resp.ERROR(c, "不能赠送给自己")
		return
	}

	// 赠送服务费固定 1%（另计，对方仍按赠送额激活）
	const giftFeePct = 1.0
	feeAmount := math.Round(data.Power*giftFeePct/100.0*1e6) / 1e6
	totalDebit := math.Round((data.Power+feeAmount)*1e6) / 1e6
	if giver.Power < totalDebit {
		resp.ERROR(c, "你当前可用算力余额不足，无法支付赠送算力及服务费")
		return
	}

	minPower := h.App.SysConfig.JihuoExMinPower
	if minPower > 0 && data.Power < minPower {
		resp.ERROR(c, fmt.Sprintf("最低投入算力为%g", minPower))
		return
	}
	if err := service.ValidateInvestPower(data.Power, h.App.SysConfig); err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	// 按投入档位硬编码复投次数（按收款人同档订单数）
	if err := service.CheckReinvestTierLimit(h.DB, receiver.Id, data.Power); err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	orderNo, err := h.snowflake.Next(false)
	if err != nil {
		resp.ERROR(c, "error with generate trade no: "+err.Error())
		return
	}

	remark := types.OrderRemark{
		Days:     1,
		Power:    data.Power,
		Name:     "赠送激活",
		Price:    3,
		Discount: 2,
	}

	txErr := h.DB.Transaction(func(tx *gorm.DB) error {
		// 扣减付款人算力：赠送额 + 1% 服务费（乐观校验余额）
		res := tx.Model(&model.User{}).
			Where("id = ? AND power >= ?", giver.Id, totalDebit).
			UpdateColumn("power", gorm.Expr("power - ?", totalDebit))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("你当前可用算力余额不足，无法支付赠送算力及服务费")
		}

		outputPower := h.App.SysConfig.InvestOutputPower(data.Power)
		order := model.Order{
			UserId:    receiver.Id,
			Username:  receiver.Username,
			ProductId: 0,
			OrderNo:   orderNo,
			Subject:   fmt.Sprintf("赠送激活：%f", data.Power),
			Amount:    float64(1),
			Status:    types.OrderZyz,
			NowPower:  outputPower,
			PayWay:    "gift",
			Power:     outputPower,
			JsPower:   0,
			Remark:    utils.JsonEncode(remark),
			Sales:     data.Power,
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		alreadyActivated := receiver.IsJh == 1
		if !alreadyActivated {
			var jhTotal int64
			tx.Raw("SELECT COUNT(*) FROM chatgpt_users WHERE zj_fid = ? AND is_jh = 1", receiver.ZjFid).Scan(&jhTotal)
			if err := tx.Model(&model.User{}).Where("id = ?", receiver.Id).UpdateColumn("is_jh", 1).Error; err != nil {
				return err
			}
			if jhTotal != 0 && receiver.ZjFid > 0 {
				if err := tx.Model(&model.User{}).Where("id = ?", receiver.ZjFid).
					UpdateColumn("agent_num", gorm.Expr("agent_num + ?", 1)).Error; err != nil {
					return err
				}
			}
		} else {
			if err := tx.Model(&model.User{}).Where("id = ?", receiver.Id).
				UpdateColumn("jihuo_ex_reinvest_count", gorm.Expr("jihuo_ex_reinvest_count + ?", 1)).Error; err != nil {
				return err
			}
		}

		var updatedGiver model.User
		giverBalance := math.Round((giver.Power-totalDebit)*1e6) / 1e6
		if err := tx.Select("power").First(&updatedGiver, giver.Id).Error; err == nil {
			giverBalance = updatedGiver.Power
		}
		// 赠送本金流水：余额展示为扣完本金后、扣手续费前
		balanceAfterGift := math.Round((giverBalance+feeAmount)*1e6) / 1e6
		t1 := time.Now()
		if err := tx.Create(&model.PowerLog{
			UserId:    giver.Id,
			Username:  giver.Username,
			Type:      types.PowerZy,
			Amount:    data.Power,
			Balance:   balanceAfterGift,
			Mark:      types.PowerSub,
			Model:     fmt.Sprintf("赠送激活:受益人(%s)", receiver.Username),
			Remark:    fmt.Sprintf("为用户 %s 赠送激活 %.6f，订单号：%s", receiver.Username, data.Power, orderNo),
			CreatedAt: t1,
		}).Error; err != nil {
			return err
		}
		if feeAmount > 0 {
			if err := tx.Create(&model.PowerLog{
				UserId:    giver.Id,
				Username:  giver.Username,
				Type:      types.PowerZy,
				Amount:    feeAmount,
				Balance:   giverBalance,
				Mark:      types.PowerSub,
				Model:     fmt.Sprintf("赠送服务费:受益人(%s)", receiver.Username),
				Remark:    fmt.Sprintf("赠送服务费 %.6f（%.4f%%），订单号：%s", feeAmount, giftFeePct, orderNo),
				CreatedAt: t1.Add(time.Millisecond),
			}).Error; err != nil {
				return err
			}
			// 服务费按配置比例静默回流认购池（默认 100%）
			poolPct := 100.0
			if h.App.SysConfig != nil {
				poolPct = h.App.SysConfig.TransferFeeToPoolPercent
				if poolPct < 0 {
					poolPct = 0
				}
				if poolPct > 100 {
					poolPct = 100
				}
			}
			poolIn := math.Round(feeAmount*poolPct/100.0*1e6) / 1e6
			if poolIn > 0 {
				var pool model.PowerPool
				_ = tx.Where("id = ?", 1).FirstOrCreate(&pool, model.PowerPool{
					BaseModel:  model.BaseModel{Id: 1},
					TotalPower: 0,
					TotalUsdt:  0,
				})
				if err := tx.Model(&model.PowerPool{}).Where("id = ?", 1).
					UpdateColumn("total_power", gorm.Expr("total_power + ?", poolIn)).Error; err != nil {
					return err
				}
			}
		}

		if err := tx.Create(&model.PowerLog{
			UserId:    receiver.Id,
			Username:  receiver.Username,
			Type:      types.PowerZy,
			Amount:    data.Power,
			Balance:   receiver.Power,
			Mark:      types.PowerAdd,
			Model:     fmt.Sprintf("收到赠送激活:赠送人(%s)", giver.Username),
			Remark:    fmt.Sprintf("收到用户 %s 赠送激活 %.6f，订单号：%s", giver.Username, data.Power, orderNo),
			CreatedAt: time.Now(),
		}).Error; err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		resp.ERROR(c, txErr.Error())
		return
	}

	// 安置与分红按收款人（被激活方）计算
	if err := h.recordBinaryPlacement(receiver, receiver.ZjFid, data.Power); err != nil {
		logger.Error("赠送激活：处理安置树失败:", err)
	}
	// 授信：按收款人投入刷新其 Max，并给其直推上级加可用额度
	if h.creditService != nil {
		if err := h.creditService.AfterInvest(h.DB, receiver, data.Power, h.App.SysConfig); err != nil {
			logger.Error("赠送激活后授信额度更新失败:", err)
		}
	}
	go h.processConsumeRewardsAsync(receiver, data.Power, "赠送激活")

	resp.SUCCESS(c, gin.H{
		"order_no":   orderNo,
		"power":      data.Power,
		"fee":        feeAmount,
		"total":      totalDebit,
		"receiver":   receiver.Username,
	})
}

// ConfirmJihuoCreditGift 授信报单：扣付款人授信可用额度（免手续费），为收款人创建激活/复投订单
func (h *PaymentHandler) ConfirmJihuoCreditGift(c *gin.Context) {
	var data struct {
		Code     string  `json:"code"`
		Power    float64 `json:"power"`
		Password string  `json:"password"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	data.Code = strings.TrimSpace(data.Code)
	if data.Code == "" || data.Power <= 0 || strings.TrimSpace(data.Password) == "" {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	if !service.ActivationEnabled(h.App.SysConfig) {
		resp.ERROR(c, "系统暂未开放激活")
		return
	}
	if h.creditService == nil {
		resp.ERROR(c, "授信服务未启用")
		return
	}

	giver, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}
	if utils.GenPassword(data.Password, giver.Salt) != giver.Password {
		resp.ERROR(c, "密码输入错误")
		return
	}

	var giverOrders []model.Order
	h.DB.Where("user_id = ?", giver.Id).Find(&giverOrders)
	if len(giverOrders) == 0 {
		resp.ERROR(c, "请先激活产品才能使用授信报单")
		return
	}

	// 刷新本人 Max 后再校验可用额度
	_ = h.creditService.RecalcMax(h.DB, giver.Id, h.App.SysConfig)
	_ = h.DB.First(&giver, giver.Id)

	amount := math.Round(data.Power*1e6) / 1e6
	if giver.CreditAvailable < amount {
		resp.ERROR(c, fmt.Sprintf("授信可用额度不足：需要 %.6f，当前可用 %.6f", amount, giver.CreditAvailable))
		return
	}

	inviteCode := model.InviteCode{}
	if err := h.DB.Where("code = ?", data.Code).First(&inviteCode).Error; err != nil {
		resp.ERROR(c, "赠送码不存在，请核对后再操作")
		return
	}
	var receiver model.User
	if err := h.DB.First(&receiver, inviteCode.UserId).Error; err != nil {
		resp.ERROR(c, "收款用户不存在")
		return
	}
	if receiver.Id == giver.Id {
		resp.ERROR(c, "不能赠送给自己")
		return
	}

	// 授信报单只能报给自己推荐伞下的人（zj_fid 链上的任意深度后代），
	// 禁止跨区/旁部门报单；报单后可用额度只加给收款人的直推上级。
	inSubtree, relErr := service.IsInReferralSubtree(h.DB, giver.Id, uint(receiver.Id))
	if relErr != nil {
		logger.Error("授信报单：校验推荐关系失败:", relErr)
		resp.ERROR(c, "校验推荐关系失败，请稍后重试")
		return
	}
	if !inSubtree {
		resp.ERROR(c, "只能给自己伞下的用户报单")
		return
	}

	minPower := h.App.SysConfig.JihuoExMinPower
	if minPower > 0 && amount < minPower {
		resp.ERROR(c, fmt.Sprintf("最低投入算力为%g", minPower))
		return
	}
	if err := service.ValidateInvestPower(amount, h.App.SysConfig); err != nil {
		resp.ERROR(c, err.Error())
		return
	}
	if err := service.CheckReinvestTierLimit(h.DB, receiver.Id, amount); err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	orderNo, err := h.snowflake.Next(false)
	if err != nil {
		resp.ERROR(c, "error with generate trade no: "+err.Error())
		return
	}

	remark := types.OrderRemark{
		Days:     1,
		Power:    amount,
		Name:     "授信报单",
		Price:    3,
		Discount: 2,
	}

	txErr := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := h.creditService.SpendForGift(tx, giver.Id, giver.Username, amount, orderNo, receiver.Username, h.App.SysConfig); err != nil {
			return err
		}

		outputPower := h.App.SysConfig.InvestOutputPower(amount)
		order := model.Order{
			UserId:    receiver.Id,
			Username:  receiver.Username,
			ProductId: 0,
			OrderNo:   orderNo,
			Subject:   fmt.Sprintf("授信报单：%f", amount),
			Amount:    float64(1),
			Status:    types.OrderZyz,
			NowPower:  outputPower,
			PayWay:    "credit_gift",
			Power:     outputPower,
			JsPower:   0,
			Remark:    utils.JsonEncode(remark),
			Sales:     amount,
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		alreadyActivated := receiver.IsJh == 1
		if !alreadyActivated {
			var jhTotal int64
			tx.Raw("SELECT COUNT(*) FROM chatgpt_users WHERE zj_fid = ? AND is_jh = 1", receiver.ZjFid).Scan(&jhTotal)
			if err := tx.Model(&model.User{}).Where("id = ?", receiver.Id).UpdateColumn("is_jh", 1).Error; err != nil {
				return err
			}
			if jhTotal != 0 && receiver.ZjFid > 0 {
				if err := tx.Model(&model.User{}).Where("id = ?", receiver.ZjFid).
					UpdateColumn("agent_num", gorm.Expr("agent_num + ?", 1)).Error; err != nil {
					return err
				}
			}
		} else {
			if err := tx.Model(&model.User{}).Where("id = ?", receiver.Id).
				UpdateColumn("jihuo_ex_reinvest_count", gorm.Expr("jihuo_ex_reinvest_count + ?", 1)).Error; err != nil {
				return err
			}
		}

		// 收款人侧明细（不增加可用算力，仅标记收到授信报单）
		if err := tx.Create(&model.PowerLog{
			UserId:    receiver.Id,
			Username:  receiver.Username,
			Type:      types.PowerZy,
			Amount:    amount,
			Balance:   receiver.Power,
			Mark:      types.PowerAdd,
			Model:     fmt.Sprintf("收到授信报单:赠送人(%s)", giver.Username),
			Remark:    fmt.Sprintf("收到用户 %s 授信报单 %.6f，订单号：%s", giver.Username, amount, orderNo),
			CreatedAt: time.Now(),
		}).Error; err != nil {
			return err
		}

		// 同一事务内：收款人侧不因授信单抬高 Max；可用额度只加给收款人直推上级
		return h.creditService.AfterGift(tx, giver, receiver, amount, h.App.SysConfig)
	})
	if txErr != nil {
		resp.ERROR(c, txErr.Error())
		return
	}

	if err := h.recordBinaryPlacement(receiver, receiver.ZjFid, amount); err != nil {
		logger.Error("授信报单：处理安置树失败:", err)
	}
	go h.processConsumeRewardsAsync(receiver, amount, "授信报单")

	var updatedGiver model.User
	_ = h.DB.Select("credit_max", "credit_available", "credit_debt").First(&updatedGiver, giver.Id)

	resp.SUCCESS(c, gin.H{
		"order_no":          orderNo,
		"power":             amount,
		"fee":               0,
		"total":             amount,
		"receiver":          receiver.Username,
		"credit_max":        updatedGiver.CreditMax,
		"credit_available":  updatedGiver.CreditAvailable,
		"credit_debt":       updatedGiver.CreditDebt,
	})
}

// OnAppPlazaPowerConsumed 应用广场算力成功消耗后分红（直推 + 团队弱区，同 confirmJihuoEx）
func (h *PaymentHandler) OnAppPlazaPowerConsumed(userId uint, power float64, sourceName string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("应用广场分红 panic, userID:", userId, "panic:", r)
		}
	}()
	if userId == 0 || power <= 0 {
		return
	}
	var user model.User
	if err := h.DB.First(&user, userId).Error; err != nil {
		logger.Error("应用广场分红：用户不存在 userID=", userId, " err=", err)
		return
	}
	if err := h.recordBinaryPlacement(user, user.ZjFid, power); err != nil {
		logger.Error("应用广场分红：安置树失败 userID=", userId, " err=", err)
	}
	h.processConsumeRewardsAsync(user, power, sourceName)
}

// ProcessMintPurchaseRewards 算力机购买后记录安置树并异步派发推荐奖/团队奖（使用 kj_zj_pro / kj_team_reward_pro）
func (h *PaymentHandler) ProcessMintPurchaseRewards(user model.User, power float64, productName string) {
	if power <= 0 {
		return
	}
	if strings.TrimSpace(productName) == "" {
		productName = "算力机"
	}
	if err := h.recordBinaryPlacement(user, user.ZjFid, power); err != nil {
		logger.Error("算力机购买：安置树失败 userID=", user.Id, " err=", err)
	}
	go h.processMintRewardsAsync(user, power, productName)
}

func (h *PaymentHandler) mintZjPro() float64 {
	if h.App != nil && h.App.SysConfig.KjZjPro > 0 {
		return h.App.SysConfig.KjZjPro
	}
	return 0.3
}

func (h *PaymentHandler) mintTeamRewardScale() float64 {
	if h.App != nil && h.App.SysConfig.KjTeamRewardPro > 0 {
		return h.App.SysConfig.KjTeamRewardPro
	}
	return h.teamRewardScale()
}

// processMintRewardsAsync 算力机购买直推推荐奖 + 团队弱区奖
func (h *PaymentHandler) processMintRewardsAsync(user model.User, power float64, productName string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("算力机异步奖励 panic, userID:", user.Id, "panic:", r)
		}
	}()

	if strings.TrimSpace(productName) == "" {
		productName = "算力机"
	}
	logger.Info("=== 开始异步处理算力机购买奖励 ===")
	logger.Info("当前用户ID:", user.Id, "用户名:", user.Username)
	logger.Info("算力机直推比例 kj_zj_pro:", h.mintZjPro(), "算力机团队奖比例 kj_team_reward_pro:", h.mintTeamRewardScale())
	logger.Info("投入算力:", power)

	if user.ZjFid <= 0 {
		logger.Info("无直推上级，跳过推荐奖")
	} else {
		var superiorUser model.User
		if err := h.DB.Model(&model.User{}).Where("id = ?", user.ZjFid).First(&superiorUser).Error; err != nil {
			logger.Info("直推上级不存在，用户ID:", user.ZjFid, "错误:", err)
		} else {
			var superiorOrders []model.Order
			if err := h.DB.Model(&model.Order{}).Where("user_id = ? AND status = 8", user.ZjFid).Find(&superiorOrders).Error; err != nil {
				logger.Info("查询上级订单失败，用户ID:", user.ZjFid, "错误:", err)
			} else if len(superiorOrders) == 0 {
				logger.Info("直推上级没有质押中的订单，用户ID:", user.ZjFid)
			} else {
				scale := h.mintZjPro()
				releasePower := power * scale
				logger.Info("算力机推荐奖释放:", releasePower, "比例:", scale)
				totalReleased := h.releaseSuperiorPowerFromMultipleOrders(superiorOrders, releasePower, user.Username, productName, "推荐奖励", user.Id)
				logger.Info("算力机推荐奖实际释放:", totalReleased)
			}
		}
	}

	teamScale := h.mintTeamRewardScale()
	if err := h.distributeTeamRewardByWeakArea(int(user.Id), power, teamScale, productName); err != nil {
		logger.Error("算力机团队奖励失败:", err)
	}
	logger.Info("=== 算力机异步奖励处理完成 userID=", user.Id, " ===")
}

// processConsumeRewardsAsync 直推推荐奖 + 团队弱区奖（与自定义套餐/激活逻辑一致）
func (h *PaymentHandler) processConsumeRewardsAsync(user model.User, power float64, productName string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("异步 user rewards panic, userID:", user.Id, "panic:", r)
		}
	}()

	if strings.TrimSpace(productName) == "" {
		productName = "应用消费"
	}
	logger.Info("=== 开始异步处理上级奖励释放 ===")
	logger.Info("当前用户ID:", user.Id, "用户名:", user.Username)
	logger.Info("系统配置 - 直推加速比例 ZjPro:", h.App.SysConfig.ZjPro, "（间推奖励已关闭）原 jj_pro:", h.App.SysConfig.JjPro, "SanPro:", h.App.SysConfig.SanPro, "OtherPro:", h.App.SysConfig.OtherPro)
	logger.Info("投入算力:", power)

	var ids [1]int
	ids[0] = user.ZjFid
	logger.Info("上级ID数组:", ids)

	for i := 0; i < len(ids); i++ {
		logger.Info("=== 处理第", i+1, "代上级 ===")
		logger.Info("当前处理的上级ID:", ids[i])
		if ids[i] <= 0 {
			logger.Info("跳过无效上级ID:", ids[i])
			continue
		}

		var superiorUser model.User
		res := h.DB.Model(&model.User{}).Where("id = ?", ids[i]).First(&superiorUser)
		if res.Error != nil {
			logger.Info("上级用户不存在，用户ID:", ids[i], "错误:", res.Error)
			continue
		}
		logger.Info("处理上级用户:", superiorUser.Username, "代数为:", i+1, "用户ID:", superiorUser.Id)

		var superiorOrders []model.Order
		res = h.DB.Model(&model.Order{}).Where("user_id = ? AND status = 8", ids[i]).Find(&superiorOrders)
		if res.Error != nil {
			logger.Info("查询上级订单失败，用户ID:", ids[i], "错误:", res.Error)
			continue
		}
		if len(superiorOrders) == 0 {
			logger.Info("上级用户没有质押中的订单，用户ID:", ids[i])
			continue
		}
		logger.Info("上级用户质押中订单数量:", len(superiorOrders))

		for _, order := range superiorOrders {
			logger.Info("上级订单详情 - 订单号:", order.OrderNo, "power:", order.Power, "剩余算力:", order.NowPower, "已释放:", order.JsPower)
		}

		scale := h.App.SysConfig.ZjPro
		logger.Info("直推上级释放比例:", scale)

		releasePower := power * scale
		logger.Info("计算释放算力:", releasePower, "比例:", scale, "自定义投入:", power)

		logger.Info("=== 开始执行算力释放（多订单顺序释放） ===")
		totalReleased := h.releaseSuperiorPowerFromMultipleOrders(superiorOrders, releasePower, user.Username, productName, "推荐奖励", user.Id)
		logger.Info(fmt.Sprintf("=== 算力释放完成，实际释放: %.2f, 需要释放: %.2f ===", totalReleased, releasePower))
	}
	logger.Info("=== 自定义套餐上级奖励释放处理完成 ===")

	teamScale := h.teamRewardScale()
	if err := h.distributeTeamRewardByWeakArea(int(user.Id), power, teamScale, productName); err != nil {
		logger.Error("派发团队奖励失败:", err)
	}
	logger.Info("=== 自定义套餐异步奖励处理完成 userID=", user.Id, " ===")
}

// recordBinaryPlacement 将用户按照双轨弱区规则安置，并为本次业绩做团队累加
func (h *PaymentHandler) recordBinaryPlacement(user model.User, referrerID int, performance float64) error {
	// 没有推荐人则不做安置
	if referrerID <= 0 {
		return nil
	}

	return h.DB.Transaction(func(tx *gorm.DB) error {
		// 确保推荐人在安置树上
		if _, err := h.ensurePlacementNode(tx, uint(referrerID)); err != nil {
			return err
		}

		// 获取/创建当前用户的安置节点
		node, err := h.ensurePlacementNode(tx, user.Id)
		if err != nil {
			return err
		}

		var parentID int
		var parentSide string

		if node.ParentID == 0 {
			// 需要安置：判断是否是直推关系
			// 如果是直推（referrerID == user.ZjFid），使用弱区规则滑落
			// 如果不是直推（滑落下来的），永远只滑落到最左侧的左区
			isDirectReferral := (referrerID == int(user.ZjFid))
			// originReferrerID 表示本次安置的“实际推荐人”（根节点）
			// 用于实现：只有“实际推荐人”自己直推的用户，才允许占用其右区；
			// 从上级滑落到其名下的用户，只能继续往其左区链路滑落。
			// firstDirect：本次是否是该推荐人的“第一个直推用户”。
			// 第一个直推必须强制沿左区链路滑落（即使左子已被滑落者占用、右子为空），
			// 不得占用其右区；第二个及以后才回归弱区规则。
			firstDirect := isDirectReferral && h.countPlacedDirectReferrals(tx, referrerID) == 0
			parentNode, side, err := h.findPlacementSlot(tx, referrerID, isDirectReferral, referrerID, firstDirect)
			if err != nil {
				return err
			}

			// 绑定父子关系
			childColumn := "left_child_id"
			if side == "right" {
				childColumn = "right_child_id"
			}
			if err := tx.Model(&PlacementNode{}).Where("user_id = ?", parentNode.UserID).
				UpdateColumn(childColumn, user.Id).Error; err != nil {
				return err
			}
			if err := tx.Model(&PlacementNode{}).Where("user_id = ?", user.Id).
				UpdateColumn("parent_id", parentNode.UserID).Error; err != nil {
				return err
			}

			parentID = parentNode.UserID
			parentSide = side
		} else {
			// 已经安置过，找到父节点与方向
			var parentNode PlacementNode
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("user_id = ?", node.ParentID).First(&parentNode).Error; err != nil {
				return err
			}
			parentID = parentNode.UserID
			if parentNode.LeftChildID == node.UserID {
				parentSide = "left"
			} else if parentNode.RightChildID == node.UserID {
				parentSide = "right"
			}
		}

		if parentID == 0 || parentSide == "" {
			return nil
		}

		// 团队业绩自下向上累加
		return h.bubbleUpPerformance(tx, parentID, parentSide, performance)
	})
}

// ensurePlacementNode 获取或创建安置节点（带行级锁）
func (h *PaymentHandler) ensurePlacementNode(tx *gorm.DB, userID uint) (PlacementNode, error) {
	var node PlacementNode
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).First(&node).Error
	if err == nil {
		return node, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		node = PlacementNode{UserID: int(userID)}
		if err := tx.Create(&node).Error; err != nil {
			return node, err
		}
		return node, nil
	}
	return node, err
}

// findPlacementSlot 在以 rootID 为根的安置树中寻找插入位置
// isDirectReferral: true 表示直推关系（使用弱区规则），false 表示滑落关系（永远只滑落到最左侧）
// originReferrerID: 本次安置的实际推荐人 ID（根节点）。
// firstDirect: 本次是否是该推荐人的第一个直推用户；为 true 时强制沿左区链路滑落，绝不占用右区。
// 业务规则：
//   - 推荐人的第一个直推用户：必须强制沿左区链路滑落，不管左区多大，都不允许占用右区；
//   - 只有 originReferrerID 本人直推的用户，才允许占用其右区（按弱区规则左右滑落）；
//   - 从上级滑落到其名下的用户，只能继续沿着其左区链路滑落，不能占用其右区。
func (h *PaymentHandler) findPlacementSlot(tx *gorm.DB, rootID int, isDirectReferral bool, originReferrerID int, firstDirect bool) (PlacementNode, string, error) {
	node, err := h.ensurePlacementNode(tx, uint(rootID))
	if err != nil {
		return node, "", err
	}

	// 第一个直推：左子为空则占左；否则沿左区链路继续下滑，绝不占用右区
	if firstDirect {
		if node.LeftChildID == 0 {
			return node, "left", nil
		}
		return h.findPlacementSlot(tx, node.LeftChildID, isDirectReferral, originReferrerID, firstDirect)
	}

	// 有空位先左后右：
	// - 左区：任何情况下都可以优先占用；
	// - 右区：只有当当前节点就是实际推荐人（originReferrerID）时，才允许占用。
	if node.LeftChildID == 0 {
		return node, "left", nil
	}
	if node.RightChildID == 0 {
		// 只有实际推荐人节点，才允许把新用户直接安置在其右区
		if node.UserID == originReferrerID {
			return node, "right", nil
		}
		// 若不是实际推荐人，则继续沿着左区链路向下滑落
		return h.findPlacementSlot(tx, node.LeftChildID, isDirectReferral, originReferrerID, firstDirect)
	}

	// 左右都满，根据关系类型决定滑落规则
	if isDirectReferral {
		// 直推关系：
		//   - 对于实际推荐人节点：使用弱区规则（业绩小的一侧）；
		//   - 对于其下级节点：视为滑落关系，只允许继续走左区链路。
		if node.UserID == originReferrerID {
			// 实际推荐人：弱区规则
			if node.LeftPerf <= node.RightPerf {
				return h.findPlacementSlot(tx, node.LeftChildID, isDirectReferral, originReferrerID, firstDirect)
			}
			return h.findPlacementSlot(tx, node.RightChildID, isDirectReferral, originReferrerID, firstDirect)
		}
		// 非实际推荐人：当作滑落关系处理，只走左区
		return h.findPlacementSlot(tx, node.LeftChildID, isDirectReferral, originReferrerID, firstDirect)
	}

	// 滑落关系：永远只滑落到最左侧的左区
	return h.findPlacementSlot(tx, node.LeftChildID, isDirectReferral, originReferrerID, firstDirect)
}

// countPlacedDirectReferrals 统计某推荐人名下“已经安置进安置树”的直推用户数（不含滑落下来的用户）
// 判定条件：users.zj_fid = referrerID 且该用户已在安置树中真正挂上父节点（parent_id > 0）。
// 注意：ensurePlacementNode 会先给“正在安置的当前用户”建一条 parent_id=0 的节点，
// 若不排除 parent_id=0，会把本人也算进去，导致 firstDirect 永远为 false，第一个直推误落右区。
func (h *PaymentHandler) countPlacedDirectReferrals(tx *gorm.DB, referrerID int) int64 {
	var cnt int64
	tx.Table("chatgpt_users AS u").
		Joins("JOIN chatgpt_placement_nodes AS p ON p.user_id = u.id").
		Where("u.zj_fid = ? AND p.parent_id > 0", referrerID).
		Count(&cnt)
	return cnt
}

// bubbleUpPerformance 向上累加团队业绩
func (h *PaymentHandler) bubbleUpPerformance(tx *gorm.DB, startParentID int, side string, delta float64) error {
	currentID := startParentID
	currentSide := side

	for currentID != 0 {
		var node PlacementNode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", currentID).First(&node).Error; err != nil {
			return err
		}

		updates := map[string]any{
			"total_perf": gorm.Expr("total_perf + ?", delta),
		}
		if currentSide == "left" {
			updates["left_perf"] = gorm.Expr("left_perf + ?", delta)
		} else if currentSide == "right" {
			updates["right_perf"] = gorm.Expr("right_perf + ?", delta)
		}

		if err := tx.Model(&PlacementNode{}).
			Where("user_id = ?", currentID).
			Updates(updates).Error; err != nil {
			return err
		}

		// 准备上溯
		if node.ParentID == 0 {
			break
		}
		var parent PlacementNode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", node.ParentID).First(&parent).Error; err != nil {
			return err
		}
		if parent.LeftChildID == node.UserID {
			currentSide = "left"
		} else if parent.RightChildID == node.UserID {
			currentSide = "right"
		} else {
			currentSide = ""
		}
		currentID = node.ParentID
	}
	return nil
}

// collectPlacementUplines 获取安置树上方的父链（不含自己）
func (h *PaymentHandler) collectPlacementUplines(userID int, maxDepth int) ([]int, error) {
	var chain []int

	var node PlacementNode
	if err := h.DB.Where("user_id = ?", userID).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return chain, nil
		}
		return chain, err
	}

	parentID := node.ParentID
	depth := 0
	for parentID != 0 && depth < maxDepth {
		chain = append(chain, parentID)
		var parent PlacementNode
		if err := h.DB.Where("user_id = ?", parentID).First(&parent).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			return chain, err
		}
		parentID = parent.ParentID
		depth++
	}
	return chain, nil
}

func (h *PaymentHandler) teamRewardScale() float64 {
	n := h.App.SysConfig.TeamRewardPro
	// 填 0 就是关闭（按 0 发放）；仅负值才回退默认 0.2
	if n < 0 {
		return 0.2
	}
	return n
}

func (h *PaymentHandler) teamRewardMeasureLayerDepth() int {
	n := h.App.SysConfig.TeamRewardMeasureLayerDepth
	if n <= 0 {
		return 4
	}
	return n
}

func (h *PaymentHandler) teamRewardLayer4WidthLimit() int {
	n := h.App.SysConfig.TeamRewardLayer4WidthLimit
	if n >= 2 {
		return n
	}
	return 8
}

// placementIsBinary 左右区均有直属下级（真正开出第二个区）
func placementIsBinary(n PlacementNode) bool {
	return n.LeftChildID != 0 && n.RightChildID != 0
}

// subjectLayer4WidthGate 以主体为根，测算其弱区向下第 N 逻辑层的宽度（量碰）
// 逻辑层：单边下行（一条线）不加深，整条线相对上一分叉同属一层；仅父节点开出双区时子节点进入下一层
type subjectLayer4WidthGate struct {
	MeasureLayerDepth   int
	DepthToRecharge     int
	ReachedMeasureLayer bool
	LayerNodeUserID     int   // 该层仅 1 个节点时记录 ID
	LayerNodeIDs        []int // 弱区第 N 逻辑层入口节点 ID（不含同层单边续链）
	LayerWidth          int   // 弱区第 N 逻辑层宽度
	WidthLimit          int
	BlocksTeamReward    bool // 弱区第 N 层节点数 > 阈值：主体及安置树往上直至根不发团队奖
	WeakSide            string
}

// pathBelowSubject 主体朝向充值侧的下级链（不含主体本人；第1层=直推下级）
func pathBelowSubject(subjectID int, fullPath []int) []int {
	if len(fullPath) == 0 {
		return nil
	}
	if fullPath[0] == subjectID {
		return fullPath[1:]
	}
	return fullPath
}

// subjectWeakChildID 主体当前弱区（小区）直属下级：左右业绩较小侧；相等取左
func subjectWeakChildID(node PlacementNode) (childID int, side string) {
	if node.LeftPerf < node.RightPerf {
		return node.LeftChildID, "left"
	}
	if node.RightPerf < node.LeftPerf {
		return node.RightChildID, "right"
	}
	// 相等：与发奖侧惯例一致，视左区为弱区
	return node.LeftChildID, "left"
}

// placementNodeIDsAtWeakLayerBelowSubject 主体弱区下第 layerDepth 逻辑层的入口节点
// 例：1 双区，弱区 A→A2→A3 单边下行 → A/A2/A3 同属第1层，宽度只计入口 A（=1）
func (h *PaymentHandler) placementNodeIDsAtWeakLayerBelowSubject(subjectID, layerDepth int) (ids []int, weakSide string) {
	if layerDepth <= 0 {
		return nil, ""
	}
	var subjectNode PlacementNode
	if err := h.DB.Where("user_id = ?", subjectID).First(&subjectNode).Error; err != nil {
		return nil, ""
	}
	weakChildID, weakSide := subjectWeakChildID(subjectNode)
	if weakChildID == 0 {
		return nil, weakSide
	}

	type item struct {
		id    int
		layer int
	}
	queue := []item{{id: weakChildID, layer: 1}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.layer > layerDepth {
			continue
		}
		if cur.layer == layerDepth {
			ids = append(ids, cur.id)
			// 同层单边续链不计入宽度，也不再向下扩展（下一逻辑层不会从续链「入口」重复产生）
			continue
		}
		var n PlacementNode
		if err := h.DB.Where("user_id = ?", cur.id).First(&n).Error; err != nil {
			continue
		}
		left, right := n.LeftChildID, n.RightChildID
		if left != 0 && right != 0 {
			// 开出双区：左右子进入下一逻辑层
			queue = append(queue, item{id: left, layer: cur.layer + 1}, item{id: right, layer: cur.layer + 1})
			continue
		}
		// 单边下行：子节点仍属当前逻辑层，继续往下找目标层
		if left != 0 {
			queue = append(queue, item{id: left, layer: cur.layer})
		}
		if right != 0 {
			queue = append(queue, item{id: right, layer: cur.layer})
		}
	}
	return ids, weakSide
}

// placementNodeIDsAtLayerBelowSubject 保留旧接口：整层左右合计，同样按逻辑层（单边同层）计算入口宽度
func (h *PaymentHandler) placementNodeIDsAtLayerBelowSubject(subjectID, layerDepth int) []int {
	if layerDepth <= 0 {
		return nil
	}
	var subjectNode PlacementNode
	if err := h.DB.Where("user_id = ?", subjectID).First(&subjectNode).Error; err != nil {
		return nil
	}
	type item struct {
		id    int
		layer int
	}
	var queue []item
	if subjectNode.LeftChildID != 0 {
		queue = append(queue, item{id: subjectNode.LeftChildID, layer: 1})
	}
	if subjectNode.RightChildID != 0 {
		queue = append(queue, item{id: subjectNode.RightChildID, layer: 1})
	}
	var ids []int
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.layer > layerDepth {
			continue
		}
		if cur.layer == layerDepth {
			ids = append(ids, cur.id)
			continue
		}
		var n PlacementNode
		if err := h.DB.Where("user_id = ?", cur.id).First(&n).Error; err != nil {
			continue
		}
		left, right := n.LeftChildID, n.RightChildID
		if left != 0 && right != 0 {
			queue = append(queue, item{id: left, layer: cur.layer + 1}, item{id: right, layer: cur.layer + 1})
			continue
		}
		if left != 0 {
			queue = append(queue, item{id: left, layer: cur.layer})
		}
		if right != 0 {
			queue = append(queue, item{id: right, layer: cur.layer})
		}
	}
	return ids
}

// measureSubjectLayer4WidthGate 主体→充值逻辑层深>=N 时，统计主体弱区第 N 逻辑层入口宽度；仅当宽度 > 阈值才拦截
func (h *PaymentHandler) measureSubjectLayer4WidthGate(subjectID, rechargeUserID int) subjectLayer4WidthGate {
	measureDepth := h.teamRewardMeasureLayerDepth()
	widthLimit := h.teamRewardLayer4WidthLimit()
	g := subjectLayer4WidthGate{
		MeasureLayerDepth: measureDepth,
		WidthLimit:        widthLimit,
	}
	g.DepthToRecharge = h.calculatePlacementDepthFromRecharge(rechargeUserID, subjectID)
	if g.DepthToRecharge < measureDepth {
		return g
	}
	layerNodes, weakSide := h.placementNodeIDsAtWeakLayerBelowSubject(subjectID, measureDepth)
	g.WeakSide = weakSide
	g.LayerNodeIDs = layerNodes
	g.LayerWidth = len(layerNodes)
	if g.LayerWidth <= 0 {
		return g
	}
	g.ReachedMeasureLayer = true
	if len(layerNodes) == 1 {
		g.LayerNodeUserID = layerNodes[0]
	}
	// 等于阈值不拦截；只有严格大于才拦截（如 4,4 时弱区第4层有4人不拦，有5人才拦）
	g.BlocksTeamReward = g.LayerWidth > widthLimit
	return g
}

// markSubjectAndPlacementAncestorsBlocked 主体本人 + 沿安置树 ParentID 往上直至根，均标记不发团队奖
func (h *PaymentHandler) markSubjectAndPlacementAncestorsBlocked(blocked map[int]bool, subjectID int) {
	blocked[subjectID] = true
	var node PlacementNode
	if err := h.DB.Where("user_id = ?", subjectID).First(&node).Error; err != nil {
		return
	}
	pid := node.ParentID
	const maxDepth = 200
	for depth := 0; pid != 0 && depth < maxDepth; depth++ {
		blocked[pid] = true
		var parent PlacementNode
		if err := h.DB.Where("user_id = ?", pid).First(&parent).Error; err != nil {
			break
		}
		pid = parent.ParentID
	}
}

// applyLayer4WidthBlocks 按路径上每个主体测算弱区第 N 层宽度；超过阈值则合并标记主体及安置树全部上级
func (h *PaymentHandler) applyLayer4WidthBlocks(blocked map[int]bool, releasePath []int, rechargeUserID int) {
	for _, subjectID := range releasePath {
		gate := h.measureSubjectLayer4WidthGate(subjectID, rechargeUserID)
		logger.Info(fmt.Sprintf("【量碰-主体弱区第%d逻辑层】主体=%d 弱区=%s 充值逻辑深度=%d 弱区第%d层入口=%v 宽=%d 阈值(>)%d 拦截=%v",
			gate.MeasureLayerDepth, subjectID, gate.WeakSide, gate.DepthToRecharge, gate.MeasureLayerDepth,
			gate.LayerNodeIDs, gate.LayerWidth, gate.WidthLimit, gate.BlocksTeamReward))
		if !gate.BlocksTeamReward {
			continue
		}
		h.markSubjectAndPlacementAncestorsBlocked(blocked, subjectID)
		logger.Info(fmt.Sprintf("【量碰拦截】主体=%d 弱区第%d层宽>%d，本人及安置树往上(P→Q→…→根)不发团队奖",
			subjectID, gate.MeasureLayerDepth, gate.WidthLimit))
	}
}

// distributeTeamRewardByWeakArea 弱区团队奖 + 量碰：以各收奖候选为主体，测算其弱区第 N 层宽度；仅当宽度>阈值时主体及安置树往上直至根不发
func (h *PaymentHandler) distributeTeamRewardByWeakArea(currentUserID int, performance float64, rewardScale float64, productName string) error {

	measureDepth := h.teamRewardMeasureLayerDepth()
	widthLimit := h.teamRewardLayer4WidthLimit()
	logger.Info("=== 【量碰】distributeTeamRewardByWeakArea 开始 ===")
	logger.Info("【量碰】充值用户ID:", currentUserID, "业绩:", performance, "奖励比例:", rewardScale,
		"测算逻辑层深(主体弱区):", measureDepth, "弱区第", measureDepth, "逻辑层宽度阈值(>):", widthLimit)

	// 为了防止同一次团队奖励中，同一个用户在不同路径上被重复发放奖励，这里做一次去重控制
	// key: userID, value: 是否已经给该用户发放过本次团队奖励
	rewardedUsers := make(map[int]bool)
	// 本次充值触发的量碰拦截：主体弱区第 N 层宽>阈值时，主体及安置树全部上级直至根不发团队奖
	blockedByLayer4 := make(map[int]bool)

	// 获取当前充值用户信息（团队奖流水需写入下级用户ID/用户名，供当晚管理奖静态关联）
	var investor model.User
	investorUsername := ""
	if err := h.DB.Select("id, username").Where("id = ?", currentUserID).First(&investor).Error; err == nil {
		investorUsername = investor.Username
	}

	// 获取当前用户节点
	var currentNode PlacementNode
	if err := h.DB.Where("user_id = ?", currentUserID).First(&currentNode).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Info("当前用户不在安置树中，跳过团队奖励")
			return nil
		}
		return err
	}

	// 从当前用户的父节点开始向上查找
	// 需要跟踪当前用户所在的子树，以便正确判断业绩累加到哪个区
	checkingNodeID := currentUserID // 当前检查的节点ID（从当前用户开始）
	parentID := currentNode.ParentID
	weakAreaEventNo := 0 // 向上遍历过程中触发的弱区对碰事件序号

	for parentID != 0 {
		var parentNode PlacementNode
		if err := h.DB.Where("user_id = ?", parentID).First(&parentNode).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			return err
		}

		logger.Info("检查父节点，用户ID:", parentID, "左区业绩:", parentNode.LeftPerf, "右区业绩:", parentNode.RightPerf, "检查节点ID:", checkingNodeID)

		// 关键检查：只有当父节点是完整的二叉树节点（左右都有子节点）时，才发放团队奖励
		isCompleteBinaryNode := parentNode.LeftChildID != 0 && parentNode.RightChildID != 0
		if !isCompleteBinaryNode {
			logger.Info("父节点不是完整的二叉树节点（左子节点:", parentNode.LeftChildID, "右子节点:", parentNode.RightChildID, "），跳过团队奖励，继续向上查找")
			checkingNodeID = parentID
			parentID = parentNode.ParentID
			continue
		}

		// 判断checkingNodeID是在父节点的左区还是右区
		var checkingSide string
		if parentNode.LeftChildID == checkingNodeID {
			checkingSide = "left"
		} else if parentNode.RightChildID == checkingNodeID {
			checkingSide = "right"
		} else {
			// checkingNodeID不是父节点的直接子节点，需要判断它在哪个子树
			if h.isDescendant(parentNode.LeftChildID, checkingNodeID) {
				checkingSide = "left"
			} else if h.isDescendant(parentNode.RightChildID, checkingNodeID) {
				checkingSide = "right"
			} else {
				logger.Info("检查节点不在父节点的子树中，跳过")
				checkingNodeID = parentID
				parentID = parentNode.ParentID
				continue
			}
		}

		// 计算累加当前用户充值业绩后的区业绩
		// 注意：此时业绩可能已经累加过了，需要先减去本次业绩得到累加前的业绩，然后再计算累加后的业绩
		var leftPerfBefore, rightPerfBefore float64
		var leftPerfAfter, rightPerfAfter float64

		// 先计算累加前的业绩（从当前业绩减去本次投入）
		if checkingSide == "left" {
			leftPerfBefore = parentNode.LeftPerf - performance
			rightPerfBefore = parentNode.RightPerf
			// 累加后的业绩
			leftPerfAfter = parentNode.LeftPerf
			rightPerfAfter = parentNode.RightPerf
		} else {
			leftPerfBefore = parentNode.LeftPerf
			rightPerfBefore = parentNode.RightPerf - performance
			// 累加后的业绩
			leftPerfAfter = parentNode.LeftPerf
			rightPerfAfter = parentNode.RightPerf
		}

		// 计算累加前两个区的差的绝对值
		diffBefore := math.Abs(leftPerfBefore - rightPerfBefore)

		logger.Info("累加前业绩 - 左区:", leftPerfBefore, "右区:", rightPerfBefore, "差的绝对值:", diffBefore)
		logger.Info("累加后业绩 - 左区:", leftPerfAfter, "右区:", rightPerfAfter, "检查节点在:", checkingSide)

		// 判断检查节点所在区是否是最小区（弱区）
		isWeakArea := false
		if checkingSide == "left" && leftPerfAfter <= rightPerfAfter {
			isWeakArea = true
		} else if checkingSide == "right" && rightPerfAfter <= leftPerfAfter {
			isWeakArea = true
		}

		// 如果不是最小区，但左右业绩相等，且当前区是左区，也应该释放奖励
		shouldReleaseReward := false
		if isWeakArea {
			shouldReleaseReward = true
		} else if leftPerfAfter == rightPerfAfter && checkingSide == "left" {
			// 左右业绩相等，且当前区是左区，释放奖励
			shouldReleaseReward = true
			logger.Info("左右业绩相等，且当前区是左区，释放奖励")
		}

		logger.Info("shouldReleaseReward", shouldReleaseReward)

		if !shouldReleaseReward {
			logger.Info("检查节点所在区不是最小区(大区)，本层不发团队奖，继续向上查找；父节点ID:", parentID, "检查区:", checkingSide)
			checkingNodeID = parentID
			parentID = parentNode.ParentID
			continue
		}

		weakAreaEventNo++
		if isWeakArea {
			logger.Info("【量碰事件】第", weakAreaEventNo, "次弱区对碰，父节点ID:", parentID, "检查子树节点:", checkingNodeID, "所在区:", checkingSide, "为小区")
		} else {
			logger.Info("【量碰事件】第", weakAreaEventNo, "次弱区对碰(左右业绩相等取左区)，父节点ID:", parentID)
		}
		releasePath := h.findPathFromTo(parentID, currentUserID)
		h.applyLayer4WidthBlocks(blockedByLayer4, releasePath, currentUserID)
		logger.Info("【量碰】释放路径(父→充值用户，不含充值用户):", releasePath, "路径人数:", len(releasePath),
			"弱区第", measureDepth, "层宽>", widthLimit, "累计拦截(含安置树上级):", blockedByLayer4)

		for pathIdx, userID := range releasePath {
			layerNo := pathIdx + 1
			upDepth := h.calculatePlacementDepthFromRecharge(currentUserID, userID)
			logger.Info(fmt.Sprintf("【量碰核算】收奖用户=%d 路径层号=%d号(近父→近充值) 逻辑深度=%d",
				userID, layerNo, upDepth))
			if blockedByLayer4[userID] {
				logger.Info(fmt.Sprintf("【量碰跳过-主体弱区第%d逻辑层超宽】收奖用户=%d 其弱区第%d逻辑层宽度>%d，或其为某主体安置树上级，不发团队奖",
					measureDepth, userID, measureDepth, widthLimit))
				continue
			}
			// 获取该用户的节点信息
			var pathUserNode PlacementNode
			if err := h.DB.Where("user_id = ?", userID).First(&pathUserNode).Error; err != nil {
				logger.Info("路径用户节点不存在，跳过，用户ID:", userID)
				continue
			}

			// 如果该用户在本次团队奖励中已经拿过奖励，则跳过，避免重复释放
			if rewardedUsers[userID] {
				logger.Info("团队奖励已发放过，跳过重复用户，用户ID:", userID)
				continue
			}

			// 未激活用户不参与团队奖励（与层数上限无关）
			var pathUser model.User
			if err := h.DB.Where("id = ?", userID).First(&pathUser).Error; err != nil {
				logger.Info("团队奖励跳过，用户不存在，用户ID:", userID)
				continue
			}
			if pathUser.IsJh != 1 {
				logger.Info("团队奖励跳过，用户未激活，用户ID:", userID)
				continue
			}

			receiverScale := service.TeamRewardScaleForReceiver(h.DB, &pathUser, h.App.SysConfig, rewardScale)
			if receiverScale < 0 {
				logger.Info("团队奖励跳过，用户不符合会员档位/截止规则，用户ID:", userID)
				continue
			}

			// 获取该用户的所有质押中订单
			var userOrders []model.Order
			res := h.DB.Model(&model.Order{}).Where("user_id = ? AND status = 8", userID).Find(&userOrders)
			if res.Error != nil || len(userOrders) == 0 {
				logger.Info("团队奖励跳过，无在投订单，用户ID:", userID)
				continue
			}

			var rewardAmount float64
			{
				// 小区按本次业绩比例奖励；小区变大区按累加前差额比例奖励（单区节点同样按左右业绩判断）
				// 判断当前投入用户相对于该节点是在左区还是右区
				var pathUserCheckingSide string
				if pathUserNode.LeftChildID == currentUserID {
					pathUserCheckingSide = "left"
				} else if pathUserNode.RightChildID == currentUserID {
					pathUserCheckingSide = "right"
				} else {
					// 当前投入用户不是该用户的直接子节点，需要判断它在哪个子树
					if h.isDescendant(pathUserNode.LeftChildID, currentUserID) {
						pathUserCheckingSide = "left"
					} else if h.isDescendant(pathUserNode.RightChildID, currentUserID) {
						pathUserCheckingSide = "right"
					} else {
						logger.Info("当前投入用户不在路径用户的子树中，跳过，路径用户ID:", userID)
						continue
					}
				}

				// 计算该节点累加前、累加后的区业绩
				var pathUserLeftPerfBefore, pathUserRightPerfBefore float64
				var pathUserLeftPerfAfter, pathUserRightPerfAfter float64

				if pathUserCheckingSide == "left" {
					pathUserLeftPerfBefore = pathUserNode.LeftPerf - performance
					pathUserRightPerfBefore = pathUserNode.RightPerf
					pathUserLeftPerfAfter = pathUserNode.LeftPerf
					pathUserRightPerfAfter = pathUserNode.RightPerf
				} else {
					pathUserLeftPerfBefore = pathUserNode.LeftPerf
					pathUserRightPerfBefore = pathUserNode.RightPerf - performance
					pathUserLeftPerfAfter = pathUserNode.LeftPerf
					pathUserRightPerfAfter = pathUserNode.RightPerf
				}

				// 计算该节点累加前两个区的差的绝对值
				pathUserDiffBefore := math.Abs(pathUserLeftPerfBefore - pathUserRightPerfBefore)

				// 当前充值区累加前后是否是大区/小区
				var pathUserSidePerfBefore, pathUserOtherPerfBefore, pathUserSidePerfAfter, pathUserOtherPerfAfter float64
				if pathUserCheckingSide == "left" {
					pathUserSidePerfBefore = pathUserLeftPerfBefore
					pathUserOtherPerfBefore = pathUserRightPerfBefore
					pathUserSidePerfAfter = pathUserLeftPerfAfter
					pathUserOtherPerfAfter = pathUserRightPerfAfter
				} else {
					pathUserSidePerfBefore = pathUserRightPerfBefore
					pathUserOtherPerfBefore = pathUserLeftPerfBefore
					pathUserSidePerfAfter = pathUserRightPerfAfter
					pathUserOtherPerfAfter = pathUserLeftPerfAfter
				}

				logger.Info("团队奖励节点用户ID:", userID, "累加前区业绩(当前区/另一区):", pathUserSidePerfBefore, "/", pathUserOtherPerfBefore, "累加后:", pathUserSidePerfAfter, "/", pathUserOtherPerfAfter)

				// 新增兼容规则：
				// 1) 累加前已是大区：不发放团队奖
				// 2) 累加前是小区，累加后变大区：使用累加前差值 * 奖励比例
				// 3) 累加前是小区，累加后仍是小区：使用本次投入金额 * 奖励比例
				if pathUserSidePerfBefore > pathUserOtherPerfBefore {
					logger.Info("团队奖励用户ID:", userID, "充值前所在区已是大区，本次不发团队奖")
					continue
				}

				useDiffBefore := pathUserSidePerfAfter > pathUserOtherPerfAfter
				if useDiffBefore {
					rewardAmount = pathUserDiffBefore * receiverScale
					logger.Info("团队奖励用户ID:", userID, "小区变大区，使用累加前差值计算奖励:", rewardAmount, "差值:", pathUserDiffBefore, "比例:", receiverScale)
				} else {
					rewardAmount = performance * receiverScale
					logger.Info("团队奖励用户ID:", userID, "仍为小区，使用本次投入计算奖励:", rewardAmount, "投入金额:", performance, "比例:", receiverScale)
				}
			}

			if rewardAmount <= 0 {
				logger.Info("【量碰跳过】计算奖励为0，用户ID:", userID, "层号:", layerNo)
				continue
			}

			rewardAmount = h.clampTeamRewardByWeeklyCap(userID, rewardAmount)
			if rewardAmount <= 0 {
				logger.Info("【量碰跳过-周封顶】用户ID:", userID, "本周团队奖励已达封顶")
				continue
			}

			logger.Info(fmt.Sprintf("【量碰发放】%d号层有效 收奖用户=%d 应释放=%.2f 充值用户=%d",
				layerNo, userID, rewardAmount, currentUserID))
			totalReleased := h.releaseSuperiorPowerFromMultipleOrders(userOrders, rewardAmount, investorUsername, productName, "团队奖励", uint(currentUserID))
			logger.Info(fmt.Sprintf("【量碰完成】收奖用户=%d %d号层 应发=%.2f 实发=%.2f 烧伤=%.2f",
				userID, layerNo, rewardAmount, totalReleased, rewardAmount-totalReleased))

			// 标记该用户已经领取过本次团队奖励
			rewardedUsers[userID] = true
		}

		// 找到弱区后，继续向上查找（可能还有更上层的弱区）
		checkingNodeID = parentID
		parentID = parentNode.ParentID
	}

	logger.Info("=== 【量碰】distributeTeamRewardByWeakArea 完成 充值用户=", currentUserID, "弱区事件次数=", weakAreaEventNo, "===")
	return nil
}

// calculatePlacementDepthFromRecharge 充值用户相对收奖用户的逻辑层深
// 单边下行不加深（A→A2→A3 相对 1 均为第 1 层）；仅途经「左右双区都已开」的节点时层深 +1
// 不在上级链上返回 999；本人返回 0
func (h *PaymentHandler) calculatePlacementDepthFromRecharge(rechargeUserID int, rewardUserID int) int {
	if rechargeUserID == rewardUserID {
		return 0
	}
	// 自下而上收集到收奖用户为止的链（含充值用户，不含收奖用户）
	var upChain []int
	currentID := rechargeUserID
	found := false
	const maxDepth = 200
	for depth := 0; currentID != 0 && depth < maxDepth; depth++ {
		if currentID == rewardUserID {
			found = true
			break
		}
		var node PlacementNode
		if err := h.DB.Where("user_id = ?", currentID).First(&node).Error; err != nil {
			return 999
		}
		upChain = append(upChain, currentID)
		if node.ParentID == 0 {
			break
		}
		currentID = node.ParentID
	}
	if !found {
		return 999
	}
	if len(upChain) == 0 {
		return 0
	}
	// 转为自上而下：收奖用户下方第一人 … 充值用户
	pathDown := make([]int, len(upChain))
	for i := range upChain {
		pathDown[i] = upChain[len(upChain)-1-i]
	}
	layer := 1
	for i := 1; i < len(pathDown); i++ {
		parentID := pathDown[i-1]
		var parent PlacementNode
		if err := h.DB.Where("user_id = ?", parentID).First(&parent).Error; err != nil {
			break
		}
		if placementIsBinary(parent) {
			layer++
		}
	}
	return layer
}

// findPathFromTo 找到从fromUserID到toUserID的路径（不包括toUserID）
// 返回从fromUserID开始，沿着到toUserID路径上的所有节点（不包括toUserID）
func (h *PaymentHandler) findPathFromTo(fromUserID int, toUserID int) []int {
	var path []int

	// 如果fromUserID就是toUserID，返回空路径
	if fromUserID == toUserID {
		return path
	}

	// 从fromUserID开始，沿着到toUserID的路径向下查找
	currentID := fromUserID
	maxDepth := 100 // 防止无限循环
	depth := 0

	for currentID != 0 && currentID != toUserID && depth < maxDepth {
		path = append(path, currentID)

		var node PlacementNode
		if err := h.DB.Where("user_id = ?", currentID).First(&node).Error; err != nil {
			break
		}

		// 判断toUserID在左子树还是右子树
		if node.LeftChildID == toUserID {
			// toUserID是直接左子节点，路径结束
			break
		} else if node.RightChildID == toUserID {
			// toUserID是直接右子节点，路径结束
			break
		} else if node.LeftChildID != 0 && h.isDescendant(node.LeftChildID, toUserID) {
			currentID = node.LeftChildID
		} else if node.RightChildID != 0 && h.isDescendant(node.RightChildID, toUserID) {
			currentID = node.RightChildID
		} else {
			// 找不到路径
			break
		}
		depth++
	}

	return path
}

// isDescendant 判断descendantID是否是ancestorID的后代
func (h *PaymentHandler) isDescendant(ancestorID int, descendantID int) bool {
	if ancestorID == descendantID {
		return true
	}

	var node PlacementNode
	if err := h.DB.Where("user_id = ?", ancestorID).First(&node).Error; err != nil {
		return false
	}

	if node.LeftChildID != 0 && h.isDescendant(node.LeftChildID, descendantID) {
		return true
	}
	if node.RightChildID != 0 && h.isDescendant(node.RightChildID, descendantID) {
		return true
	}

	return false
}

// getUserRewardLayerLimit 获取用户的团队奖励层级限制
// 返回层数：-1 未激活；10 激活；20～100 对应 1～9 钻
// 性能：直推钻级统计改用库内已存 lev 一次聚合，避免远程库上递归 N+1 超时
func (h *PaymentHandler) getUserRewardLayerLimit(userID int) int {
	var user model.User
	if err := h.DB.Select("id, is_jh").Where("id = ?", userID).First(&user).Error; err != nil {
		return -1
	}
	if user.IsJh != 1 {
		return -1
	}

	var cnt struct {
		C1 int64 `gorm:"column:c1"`
		C2 int64 `gorm:"column:c2"`
		C3 int64 `gorm:"column:c3"`
		C4 int64 `gorm:"column:c4"`
		C5 int64 `gorm:"column:c5"`
		C6 int64 `gorm:"column:c6"`
		C7 int64 `gorm:"column:c7"`
		C8 int64 `gorm:"column:c8"`
	}
	_ = h.DB.Model(&model.User{}).
		Select(`
			COALESCE(SUM(CASE WHEN lev >= 1 THEN 1 ELSE 0 END), 0) as c1,
			COALESCE(SUM(CASE WHEN lev >= 2 THEN 1 ELSE 0 END), 0) as c2,
			COALESCE(SUM(CASE WHEN lev >= 3 THEN 1 ELSE 0 END), 0) as c3,
			COALESCE(SUM(CASE WHEN lev >= 4 THEN 1 ELSE 0 END), 0) as c4,
			COALESCE(SUM(CASE WHEN lev >= 5 THEN 1 ELSE 0 END), 0) as c5,
			COALESCE(SUM(CASE WHEN lev >= 6 THEN 1 ELSE 0 END), 0) as c6,
			COALESCE(SUM(CASE WHEN lev >= 7 THEN 1 ELSE 0 END), 0) as c7,
			COALESCE(SUM(CASE WHEN lev >= 8 THEN 1 ELSE 0 END), 0) as c8
		`).
		Where("zj_fid = ? AND is_jh = 1", userID).
		Scan(&cnt).Error

	// 9钻→1钻：与原规则人数门槛一致，依据直推已落库的 lev
	if cnt.C8 >= 5 {
		return 100
	}
	if cnt.C7 >= 5 {
		return 90
	}
	if cnt.C6 >= 4 {
		return 80
	}
	if cnt.C5 >= 4 {
		return 70
	}
	if cnt.C4 >= 3 {
		return 60
	}
	if cnt.C3 >= 3 {
		return 50
	}
	if cnt.C2 >= 2 {
		return 40
	}
	if cnt.C1 >= 2 {
		return 30
	}
	if h.isDiamond1(userID) {
		return 20
	}
	return 10
}

// isDiamond1 判断用户是否是1钻（小区业绩 >= 100万算力）
func (h *PaymentHandler) isDiamond1(userID int) bool {
	var node PlacementNode
	if err := h.DB.Where("user_id = ?", userID).First(&node).Error; err != nil {
		return false
	}

	// 计算小区业绩（左右区中较小的那个）
	minPerf := node.LeftPerf
	if node.RightPerf < minPerf {
		minPerf = node.RightPerf
	}

	// 1钻条件：小区业绩 >= 100万算力
	isDiamond1 := minPerf >= 1000000.0
	if isDiamond1 {
		logger.Info("用户是1钻，用户ID:", userID, "小区业绩:", minPerf)
	}
	return isDiamond1
}

// isDiamond2 判断用户是否是2钻（有2个1钻直接推下级）
func (h *PaymentHandler) isDiamond2(userID int) bool {
	// 查询所有直推下级（zj_fid = userID）
	var directReferrals []model.User
	if err := h.DB.Where("zj_fid = ?", userID).Find(&directReferrals).Error; err != nil {
		logger.Info("查询直推下级失败，用户ID:", userID, "错误:", err)
		return false
	}

	// 统计直推下级中1钻的数量
	diamond1Count := 0
	for _, referral := range directReferrals {
		if h.isDiamond1(int(referral.Id)) {
			diamond1Count++
		}
	}

	// 2钻条件：有2个1钻直推下级
	isDiamond2 := diamond1Count >= 2
	if isDiamond2 {
		logger.Info("用户是2钻，用户ID:", userID, "1钻直推下级数量:", diamond1Count)
	}
	return isDiamond2
}

// isDiamond3 判断用户是否是3钻（有2个2钻直接推下级）
func (h *PaymentHandler) isDiamond3(userID int) bool {
	// 查询所有直推下级（zj_fid = userID）
	var directReferrals []model.User
	if err := h.DB.Where("zj_fid = ?", userID).Find(&directReferrals).Error; err != nil {
		logger.Info("查询直推下级失败，用户ID:", userID, "错误:", err)
		return false
	}

	// 统计直推下级中2钻的数量
	diamond2Count := 0
	for _, referral := range directReferrals {
		if h.isDiamond2(int(referral.Id)) {
			diamond2Count++
		}
	}

	// 3钻条件：有2个2钻直推下级
	isDiamond3 := diamond2Count >= 2
	if isDiamond3 {
		logger.Info("用户是3钻，用户ID:", userID, "2钻直推下级数量:", diamond2Count)
	}
	return isDiamond3
}

// isDiamond4 判断用户是否是4钻（有3个3钻直推下级）
func (h *PaymentHandler) isDiamond4(userID int) bool {
	// 查询所有直推下级（zj_fid = userID）
	var directReferrals []model.User
	if err := h.DB.Where("zj_fid = ?", userID).Find(&directReferrals).Error; err != nil {
		logger.Info("查询直推下级失败，用户ID:", userID, "错误:", err)
		return false
	}

	// 统计直推下级中3钻的数量
	diamond3Count := 0
	for _, referral := range directReferrals {
		if h.isDiamond3(int(referral.Id)) {
			diamond3Count++
		}
	}

	// 4钻条件：有3个3钻直推下级
	isDiamond4 := diamond3Count >= 3
	if isDiamond4 {
		logger.Info("用户是4钻，用户ID:", userID, "3钻直推下级数量:", diamond3Count)
	}
	return isDiamond4
}

// isDiamond5 判断用户是否是5钻（有3个4钻直推下级）
func (h *PaymentHandler) isDiamond5(userID int) bool {
	// 查询所有直推下级（zj_fid = userID）
	var directReferrals []model.User
	if err := h.DB.Where("zj_fid = ?", userID).Find(&directReferrals).Error; err != nil {
		logger.Info("查询直推下级失败，用户ID:", userID, "错误:", err)
		return false
	}

	// 统计直推下级中4钻的数量
	diamond4Count := 0
	for _, referral := range directReferrals {
		if h.isDiamond4(int(referral.Id)) {
			diamond4Count++
		}
	}

	// 5钻条件：有3个4钻直推下级
	isDiamond5 := diamond4Count >= 3
	if isDiamond5 {
		logger.Info("用户是5钻，用户ID:", userID, "4钻直推下级数量:", diamond4Count)
	}
	return isDiamond5
}

// isDiamond6 判断用户是否是6钻（有4个5钻直推下级）
func (h *PaymentHandler) isDiamond6(userID int) bool {
	// 查询所有直推下级（zj_fid = userID）
	var directReferrals []model.User
	if err := h.DB.Where("zj_fid = ?", userID).Find(&directReferrals).Error; err != nil {
		logger.Info("查询直推下级失败，用户ID:", userID, "错误:", err)
		return false
	}

	// 统计直推下级中5钻的数量
	diamond5Count := 0
	for _, referral := range directReferrals {
		if h.isDiamond5(int(referral.Id)) {
			diamond5Count++
		}
	}

	// 6钻条件：有4个5钻直推下级
	isDiamond6 := diamond5Count >= 4
	if isDiamond6 {
		logger.Info("用户是6钻，用户ID:", userID, "5钻直推下级数量:", diamond5Count)
	}
	return isDiamond6
}

// isDiamond7 判断用户是否是7钻（有4个6钻直推下级）
func (h *PaymentHandler) isDiamond7(userID int) bool {
	// 查询所有直推下级（zj_fid = userID）
	var directReferrals []model.User
	if err := h.DB.Where("zj_fid = ?", userID).Find(&directReferrals).Error; err != nil {
		logger.Info("查询直推下级失败，用户ID:", userID, "错误:", err)
		return false
	}

	// 统计直推下级中6钻的数量
	diamond6Count := 0
	for _, referral := range directReferrals {
		if h.isDiamond6(int(referral.Id)) {
			diamond6Count++
		}
	}

	// 7钻条件：有4个6钻直推下级
	isDiamond7 := diamond6Count >= 4
	if isDiamond7 {
		logger.Info("用户是7钻，用户ID:", userID, "6钻直推下级数量:", diamond6Count)
	}
	return isDiamond7
}

// isDiamond8 判断用户是否是8钻（有5个7钻直推下级）
func (h *PaymentHandler) isDiamond8(userID int) bool {
	// 查询所有直推下级（zj_fid = userID）
	var directReferrals []model.User
	if err := h.DB.Where("zj_fid = ?", userID).Find(&directReferrals).Error; err != nil {
		logger.Info("查询直推下级失败，用户ID:", userID, "错误:", err)
		return false
	}

	// 统计直推下级中7钻的数量
	diamond7Count := 0
	for _, referral := range directReferrals {
		if h.isDiamond7(int(referral.Id)) {
			diamond7Count++
		}
	}

	// 8钻条件：有5个7钻直推下级
	isDiamond8 := diamond7Count >= 5
	if isDiamond8 {
		logger.Info("用户是8钻，用户ID:", userID, "7钻直推下级数量:", diamond7Count)
	}
	return isDiamond8
}

// isDiamond9 判断用户是否是9钻（有5个8钻直推下级）
func (h *PaymentHandler) isDiamond9(userID int) bool {
	// 查询所有直推下级（zj_fid = userID）
	var directReferrals []model.User
	if err := h.DB.Where("zj_fid = ?", userID).Find(&directReferrals).Error; err != nil {
		logger.Info("查询直推下级失败，用户ID:", userID, "错误:", err)
		return false
	}

	// 统计直推下级中8钻的数量
	diamond8Count := 0
	for _, referral := range directReferrals {
		if h.isDiamond8(int(referral.Id)) {
			diamond8Count++
		}
	}

	// 9钻条件：有5个8钻直推下级
	isDiamond9 := diamond8Count >= 5
	if isDiamond9 {
		logger.Info("用户是9钻，用户ID:", userID, "8钻直推下级数量:", diamond8Count)
	}
	return isDiamond9
}

// pickActiveOrder 选择用户在投订单（优先剩余算力最大）
func (h *PaymentHandler) pickActiveOrder(userID int) (model.Order, bool) {
	var orders []model.Order
	res := h.DB.Model(&model.Order{}).Where("user_id = ? AND status = 8", userID).Find(&orders)
	if res.Error != nil || len(orders) == 0 {
		return model.Order{}, false
	}

	var target model.Order
	maxNowPower := -1.0
	for _, o := range orders {
		if o.NowPower > maxNowPower {
			maxNowPower = o.NowPower
			target = o
		}
	}
	if maxNowPower <= 0 {
		return model.Order{}, false
	}
	return target, true
}

// determineTargetLevel 确定要释放的目标套餐等级
// superiorLevels: 上级用户的套餐等级列表（已排序）
// currentLevel: 当前用户购买的套餐等级
func (h *PaymentHandler) determineTargetLevel(superiorLevels []int, currentLevel int) int {
	logger.Info("=== determineTargetLevel 开始 ===")
	logger.Info("上级套餐等级列表:", superiorLevels)
	logger.Info("当前用户套餐等级:", currentLevel)

	if len(superiorLevels) == 0 {
		logger.Info("上级没有套餐，返回0")
		return 0
	}

	// 情况1：当前套餐等级大于上级最大套餐，释放上级最大套餐
	if currentLevel > superiorLevels[len(superiorLevels)-1] {
		target := superiorLevels[len(superiorLevels)-1]
		logger.Info("情况1：当前套餐大于上级最大套餐，释放上级最大套餐:", target)
		return target
	}

	// 情况2：当前套餐等级小于上级最小套餐，释放上级最小套餐
	if currentLevel < superiorLevels[0] {
		target := superiorLevels[0]
		logger.Info("情况2：当前套餐小于上级最小套餐，释放上级最小套餐:", target)
		return target
	}

	// 情况3：当前套餐等级在上级套餐范围内，释放对应等级的套餐
	for _, level := range superiorLevels {
		if level == currentLevel {
			logger.Info("情况3：找到完全匹配的套餐等级:", level)
			return level
		}
	}

	// 情况4：当前套餐等级在上级套餐范围内但没有完全匹配，释放小于当前等级的最大套餐
	for i := len(superiorLevels) - 1; i >= 0; i-- {
		if superiorLevels[i] < currentLevel {
			logger.Info("情况4：释放小于当前等级的最大套餐:", superiorLevels[i])
			return superiorLevels[i]
		}
	}

	logger.Info("未找到合适的套餐等级，返回0")
	return 0
}

func teamRewardWeeklyCapPick(configured, fallback float64) float64 {
	if configured > 0 {
		return configured
	}
	return fallback
}

// teamRewardWeeklyCapMap 团队奖励周封顶（后台 team_reward_weekly_cap，仅团队奖励流水计入）
func (h *PaymentHandler) teamRewardWeeklyCapMap() map[int]float64 {
	def := types.DefaultTeamRewardWeeklyCap()
	c := h.App.SysConfig.TeamRewardWeeklyCap
	return map[int]float64{
		10:  teamRewardWeeklyCapPick(c.Vip, def.Vip),
		20:  teamRewardWeeklyCapPick(c.D1, def.D1),
		30:  teamRewardWeeklyCapPick(c.D2, def.D2),
		40:  teamRewardWeeklyCapPick(c.D3, def.D3),
		50:  teamRewardWeeklyCapPick(c.D4, def.D4),
		60:  teamRewardWeeklyCapPick(c.D5, def.D5),
		70:  teamRewardWeeklyCapPick(c.D6, def.D6),
		80:  teamRewardWeeklyCapPick(c.D7, def.D7),
		90:  teamRewardWeeklyCapPick(c.D8, def.D8),
		100: teamRewardWeeklyCapPick(c.D9, def.D9),
	}
}

func teamRewardWeekStartAt(t time.Time) time.Time {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	t = t.In(loc)
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := t.AddDate(0, 0, -(weekday - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, loc)
}

// teamRewardWeeklyCapAmount 按用户当前团队层级返回本周团队奖励算力封顶
func (h *PaymentHandler) teamRewardWeeklyCapAmount(userID int) float64 {
	limit := h.getUserRewardLayerLimit(userID)
	if cap, ok := h.teamRewardWeeklyCapMap()[limit]; ok {
		return cap
	}
	return 0
}

func (h *PaymentHandler) teamRewardWeeklyReleased(userID int, weekStart time.Time) float64 {
	var sum float64
	res := h.DB.Model(&model.PowerLog{}).
		Where("user_id = ? AND model = ? AND created_at >= ?", userID, "团队奖励释放", weekStart).
		Select("COALESCE(SUM(amount), 0)").Scan(&sum)
	if res.Error != nil {
		logger.Error("统计本周团队奖励已释放失败，用户ID:", userID, "错误:", res.Error)
		return 0
	}
	return sum
}

// teamRewardWeeklyRemaining 本周团队奖励剩余可释放额度（封顶 - 已释放）
func (h *PaymentHandler) teamRewardWeeklyRemaining(userID int) (remaining float64, cap float64, released float64) {
	cap = h.teamRewardWeeklyCapAmount(userID)
	if cap <= 0 {
		return 0, cap, 0
	}
	weekStart := teamRewardWeekStartAt(time.Now())
	released = h.teamRewardWeeklyReleased(userID, weekStart)
	remaining = cap - released
	if remaining < 0 {
		remaining = 0
	}
	return remaining, cap, released
}

// clampTeamRewardByWeeklyCap 团队奖励周封顶裁剪；返回允许释放的算力（0 表示本周已满或未配置封顶）
func (h *PaymentHandler) clampTeamRewardByWeeklyCap(userID int, requested float64) float64 {
	if requested <= 0 {
		return 0
	}
	remaining, cap, released := h.teamRewardWeeklyRemaining(userID)
	if cap <= 0 {
		logger.Info("【团队奖励周封顶】用户ID:", userID, "无有效封顶档位，跳过释放")
		return 0
	}
	if remaining <= 0 {
		logger.Info(fmt.Sprintf("【团队奖励周封顶】用户ID:%d 本周已释放%.2f/封顶%.2f，不再发放", userID, released, cap))
		return 0
	}
	if requested > remaining {
		logger.Info(fmt.Sprintf("【团队奖励周封顶】用户ID:%d 请求%.2f 裁剪为%.2f（本周已释放%.2f/封顶%.2f）",
			userID, requested, remaining, released, cap))
		return remaining
	}
	return requested
}

// releaseSuperiorPower 释放上级用户的算力
// rewardType: "推荐奖励" 或 "团队奖励"
// investorUserID: 触发奖励的下级用户ID（团队奖写入流水，供管理奖静态关联；推荐奖可传 0）
func (h *PaymentHandler) releaseSuperiorPower(order model.Order, releasePower float64, userName string, productName string, rewardType string, investorUserID uint) {
	logger.Info("=== releaseSuperiorPower 开始 ===")
	logger.Info("订单信息 - 订单号:", order.OrderNo, "用户ID:", order.UserId, "用户名:", order.Username)
	logger.Info("当前订单剩余算力:", order.NowPower, "已释放算力:", order.JsPower)
	logger.Info("请求释放算力:", releasePower, "下级用户:", userName, "产品:", productName, "奖励类型:", rewardType)

	// 计算实际可释放的算力（不能超过剩余算力）
	actualReleasePower := releasePower
	if order.NowPower < releasePower {
		actualReleasePower = order.NowPower
		logger.Info("实际可释放算力不足，调整为:", actualReleasePower)
	}

	if actualReleasePower <= 0 {
		logger.Info("没有可释放的算力，订单号:", order.OrderNo)
		return
	}

	// 团队奖励：周封顶（仅统计 model=团队奖励释放 的流水）
	if rewardType == "团队奖励" {
		afterCap := h.clampTeamRewardByWeeklyCap(int(order.UserId), actualReleasePower)
		if afterCap <= 0 {
			logger.Info("团队奖励周封顶已满，订单号:", order.OrderNo, "用户ID:", order.UserId)
			return
		}
		if afterCap < actualReleasePower {
			actualReleasePower = afterCap
		}
	}

	// 计算释放后的剩余算力
	remainingPower := order.NowPower - actualReleasePower

	logger.Info("释放算力详情 - 订单号:", order.OrderNo, "应释放:", releasePower, "实际释放:", actualReleasePower, "剩余:", remainingPower)

	// 更新订单的已释放算力和剩余算力
	logger.Info("开始更新订单数据...")
	res := h.DB.Model(&model.Order{}).Where("order_no = ?", order.OrderNo).UpdateColumn("js_power", gorm.Expr("js_power + ?", actualReleasePower))
	if res.Error != nil {
		logger.Error("更新已释放算力失败:", res.Error)
		return
	}
	logger.Info("已释放算力更新成功")

	res = h.DB.Model(&model.Order{}).Where("order_no = ?", order.OrderNo).UpdateColumn("now_power", remainingPower)
	if res.Error != nil {
		logger.Error("更新剩余算力失败:", res.Error)
		return
	}
	logger.Info("剩余算力更新成功")

	// 如果算力完全释放，更新订单状态为已完成
	if remainingPower == 0.0 {
		logger.Info("算力完全释放，更新订单状态为已完成")
		res = h.DB.Model(&model.Order{}).Where("order_no = ?", order.OrderNo).UpdateColumn("status", "9")
		if res.Error != nil {
			logger.Error("更新订单状态失败:", res.Error)
		} else {
			logger.Info("订单状态更新成功")
		}
	}

	// 先读取上级当前余额（用于流水余额字段，避免同一时间多次释放时多条流水显示相同余额）
	var superiorUser model.User
	res = h.DB.First(&superiorUser, order.UserId)
	if res.Error != nil {
		logger.Error("获取上级用户信息失败:", res.Error)
		return
	}
	balanceBefore := superiorUser.Power
	logger.Info("释放前上级余额:", balanceBefore)

	// 增加上级用户的可用算力
	logger.Info("开始增加上级用户可用算力...")
	res = h.DB.Model(&model.User{}).Where("id = ?", order.UserId).UpdateColumn("power", gorm.Expr("power + ?", actualReleasePower))
	if res.Error != nil {
		logger.Error("增加用户可用算力失败:", res.Error)
		return
	}
	logger.Info("用户可用算力增加成功")

	// 流水中的余额 = 本笔释放前余额 + 本笔释放量，避免同一秒内多笔释放都读到“最终余额”导致多条显示相同
	balanceForLog := balanceBefore + actualReleasePower

	// 记录算力释放流水
	logger.Info("开始记录算力释放流水...")
	// 根据奖励类型设置不同的Model和Remark
	var modelText string
	var remarkText string
	if rewardType == "团队奖励" {
		modelText = "团队奖励释放"
		_, weekCap, weekReleased := h.teamRewardWeeklyRemaining(int(order.UserId))
		capNote := ""
		if weekCap > 0 {
			capNote = fmt.Sprintf("（本周团队奖励累计%.0f/封顶%.0f）", weekReleased+actualReleasePower, weekCap)
		}
		if investorUserID > 0 && userName != "" {
			remarkText = fmt.Sprintf("团队奖励：下级用户ID：%d 用户名：%s 充值:%s 产品  加速释放算力：%f%s，", investorUserID, userName, productName, actualReleasePower, capNote)
		} else if investorUserID > 0 {
			remarkText = fmt.Sprintf("团队奖励：下级用户ID：%d 充值:%s 产品  加速释放算力：%f%s，", investorUserID, productName, actualReleasePower, capNote)
		} else if userName != "" {
			remarkText = fmt.Sprintf("团队奖励：下级用户 用户名：%s 充值:%s 产品  加速释放算力：%f%s，", userName, productName, actualReleasePower, capNote)
		} else {
			remarkText = fmt.Sprintf("团队奖励：下级用户充值:%s 产品  加速释放算力：%f%s，", productName, actualReleasePower, capNote)
		}
	} else {
		modelText = "推荐奖励释放"
		remarkText = fmt.Sprintf("推荐奖励：下级用户 用户名：%s 充值:%s 产品  加速释放算力：%f，", userName, productName, actualReleasePower)
	}

	powerLog := &model.PowerLog{
		UserId:    order.UserId,
		Username:  order.Username,
		Type:      types.PowerJT, // 释放
		Amount:    actualReleasePower,
		Balance:   balanceForLog,
		Mark:      types.PowerAdd,
		Model:     modelText,
		OrderNo:   order.OrderNo,
		Remark:    remarkText,
		CreatedAt: time.Now(),
	}

	res = h.DB.Create(powerLog)
	if res.Error != nil {
		logger.Error("创建算力释放流水失败:", res.Error)
		return
	}
	logger.Info("算力释放流水记录成功，流水ID:", powerLog.Id)

	// 有授信欠款时：收益入账后立刻还款
	if h.creditService != nil && actualReleasePower > 0 {
		if _, err := h.creditService.RepayFromIncome(h.DB, order.UserId, order.Username, actualReleasePower, order.OrderNo); err != nil {
			logger.Error("推荐/团队奖励授信还款失败:", err)
		}
	}

	logger.Info("算力释放完成 - 用户:", order.Username, "释放算力:", actualReleasePower)
	logger.Info("=== releaseSuperiorPower 结束 ===")
}

// releaseSuperiorPowerFromMultipleOrders 从多个订单中按顺序释放算力，实现0烧伤
// orders: 订单列表（按剩余算力从大到小排序）
// totalReleasePower: 需要释放的总算力
// userName: 下级用户名
// productName: 产品名称
// rewardType: "推荐奖励" 或 "团队奖励"
// investorUserID: 触发奖励的下级用户ID（团队奖写入流水；推荐奖传 0）
// 返回实际释放的总算力
func (h *PaymentHandler) releaseSuperiorPowerFromMultipleOrders(orders []model.Order, totalReleasePower float64, userName string, productName string, rewardType string, investorUserID uint) float64 {
	logger.Info("=== releaseSuperiorPowerFromMultipleOrders 开始 ===")
	logger.Info("订单数量:", len(orders), "需要释放总算力:", totalReleasePower)
	logger.Info("下级用户:", userName, "产品:", productName, "奖励类型:", rewardType)

	if len(orders) == 0 {
		logger.Info("没有可用订单，返回0")
		return 0
	}

	if totalReleasePower <= 0 {
		logger.Info("需要释放的算力为0，返回0")
		return 0
	}

	// 按剩余算力从大到小排序订单（确保优先释放剩余算力多的订单）
	sortedOrders := make([]model.Order, len(orders))
	copy(sortedOrders, orders)
	sort.Slice(sortedOrders, func(i, j int) bool {
		return sortedOrders[i].NowPower > sortedOrders[j].NowPower
	})

	var totalReleased float64 = 0
	remainingPower := totalReleasePower

	// 遍历所有订单，按顺序释放算力
	for i, order := range sortedOrders {
		if remainingPower <= 0 {
			logger.Info("已释放完所有需要的算力，停止释放")
			break
		}

		logger.Info(fmt.Sprintf("=== 处理第 %d 个订单 ===", i+1))
		logger.Info("订单号:", order.OrderNo, "剩余算力:", order.NowPower, "还需释放:", remainingPower)

		// 计算从这个订单中需要释放的算力
		releaseFromThisOrder := remainingPower
		if order.NowPower < remainingPower {
			releaseFromThisOrder = order.NowPower
			logger.Info("订单剩余算力不足，只能释放:", releaseFromThisOrder)
		}

		if releaseFromThisOrder <= 0 {
			logger.Info("订单剩余算力为0，跳过")
			continue
		}

		// 重新查询订单的最新状态（确保数据是最新的）
		var currentOrder model.Order
		res := h.DB.Where("order_no = ?", order.OrderNo).First(&currentOrder)
		if res.Error != nil {
			logger.Error("查询订单失败，订单号:", order.OrderNo, "错误:", res.Error)
			continue
		}

		// 再次检查剩余算力（防止并发问题）
		if currentOrder.NowPower <= 0 {
			logger.Info("订单剩余算力为0，跳过")
			continue
		}

		// 如果订单剩余算力小于预期，调整释放量
		if currentOrder.NowPower < releaseFromThisOrder {
			releaseFromThisOrder = currentOrder.NowPower
			logger.Info("订单实际剩余算力不足，调整释放量为:", releaseFromThisOrder)
		}

		// 记录释放前的剩余算力
		powerBeforeRelease := currentOrder.NowPower

		// 调用 releaseSuperiorPower 释放算力
		logger.Info("开始释放算力，订单号:", currentOrder.OrderNo, "释放量:", releaseFromThisOrder)
		h.releaseSuperiorPower(currentOrder, releaseFromThisOrder, userName, productName, rewardType, investorUserID)

		// 查询释放后的订单状态，计算实际释放的算力
		var orderAfterRelease model.Order
		res = h.DB.Where("order_no = ?", currentOrder.OrderNo).First(&orderAfterRelease)
		if res.Error != nil {
			logger.Error("查询释放后订单状态失败，订单号:", currentOrder.OrderNo, "错误:", res.Error)
			// 如果查询失败，使用预期释放量
			totalReleased += releaseFromThisOrder
			remainingPower -= releaseFromThisOrder
		} else {
			// 计算实际释放的算力
			actualReleasedFromThisOrder := powerBeforeRelease - orderAfterRelease.NowPower
			if actualReleasedFromThisOrder < 0 {
				actualReleasedFromThisOrder = 0
			}
			totalReleased += actualReleasedFromThisOrder
			remainingPower -= actualReleasedFromThisOrder
			logger.Info(fmt.Sprintf("订单 %s 实际释放算力: %.2f (预期: %.2f)", currentOrder.OrderNo, actualReleasedFromThisOrder, releaseFromThisOrder))
		}

		logger.Info(fmt.Sprintf("累计已释放算力: %.2f, 还需释放: %.2f", totalReleased, remainingPower))
	}

	logger.Info("=== releaseSuperiorPowerFromMultipleOrders 结束 ===")
	logger.Info(fmt.Sprintf("总计释放算力: %.2f, 需要释放: %.2f, 烧伤: %.2f", totalReleased, totalReleasePower, totalReleasePower-totalReleased))

	return totalReleased
}

// 辅助方法：自定义套餐power等级匹配
func determineTargetPower(superiorPowers []float64, currentPower float64) float64 {
	logger.Info("=== determineTargetPower 开始 ===")
	logger.Info("上级power列表:", superiorPowers)
	logger.Info("当前用户power:", currentPower)
	if len(superiorPowers) == 0 {
		logger.Info("上级没有套餐，返回0")
		return 0
	}
	if currentPower > superiorPowers[len(superiorPowers)-1] {
		logger.Info("情况1：当前power大于上级最大power，释放最大power:", superiorPowers[len(superiorPowers)-1])
		return superiorPowers[len(superiorPowers)-1]
	}
	if currentPower < superiorPowers[0] {
		logger.Info("情况2：当前power小于上级最小power，释放最小power:", superiorPowers[0])
		return superiorPowers[0]
	}
	for _, p := range superiorPowers {
		if p == currentPower {
			logger.Info("情况3：找到完全匹配power:", p)
			return p
		}
	}
	for i := len(superiorPowers) - 1; i >= 0; i-- {
		if superiorPowers[i] < currentPower {
			logger.Info("情况4：释放小于当前power的最大power:", superiorPowers[i])
			return superiorPowers[i]
		}
	}
	logger.Info("未找到合适的power，返回0")
	return 0
}
