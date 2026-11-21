package service

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"chatplus/store/model"

	"github.com/BurntSushi/toml"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// 仅在显式开启时跑：go test ./service -run TestCreditAPISmoke -v -vet=off -count=1
// 需要本机后端已启动，且能连 config.toml 中的 MySQL。
func TestCreditAPISmoke(t *testing.T) {
	if os.Getenv("CREDIT_API_SMOKE") != "1" {
		t.Skip("set CREDIT_API_SMOKE=1 to run live API smoke")
	}

	type appCfg struct {
		MysqlDns string
		Session  struct {
			SecretKey string
			MaxAge    int
		}
		Listen string
	}
	var cfg appCfg
	if _, err := toml.DecodeFile("../config.toml", &cfg); err != nil {
		// try cwd = api
		if _, err2 := toml.DecodeFile("config.toml", &cfg); err2 != nil {
			t.Fatalf("load config: %v / %v", err, err2)
		}
	}
	if cfg.Session.SecretKey == "" || cfg.MysqlDns == "" {
		t.Fatal("missing session/mysql in config")
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

	var giver model.User
	if err := db.Where("username = ?", "test").First(&giver).Error; err != nil {
		t.Fatalf("find user test: %v", err)
	}
	var receiver model.User
	if err := db.Where("username <> ? AND status = 1", "test").Order("id asc").First(&receiver).Error; err != nil {
		t.Fatalf("find receiver: %v", err)
	}
	var invite model.InviteCode
	if err := db.Where("user_id = ?", receiver.Id).First(&invite).Error; err != nil {
		t.Fatalf("receiver invite code: %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": giver.Id,
		"expired": time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(cfg.Session.SecretKey))
	if err != nil {
		t.Fatal(err)
	}

	base := "http://127.0.0.1:5678"
	client := &http.Client{Timeout: 20 * time.Second}

	// 1) profile 应返回授信三字段
	req, _ := http.NewRequest(http.MethodGet, base+"/api/user/profile", nil)
	req.Header.Set("Authorization", tokenStr)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("profile request: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("profile status %d body=%s", resp.StatusCode, body)
	}
	var profileWrap struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	_ = json.Unmarshal(body, &profileWrap)
	var profile map[string]interface{}
	_ = json.Unmarshal(profileWrap.Data, &profile)
	for _, k := range []string{"credit_max", "credit_available", "credit_debt"} {
		if _, ok := profile[k]; !ok {
			t.Fatalf("profile missing %s, data=%s", k, profileWrap.Data)
		}
	}
	t.Logf("profile credit_max=%v credit_available=%v credit_debt=%v",
		profile["credit_max"], profile["credit_available"], profile["credit_debt"])

	// 2) 无 token 应失败
	payload, _ := json.Marshal(map[string]interface{}{
		"code": invite.Code, "power": 1000, "password": "wrong",
	})
	req2, _ := http.NewRequest(http.MethodPost, base+"/api/payment/confirmJihuoCreditGift", bytes.NewReader(payload))
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("unauth request: %v", err)
	}
	b2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	t.Logf("unauth status=%d body=%s", resp2.StatusCode, b2)

	// 3) 有 token 但额度不足 / 密码错误 —— 应返回业务错误而不是 500
	req3, _ := http.NewRequest(http.MethodPost, base+"/api/payment/confirmJihuoCreditGift", bytes.NewReader(payload))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Authorization", tokenStr)
	resp3, err := client.Do(req3)
	if err != nil {
		t.Fatalf("credit gift request: %v", err)
	}
	b3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	t.Logf("creditGift(auth) status=%d body=%s", resp3.StatusCode, b3)
	if resp3.StatusCode >= 500 {
		t.Fatalf("unexpected 5xx: %s", b3)
	}

	// 4) 路由存在性：错误 body 也不应 404
	if resp3.StatusCode == 404 {
		t.Fatal("confirmJihuoCreditGift route 404")
	}

	t.Logf("invite_code=%s giver=%d receiver=%d", invite.Code, giver.Id, receiver.Id)
}
