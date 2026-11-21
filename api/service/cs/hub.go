package cs

import (
	"chatplus/core/types"
	"chatplus/store/model"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

const (
	RoleUser   = "user"
	RoleAdmin  = "admin"
	RoleSystem = "system"

	MsgTypeText  = "text"
	MsgTypeImage = "image"

	StatusOpen   = 0
	StatusClosed = 1

	OfflineTip = "客服暂时不在线，您可以先留言，上线后会尽快回复。"

	// AgentAdminIDBase 前台客服用 BindAdmin 时的 ID 偏移，避免与后台 AdminUser.id 冲突
	AgentAdminIDBase = uint(1_000_000_000)
)

// AgentBindID 前台客服在 adminClients 中的绑定键
func AgentBindID(userID uint) uint {
	return AgentAdminIDBase + userID
}

// NormalizeMsgType 校验消息类型
func NormalizeMsgType(msgType string) string {
	msgType = strings.TrimSpace(strings.ToLower(msgType))
	if msgType == MsgTypeImage {
		return MsgTypeImage
	}
	return MsgTypeText
}

// ValidMessageContent 校验消息内容
func ValidMessageContent(content, msgType string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return "消息不能为空"
	}
	if msgType == MsgTypeImage {
		if !isValidImageURL(content) {
			return "图片地址无效"
		}
		if len(content) > 1024 {
			return "图片地址过长"
		}
		return ""
	}
	if len([]rune(content)) > 2000 {
		return "消息过长"
	}
	return ""
}

func isValidImageURL(content string) bool {
	if strings.HasPrefix(content, "http://") || strings.HasPrefix(content, "https://") {
		return true
	}
	// 本地上传接口返回站内相对路径，如 /static/upload/2026/8/xxx.png
	if strings.HasPrefix(content, "/static/upload/") && !strings.Contains(content, "..") {
		return true
	}
	return false
}

// Packet WebSocket 通信包
type Packet struct {
	Type        string      `json:"type"`
	Content     string      `json:"content,omitempty"`
	MsgType     string      `json:"msg_type,omitempty"`
	Message     *MsgVO      `json:"message,omitempty"`
	Messages    []MsgVO     `json:"messages,omitempty"`
	Session     *SessionVO  `json:"session,omitempty"`
	Sessions    []SessionVO `json:"sessions,omitempty"`
	AdminOnline bool        `json:"admin_online"`
	UserOnline  bool        `json:"user_online"`
	UserId      uint        `json:"user_id,omitempty"`
	SessionId   uint        `json:"session_id,omitempty"`
	Error       string      `json:"error,omitempty"`
}

type MsgVO struct {
	Id        uint   `json:"id"`
	SessionId uint   `json:"session_id"`
	UserId    uint   `json:"user_id"`
	AdminId   uint   `json:"admin_id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	MsgType   string `json:"msg_type"`
	CreatedAt int64  `json:"created_at"`
}

type SessionVO struct {
	Id          uint   `json:"id"`
	UserId      uint   `json:"user_id"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname"`
	Avatar      string `json:"avatar"`
	Status      int    `json:"status"`
	AdminId     uint   `json:"admin_id"`
	LastMsg     string `json:"last_msg"`
	LastMsgAt   int64  `json:"last_msg_at"`
	UserUnread  int    `json:"user_unread"`
	AdminUnread int    `json:"admin_unread"`
	UserOnline  bool   `json:"user_online"`
	CreatedAt   int64  `json:"created_at"`
}

type Hub struct {
	db          *gorm.DB
	mu          sync.RWMutex
	userClients map[uint]*types.WsClient // userId -> client
	adminClients map[uint]*types.WsClient // adminId -> client
}

func NewHub(db *gorm.DB) *Hub {
	return &Hub{
		db:           db,
		userClients:  make(map[uint]*types.WsClient),
		adminClients: make(map[uint]*types.WsClient),
	}
}

func (h *Hub) IsAdminOnline() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.adminClients {
		if c != nil && !c.Closed {
			return true
		}
	}
	return false
}

func (h *Hub) BindUser(userId uint, client *types.WsClient) {
	h.mu.Lock()
	if old, ok := h.userClients[userId]; ok && old != nil && old != client {
		old.Close()
	}
	h.userClients[userId] = client
	h.mu.Unlock()
	h.broadcastUserPresence(userId, true)
}

func (h *Hub) UnbindUser(userId uint, client *types.WsClient) {
	h.mu.Lock()
	if cur, ok := h.userClients[userId]; ok && cur == client {
		delete(h.userClients, userId)
	}
	h.mu.Unlock()
	h.broadcastUserPresence(userId, false)
}

func (h *Hub) IsUserOnline(userId uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c := h.userClients[userId]
	return c != nil && !c.Closed
}

func (h *Hub) broadcastUserPresence(userId uint, online bool) {
	h.BroadcastAdmins(Packet{
		Type:       "user_presence",
		UserId:     userId,
		UserOnline: online,
	})
}

func (h *Hub) BindAdmin(adminId uint, client *types.WsClient) {
	wasOnline := h.IsAdminOnline()
	h.mu.Lock()
	if old, ok := h.adminClients[adminId]; ok && old != nil && old != client {
		old.Close()
	}
	h.adminClients[adminId] = client
	h.mu.Unlock()
	if !wasOnline {
		h.broadcastAdminStatus(true)
	}
}

func (h *Hub) UnbindAdmin(adminId uint, client *types.WsClient) {
	h.mu.Lock()
	if cur, ok := h.adminClients[adminId]; ok && cur == client {
		delete(h.adminClients, adminId)
	}
	stillOnline := false
	for _, c := range h.adminClients {
		if c != nil && !c.Closed {
			stillOnline = true
			break
		}
	}
	h.mu.Unlock()
	if !stillOnline {
		h.broadcastAdminStatus(false)
	}
}

func (h *Hub) broadcastAdminStatus(online bool) {
	pkt := Packet{Type: "status", AdminOnline: online}
	if !online {
		pkt.Content = OfflineTip
	}
	h.mu.RLock()
	clients := make([]*types.WsClient, 0, len(h.userClients))
	for _, c := range h.userClients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()
	for _, c := range clients {
		_ = c.SendJson(pkt)
	}
}

func (h *Hub) SendToUser(userId uint, pkt Packet) {
	h.mu.RLock()
	c := h.userClients[userId]
	h.mu.RUnlock()
	if c != nil && !c.Closed {
		_ = c.SendJson(pkt)
	}
}

func (h *Hub) BroadcastAdmins(pkt Packet) {
	h.mu.RLock()
	clients := make([]*types.WsClient, 0, len(h.adminClients))
	for _, c := range h.adminClients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()
	for _, c := range clients {
		_ = c.SendJson(pkt)
	}
}

func ToMsgVO(m model.CsMessage) MsgVO {
	return MsgVO{
		Id:        m.Id,
		SessionId: m.SessionId,
		UserId:    m.UserId,
		AdminId:   m.AdminId,
		Role:      m.Role,
		Content:   m.Content,
		MsgType:   m.MsgType,
		CreatedAt: m.CreatedAt.Unix(),
	}
}

func ToSessionVO(s model.CsSession) SessionVO {
	return SessionVO{
		Id:          s.Id,
		UserId:      s.UserId,
		Username:    s.Username,
		Nickname:    s.Nickname,
		Avatar:      s.Avatar,
		Status:      s.Status,
		AdminId:     s.AdminId,
		LastMsg:     s.LastMsg,
		LastMsgAt:   s.LastMsgAt,
		UserUnread:  s.UserUnread,
		AdminUnread: s.AdminUnread,
		CreatedAt:   s.CreatedAt.Unix(),
	}
}

func (h *Hub) ToSessionVOWithOnline(s model.CsSession) SessionVO {
	vo := ToSessionVO(s)
	vo.UserOnline = h.IsUserOnline(s.UserId)
	return vo
}

func Truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

// GetOrCreateOpenSession 获取或创建用户进行中的会话
func (h *Hub) GetOrCreateOpenSession(user model.User) (model.CsSession, error) {
	var session model.CsSession
	err := h.db.Where("user_id = ? AND status = ?", user.Id, StatusOpen).
		Order("id DESC").First(&session).Error
	if err == nil {
		return session, nil
	}
	if err != gorm.ErrRecordNotFound {
		return session, err
	}
	nickname := user.Nickname
	if nickname == "" {
		nickname = user.Username
	}
	session = model.CsSession{
		UserId:   user.Id,
		Username: user.Username,
		Nickname: nickname,
		Avatar:   user.Avatar,
		Status:   StatusOpen,
	}
	if err := h.db.Create(&session).Error; err != nil {
		return session, err
	}
	return session, nil
}

func (h *Hub) ListMessages(sessionId uint, limit int) ([]model.CsMessage, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var list []model.CsMessage
	err := h.db.Where("session_id = ?", sessionId).Order("id ASC").Limit(limit).Find(&list).Error
	return list, err
}

func (h *Hub) SaveMessage(sessionId, userId, adminId uint, role, content, msgType string) (model.CsMessage, error) {
	if msgType == "" {
		msgType = "text"
	}
	msg := model.CsMessage{
		SessionId: sessionId,
		UserId:    userId,
		AdminId:   adminId,
		Role:      role,
		Content:   content,
		MsgType:   msgType,
	}
	if err := h.db.Create(&msg).Error; err != nil {
		return msg, err
	}
	now := time.Now().Unix()
	preview := content
	if msgType == "image" {
		preview = "[图片]"
	}
	updates := map[string]interface{}{
		"last_msg":    Truncate(preview, 200),
		"last_msg_at": now,
	}
	if role == RoleUser {
		updates["admin_unread"] = gorm.Expr("admin_unread + 1")
		updates["user_unread"] = 0
	} else if role == RoleAdmin {
		updates["user_unread"] = gorm.Expr("user_unread + 1")
		updates["admin_unread"] = 0
		updates["admin_id"] = adminId
	}
	_ = h.db.Model(&model.CsSession{}).Where("id = ?", sessionId).Updates(updates)
	return msg, nil
}

func (h *Hub) ClearUserUnread(sessionId uint) {
	_ = h.db.Model(&model.CsSession{}).Where("id = ?", sessionId).Update("user_unread", 0)
}

func (h *Hub) ClearAdminUnread(sessionId uint) {
	_ = h.db.Model(&model.CsSession{}).Where("id = ?", sessionId).Update("admin_unread", 0)
}

func ParsePacket(data []byte) (Packet, error) {
	var p Packet
	err := json.Unmarshal(data, &p)
	return p, err
}

func (h *Hub) ListSessions(status int, page, pageSize int) ([]model.CsSession, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	session := h.db.Model(&model.CsSession{})
	if status >= 0 {
		session = session.Where("status = ?", status)
	}
	var total int64
	session.Count(&total)
	var list []model.CsSession
	err := session.Order("last_msg_at DESC, id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}
