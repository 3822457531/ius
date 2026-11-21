package service

import (
	"testing"
	"time"

	"chatplus/core/types"
	"chatplus/store/model"

	"gorm.io/gorm"
)

// legacyCfg 分界时刻 D0；门槛 1 万、老会员基数 5 万、杠杆 5
func legacyCfg() *types.SystemConfig {
	return &types.SystemConfig{
		CreditLeverage:       5,
		CreditDirectRate:     0.5,
		CreditLegacyCutoff:   "2026-01-01 00:00:00",
		CreditLegacyMinPower: 10000,
		CreditLegacyMax:      50000,
	}
}

func cutoffTime(t *testing.T) time.Time {
	t.Helper()
	ts, ok := legacyCfg().CreditLegacyCutoffTime()
	if !ok {
		t.Fatal("cutoff parse failed")
	}
	return ts
}

// orderAt 造一笔指定创建时刻的订单（gorm 会忽略零值时间，必须显式写 created_at）
func orderAt(t *testing.T, db *gorm.DB, userID uint, sales float64, status types.OrderStatus, at time.Time) {
	t.Helper()
	o := &model.Order{
		UserId: userID, Username: "u", OrderNo: at.Format("150405.000000") + "-" + string(rune('0'+int(sales)%10)),
		Status: status, Sales: sales, Power: sales * 5, NowPower: sales * 5,
	}
	if err := db.Create(o).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}
	if err := db.Model(&model.Order{}).Where("id = ?", o.Id).
		UpdateColumn("created_at", at).Error; err != nil {
		t.Fatalf("set created_at: %v", err)
	}
}

// maxOf 重算并返回额度构成
func maxOf(t *testing.T, db *gorm.DB, userID uint, cfg *types.SystemConfig) CreditSnapshot {
	t.Helper()
	snap, err := NewCreditService(db).RecalcMaxAndSnapshot(db, userID, cfg)
	if err != nil {
		t.Fatalf("recalc: %v", err)
	}
	return snap
}

// 场景 1：以前投 10 万，之后不再投 → max 恒为 5 万
func TestLegacy_OldInvestOnly_CappedAtLegacyBase(t *testing.T) {
	db := setupCreditTestDB(t)
	cutoff := cutoffTime(t)

	mustUser(t, db, &model.User{Username: "old10w", Power: 0})
	orderAt(t, db, 1, 100000, types.OrderZyz, cutoff.Add(-time.Hour))

	if snap := maxOf(t, db, 1, legacyCfg()); !snap.Legacy || snap.Max != 50000 {
		t.Fatalf("want legacy max 50000, got legacy=%v max=%v", snap.Legacy, snap.Max)
	}
}

// 场景 2：以前没投，之后投 1 万 → 走当前机制 1万×5 = 5 万
func TestLegacy_NoOldInvest_UsesCurrentMechanism(t *testing.T) {
	db := setupCreditTestDB(t)
	cutoff := cutoffTime(t)

	mustUser(t, db, &model.User{Username: "newbie", Power: 0})
	orderAt(t, db, 1, 10000, types.OrderZyz, cutoff.Add(time.Hour))

	if snap := maxOf(t, db, 1, legacyCfg()); snap.Legacy || snap.Max != 50000 {
		t.Fatalf("want non-legacy max 50000, got legacy=%v max=%v", snap.Legacy, snap.Max)
	}
}

// 场景 3：以前投 10 万 + 之后投 1 万 → 5万 + 1万×5 = 10 万
func TestLegacy_OldPlusNew_AddsOnTopOfLegacyBase(t *testing.T) {
	db := setupCreditTestDB(t)
	cutoff := cutoffTime(t)

	mustUser(t, db, &model.User{Username: "zhangsan", Power: 0})
	orderAt(t, db, 1, 100000, types.OrderZyz, cutoff.Add(-24*time.Hour))
	orderAt(t, db, 1, 10000, types.OrderZyz, cutoff.Add(24*time.Hour))

	if snap := maxOf(t, db, 1, legacyCfg()); !snap.Legacy || snap.Max != 100000 {
		t.Fatalf("want legacy max 100000, got legacy=%v max=%v", snap.Legacy, snap.Max)
	}
}

// 回归：分界前多笔小额、无单笔达标的用户必须被封顶，不能按合计×杠杆拿超额
// 现场数据：4 笔共 1.2 万 → 旧口径 60000（泄漏），新口径 50000
func TestLegacy_MultiSmallOldOrders_NowCapped(t *testing.T) {
	db := setupCreditTestDB(t)
	cutoff := cutoffTime(t)

	mustUser(t, db, &model.User{Username: "test", Power: 0})
	for _, s := range []float64{5000, 5000, 1000, 1000} {
		orderAt(t, db, 1, s, types.OrderZyz, cutoff.Add(-time.Hour))
	}

	snap := maxOf(t, db, 1, legacyCfg())
	if !snap.Legacy {
		t.Fatalf("pre-cutoff total 12000 >= 10000 should qualify as legacy")
	}
	if snap.Max != 50000 {
		t.Fatalf("want 50000 got %v", snap.Max)
	}
	if snap.Split.Before != 12000 || snap.Split.After != 0 {
		t.Fatalf("split want before=12000 after=0, got %+v", snap.Split)
	}
}

// 封顶不产生倒挂：分界前投入越多，额度不下降
func TestLegacy_NoInversionAcrossThreshold(t *testing.T) {
	db := setupCreditTestDB(t)
	cutoff := cutoffTime(t)

	cases := []struct {
		sales float64
		want  float64
	}{
		{5000, 25000},   // 未达门槛 → 合计×杠杆
		{9999, 49995},   // 仍未达门槛
		{10000, 50000},  // 刚达门槛 → 固定上限
		{90000, 50000},  // 远超门槛 → 仍封顶
	}
	for i, c := range cases {
		uid := uint(i + 1)
		mustUser(t, db, &model.User{Username: "u", Power: 0})
		orderAt(t, db, uid, c.sales, types.OrderZyz, cutoff.Add(-time.Hour))
		if snap := maxOf(t, db, uid, legacyCfg()); snap.Max != c.want {
			t.Fatalf("sales=%v want %v got %v", c.sales, c.want, snap.Max)
		}
	}
}

// 老订单到期（status 8→9）不能让额度回弹：已结束订单仍计入 Before
func TestLegacy_MaxStableAfterOrderExpiry(t *testing.T) {
	db := setupCreditTestDB(t)
	cutoff := cutoffTime(t)

	mustUser(t, db, &model.User{Username: "old", Power: 0})
	orderAt(t, db, 1, 100000, types.OrderZyz, cutoff.Add(-time.Hour))
	if snap := maxOf(t, db, 1, legacyCfg()); snap.Max != 50000 {
		t.Fatalf("before expiry want 50000, got %v", snap.Max)
	}

	// 订单跑完
	if err := db.Model(&model.Order{}).Where("user_id = ?", 1).
		UpdateColumn("status", int(types.OrderYjs)).Error; err != nil {
		t.Fatal(err)
	}
	if snap := maxOf(t, db, 1, legacyCfg()); !snap.Legacy || snap.Max != 50000 {
		t.Fatalf("after expiry must stay legacy 50000, got legacy=%v max=%v", snap.Legacy, snap.Max)
	}
}

// 未配置分界时刻：全员按当前机制
func TestLegacy_CutoffUnset_FallsBackToCurrentMechanism(t *testing.T) {
	db := setupCreditTestDB(t)
	cfg := &types.SystemConfig{CreditLeverage: 5, CreditDirectRate: 0.5}

	mustUser(t, db, &model.User{Username: "u", Power: 0})
	orderAt(t, db, 1, 100000, types.OrderZyz, time.Now().Add(-24*time.Hour))

	snap := maxOf(t, db, 1, cfg)
	if snap.Legacy {
		t.Fatal("legacy must be inactive without cutoff")
	}
	if snap.Max != 500000 {
		t.Fatalf("want 500000 got %v", snap.Max)
	}
}

// max 被下调时可用额度同步截断，债务不变
func TestLegacy_AvailableTruncatedButDebtKept(t *testing.T) {
	db := setupCreditTestDB(t)
	cutoff := cutoffTime(t)

	// 改造前按旧机制已累积到 50 万 max、20 万可用、3 万债务
	mustUser(t, db, &model.User{
		Username: "old", Power: 0,
		CreditMax: 500000, CreditAvailable: 200000, CreditDebt: 30000,
	})
	orderAt(t, db, 1, 100000, types.OrderZyz, cutoff.Add(-time.Hour))

	maxOf(t, db, 1, legacyCfg())

	u := loadUser(t, db, 1)
	if u.CreditMax != 50000 {
		t.Fatalf("max want 50000 got %v", u.CreditMax)
	}
	if u.CreditAvailable != 50000 {
		t.Fatalf("available should truncate to 50000, got %v", u.CreditAvailable)
	}
	if u.CreditDebt != 30000 {
		t.Fatalf("debt must stay 30000, got %v", u.CreditDebt)
	}
}

// EvaluateCredit 纯函数分支
func TestEvaluateCredit_Pure(t *testing.T) {
	p := CreditMaxParamsFrom(legacyCfg())

	cases := []struct {
		name     string
		split    InvestSplit
		wantMax  float64
		wantBase float64
	}{
		{"分界前达标 + 分界后达标", InvestSplit{Before: 100000, After: 10000, Total: 110000}, 100000, 50000},
		{"分界前达标、无新增", InvestSplit{Before: 100000, Total: 100000}, 50000, 50000},
		{"分界前未达门槛", InvestSplit{Before: 9999, Total: 9999}, 0, 0},
		{"分界前为零、分界后达标", InvestSplit{After: 10000, Total: 10000}, 50000, 0},
		{"完全无投入", InvestSplit{}, 0, 0},
	}
	for _, c := range cases {
		got := EvaluateCredit(p, c.split)
		if got.Max != c.wantMax || got.Base != c.wantBase {
			t.Fatalf("%s: want max=%v base=%v, got max=%v base=%v",
				c.name, c.wantMax, c.wantBase, got.Max, got.Base)
		}
	}
}

// 分界后投入不足门槛时不计入：不按比例折算，也不能救活未达标的普通用户。
func TestEvaluateCredit_AfterBelowThreshold(t *testing.T) {
	p := CreditMaxParamsFrom(legacyCfg())

	cases := []struct {
		name        string
		split       InvestSplit
		wantMax     float64
		wantLegacy  bool
		wantCounted float64
	}{
		// A：T 前有 ≥1 万，T 后只投 8000 → 固定上限保住，新增不放大
		{"老会员 + 分界后 8000", InvestSplit{Before: 12000, After: 8000, Total: 20000}, 50000, true, 0},
		// A：T 前无订单，T 后只投 8000 → 完全无额度
		{"普通会员 + 分界后 8000", InvestSplit{After: 8000, Total: 8000}, 0, false, 0},
		// B：T 前无订单，T 后投 20000 → 达门槛，按杠杆计算
		{"普通会员 + 分界后 20000", InvestSplit{After: 20000, Total: 20000}, 100000, false, 20000},
		// B：T 前有 ≥1 万，T 后投 20000 → 基数 + 新增
		{"老会员 + 分界后 20000", InvestSplit{Before: 12000, After: 20000, Total: 32000}, 150000, true, 20000},
		// 边界：正好等于门槛即达标
		{"老会员 + 分界后恰好 10000", InvestSplit{Before: 12000, After: 10000, Total: 22000}, 100000, true, 10000},
		{"普通会员 + 分界后恰好 10000", InvestSplit{After: 10000, Total: 10000}, 50000, false, 10000},
	}
	for _, c := range cases {
		got := EvaluateCredit(p, c.split)
		if got.Max != c.wantMax || got.Legacy != c.wantLegacy || got.CountedAfter != c.wantCounted {
			t.Fatalf("%s: want max=%v legacy=%v counted=%v, got max=%v legacy=%v counted=%v",
				c.name, c.wantMax, c.wantLegacy, c.wantCounted,
				got.Max, got.Legacy, got.CountedAfter)
		}
	}
}

// 未启用分界时刻时，不足单笔门槛的投入仍不计入 Max。
func TestEvaluateCredit_NoCutoffStillRespectsMin(t *testing.T) {
	cfg := legacyCfg()
	cfg.CreditLegacyCutoff = ""
	p := CreditMaxParamsFrom(cfg)

	got := EvaluateCredit(p, InvestSplit{After: 8000, Total: 8000})
	if got.Max != 0 {
		t.Fatalf("without cutoff, 8000 below min should yield 0, got %v", got.Max)
	}
	got = EvaluateCredit(p, InvestSplit{After: 10000, Total: 10000})
	if got.Max != 50000 {
		t.Fatalf("without cutoff, 10000 should yield 50000, got %v", got.Max)
	}
}

// 门槛配成极小值（1）等价于「分界前任意投入即算老会员」，但零投入不得白拿基数。
// 注意：门槛 0 会被 OrDefault 回退成默认 1 万，避免配置缺字段时全员变成老会员。
func TestEvaluateCredit_TinyThresholdRequiresInvest(t *testing.T) {
	cfg := legacyCfg()
	cfg.CreditLegacyMinPower = 1
	p := CreditMaxParamsFrom(cfg)

	if got := EvaluateCredit(p, InvestSplit{Before: 1, Total: 1}); !got.Legacy || got.Max != 50000 {
		t.Fatalf("before=1 with threshold 1 should be legacy 50000, got legacy=%v max=%v", got.Legacy, got.Max)
	}
	if got := EvaluateCredit(p, InvestSplit{}); got.Legacy {
		t.Fatal("zero invest must never qualify as legacy")
	}
}