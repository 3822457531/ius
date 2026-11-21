package chatimpl

import (
	"chatplus/core/types"
	"chatplus/store/model"
	"chatplus/utils"
)

const ephemeralFocusSystem = "请只回答用户最后一条消息中的问题，不要重复回应或附带总结历史里已经讨论过、且与当前问题无关的话题（除非用户明确要求回顾）。"
const ephemeralWebSearchSystem = "用户已开启联网搜索：涉及实时信息、新闻、天气、股价等，请先调用 web_search 工具获取结果后再组织回答；不要声称无法联网。"

// buildChatContext 组装可持久化的多轮上下文（角色设定在前，历史在后）
func (h *ChatHandler) buildChatContext(session *types.ChatSession, role model.ChatRole) []types.Message {
	if !h.App.SysConfig.EnableContext {
		return nil
	}

	if h.App.ChatContexts.Has(session.ChatId) {
		return h.App.ChatContexts.Get(session.ChatId)
	}

	var rolePreset []types.Message
	_ = utils.JsonDecode(role.Context, &rolePreset)
	chatCtx := make([]types.Message, 0, len(rolePreset)+8)
	chatCtx = append(chatCtx, rolePreset...)

	if h.App.SysConfig.ContextDeep > 0 {
		var historyMessages []model.ChatMessage
		res := h.DB.Where("chat_id = ? and use_context = 1", session.ChatId).
			Limit(h.App.SysConfig.ContextDeep).Order("id DESC").Find(&historyMessages)
		if res.Error == nil {
			for i := len(historyMessages) - 1; i >= 0; i-- {
				msg := historyMessages[i]
				ms := types.Message{Role: "user", Content: msg.Content}
				if msg.Type == types.ReplyMsg {
					ms.Role = "assistant"
				}
				chatCtx = append(chatCtx, ms)
			}
		}
	}
	return chatCtx
}

func (h *ChatHandler) trimChatContext(chatCtx []types.Message, req types.ApiRequest, promptTokens int, model string, maxContext int) []types.Message {
	if len(chatCtx) == 0 {
		return chatCtx
	}
	tokens := req.MaxTokens
	tks, _ := utils.CalcTokens(utils.JsonEncode(req.Tools), model)
	tokens += tks + promptTokens

	out := make([]types.Message, 0, len(chatCtx))
	for _, v := range chatCtx {
		tks, _ := utils.CalcTokens(v.Content, model)
		if maxContext > 0 && tokens+tks >= maxContext {
			break
		}
		if h.App.SysConfig.ContextDeep > 0 && len(out) >= h.App.SysConfig.ContextDeep {
			break
		}
		tokens += tks
		out = append(out, v)
	}
	return out
}

func prependEphemeralSystem(msgs []interface{}, webSearch bool) []interface{} {
	content := ephemeralFocusSystem
	if webSearch {
		content = content + " " + ephemeralWebSearchSystem
	}
	out := make([]interface{}, 0, len(msgs)+1)
	out = append(out, types.Message{Role: "system", Content: content})
	out = append(out, msgs...)
	return out
}
