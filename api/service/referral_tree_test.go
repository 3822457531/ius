package service

import (
	"fmt"
	"testing"
	"time"

	"chatplus/store/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func setupReferralTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:referral_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
		// 与生产库保持一致：表前缀 chatgpt_（判定逻辑直接引用 chatgpt_users）
		NamingStrategy: schema.NamingStrategy{TablePrefix: "chatgpt_"},
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// 链：A(1) -> B(2) -> C(3)，旁部门 D(4) 挂在 A 名下，E(5) 是 D 的下级
func seedReferralChain(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupReferralTestDB(t)
	users := []model.User{
		{BaseModel: model.BaseModel{Id: 1}, Username: "A", ZjFid: 0},
		{BaseModel: model.BaseModel{Id: 2}, Username: "B", ZjFid: 1},
		{BaseModel: model.BaseModel{Id: 3}, Username: "C", ZjFid: 2},
		{BaseModel: model.BaseModel{Id: 4}, Username: "D", ZjFid: 1},
		{BaseModel: model.BaseModel{Id: 5}, Username: "E", ZjFid: 4},
	}
	for _, u := range users {
		row := u
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create user %s: %v", u.Username, err)
		}
	}
	return db
}

func TestIsInReferralSubtree(t *testing.T) {
	db := seedReferralChain(t)

	cases := []struct {
		name     string
		ancestor uint
		user     uint
		want     bool
	}{
		{"直推下级", 1, 2, true},
		{"隔代下级", 1, 3, true},
		{"旁部门自己", 1, 4, true},         // 对 A 而言 D 也是伞下（A 的另一个直推）
		{"旁部门的下级", 1, 5, true},        // 对 A 仍在伞下
		{"旁部门之间不可互报", 2, 4, false},   // B 不能给旁部门 D 报单
		{"旁部门的下级不可互报", 2, 5, false},
		{"隔代旁部门不可互报", 3, 5, false},
		{"不能报给自己", 1, 1, false},
		{"不能报给上级", 2, 1, false},
		{"不能报给隔代上级", 3, 1, false},
		{"用户不存在", 1, 999, false},
		{"祖先不存在", 999, 2, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := IsInReferralSubtree(db, tc.ancestor, tc.user)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tc.want {
				t.Fatalf("IsInReferralSubtree(%d,%d)=%v, want %v", tc.ancestor, tc.user, got, tc.want)
			}
		})
	}
}

func TestIsInReferralSubtreeDeepChain(t *testing.T) {
	db := setupReferralTestDB(t)
	const depth = 200
	for id := 1; id <= depth; id++ {
		zj := id - 1
		if id == 1 {
			zj = 0
		}
		if err := db.Create(&model.User{
			BaseModel: model.BaseModel{Id: uint(id)},
			Username:  fmt.Sprintf("u%d", id),
			ZjFid:     zj,
		}).Error; err != nil {
			t.Fatalf("create user %d: %v", id, err)
		}
	}

	got, err := IsInReferralSubtree(db, 1, depth)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !got {
		t.Fatalf("深度 %d 的后代应判定为伞下", depth)
	}
}