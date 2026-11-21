package service

import (
	"fmt"
	"math"
	"time"

	"chatplus/core/types"
	"chatplus/store/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	CreditLogModelGrant   = "授信额度增加" // 可用额度增加（直推）
	CreditLogModelSpend   = "授信报单"   // 可用额度消耗 + 待还增加
	CreditLogModelRepay   = "授信还款"   // 待还额度减少
	CreditLogModelRecalc  = "授信额度调整" // 兼容旧流水
	CreditLogModelMaxUp   = "最大额度增加" // 自投抬高最大额度
	CreditLogModelMaxDown = "最大额度减少" // 报单消耗等导致最大额度下调
)

// CreditService 授信额度：独立于可用算力，仅可报单，还款不回补可用额度
type CreditService struct {
	db *gorm.DB
}

func NewCreditService(db *gorm.DB) *CreditService {
	return &CreditService{db: db}
}

func round6(v float64) float64 {
	return math.Round(v*1e6) / 1e6
}

// lockRow SQLite 不支持 FOR UPDATE，测试/兼容场景下跳过行锁
func lockRow(tx *gorm.DB) *gorm.DB {
	if tx == nil || tx.Dialector == nil {
		return tx
	}
	if name := tx.Dialector.Name(); name == "sqlite" {
		return tx
	}
	return tx.Clauses(clause.Locking{Strength: "UPDATE"})
}

// RecalcMax 按额度策略重算最大额度；若可用超过新 Max 则下调可用（不改债务）
func (s *CreditService) RecalcMax(tx *gorm.DB, userId uint, cfg *types.SystemConfig) error {
	_, err := s.RecalcMaxAndSnapshot(tx, userId, cfg)
	return err
}

// RecalcMaxAndSnapshot 同 RecalcMax，并返回推导出的额度构成，供调用方解释额度来源。
//
//	最大额度 = max(新规则基数, CreditMaxFloor) − 累计授信报单消耗
//
// Floor 是「历史基数的快照」，只升不降：
// 新规则上线后老会员的历史投入仍按固定 5 万计，不会被「单笔≥1万」的门槛清零。
func (s *CreditService) RecalcMaxAndSnapshot(tx *gorm.DB, userId uint, cfg *types.SystemConfig) (CreditSnapshot, error) {
	if tx == nil {
		tx = s.db
	}
	if userId == 0 {
		return CreditSnapshot{}, nil
	}
	params := CreditMaxParamsFrom(cfg)

	var user model.User
	if err := lockRow(tx).
		Select("id", "username", "credit_max", "credit_available", "credit_max_floor").
		First(&user, userId).Error; err != nil {
		return CreditSnapshot{}, err
	}

	split, err := s.loadInvestSplit(tx, userId, params)
	if err != nil {
		return CreditSnapshot{}, err
	}
	snap := EvaluateCredit(params, split)
	investBase := snap.Max // 投入侧基数（未扣报单消耗）

	spent, err := s.loadCreditSpent(tx, userId)
	if err != nil {
		return CreditSnapshot{}, err
	}

	oldMax := round6(user.CreditMax)
	floor := round6(user.CreditMaxFloor)
	freeze := true // 能否把 floor 固化到库里
	if floor <= 0 {
		floor, freeze = s.resolveFloor(tx, userId, investBase, oldMax, spent)
	}
	if investBase > floor {
		floor = investBase
	}

	finalMax := round6(floor - spent)
	if finalMax < 0 {
		finalMax = 0
	}
	snap.Max = finalMax

	newAvail := user.CreditAvailable
	if newAvail > snap.Max {
		newAvail = snap.Max
	}
	// 未固化时不能把本次结果当基线，否则一次读取失败就被写死
	floorChanged := freeze && round6(user.CreditMaxFloor) != floor
	if oldMax == snap.Max && round6(user.CreditAvailable) == newAvail && !floorChanged {
		return snap, nil
	}
	updates := map[string]interface{}{
		"credit_max":       snap.Max,
		"credit_available": newAvail,
	}
	if freeze {
		updates["credit_max_floor"] = floor
	}
	if err := tx.Model(&model.User{}).Where("id = ?", userId).Updates(updates).Error; err != nil {
		return CreditSnapshot{}, err
	}
	// 最大额度变动记明细（Floor 恢复导致的回升也会记「增加」）
	deltaMax := round6(snap.Max - oldMax)
	if deltaMax > 0 {
		_ = tx.Create(&model.PowerLog{
			UserId:    userId,
			Username:  user.Username,
			Type:      types.PowerCredit,
			Amount:    deltaMax,
			Balance:   snap.Max,
			Mark:      types.PowerAdd,
			Model:     CreditLogModelMaxUp,
			Remark:    fmt.Sprintf("最大额度增加 +%.6f → %.6f", deltaMax, snap.Max),
			CreatedAt: time.Now(),
		}).Error
	} else if deltaMax < 0 {
		_ = tx.Create(&model.PowerLog{
			UserId:    userId,
			Username:  user.Username,
			Type:      types.PowerCredit,
			Amount:    -deltaMax,
			Balance:   snap.Max,
			Mark:      types.PowerSub,
			Model:     CreditLogModelMaxDown,
			Remark:    fmt.Sprintf("最大额度减少 %.6f → %.6f", -deltaMax, snap.Max),
			CreatedAt: time.Now(),
		}).Error
	}
	return snap, nil
}

// resolveFloor 计算本轮的投入侧基数，并给出「能否把结果固化到 credit_max_floor」。
//
//	floor = max(新规则基数, 被打成 0 前的额度)
//
// 第二项有两个证据来源，谁大取谁：
//
//	oldMax + spent  —— 额度还在或仍有报单消耗时，反推出基线
//	wipedBase       —— 已被打成 0 时，用「最大额度减少」的流水金额（订单被清理也兜得住）
//
// 返回 ok=false 表示证据读不到（查询失败）。此时仍然用新规则基数跑完本次计算，
// 但**不落库**：floor 一旦写入就只升不降，把一次抖动的结果写死等于永久丢额度。
func (s *CreditService) resolveFloor(tx *gorm.DB, userId uint, investBase, oldMax, spent float64) (floor float64, ok bool) {
	floor = investBase
	if oldMax > 0 || spent > 0 {
		if v := round6(oldMax + spent); v > floor {
			floor = v
		}
		return round6(floor), true
	}
	// 已归零：没有任何流水痕迹说明是全新用户，不能凭空给额度
	wiped, ok := s.loadWipedBase(tx, userId)
	if !ok {
		return round6(floor), false
	}
	if wiped > floor {
		floor = wiped
	}
	return round6(floor), true
}

// loadWipedBase 取历史「最大额度减少」流水的最大单笔金额：被打成 0 的那一刻，它就是原本的额度。
// 用作保守下限（减少额不会超过当时的基数），订单被清理/改状态后仍能兜住历史额度。
func (s *CreditService) loadWipedBase(tx *gorm.DB, userId uint) (float64, bool) {
	if tx == nil {
		tx = s.db
	}
	var v float64
	if err := tx.Model(&model.PowerLog{}).
		Where("user_id = ? AND model = ? AND mark = ? AND deleted_at IS NULL",
			userId, CreditLogModelMaxDown, types.PowerSub).
		Select("COALESCE(MAX(amount), 0)").Scan(&v).Error; err != nil {
		return 0, false
	}
	return round6(v), true
}

// loadCreditSpent 累计已消耗的授信报单额度（用多少扣多少最大额度，还款不回补）
func (s *CreditService) loadCreditSpent(tx *gorm.DB, userId uint) (float64, error) {
	if tx == nil {
		tx = s.db
	}
	var spent float64
	q := tx.Model(&model.PowerLog{}).
		Where("user_id = ? AND type = ? AND mark = ? AND model = ?",
			userId, types.PowerCredit, types.PowerSub, CreditLogModelSpend).
		Where("deleted_at IS NULL")
	err := q.Select("COALESCE(SUM(amount), 0)").Scan(&spent).Error
	return round6(spent), err
}

// GrantAvailableFromDirect 直推下级投入后，给上级增加可用额度（≤ Max）；还款不回补
func (s *CreditService) GrantAvailableFromDirect(tx *gorm.DB, superiorId uint, investSales float64, cfg *types.SystemConfig, fromUsername string) error {
	if tx == nil {
		tx = s.db
	}
	if superiorId == 0 || investSales <= 0 {
		return nil
	}
	rate := 0.5
	if cfg != nil {
		rate = cfg.CreditDirectRateOrDefault()
	}
	// 先确保上级 Max 已按当前自投刷新
	if err := s.RecalcMax(tx, superiorId, cfg); err != nil {
		return err
	}

	var user model.User
	if err := lockRow(tx).First(&user, superiorId).Error; err != nil {
		return err
	}
	room := round6(user.CreditMax - user.CreditAvailable)
	if room <= 0 {
		return nil
	}
	delta := round6(investSales * rate)
	if delta > room {
		delta = room
	}
	if delta <= 0 {
		return nil
	}
	newAvail := round6(user.CreditAvailable + delta)
	if err := tx.Model(&model.User{}).Where("id = ?", superiorId).
		UpdateColumn("credit_available", newAvail).Error; err != nil {
		return err
	}
	return tx.Create(&model.PowerLog{
		UserId:    superiorId,
		Username:  user.Username,
		Type:      types.PowerCredit,
		Amount:    delta,
		Balance:   newAvail,
		Mark:      types.PowerAdd,
		Model:     CreditLogModelGrant,
		Remark:    fmt.Sprintf("直推用户 %s 投入 %.6f，获得可用授信 %.6f（比例 %.4f）", fromUsername, investSales, delta, rate),
		CreatedAt: time.Now(),
	}).Error
}

// AfterInvest 真实入单完成后：刷新本人 Max，并给其直推上级增加可用额度（推荐奖）。
func (s *CreditService) AfterInvest(tx *gorm.DB, investor model.User, sales float64, cfg *types.SystemConfig) error {
	if tx == nil {
		tx = s.db
	}
	if sales <= 0 || investor.Id == 0 {
		return nil
	}
	if err := s.RecalcMax(tx, investor.Id, cfg); err != nil {
		return err
	}
	if investor.ZjFid > 0 {
		return s.GrantAvailableFromDirect(tx, uint(investor.ZjFid), sales, cfg, investor.Username)
	}
	return nil
}

// AfterGift 授信报单完成后的授信处理。
// 规则：不管谁给谁报单，可用额度只增加给收款人的直推上级（受 Max 天花板限制）；
// 收款人最大额度不因授信/赠送订单增加（这类订单已从自投统计中排除）。
func (s *CreditService) AfterGift(tx *gorm.DB, giver, receiver model.User, sales float64, cfg *types.SystemConfig) error {
	_ = giver // 付款人已在 SpendForGift 中扣减可用/最大额度，此处不再回补
	return s.AfterInvest(tx, receiver, sales, cfg)
}

// SpendForGift 授信报单：扣可用额度、增加待还债务，并按消耗同步下调最大额度（还款不回补）
func (s *CreditService) SpendForGift(tx *gorm.DB, userId uint, username string, amount float64, orderNo string, receiverName string, cfg *types.SystemConfig) error {
	if tx == nil {
		tx = s.db
	}
	amount = round6(amount)
	if userId == 0 || amount <= 0 {
		return fmt.Errorf("授信报单金额无效")
	}
	// 先固化 Floor，避免「先写消耗流水再 bootstrap」把 Floor 抬成 oldMax+新spent
	if err := s.RecalcMax(tx, userId, cfg); err != nil {
		return err
	}
	res := tx.Model(&model.User{}).
		Where("id = ? AND credit_available >= ?", userId, amount).
		Updates(map[string]interface{}{
			"credit_available": gorm.Expr("credit_available - ?", amount),
			"credit_debt":      gorm.Expr("credit_debt + ?", amount),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("授信可用额度不足")
	}
	var user model.User
	if err := tx.Select("credit_available", "credit_debt").First(&user, userId).Error; err != nil {
		return err
	}
	if err := tx.Create(&model.PowerLog{
		UserId:    userId,
		Username:  username,
		Type:      types.PowerCredit,
		Amount:    amount,
		Balance:   user.CreditAvailable,
		Mark:      types.PowerSub,
		Model:     CreditLogModelSpend,
		OrderNo:   orderNo,
		Remark:    fmt.Sprintf("授信报单 %.6f 给 %s：可用-%.6f，待还+%.6f（当前待还 %.6f），订单号：%s", amount, receiverName, amount, amount, user.CreditDebt, orderNo),
		CreatedAt: time.Now(),
	}).Error; err != nil {
		return err
	}
	// 流水落库后再重算 Max（基数 − 累计消耗）
	return s.RecalcMax(tx, userId, cfg)
}

// RepayFromIncome 日收益入账后立刻还款：扣减 user.power 与 credit_debt，记「授信还款」流水。
// 不回补 credit_available。返回实际还款额。
func (s *CreditService) RepayFromIncome(tx *gorm.DB, userId uint, username string, grossIncome float64, orderNo string) (float64, error) {
	if tx == nil {
		tx = s.db
	}
	grossIncome = round6(grossIncome)
	if userId == 0 || grossIncome <= 0 {
		return 0, nil
	}
	var user model.User
	if err := lockRow(tx).
		Select("id", "username", "power", "credit_debt").
		First(&user, userId).Error; err != nil {
		return 0, err
	}
	if user.CreditDebt <= 0 {
		return 0, nil
	}
	repay := grossIncome
	if repay > user.CreditDebt {
		repay = user.CreditDebt
	}
	repay = round6(repay)
	if repay <= 0 {
		return 0, nil
	}
	// 算力可能因并发略少，按实际余额封顶
	if user.Power < repay {
		repay = round6(user.Power)
	}
	if repay <= 0 {
		return 0, nil
	}

	res := tx.Model(&model.User{}).
		Where("id = ? AND power >= ? AND credit_debt >= ?", userId, repay, repay).
		Updates(map[string]interface{}{
			"power":       gorm.Expr("power - ?", repay),
			"credit_debt": gorm.Expr("credit_debt - ?", repay),
		})
	if res.Error != nil {
		return 0, res.Error
	}
	if res.RowsAffected == 0 {
		// 并发下可能已还完，忽略
		return 0, nil
	}
	var after model.User
	_ = tx.Select("power", "credit_debt").First(&after, userId)
	if username == "" {
		username = user.Username
	}
	if err := tx.Create(&model.PowerLog{
		UserId:    userId,
		Username:  username,
		Type:      types.PowerCredit,
		Amount:    repay,
		Balance:   after.Power,
		Mark:      types.PowerSub,
		Model:     CreditLogModelRepay,
		OrderNo:   orderNo,
		Remark:    fmt.Sprintf("收益自动偿还授信 %.6f（原收益 %.6f），剩余待还 %.6f", repay, grossIncome, after.CreditDebt),
		CreatedAt: time.Now().Add(time.Millisecond),
	}).Error; err != nil {
		return 0, err
	}
	return repay, nil
}
