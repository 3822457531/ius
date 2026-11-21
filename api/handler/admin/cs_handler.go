package admin

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/handler"
	"chatplus/service/cs"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils/resp"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

type CsHandler struct {
	handler.BaseHandler
	hub *cs.Hub
}

func NewCsHandler(app *core.AppServer, db *gorm.DB, hub *cs.Hub) *CsHandler {
	return &CsHandler{
		BaseHandler: handler.BaseHandler{App: app, DB: db},
		hub:         hub,
	}
}

// Sessions 会话列表
func (h *CsHandler) Sessions(c *gin.Context) {
	var data struct {
		Status   int `json:"status"`
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	}
	_ = c.ShouldBindJSON(&data)
	if data.Page == 0 {
		data.Page = 1
	}
	if data.PageSize == 0 {
		data.PageSize = 50
	}
	// status 缺省传 -1 表示全部；JSON 默认 0 表示进行中。用 query 兼容
	status := data.Status
	if c.Query("all") == "1" {
		status = -1
	}
	list, total, err := h.hub.ListSessions(status, data.Page, data.PageSize)
	if err != nil {
		resp.ERROR(c, "加载失败")
		return
	}
	vos := make([]cs.SessionVO, 0, len(list))
	for _, s := range list {
		vos = append(vos, h.hub.ToSessionVOWithOnline(s))
	}
	resp.SUCCESS(c, vo.NewPage(total, data.Page, data.PageSize, vos))
}

// Unread 管理端未读会话汇总
func (h *CsHandler) Unread(c *gin.Context) {
	var totalUnread int64
	var sessionCount int64
	h.DB.Model(&model.CsSession{}).
		Where("status = ? AND admin_unread > 0", cs.StatusOpen).
		Count(&sessionCount)
	h.DB.Model(&model.CsSession{}).
		Where("status = ? AND admin_unread > 0", cs.StatusOpen).
		Select("COALESCE(SUM(admin_unread),0)").Scan(&totalUnread)
	resp.SUCCESS(c, gin.H{
		"unread":         totalUnread,
		"session_count":  sessionCount,
	})
}

// Messages 某会话消息
func (h *CsHandler) Messages(c *gin.Context) {
	sessionId := uint(h.GetInt(c, "session_id", 0))
	if sessionId == 0 {
		resp.ERROR(c, "session_id required")
		return
	}
	msgs, err := h.hub.ListMessages(sessionId, 200)
	if err != nil {
		resp.ERROR(c, "加载失败")
		return
	}
	list := make([]cs.MsgVO, 0, len(msgs))
	for _, m := range msgs {
		list = append(list, cs.ToMsgVO(m))
	}
	var session model.CsSession
	_ = h.DB.First(&session, sessionId)
	h.hub.ClearAdminUnread(sessionId)
	sessionVO := h.hub.ToSessionVOWithOnline(session)
	sessionVO.AdminUnread = 0
	resp.SUCCESS(c, gin.H{
		"session":  sessionVO,
		"messages": list,
	})
}

// Reply 管理端 HTTP 回复（用户离线也可留言回复）
func (h *CsHandler) Reply(c *gin.Context) {
	adminId := h.GetLoginUserId(c)
	if adminId == 0 {
		resp.NotAuth(c)
		return
	}
	var data struct {
		SessionId uint   `json:"session_id"`
		Content   string `json:"content"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	content := strings.TrimSpace(data.Content)
	if content == "" || data.SessionId == 0 {
		resp.ERROR(c, "参数错误")
		return
	}
	if len([]rune(content)) > 2000 {
		resp.ERROR(c, "消息过长")
		return
	}
	var session model.CsSession
	if err := h.DB.First(&session, data.SessionId).Error; err != nil {
		resp.ERROR(c, "会话不存在")
		return
	}
	if session.Status == cs.StatusClosed {
		// 离线留言回复时自动重新打开
		_ = h.DB.Model(&session).Update("status", cs.StatusOpen)
		session.Status = cs.StatusOpen
	}
	msg, err := h.hub.SaveMessage(session.Id, session.UserId, adminId, cs.RoleAdmin, content, cs.MsgTypeText)
	if err != nil {
		resp.ERROR(c, "发送失败")
		return
	}
	voMsg := cs.ToMsgVO(msg)
	_ = h.DB.First(&session, session.Id)
	sessionVO := h.hub.ToSessionVOWithOnline(session)
	out := cs.Packet{Type: "message", Message: &voMsg, Session: &sessionVO}
	h.hub.BroadcastAdmins(out)
	h.hub.SendToUser(session.UserId, out)
	resp.SUCCESS(c, gin.H{"message": voMsg, "session": sessionVO})
}

// Close 关闭会话
func (h *CsHandler) Close(c *gin.Context) {
	sessionId := uint(h.GetInt(c, "session_id", 0))
	if sessionId == 0 {
		resp.ERROR(c, "session_id required")
		return
	}
	res := h.DB.Model(&model.CsSession{}).Where("id = ?", sessionId).Update("status", cs.StatusClosed)
	if res.Error != nil {
		resp.ERROR(c, "关闭失败")
		return
	}
	resp.SUCCESS(c)
}

// Client 管理端 WebSocket
func (h *CsHandler) Client(c *gin.Context) {
	adminId := h.GetLoginUserId(c)
	if adminId == 0 {
		resp.NotAuth(c)
		return
	}

	ws, err := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := types.NewWsClient(ws)
	h.hub.BindAdmin(adminId, client)
	defer func() {
		h.hub.UnbindAdmin(adminId, client)
		client.Close()
	}()

	list, _, _ := h.hub.ListSessions(cs.StatusOpen, 1, 100)
	vos := make([]cs.SessionVO, 0, len(list))
	for _, s := range list {
		vos = append(vos, h.hub.ToSessionVOWithOnline(s))
	}
	_ = client.SendJson(cs.Packet{
		Type:        "init",
		Sessions:    vos,
		AdminOnline: true,
	})

	for {
		_, data, err := client.Receive()
		if err != nil {
			break
		}
		pkt, err := cs.ParsePacket(data)
		if err != nil {
			continue
		}
		switch pkt.Type {
		case "ping":
			_ = client.SendJson(cs.Packet{Type: "pong"})
		case "read":
			if pkt.SessionId > 0 {
				h.hub.ClearAdminUnread(pkt.SessionId)
			}
		case "message":
			content := strings.TrimSpace(pkt.Content)
			if content == "" || pkt.SessionId == 0 {
				continue
			}
			if len([]rune(content)) > 2000 {
				_ = client.SendJson(cs.Packet{Type: "error", Error: "消息过长"})
				continue
			}
			var session model.CsSession
			if err := h.DB.First(&session, pkt.SessionId).Error; err != nil {
				_ = client.SendJson(cs.Packet{Type: "error", Error: "会话不存在"})
				continue
			}
			if session.Status == cs.StatusClosed {
				_ = h.DB.Model(&session).Update("status", cs.StatusOpen)
				session.Status = cs.StatusOpen
			}
			msg, err := h.hub.SaveMessage(session.Id, session.UserId, adminId, cs.RoleAdmin, content, cs.MsgTypeText)
			if err != nil {
				_ = client.SendJson(cs.Packet{Type: "error", Error: "发送失败"})
				continue
			}
			voMsg := cs.ToMsgVO(msg)
			_ = h.DB.First(&session, session.Id)
			sessionVO := h.hub.ToSessionVOWithOnline(session)
			out := cs.Packet{Type: "message", Message: &voMsg, Session: &sessionVO}
			h.hub.BroadcastAdmins(out)
			h.hub.SendToUser(session.UserId, out)
		}
	}
}
