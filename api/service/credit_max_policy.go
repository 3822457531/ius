package service

import (
	"time"

	"chatplus/core/types"
	"chatplus/store/model"

	"gorm.io/gorm"
)

// 授信额度策略（CreditMaxPolicy）
//
// 唯一输入是「订单投入在分界时刻前后的切分」，额度由此推导，不依赖任何用户状态字段：
//
//	分界前投入合计 ≥ 门槛  →  老会员   max = 固定上限 + 分界后投入 × 杠杆
//	否则                  →  普通会员  max = 分界前投入 × 杠杆
//
// 计入口径：只统计「单笔 sales ≥ LegacyMin（默认 1 万）」的自费订单。
// 多笔小额（每笔都 < 1 万）即使合计很大，也不抬高 Max。
//
// 分界后合计仍受 CountedAfter 门槛约束（不足门槛视为 0，不按比例折算）：
//
//	A：T 前单笔 ≥1 万 + T 后仅有小额  →  老会员  50000 + 0
//	B：T 前无大单、T 后单笔 2 万       →  普通会员 20000×5 = 100000
//
// 不固化「老会员资格」快照的原因：
//
//  1. loadInvestSplit 统计 status IN (8, 9)，已结束订单同样计入 Before，
//     所以订单到期不会让 Before 归零，额度不会回弹；
//  2. 资格与额度来自同一个 split，不存在「打标口径 / 算额度口径」两套规则。
const (
	defaultCreditLegacyMinPower = 10000.0
	defaultCreditLegacyMax      = 50000.0
)

// CreditMaxParams 额度策略参数（从 SystemConfig 归一化得到，避免下游反复判空）
type CreditMaxParams struct {
	Leverage     float64   // 杠杆倍数
	LegacyMax    float64   // 老会员固定上限基数
	LegacyMin    float64   // 单笔计入门槛（默认 1 万）；也用于老会员分界前合计门槛
	LegacyActive bool      // 是否启用分界（未配置分界时刻时为 false）
	Cutoff       time.Time // 分界时刻
}

func CreditMaxParamsFrom(cfg *types.SystemConfig) CreditMaxParams {
	p := CreditMaxParams{
		Leverage:  5,
		LegacyMax: defaultCreditLegacyMax,
		LegacyMin: defaultCreditLegacyMinPower,
	}
	if cfg == nil {
		return p
	}
	p.Leverage = cfg.CreditLeverageOrDefault()
	p.LegacyMax = cfg.CreditLegacyMaxOrDefault()
	p.LegacyMin = cfg.CreditLegacyMinPowerOrDefault()
	p.Cutoff, p.LegacyActive = cfg.CreditLegacyCutoffTime()
	return p
}

// InvestSplit 自投在分界时刻前后的切分；Before + After 恒等于 Total
type InvestSplit struct {
	Before float64
	After  float64
	Total  float64
}

// IsLegacyInvest 老会员资格：分界前投入合计达到门槛。
// 必须要求 Before > 0，否则门槛配成 0 时「从未投入」的新用户也会被判成老会员白拿固定上限。
func IsLegacyInvest(p CreditMaxParams, split InvestSplit) bool {
	return p.LegacyActive && split.Before > 0 && split.Before >= p.LegacyMin
}

// CountedAfter 参与额度计算的分界后投入。
// 合计不足 LegacyMin 视为 0（不按比例折算）；与 loadInvestSplit 的「单笔门槛」互补。
func CountedAfter(p CreditMaxParams, split InvestSplit) float64 {
	if p.LegacyMin > 0 && split.After < p.LegacyMin {
		return 0
	}
	return split.After
}

// CountedBefore 参与额度计算的分界前投入（非老会员路径用）；不足门槛视为 0。
func CountedBefore(p CreditMaxParams, split InvestSplit) float64 {
	if p.LegacyMin > 0 && split.Before < p.LegacyMin {
		return 0
	}
	return split.Before
}

// CreditSnapshot 额度推导结果，供落库与前端解释额度构成共用
type CreditSnapshot struct {
	Max          float64     // 额度上限
	Legacy       bool        // 是否按老会员基数计算
	Base         float64     // 老会员固定上限（非老会员为 0）
	CountedAfter float64     // 实际参与计算的（已过门槛的）分界后投入
	Split        InvestSplit // 投入原始切分
}

// EvaluateCredit 纯函数：策略参数 + 投入切分 → 额度构成
func EvaluateCredit(p CreditMaxParams, split InvestSplit) CreditSnapshot {
	after := CountedAfter(p, split)
	if IsLegacyInvest(p, split) {
		return CreditSnapshot{
			Max:          round6(p.LegacyMax + after*p.Leverage),
			Legacy:       true,
			Base:         p.LegacyMax,
			CountedAfter: after,
			Split:        split,
		}
	}
	before := CountedBefore(p, split)
	return CreditSnapshot{
		Max:          round6((before + after) * p.Leverage),
		CountedAfter: after,
		Split:        split,
	}
}

// CalcCreditMax 只要额度数值时的便捷入口
func CalcCreditMax(p CreditMaxParams, split InvestSplit) float64 {
	return EvaluateCredit(p, split).Max
}

// loadInvestSplit 统计用户「自费且单笔达标」投入（在投 + 已结束），并按分界时刻切分。
// 排除 gift / credit_gift；排除单笔 sales < LegacyMin 的小额单（不抬高 Max）。
func (s *CreditService) loadInvestSplit(tx *gorm.DB, userId uint, p CreditMaxParams) (InvestSplit, error) {
	var split InvestSplit
	if tx == nil {
		tx = s.db
	}
	minSales := p.LegacyMin
	if minSales <= 0 {
		minSales = defaultCreditLegacyMinPower
	}
	base := func() *gorm.DB {
		return tx.Model(&model.Order{}).
			Where("user_id = ? AND sales >= ? AND status IN ?", userId, minSales,
				[]int{int(types.OrderZyz), int(types.OrderYjs)}).
			Where("(pay_way IS NULL OR pay_way = '' OR pay_way NOT IN ?)",
				[]string{"gift", "credit_gift"})
	}
	if err := base().Select("COALESCE(SUM(sales), 0)").Scan(&split.Total).Error; err != nil {
		return split, err
	}
	if p.LegacyActive {
		if err := base().Where("created_at >= ?", p.Cutoff).
			Select("COALESCE(SUM(sales), 0)").Scan(&split.After).Error; err != nil {
			return split, err
		}
	} else {
		// 未启用分界：全部视为「分界后」，等价于当前机制
		split.After = split.Total
	}
	split.Before = round6(split.Total - split.After)
	return split, nil
}

// CreditSnapshotOf 只读推导某用户的额度构成，不落库
func (s *CreditService) CreditSnapshotOf(tx *gorm.DB, userId uint, cfg *types.SystemConfig) (CreditSnapshot, error) {
	if tx == nil {
		tx = s.db
	}
	if userId == 0 {
		return CreditSnapshot{}, nil
	}
	params := CreditMaxParamsFrom(cfg)
	split, err := s.loadInvestSplit(tx, userId, params)
	if err != nil {
		return CreditSnapshot{}, err
	}
	return EvaluateCredit(params, split), nil
}