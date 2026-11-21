package handler

import (
	"chatplus/core/types"
	"chatplus/service/cs"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils/resp"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// requireCsAgent 校验当前登录用户是否为客服
func (h *CsHandler) requireCsAgent(c *gin.Context) (model.User, bool) {
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return model.User{}, false
	}
	if !user.IsCs {
		resp.ERROR(c, "无客服权限")
		return model.User{}, false
	}
	return user, true
}

// AgentSessions 客服会话列表
func (h *CsHandler) AgentSessions(c *gin.Context) {
	user, ok := h.requireCsAgent(c)
	if !ok {
		return
	}
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
		if s.UserId == user.Id {
			continue
		}
		vos = append(vos, h.hub.ToSessionVOWithOnline(s))
	}
	resp.SUCCESS(c, vo.NewPage(total, data.Page, data.PageSize, vos))
}

// AgentUnread 客服未读汇总
func (h *CsHandler) AgentUnread(c *gin.Context) {
	if _, ok := h.requireCsAgent(c); !ok {
		return
	}
	var totalUnread int64
	var sessionCount int64
	h.DB.Model(&model.CsSession{}).
		Where("status = ? AND admin_unread > 0", cs.StatusOpen).
		Count(&sessionCount)
	h.DB.Model(&model.CsSession{}).
		Where("status = ? AND admin_unread > 0", cs.StatusOpen).
		Select("COALESCE(SUM(admin_unread),0)").Scan(&totalUnread)
	resp.SUCCESS(c, gin.H{
		"unread":        totalUnread,
		"session_count": sessionCount,
	})
}

// AgentMessages 客服拉取某会话消息
func (h *CsHandler) AgentMessages(c *gin.Context) {
	if _, ok := h.requireCsAgent(c); !ok {
		return
	}
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

// AgentReply 客服 HTTP 回复
func (h *CsHandler) AgentReply(c *gin.Context) {
	user, ok := h.requireCsAgent(c)
	if !ok {
		return
	}
	var data struct {
		SessionId uint   `json:"session_id"`
		Content   string `json:"content"`
		MsgType   string `json:"msg_type"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	msgType := cs.NormalizeMsgType(data.MsgType)
	content := strings.TrimSpace(data.Content)
	if data.SessionId == 0 {
		resp.ERROR(c, "参数错误")
		return
	}
	if errMsg := cs.ValidMessageContent(content, msgType); errMsg != "" {
		resp.ERROR(c, errMsg)
		return
	}
	var session model.CsSession
	if err := h.DB.First(&session, data.SessionId).Error; err != nil {
		resp.ERROR(c, "会话不存在")
		return
	}
	if session.UserId == user.Id {
		resp.ERROR(c, "不能回复自己的会话")
		return
	}
	if session.Status == cs.StatusClosed {
		_ = h.DB.Model(&session).Update("status", cs.StatusOpen)
		session.Status = cs.StatusOpen
	}
	msg, err := h.hub.SaveMessage(session.Id, session.UserId, user.Id, cs.RoleAdmin, content, msgType)
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

// AgentClose 客服结束会话
func (h *CsHandler) AgentClose(c *gin.Context) {
	if _, ok := h.requireCsAgent(c); !ok {
		return
	}
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

// AgentClient 前台客服 WebSocket（接入后计入客服在线）
func (h *CsHandler) AgentClient(c *gin.Context) {
	user, ok := h.requireCsAgent(c)
	if !ok {
		return
	}

	ws, err := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := types.NewWsClient(ws)
	bindID := cs.AgentBindID(user.Id)
	h.hub.BindAdmin(bindID, client)
	defer func() {
		h.hub.UnbindAdmin(bindID, client)
		client.Close()
	}()

	list, _, _ := h.hub.ListSessions(cs.StatusOpen, 1, 100)
	vos := make([]cs.SessionVO, 0, len(list))
	for _, s := range list {
		if s.UserId == user.Id {
			continue
		}
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
			msgType := cs.NormalizeMsgType(pkt.MsgType)
			if pkt.SessionId == 0 {
				continue
			}
			if errMsg := cs.ValidMessageContent(content, msgType); errMsg != "" {
				_ = client.SendJson(cs.Packet{Type: "error", Error: errMsg})
				continue
			}
			var session model.CsSession
			if err := h.DB.First(&session, pkt.SessionId).Error; err != nil {
				_ = client.SendJson(cs.Packet{Type: "error", Error: "会话不存在"})
				continue
			}
			if session.UserId == user.Id {
				_ = client.SendJson(cs.Packet{Type: "error", Error: "不能回复自己的会话"})
				continue
			}
			if session.Status == cs.StatusClosed {
				_ = h.DB.Model(&session).Update("status", cs.StatusOpen)
				session.Status = cs.StatusOpen
			}
			msg, err := h.hub.SaveMessage(session.Id, session.UserId, user.Id, cs.RoleAdmin, content, msgType)
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
