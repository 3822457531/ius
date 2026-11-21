package service

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"chatplus/core/types"
	"chatplus/store/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// creditTestDBSeq 保证每个测试拿独立的内存库名。
// 不能只靠 time.Now().UnixNano()：Windows 时钟精度约 15ms，相邻测试会撞名，
// 共享 cache=shared 的库会让彼此的用户/订单互相污染。
var creditTestDBSeq int64

func setupCreditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	seq := atomic.AddInt64(&creditTestDBSeq, 1)
	dsn := fmt.Sprintf("file:credit_test_%d_%d?mode=memory&cache=shared", time.Now().UnixNano(), seq)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Order{}, &model.PowerLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func creditCfg() *types.SystemConfig {
	return &types.SystemConfig{
		CreditLeverage:   5,
		CreditDirectRate: 0.5,
	}
}

func mustUser(t *testing.T, db *gorm.DB, u *model.User) {
	t.Helper()
	if err := db.Create(u).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func mustOrder(t *testing.T, db *gorm.DB, o *model.Order) {
	t.Helper()
	if err := db.Create(o).Error; err != nil {
		t.Fatalf("create order: %v", err)
	}
}

func loadUser(t *testing.T, db *gorm.DB, id uint) model.User {
	t.Helper()
	var u model.User
	if err := db.First(&u, id).Error; err != nil {
		t.Fatalf("load user %d: %v", id, err)
	}
	return u
}

func TestRecalcMax_FromSelfInvest(t *testing.T) {
	db := setupCreditTestDB(t)
	svc := NewCreditService(db)
	mustUser(t, db, &model.User{Username: "boss", Power: 1000})
	mustOrder(t, db, &model.Order{
		UserId: 1, Username: "boss", OrderNo: "o1",
		Status: types.OrderZyz, Sales: 10000, Power: 50000, NowPower: 50000,
	})

	if err := svc.RecalcMax(db, 1, creditCfg()); err != nil {
		t.Fatal(err)
	}
	u := loadUser(t, db, 1)
	if u.CreditMax != 50000 {
		t.Fatalf("CreditMax want 50000 got %v", u.CreditMax)
	}
	if u.CreditAvailable != 0 {
		t.Fatalf("CreditAvailable want 0 got %v", u.CreditAvailable)
	}
}

func TestGrantAvailable_CapByMax(t *testing.T) {
	db := setupCreditTestDB(t)
	svc := NewCreditService(db)
	mustUser(t, db, &model.User{Username: "boss", Power: 1000})
	mustOrder(t, db, &model.Order{
		UserId: 1, Username: "boss", OrderNo: "o1",
		Status: types.OrderZyz, Sales: 10000, Power: 50000, NowPower: 50000,
	})
	if err := svc.RecalcMax(db, 1, creditCfg()); err != nil {
		t.Fatal(err)
	}

	// 直推投入 10万 → 50% = 5万，恰好顶满 Max
	if err := svc.GrantAvailableFromDirect(db, 1, 100000, creditCfg(), "down1"); err != nil {
		t.Fatal(err)
	}
	u := loadUser(t, db, 1)
	if u.CreditAvailable != 50000 {
		t.Fatalf("after 10w invest avail want 50000 got %v", u.CreditAvailable)
	}

	// 再投入 20万也不超过 Max
	if err := svc.GrantAvailableFromDirect(db, 1, 200000, creditCfg(), "down2"); err != nil {
		t.Fatal(err)
	}
	u = loadUser(t, db, 1)
	if u.CreditAvailable != 50000 {
		t.Fatalf("cap: avail want 50000 got %v", u.CreditAvailable)
	}
	if u.CreditMax != 50000 {
		t.Fatalf("max should stay 50000 got %v", u.CreditMax)
	}
}

func TestSpendForGift_DebtAndNoRestoreOnRepay(t *testing.T) {
	db := setupCreditTestDB(t)
	svc := NewCreditService(db)
	mustUser(t, db, &model.User{
		Username: "boss", Power: 1000,
		CreditMax: 50000, CreditAvailable: 50000, CreditDebt: 0,
	})
	// 自投 1 万 → 基数 Max=5 万；报单消耗会从 Max 再扣
	mustOrder(t, db, &model.Order{
		UserId: 1, Username: "boss", OrderNo: "boss-self",
		Status: types.OrderZyz, Sales: 10000, Power: 50000, NowPower: 50000,
	})

	if err := svc.SpendForGift(db, 1, "boss", 20000, "ORD-C1", "recv", creditCfg()); err != nil {
		t.Fatal(err)
	}
	u := loadUser(t, db, 1)
	if u.CreditAvailable != 30000 {
		t.Fatalf("avail want 30000 got %v", u.CreditAvailable)
	}
	if u.CreditDebt != 20000 {
		t.Fatalf("debt want 20000 got %v", u.CreditDebt)
	}
	// 用掉 2 万额度 → Max 同步扣到 3 万
	if u.CreditMax != 30000 {
		t.Fatalf("max want 30000 got %v", u.CreditMax)
	}

	// 收益入账后还款：先模拟入账 +100，再还款
	if err := db.Model(&model.User{}).Where("id = ?", 1).
		UpdateColumn("power", gorm.Expr("power + ?", 100)).Error; err != nil {
		t.Fatal(err)
	}
	repay, err := svc.RepayFromIncome(db, 1, "boss", 100, "ORD-INCOME")
	if err != nil {
		t.Fatal(err)
	}
	if repay != 100 {
		t.Fatalf("repay want 100 got %v", repay)
	}
	u = loadUser(t, db, 1)
	if u.CreditDebt != 19900 {
		t.Fatalf("debt want 19900 got %v", u.CreditDebt)
	}
	// 方案 A：还款不回补可用额度
	if u.CreditAvailable != 30000 {
		t.Fatalf("avail must NOT restore, want 30000 got %v", u.CreditAvailable)
	}
	if u.Power != 1000 { // 1000+100-100
		t.Fatalf("power want 1000 got %v", u.Power)
	}

	var repayLog int64
	db.Model(&model.PowerLog{}).Where("user_id = ? AND model = ?", 1, CreditLogModelRepay).Count(&repayLog)
	if repayLog != 1 {
		t.Fatalf("expect 1 repay log, got %d", repayLog)
	}
	var spendLog int64
	db.Model(&model.PowerLog{}).Where("user_id = ? AND model = ?", 1, CreditLogModelSpend).Count(&spendLog)
	if spendLog != 1 {
		t.Fatalf("expect 1 spend log, got %d", spendLog)
	}
}

func TestSpendForGift_Insufficient(t *testing.T) {
	db := setupCreditTestDB(t)
	svc := NewCreditService(db)
	mustUser(t, db, &model.User{
		Username: "boss", Power: 100,
		CreditMax: 5000, CreditAvailable: 1000, CreditDebt: 0,
	})
	err := svc.SpendForGift(db, 1, "boss", 2000, "ORD-X", "recv", creditCfg())
	if err == nil {
		t.Fatal("expected insufficient error")
	}
	u := loadUser(t, db, 1)
	if u.CreditAvailable != 1000 || u.CreditDebt != 0 {
		t.Fatalf("state should unchanged, avail=%v debt=%v", u.CreditAvailable, u.CreditDebt)
	}
}

func TestRepayFromIncome_NoDebt(t *testing.T) {
	db := setupCreditTestDB(t)
	svc := NewCreditService(db)
	mustUser(t, db, &model.User{Username: "boss", Power: 500, CreditDebt: 0})
	repay, err := svc.RepayFromIncome(db, 1, "boss", 50, "x")
	if err != nil {
		t.Fatal(err)
	}
	if repay != 0 {
		t.Fatalf("want 0 repay got %v", repay)
	}
	u := loadUser(t, db, 1)
	if u.Power != 500 {
		t.Fatalf("power should unchanged got %v", u.Power)
	}
}

func TestRepayFromIncome_PartialAndFull(t *testing.T) {
	db := setupCreditTestDB(t)
	svc := NewCreditService(db)
	mustUser(t, db, &model.User{
		Username: "boss", Power: 0,
		CreditAvailable: 0, CreditDebt: 80, CreditMax: 50000,
	})
	// 入账 100，债务只有 80 → 还 80，余额剩 20
	_ = db.Model(&model.User{}).Where("id = ?", 1).UpdateColumn("power", 100)
	repay, err := svc.RepayFromIncome(db, 1, "boss", 100, "full")
	if err != nil {
		t.Fatal(err)
	}
	if repay != 80 {
		t.Fatalf("repay want 80 got %v", repay)
	}
	u := loadUser(t, db, 1)
	if u.CreditDebt != 0 {
		t.Fatalf("debt want 0 got %v", u.CreditDebt)
	}
	if u.Power != 20 {
		t.Fatalf("power want 20 got %v", u.Power)
	}
	if u.CreditAvailable != 0 {
		t.Fatalf("avail still 0, got %v", u.CreditAvailable)
	}
}

func TestRecalcMax_SmallOrdersDoNotIncreaseMax(t *testing.T) {
	db := setupCreditTestDB(t)
	svc := NewCreditService(db)
	mustUser(t, db, &model.User{Username: "u", Power: 1000})
	// 多笔小额合计远超 1 万，但单笔都 < 1 万 → Max 仍为 0
	for i, s := range []float64{3000, 4000, 5000, 9000} {
		mustOrder(t, db, &model.Order{
			UserId: 1, Username: "u", OrderNo: fmt.Sprintf("small-%d", i),
			Status: types.OrderZyz, Sales: s, Power: s * 5, NowPower: s * 5,
		})
	}
	if err := svc.RecalcMax(db, 1, creditCfg()); err != nil {
		t.Fatal(err)
	}
	if u := loadUser(t, db, 1); u.CreditMax != 0 {
		t.Fatalf("small orders must not raise max, got %v", u.CreditMax)
	}
	// 再加一笔单笔 ≥ 1 万 → 才计入
	mustOrder(t, db, &model.Order{
		UserId: 1, Username: "u", OrderNo: "big-1",
		Status: types.OrderZyz, Sales: 10000, Power: 50000, NowPower: 50000,
	})
	if err := svc.RecalcMax(db, 1, creditCfg()); err != nil {
		t.Fatal(err)
	}
	if u := loadUser(t, db, 1); u.CreditMax != 50000 {
		t.Fatalf("single >=10000 should raise max to 50000, got %v", u.CreditMax)
	}
}

func TestAfterInvest_UpdatesSelfAndSuperior(t *testing.T) {
	db := setupCreditTestDB(t)
	svc := NewCreditService(db)

	mustUser(t, db, &model.User{Username: "boss", Power: 100}) // id=1
	mustOrder(t, db, &model.Order{
		UserId: 1, Username: "boss", OrderNo: "boss-o1",
		Status: types.OrderZyz, Sales: 10000, Power: 50000, NowPower: 50000,
	})
	if err := svc.RecalcMax(db, 1, creditCfg()); err != nil {
		t.Fatal(err)
	}

	mustUser(t, db, &model.User{Username: "child", Power: 0, ZjFid: 1}) // id=2
	mustOrder(t, db, &model.Order{
		UserId: 2, Username: "child", OrderNo: "child-o1",
		Status: types.OrderZyz, Sales: 20000, Power: 100000, NowPower: 100000,
	})

	child := loadUser(t, db, 2)
	if err := svc.AfterInvest(db, child, 20000, creditCfg()); err != nil {
		t.Fatal(err)
	}

	boss := loadUser(t, db, 1)
	// 20000 * 0.5 = 10000 可用
	if boss.CreditAvailable != 10000 {
		t.Fatalf("boss avail want 10000 got %v", boss.CreditAvailable)
	}
	if boss.CreditMax != 50000 {
		t.Fatalf("boss max want 50000 got %v", boss.CreditMax)
	}

	child = loadUser(t, db, 2)
	// 自投 2万 ×5 = 最大 10万
	if child.CreditMax != 100000 {
		t.Fatalf("child max want 100000 got %v", child.CreditMax)
	}
}

// 授信报单：可用额度只给收款人的直推上级；隔代付款人不拿额度；收款人 Max 不因授信单增加。
// 链：giver(1) → mid(2) → receiver(3)；giver 给孙代 receiver 报单。
func TestAfterGift_TopupGoesToDirectReferrer(t *testing.T) {
	db := setupCreditTestDB(t)
	svc := NewCreditService(db)

	mustUser(t, db, &model.User{Username: "giver", Power: 0})    // id=1
	mustUser(t, db, &model.User{Username: "mid", ZjFid: 1})      // id=2
	mustUser(t, db, &model.User{Username: "receiver", ZjFid: 2}) // id=3

	// mid 自投 1万 → max=5万，才能承接可用额度
	mustOrder(t, db, &model.Order{
		UserId: 2, Username: "mid", OrderNo: "m-o1",
		Status: types.OrderZyz, Sales: 10000, Power: 50000, NowPower: 50000,
	})
	if err := svc.RecalcMax(db, 2, creditCfg()); err != nil {
		t.Fatal(err)
	}

	giver := loadUser(t, db, 1)
	receiver := loadUser(t, db, 3)

	mustOrder(t, db, &model.Order{
		UserId: 3, Username: "receiver", OrderNo: "r-o1",
		Status: types.OrderZyz, PayWay: "credit_gift", Sales: 20000, Power: 100000, NowPower: 100000,
	})
	if err := svc.AfterGift(db, giver, receiver, 20000, creditCfg()); err != nil {
		t.Fatal(err)
	}

	if g := loadUser(t, db, 1); g.CreditAvailable != 0 {
		t.Fatalf("隔代付款人不应获得额度，got %v", g.CreditAvailable)
	}
	// 直推 mid：20000×0.5=10000
	if mid := loadUser(t, db, 2); mid.CreditAvailable != 10000 {
		t.Fatalf("直推上级 avail want 10000 got %v", mid.CreditAvailable)
	}
	// 授信报单不计入收款人自投 → Max 仍为 0
	if r := loadUser(t, db, 3); r.CreditMax != 0 {
		t.Fatalf("receiver max want 0 got %v", r.CreditMax)
	}
}

// 直推上级可用额度未满时，授信报单后额度加给直推（20000×0.5=10000，受 max 限制）
func TestAfterGift_TopupCappedByDirectReferrerMax(t *testing.T) {
	db := setupCreditTestDB(t)
	svc := NewCreditService(db)

	mustUser(t, db, &model.User{Username: "giver"})             // id=1
	mustUser(t, db, &model.User{Username: "receiver", ZjFid: 1}) // id=2；giver 即直推

	mustOrder(t, db, &model.Order{
		UserId: 1, Username: "giver", OrderNo: "g-o1",
		Status: types.OrderZyz, Sales: 10000, Power: 50000, NowPower: 50000,
	})
	if err := db.Model(&model.User{}).Where("id = ?", 1).
		UpdateColumn("credit_available", 0.0).Error; err != nil {
		t.Fatal(err)
	}

	giver := loadUser(t, db, 1)
	receiver := loadUser(t, db, 2)
	mustOrder(t, db, &model.Order{
		UserId: 2, Username: "receiver", OrderNo: "r-o1",
		Status: types.OrderZyz, PayWay: "credit_gift", Sales: 20000, Power: 100000, NowPower: 100000,
	})

	if err := svc.AfterGift(db, giver, receiver, 20000, creditCfg()); err != nil {
		t.Fatal(err)
	}

	if g := loadUser(t, db, 1); g.CreditAvailable != 10000 {
		t.Fatalf("直推 avail want 10000 got %v", g.CreditAvailable)
	}
	if r := loadUser(t, db, 2); r.CreditMax != 0 {
		t.Fatalf("receiver max want 0 (credit_gift 不计自投) got %v", r.CreditMax)
	}
}

func TestCreditConfigDefaults(t *testing.T) {
	cfg := types.SystemConfig{}
	if cfg.CreditLeverageOrDefault() != 5 {
		t.Fatalf("leverage default want 5")
	}
	if cfg.CreditDirectRateOrDefault() != 0.5 {
		t.Fatalf("rate default want 0.5")
	}
	cfg.CreditDirectRate = 2
	if cfg.CreditDirectRateOrDefault() != 1 {
		t.Fatalf("rate >1 should clamp to 1")
	}
}
