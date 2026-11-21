package handler

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/service/cs"
	"chatplus/store/model"
	"chatplus/utils/resp"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

type CsHandler struct {
	BaseHandler
	hub *cs.Hub
}

func NewCsHandler(app *core.AppServer, db *gorm.DB, hub *cs.Hub) *CsHandler {
	return &CsHandler{
		BaseHandler: BaseHandler{App: app, DB: db},
		hub:         hub,
	}
}

// Status 客服在线状态
func (h *CsHandler) Status(c *gin.Context) {
	resp.SUCCESS(c, gin.H{
		"admin_online": h.hub.IsAdminOnline(),
		"tip":          cs.OfflineTip,
	})
}

// Unread 当前用户未读客服消息数（不创建会话）
func (h *CsHandler) Unread(c *gin.Context) {
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}
	var session model.CsSession
	res := h.DB.Where("user_id = ? AND status = ?", user.Id, cs.StatusOpen).
		Order("id DESC").First(&session)
	unread := 0
	if res.Error == nil {
		unread = session.UserUnread
	}
	resp.SUCCESS(c, gin.H{"unread": unread})
}

// History 当前用户会话历史
func (h *CsHandler) History(c *gin.Context) {
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}
	session, err := h.hub.GetOrCreateOpenSession(user)
	if err != nil {
		resp.ERROR(c, "创建会话失败")
		return
	}
	msgs, err := h.hub.ListMessages(session.Id, 100)
	if err != nil {
		resp.ERROR(c, "加载消息失败")
		return
	}
	list := make([]cs.MsgVO, 0, len(msgs))
	for _, m := range msgs {
		list = append(list, cs.ToMsgVO(m))
	}
	h.hub.ClearUserUnread(session.Id)
	resp.SUCCESS(c, gin.H{
		"session":      cs.ToSessionVO(session),
		"messages":     list,
		"admin_online": h.hub.IsAdminOnline(),
		"tip":          cs.OfflineTip,
	})
}

// Send 留言/发消息（HTTP，客服离线也可留言）
func (h *CsHandler) Send(c *gin.Context) {
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}
	var data struct {
		Content string `json:"content"`
		MsgType string `json:"msg_type"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	msgType := cs.NormalizeMsgType(data.MsgType)
	content := strings.TrimSpace(data.Content)
	if errMsg := cs.ValidMessageContent(content, msgType); errMsg != "" {
		resp.ERROR(c, errMsg)
		return
	}
	session, err := h.hub.GetOrCreateOpenSession(user)
	if err != nil {
		resp.ERROR(c, "创建会话失败")
		return
	}
	msg, err := h.hub.SaveMessage(session.Id, user.Id, 0, cs.RoleUser, content, msgType)
	if err != nil {
		resp.ERROR(c, "发送失败")
		return
	}
	vo := cs.ToMsgVO(msg)
	var fresh model.CsSession
	_ = h.DB.First(&fresh, session.Id)
	sessionVO := h.hub.ToSessionVOWithOnline(fresh)
	h.hub.BroadcastAdmins(cs.Packet{
		Type:    "message",
		Message: &vo,
		Session: &sessionVO,
	})
	// 同步推给自己的 WS（若在线）
	h.hub.SendToUser(user.Id, cs.Packet{Type: "message", Message: &vo})
	resp.SUCCESS(c, gin.H{
		"message":      vo,
		"session":      sessionVO,
		"admin_online": h.hub.IsAdminOnline(),
	})
}

// Client 用户端 WebSocket
func (h *CsHandler) Client(c *gin.Context) {
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return
	}

	ws, err := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("cs user ws upgrade:", err)
		return
	}
	client := types.NewWsClient(ws)
	h.hub.BindUser(user.Id, client)
	defer func() {
		h.hub.UnbindUser(user.Id, client)
		client.Close()
	}()

	session, err := h.hub.GetOrCreateOpenSession(user)
	if err != nil {
		_ = client.SendJson(cs.Packet{Type: "error", Error: "创建会话失败"})
		return
	}

	msgs, _ := h.hub.ListMessages(session.Id, 100)
	msgVOs := make([]cs.MsgVO, 0, len(msgs))
	for _, m := range msgs {
		msgVOs = append(msgVOs, cs.ToMsgVO(m))
	}
	online := h.hub.IsAdminOnline()
	_ = client.SendJson(cs.Packet{
		Type:        "init",
		Session:     ptrSession(cs.ToSessionVO(session)),
		Messages:    msgVOs,
		AdminOnline: online,
	})
	if !online {
		_ = client.SendJson(cs.Packet{
			Type:        "status",
			AdminOnline: false,
			Content:     cs.OfflineTip,
		})
	}

	h.hub.ClearUserUnread(session.Id)

	lastOfflineTipAt := time.Time{}
	if !online {
		lastOfflineTipAt = time.Now()
	}

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
			h.hub.ClearUserUnread(session.Id)
		case "message":
			content := strings.TrimSpace(pkt.Content)
			msgType := cs.NormalizeMsgType(pkt.MsgType)
			if errMsg := cs.ValidMessageContent(content, msgType); errMsg != "" {
				_ = client.SendJson(cs.Packet{Type: "error", Error: errMsg})
				continue
			}
			// 会话可能被关闭，重新打开
			if session.Status == cs.StatusClosed {
				session, err = h.hub.GetOrCreateOpenSession(user)
				if err != nil {
					_ = client.SendJson(cs.Packet{Type: "error", Error: "会话不可用"})
					continue
				}
			}
			msg, err := h.hub.SaveMessage(session.Id, user.Id, 0, cs.RoleUser, content, msgType)
			if err != nil {
				_ = client.SendJson(cs.Packet{Type: "error", Error: "发送失败"})
				continue
			}
			vo := cs.ToMsgVO(msg)
			_ = client.SendJson(cs.Packet{Type: "message", Message: &vo})

			var fresh model.CsSession
			_ = h.DB.First(&fresh, session.Id)
			sessionVO := h.hub.ToSessionVOWithOnline(fresh)
			h.hub.BroadcastAdmins(cs.Packet{
				Type:    "message",
				Message: &vo,
				Session: &sessionVO,
			})

			if !h.hub.IsAdminOnline() && time.Since(lastOfflineTipAt) > 30*time.Second {
				lastOfflineTipAt = time.Now()
				_ = client.SendJson(cs.Packet{
					Type:        "status",
					AdminOnline: false,
					Content:     cs.OfflineTip,
				})
			}
		}
	}
}

func ptrSession(s cs.SessionVO) *cs.SessionVO {
	return &s
}
