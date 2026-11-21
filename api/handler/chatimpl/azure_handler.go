package chatimpl

import (
	"bufio"
	"chatplus/core/types"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

// 微软 Azure 模型消息发送实现

func (h *ChatHandler) sendAzureMessage(
	chatCtx []types.Message,
	req types.ApiRequest,
	userVo vo.User,
	ctx context.Context,
	session *types.ChatSession,
	role model.ChatRole,
	prompt string,
	ws *types.WsClient) error {
	promptCreatedAt := time.Now() // 记录提问时间
	start := time.Now()
	var apiKey = model.ApiKey{}
	response, err := h.doRequest(ctx, req, session.Model.Platform, &apiKey)
	logger.Info("HTTP请求完成，耗时：", time.Now().Sub(start))
	if err != nil {
		if strings.Contains(err.Error(), "context canceled") {
			logger.Info("用户取消了请求：", prompt)
			return nil
		} else if strings.Contains(err.Error(), "no available key") {
			utils.ReplyMessage(ws, "抱歉😔😔😔，系统已经没有可用的 API KEY，请联系管理员！")
			return nil
		} else {
			logger.Error(err)
		}

		utils.ReplyMessage(ws, ErrorMsg)
		utils.ReplyMessage(ws, ErrImg)
		return err
	} else {
		defer response.Body.Close()
	}

	contentType := response.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		replyCreatedAt := time.Now() // 记录回复时间
		// 循环读取 Chunk 消息
		var message = types.Message{}
		var contents = make([]string, 0)
		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.Contains(line, "data:") || len(line) < 30 {
				continue
			}

			var responseBody = types.ApiResponse{}
			err = json.Unmarshal([]byte(line[6:]), &responseBody)
			if err != nil { // 数据解析出错
				logger.Error(err, line)
				utils.ReplyMessage(ws, ErrorMsg)
				utils.ReplyMessage(ws, ErrImg)
				break
			}

			if len(responseBody.Choices) == 0 {
				continue
			}

			// 初始化 role
			if responseBody.Choices[0].Delta.Role != "" && message.Role == "" {
				message.Role = responseBody.Choices[0].Delta.Role
				utils.ReplyChunkMessage(ws, types.WsMessage{Type: types.WsStart})
				continue
			} else if responseBody.Choices[0].FinishReason != "" {
				break // 输出完成或者输出中断了
			} else {
				content := responseBody.Choices[0].Delta.Content
				contents = append(contents, utils.InterfaceToString(content))
				utils.ReplyChunkMessage(ws, types.WsMessage{
					Type:    types.WsMiddle,
					Content: utils.InterfaceToString(responseBody.Choices[0].Delta.Content),
				})
			}
		} // end for

		if err := scanner.Err(); err != nil {
			if strings.Contains(err.Error(), "context canceled") {
				logger.Info("用户取消了请求：", prompt)
			} else {
				logger.Error("信息读取出错：", err)
			}
		}

		// 消息发送成功
		if len(contents) > 0 {

			if message.Role == "" {
				message.Role = "assistant"
			}
			message.Content = strings.Join(contents, "")
			useMsg := types.Message{Role: "user", Content: prompt}

			// 更新上下文消息，如果是调用函数则不需要更新上下文
			if h.App.SysConfig.EnableContext {
				chatCtx = append(chatCtx, useMsg)  // 提问消息
				chatCtx = append(chatCtx, message) // 回复消息
				h.App.ChatContexts.Put(session.ChatId, chatCtx)
			}

			// 追加聊天记录
			// for prompt
			promptToken, err := utils.CalcTokens(prompt, req.Model)
			if err != nil {
				logger.Error(err)
			}
			historyUserMsg := model.ChatMessage{
				UserId:     userVo.Id,
				ChatId:     session.ChatId,
				RoleId:     role.Id,
				Type:       types.PromptMsg,
				Icon:       userVo.Avatar,
				Content:    template.HTMLEscapeString(prompt),
				Tokens:     float64(promptToken),
				UseContext: true,
				Model:      req.Model,
			}
			historyUserMsg.CreatedAt = promptCreatedAt
			historyUserMsg.UpdatedAt = promptCreatedAt
			res := h.DB.Save(&historyUserMsg)
			if res.Error != nil {
				logger.Error("failed to save prompt history message: ", res.Error)
			}

			// 计算本次对话消耗的总 token 数量
			replyTokens, _ := utils.CalcTokens(message.Content, req.Model)
			replyTokens += getTotalTokens(req)

			historyReplyMsg := model.ChatMessage{
				UserId:     userVo.Id,
				ChatId:     session.ChatId,
				RoleId:     role.Id,
				Type:       types.ReplyMsg,
				Icon:       role.Icon,
				Content:    message.Content,
				Tokens:     float64(replyTokens),
				UseContext: true,
				Model:      req.Model,
			}
			historyReplyMsg.CreatedAt = replyCreatedAt
			historyReplyMsg.UpdatedAt = replyCreatedAt
			res = h.DB.Create(&historyReplyMsg)
			if res.Error != nil {
				logger.Error("failed to save reply history message: ", res.Error)
			}

			// 更新用户算力
			h.subUserPower(userVo, session, promptToken, replyTokens)

			// 保存当前会话
			var chatItem model.ChatItem
			res = h.DB.Where("chat_id = ?", session.ChatId).First(&chatItem)
			if res.Error != nil {
				chatItem.ChatId = session.ChatId
				chatItem.UserId = session.UserId
				chatItem.RoleId = role.Id
				chatItem.ModelId = session.Model.Id
				if utf8.RuneCountInString(prompt) > 30 {
					chatItem.Title = string([]rune(prompt)[:30]) + "..."
				} else {
					chatItem.Title = prompt
				}
				chatItem.Model = req.Model
				h.DB.Create(&chatItem)
			}
		}
	} else {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("error with reading response: %v", err)
		}
		var res types.ApiError
		err = json.Unmarshal(body, &res)
		if err != nil {
			return fmt.Errorf("error with decode response: %v", err)
		}

		if strings.Contains(res.Error.Message, "maximum context length") {
			logger.Error(res.Error.Message)
			utils.ReplyMessage(ws, "当前会话上下文长度超出限制，已为您清空会话上下文！")
			h.App.ChatContexts.Delete(session.ChatId)
			return h.sendMessage(ctx, session, role, prompt, false, ws)
		} else {
			utils.ReplyMessage(ws, "请求 Azure API 失败："+res.Error.Message)
		}
	}

	return nil
}
