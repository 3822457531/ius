package admin

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/handler"
	logger2 "chatplus/logger"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils"
	"chatplus/utils/resp"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/mojocn/base64Captcha"
	"gorm.io/gorm"
)

var logger = logger2.GetLogger()

// Manager 管理员
type Manager struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Captcha   string `json:"captcha"`    // 验证码
	CaptchaId string `json:"captcha_id"` // 验证码id
}

const SuperManagerID = 1

type ManagerHandler struct {
	handler.BaseHandler
	redis *redis.Client
}

func NewAdminHandler(app *core.AppServer, db *gorm.DB, client *redis.Client) *ManagerHandler {
	return &ManagerHandler{BaseHandler: handler.BaseHandler{DB: db, App: app}, redis: client}
}

// Login 登录
func (h *ManagerHandler) Login(c *gin.Context) {
	var data Manager
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	// add captcha
	if !base64Captcha.DefaultMemStore.Verify(data.CaptchaId, data.Captcha, true) {
		resp.ERROR(c, "验证码错误!")
		return
	}

	var manager model.AdminUser
	res := h.DB.Model(&model.AdminUser{}).Where("username = ?", data.Username).First(&manager)
	loginFailMsg := "用户名或密码错误"
	if res.Error != nil {
		resp.ERROR(c, loginFailMsg)
		return
	}
	password := utils.GenPassword(data.Password, manager.Salt)
	if password != manager.Password {
		resp.ERROR(c, loginFailMsg)
		return
	}

	// 超级管理员默认是ID:1
	if manager.Id != SuperManagerID && manager.Status == false {
		resp.ERROR(c, "该用户已被禁止登录，请联系超级管理员")
		return
	}

	secret := strings.TrimSpace(h.App.Config.AdminSession.SecretKey)
	if len(secret) < utils.MinJWTSecretLen {
		resp.ERROR(c, "后台会话密钥未正确配置")
		return
	}
	maxAge := h.App.Config.AdminSession.MaxAge
	if maxAge <= 0 {
		maxAge = 86400
	}
	ttl := time.Second * time.Duration(maxAge)
	jti := uuid.NewString()
	now := time.Now()
	// jti + admin_id 绑定；不再把 Redis 键设为可伪造的 user_id
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"jti":      jti,
		"admin_id": manager.Id,
		"user_id":  manager.Id, // 兼容现有 GetLoginUserId
		"expired":  now.Add(ttl).Unix(),
		"iat":      now.Unix(),
		"iss":      "os2ai-admin",
	})
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		resp.ERROR(c, "Failed to generate token")
		return
	}
	ctx := context.Background()
	// 单会话：踢掉旧 jti
	curKey := utils.AdminCurKey(manager.Id)
	if oldJTI, err := h.redis.Get(ctx, curKey).Result(); err == nil && oldJTI != "" {
		_ = h.redis.Del(ctx, utils.AdminSessKey(oldJTI))
	}
	if err := h.redis.Set(ctx, utils.AdminSessKey(jti), manager.Id, ttl).Err(); err != nil {
		resp.ERROR(c, "error with save token")
		return
	}
	_ = h.redis.Set(ctx, curKey, jti, ttl)

	// 更新最后登录时间和IP
	manager.LastLoginIp = c.ClientIP()
	manager.LastLoginAt = time.Now().Unix()
	h.DB.Updates(&manager)

	var result = struct {
		IsSuperAdmin bool   `json:"is_super_admin"`
		Token        string `json:"token"`
	}{
		IsSuperAdmin: manager.Id == 1,
		Token:        tokenString,
	}

	resp.SUCCESS(c, result)
}

// Logout 注销
func (h *ManagerHandler) Logout(c *gin.Context) {
	ctx := context.Background()
	if jti, ok := c.Get("ADMIN_JTI"); ok {
		jtiStr := fmt.Sprintf("%v", jti)
		_ = h.redis.Del(ctx, utils.AdminSessKey(jtiStr))
	}
	id := h.GetLoginUserId(c)
	if id > 0 {
		_ = h.redis.Del(ctx, utils.AdminCurKey(id))
		// 清理旧版 admin/{id} 键
		_ = h.redis.Del(ctx, fmt.Sprintf("admin/%d", id))
	}
	resp.SUCCESS(c)
}

// Session 会话检测
func (h *ManagerHandler) Session(c *gin.Context) {
	id := h.GetLoginUserId(c)
	if id == 0 {
		resp.NotAuth(c)
		return
	}
	var manager model.AdminUser
	res := h.DB.Where("id", id).First(&manager)
	if res.Error != nil {
		resp.NotAuth(c)
		return
	}
	// 不回传密码哈希与盐
	manager.Password = ""
	manager.Salt = ""
	resp.SUCCESS(c, manager)
}

// List 数据列表
func (h *ManagerHandler) List(c *gin.Context) {
	var items []model.AdminUser
	res := h.DB.Find(&items)
	if res.Error != nil {
		resp.ERROR(c, res.Error.Error())
		return
	}

	users := make([]vo.AdminUser, 0)
	for _, item := range items {
		var u vo.AdminUser
		err := utils.CopyObject(item, &u)
		if err != nil {
			continue
		}
		u.Id = item.Id
		u.CreatedAt = item.CreatedAt.Unix()
		users = append(users, u)
	}

	resp.SUCCESS(c, users)

}

func (h *ManagerHandler) Save(c *gin.Context) {
	var data struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Status   bool   `json:"status"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	var user model.AdminUser
	res := h.DB.Where("username", data.Username).First(&user)
	if res.Error == nil {
		resp.ERROR(c, "用户名已存在")
		return
	}

	// 生成密码
	salt := utils.RandString(8)
	password := utils.GenPassword(data.Password, salt)
	res = h.DB.Save(&model.AdminUser{
		Username: data.Username,
		Password: password,
		Salt:     salt,
		Status:   data.Status,
	})
	if res.Error != nil {
		resp.ERROR(c, "failed with update database")
		return
	}

	resp.SUCCESS(c)
}

// Remove 删除管理员
func (h *ManagerHandler) Remove(c *gin.Context) {
	id := h.GetInt(c, "id", 0)
	if id <= 0 {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	if id == SuperManagerID {
		resp.ERROR(c, "超级管理员不能删除")
		return
	}

	res := h.DB.Where("id", id).Delete(&model.AdminUser{})
	if res.Error != nil {
		resp.ERROR(c, res.Error.Error())
		return
	}

	resp.SUCCESS(c)
}

// Enable 启用/禁用
func (h *ManagerHandler) Enable(c *gin.Context) {
	var data struct {
		Id      uint `json:"id"`
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	res := h.DB.Model(&model.AdminUser{}).Where("id", data.Id).UpdateColumn("status", data.Enabled)
	if res.Error != nil {
		resp.ERROR(c, res.Error.Error())
		return
	}
	resp.SUCCESS(c)
}

// ResetPass 重置密码
func (h *ManagerHandler) ResetPass(c *gin.Context) {
	id := h.GetLoginUserId(c)
	if id != SuperManagerID {
		resp.ERROR(c, "只有超级管理员能够进行该操作")
		return
	}

	var data struct {
		Id       int    `json:"id"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	var user model.AdminUser
	res := h.DB.Where("id", data.Id).First(&user)
	if res.Error != nil {
		resp.ERROR(c, res.Error.Error())
		return
	}

	password := utils.GenPassword(data.Password, user.Salt)
	user.Password = password
	res = h.DB.Updates(&user)
	if res.Error != nil {
		resp.ERROR(c, res.Error.Error())
		return
	}

	resp.SUCCESS(c)
}

// PlacementTree 获取安置树节点数据
func (h *ManagerHandler) PlacementTree(c *gin.Context) {
	type PlacementNode struct {
		UserID       int     `json:"user_id"`
		ParentID     int     `json:"parent_id"`
		LeftChildID  int     `json:"left_child_id"`
		RightChildID int     `json:"right_child_id"`
		LeftPerf     float64 `json:"left_perf"`
		RightPerf    float64 `json:"right_perf"`
		TotalPerf    float64 `json:"total_perf"`
	}

	var nodes []PlacementNode
	res := h.DB.Table("chatgpt_placement_nodes").Find(&nodes)
	if res.Error != nil {
		resp.ERROR(c, res.Error.Error())
		return
	}

	// 获取用户信息
	type UserInfo struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Nickname string `json:"nickname"`
	}
	userMap := make(map[int]UserInfo)
	rechargeMap := make(map[int]float64) // 用户充值量映射
	if len(nodes) > 0 {
		var userIds []int
		for _, node := range nodes {
			userIds = append(userIds, node.UserID)
			if node.ParentID > 0 {
				userIds = append(userIds, node.ParentID)
			}
			if node.LeftChildID > 0 {
				userIds = append(userIds, node.LeftChildID)
			}
			if node.RightChildID > 0 {
				userIds = append(userIds, node.RightChildID)
			}
		}
		var users []model.User
		h.DB.Select("id, username, nickname").Where("id IN ?", userIds).Find(&users)
		for _, user := range users {
			userMap[int(user.Id)] = UserInfo{
				ID:       int(user.Id),
				Username: user.Username,
				Nickname: user.Nickname,
			}
		}

		// 统计每个用户的总充值量（已支付订单的总金额）
		type RechargeResult struct {
			UserId int
			Total  float64
		}
		var rechargeResults []RechargeResult
		h.DB.Table("chatgpt_orders").
			Select("user_id, COALESCE(SUM(sales), 0) as total").
			//Where("user_id IN ? AND status = ?", userIds, 8).
			Where("user_id IN ? ", userIds).
			Group("user_id").
			Scan(&rechargeResults)

		for _, r := range rechargeResults {
			rechargeMap[r.UserId] = r.Total
		}
	}

	type NodeWithUser struct {
		PlacementNode
		UserInfo       UserInfo `json:"user_info"`
		RechargeAmount float64  `json:"recharge_amount"` // 充值量
	}

	result := make([]NodeWithUser, 0)
	for _, node := range nodes {
		userInfo := userMap[node.UserID]
		rechargeAmount := rechargeMap[node.UserID]
		result = append(result, NodeWithUser{
			PlacementNode:  node,
			UserInfo:       userInfo,
			RechargeAmount: rechargeAmount,
		})
	}

	resp.SUCCESS(c, result)
}
