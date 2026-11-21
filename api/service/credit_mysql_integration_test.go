package service

import (
	"fmt"
	"os"
	"testing"
	"time"

	"chatplus/core/types"
	"chatplus/store/model"

	"github.com/BurntSushi/toml"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// go test ./service -run TestCreditMySQLIntegration -v -vet=off -count=1
// 默认开启；设 CREDIT_MYSQL_IT=0 可跳过
func TestCreditMySQLIntegration(t *testing.T) {
	if os.Getenv("CREDIT_MYSQL_IT") == "0" {
		t.Skip("CREDIT_MYSQL_IT=0")
	}

	type appCfg struct {
		MysqlDns string
	}
	var cfg appCfg
	if _, err := toml.DecodeFile("config.toml", &cfg); err != nil {
		if _, err2 := toml.DecodeFile("../config.toml", &cfg); err2 != nil {
			t.Fatalf("config: %v / %v", err, err2)
		}
	}

	db, err := gorm.Open(mysql.Open(cfg.MysqlDns), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "chatgpt_",
			SingularTable: false,
		},
	})
	if err != nil {
		t.Fatalf("mysql: %v", err)
	}

	// 确认授信列已存在
	if !db.Migrator().HasColumn(&model.User{}, "credit_max") ||
		!db.Migrator().HasColumn(&model.User{}, "credit_available") ||
		!db.Migrator().HasColumn(&model.User{}, "credit_debt") {
		t.Fatal("credit columns missing on chatgpt_users — restart backend once to migrate")
	}

	tag := fmt.Sprintf("ct%d", time.Now().Unix()%1e8)
	bossName := "__credit_boss_" + tag
	childName := "__credit_child_" + tag

	cleanup := func() {
		var ids []uint
		_ = db.Model(&model.User{}).Where("username IN ?", []string{bossName, childName}).Pluck("id", &ids)
		if len(ids) > 0 {
			_ = db.Where("user_id IN ?", ids).Delete(&model.PowerLog{}).Error
			_ = db.Where("user_id IN ?", ids).Delete(&model.Order{}).Error
			_ = db.Where("id IN ?", ids).Delete(&model.User{}).Error
		}
	}
	cleanup()
	defer cleanup()

	boss := model.User{
		Username: bossName, Nickname: "credit-boss", Password: "x", Salt: "x",
		Status: true, Power: 1000, IsJh: 1,
	}
	child := model.User{
		Username: childName, Nickname: "credit-child", Password: "x", Salt: "x",
		Status: true, Power: 0, IsJh: 0,
	}
	if err := db.Create(&boss).Error; err != nil {
		t.Fatalf("create boss: %v", err)
	}
	child.ZjFid = int(boss.Id)
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("create child: %v", err)
	}

	svc := NewCreditService(db)
	cfgSys := &types.SystemConfig{CreditLeverage: 5, CreditDirectRate: 0.5}

	// 1) boss 自投 1万 → Max=5万，Available=0
	if err := db.Create(&model.Order{
		UserId: boss.Id, Username: bossName, OrderNo: "IT-BOSS-" + tag,
		Status: types.OrderZyz, Sales: 10000, Power: 50000, NowPower: 50000, PayWay: "it",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.AfterInvest(db, loadUser(t, db, boss.Id), 10000, cfgSys); err != nil {
		t.Fatal(err)
	}
	b := loadUser(t, db, boss.Id)
	if b.CreditMax != 50000 || b.CreditAvailable != 0 {
		t.Fatalf("after self invest: max=%v avail=%v want 50000/0", b.CreditMax, b.CreditAvailable)
	}
	t.Logf("[1] boss self-invest OK max=%v avail=%v", b.CreditMax, b.CreditAvailable)

	// 2) 直推 child 投入 10万 → boss avail += 5万（封顶）
	if err := db.Create(&model.Order{
		UserId: child.Id, Username: childName, OrderNo: "IT-CHILD-" + tag,
		Status: types.OrderZyz, Sales: 100000, Power: 500000, NowPower: 500000, PayWay: "it",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.AfterInvest(db, loadUser(t, db, child.Id), 100000, cfgSys); err != nil {
		t.Fatal(err)
	}
	b = loadUser(t, db, boss.Id)
	c := loadUser(t, db, child.Id)
	if b.CreditAvailable != 50000 {
		t.Fatalf("after downline 10w: boss avail=%v want 50000", b.CreditAvailable)
	}
	if c.CreditMax != 500000 {
		t.Fatalf("child max=%v want 500000", c.CreditMax)
	}
	t.Logf("[2] downline grant OK boss.avail=%v child.max=%v", b.CreditAvailable, c.CreditMax)

	// 3) 再投 20万仍不超 Max
	if err := svc.GrantAvailableFromDirect(db, boss.Id, 200000, cfgSys, childName); err != nil {
		t.Fatal(err)
	}
	b = loadUser(t, db, boss.Id)
	if b.CreditAvailable != 50000 {
		t.Fatalf("cap failed avail=%v", b.CreditAvailable)
	}
	t.Logf("[3] cap OK avail=%v", b.CreditAvailable)

	// 4) 授信报单花掉 2万：avail↓ debt↑ max 同步扣 2万
	if err := svc.SpendForGift(db, boss.Id, bossName, 20000, "IT-GIFT-"+tag, childName, cfgSys); err != nil {
		t.Fatal(err)
	}
	b = loadUser(t, db, boss.Id)
	if b.CreditAvailable != 30000 || b.CreditDebt != 20000 || b.CreditMax != 30000 {
		t.Fatalf("after spend avail=%v debt=%v max=%v", b.CreditAvailable, b.CreditDebt, b.CreditMax)
	}
	t.Logf("[4] spend OK avail=%v debt=%v", b.CreditAvailable, b.CreditDebt)

	// 5) 收益还款：入账后再还，不回补 avail
	powerBefore := b.Power
	_ = db.Model(&model.User{}).Where("id = ?", boss.Id).UpdateColumn("power", gorm.Expr("power + ?", 5000))
	repay, err := svc.RepayFromIncome(db, boss.Id, bossName, 5000, "IT-REPAY-"+tag)
	if err != nil {
		t.Fatal(err)
	}
	if repay != 5000 {
		t.Fatalf("repay=%v want 5000", repay)
	}
	b = loadUser(t, db, boss.Id)
	if b.CreditDebt != 15000 {
		t.Fatalf("debt=%v want 15000", b.CreditDebt)
	}
	if b.CreditAvailable != 30000 {
		t.Fatalf("scheme A fail: avail restored to %v", b.CreditAvailable)
	}
	if b.Power != powerBefore {
		t.Fatalf("power after repay+credit should be %v got %v", powerBefore, b.Power)
	}
	t.Logf("[5] repay OK debt=%v avail=%v(no restore) power=%v", b.CreditDebt, b.CreditAvailable, b.Power)

	// 6) 流水校验
	var grantN, spendN, repayN int64
	_ = db.Model(&model.PowerLog{}).Where("user_id = ? AND model = ?", boss.Id, CreditLogModelGrant).Count(&grantN)
	_ = db.Model(&model.PowerLog{}).Where("user_id = ? AND model = ?", boss.Id, CreditLogModelSpend).Count(&spendN)
	_ = db.Model(&model.PowerLog{}).Where("user_id = ? AND model = ?", boss.Id, CreditLogModelRepay).Count(&repayN)
	if grantN < 1 || spendN != 1 || repayN != 1 {
		t.Fatalf("logs grant=%d spend=%d repay=%d", grantN, spendN, repayN)
	}
	t.Logf("[6] logs OK grant=%d spend=%d repay=%d", grantN, spendN, repayN)

	// 7) 真实用户 test：列可读、与 profile 口径一致
	var real model.User
	if err := db.Select("id", "username", "credit_max", "credit_available", "credit_debt").
		Where("username = ?", "test").First(&real).Error; err != nil {
		t.Logf("[7] skip real user check: %v", err)
	} else {
		_ = svc.RecalcMax(db, real.Id, cfgSys)
		real = loadUser(t, db, real.Id)
		t.Logf("[7] real user test id=%d max=%v avail=%v debt=%v", real.Id, real.CreditMax, real.CreditAvailable, real.CreditDebt)
		if real.CreditAvailable > real.CreditMax+1e-6 {
			t.Fatalf("invariant broken: avail > max")
		}
		if real.CreditDebt < 0 {
			t.Fatalf("negative debt")
		}
	}
}
