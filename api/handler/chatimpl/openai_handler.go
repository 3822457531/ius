package chatimpl

import (
	"bufio"
	"chatplus/core/types"
	"chatplus/service/websearch"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	req2 "github.com/imroc/req/v3"
)

const openAIEmptyReplyMaxRetry = 2

type openAIStreamResult struct {
	contents  []string
	message   types.Message
	toolCall  bool
	function  model.Function
	arguments []string
}

// consumeOpenAIStream 读取 SSE 流；空 choices 分片跳过，不向用户报错
func (h *ChatHandler) consumeOpenAIStream(body io.Reader, ws *types.WsClient, userVo vo.User, userPrompt string) openAIStreamResult {
	var res openAIStreamResult
	res.arguments = make([]string, 0)
	res.contents = make([]string, 0)

	scanner := bufio.NewScanner(body)
	var isNew = true
	for scanner.Scan() {
		line := scanner.Text()
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		if strings.Contains(line, "[DONE]") {
			break
		}
		if !strings.Contains(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}

		var responseBody types.ApiResponse
		if err := json.Unmarshal([]byte(payload), &responseBody); err != nil {
			logger.Warnf("OpenAI SSE 解析跳过: %v, data=%s", err, payload)
			continue
		}
		if len(responseBody.Choices) == 0 {
			continue
		}

		var tool types.ToolCall
		if len(responseBody.Choices[0].Delta.ToolCalls) > 0 {
			tool = responseBody.Choices[0].Delta.ToolCalls[0]
			if res.toolCall && tool.Function.Name == "" {
				res.arguments = append(res.arguments, tool.Function.Arguments)
				continue
			}
		}

		fun := responseBody.Choices[0].Delta.FunctionCall
		if fun.Name != "" {
			tool = *new(types.ToolCall)
			tool.Function.Name = fun.Name
		} else if res.toolCall {
			res.arguments = append(res.arguments, fun.Arguments)
			continue
		}

		if !utils.IsEmptyValue(tool) {
			if a := strings.TrimSpace(tool.Function.Arguments); a != "" {
				res.arguments = append(res.arguments, a)
			}
			if tool.Function.Name == builtinWebSearchTool {
				res.toolCall = true
				res.function = model.Function{Name: builtinWebSearchTool, Label: "联网搜索"}
				utils.ReplyChunkMessage(ws, types.WsMessage{Type: types.WsStart})
				utils.ReplyChunkMessage(ws, types.WsMessage{Type: types.WsMiddle, Content: "正在联网搜索 ...\n\n"})
			} else {
				r := h.DB.Where("name = ?", tool.Function.Name).First(&res.function)
				if r.Error == nil {
					res.toolCall = true
					utils.ReplyChunkMessage(ws, types.WsMessage{Type: types.WsStart})
					utils.ReplyChunkMessage(ws, types.WsMessage{Type: types.WsMiddle, Content: fmt.Sprintf("正在调用工具 `%s` 作答 ...\n\n", res.function.Label)})
				}
			}
			continue
		}

		if responseBody.Choices[0].FinishReason == "tool_calls" ||
			responseBody.Choices[0].FinishReason == "function_call" {
			break
		}

		if responseBody.Choices[0].Delta.Role != "" && res.message.Role == "" {
			res.message.Role = responseBody.Choices[0].Delta.Role
			continue
		}
		if responseBody.Choices[0].FinishReason != "" {
			break
		}

		contentStr := utils.InterfaceToString(responseBody.Choices[0].Delta.Content)
		res.contents = append(res.contents, contentStr)
		if contentStr != "null" && contentStr != "" && strings.TrimSpace(contentStr) != "" {
			if isNew {
				utils.ReplyChunkMessage(ws, types.WsMessage{Type: types.WsStart})
				isNew = false
			}
			utils.ReplyChunkMessage(ws, types.WsMessage{
				Type:    types.WsMiddle,
				Content: contentStr,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		if strings.Contains(err.Error(), "context canceled") {
			logger.Info("用户取消了流式读取")
		} else {
			logger.Error("信息读取出错：", err)
		}
	}

	if res.toolCall {
		var params map[string]interface{}
		_ = utils.JsonDecode(strings.Join(res.arguments, ""), &params)
		logger.Debugf("函数名称: %s, 函数参数：%s", res.function.Name, params)
		if res.function.Name == builtinWebSearchTool {
			toolQ := websearch.QueryFromParams(params)
			text, err := h.runWebSearch(toolQ, userPrompt)
			if err != nil {
				msg := "联网搜索失败：" + err.Error()
				utils.ReplyChunkMessage(ws, types.WsMessage{Type: types.WsMiddle, Content: msg})
				res.contents = append(res.contents, msg)
			} else {
				utils.ReplyChunkMessage(ws, types.WsMessage{Type: types.WsMiddle, Content: text})
				res.contents = append(res.contents, text)
			}
		} else {
			params["user_id"] = userVo.Id
			var apiRes types.BizVo
			r, err := req2.C().R().SetHeader("Content-Type", "application/json").
				SetHeader("Authorization", res.function.Token).
				SetBody(params).
				SetSuccessResult(&apiRes).Post(res.function.Action)
			errMsg := ""
			if err != nil {
				errMsg = err.Error()
			} else if r.IsErrorState() {
				errMsg = r.Status
			}
			if errMsg != "" || apiRes.Code != types.Success {
				msg := "调用函数工具出错：" + apiRes.Message + errMsg
				utils.ReplyChunkMessage(ws, types.WsMessage{Type: types.WsMiddle, Content: msg})
				res.contents = append(res.contents, msg)
			} else {
				utils.ReplyChunkMessage(ws, types.WsMessage{Type: types.WsMiddle, Content: apiRes.Data})
				res.contents = append(res.contents, utils.InterfaceToString(apiRes.Data))
			}
		}
	}

	return res
}

func cleanOpenAIContents(contents []string) []string {
	var out []string
	for _, v := range contents {
		if v != "null" && v != "" && strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}

func (h *ChatHandler) openAIStreamHasReply(res openAIStreamResult) bool {
	return len(cleanOpenAIContents(res.contents)) > 0
}

// OPenAI 消息发送实现
func (h *ChatHandler) sendOpenAiMessage(
	chatCtx []types.Message,
	req types.ApiRequest,
	userVo vo.User,
	ctx context.Context,
	session *types.ChatSession,
	role model.ChatRole,
	prompt string,
	ws *types.WsClient) error {
	promptCreatedAt := time.Now() // 记录提问时间
	var apiKey = model.ApiKey{}

	var (
		streamRes     openAIStreamResult
		replyCreatedAt time.Time
		contentType   string
	)

	for attempt := 0; attempt <= openAIEmptyReplyMaxRetry; attempt++ {
		if attempt > 0 {
			logger.Warnf("OpenAI 流式返回为空，自动重试 (%d/%d), model=%s", attempt, openAIEmptyReplyMaxRetry, req.Model)
			time.Sleep(300 * time.Millisecond)
		}

		start := time.Now()
		response, err := h.doRequest(ctx, req, session.Model.Platform, &apiKey)
		logger.Info("HTTP请求完成，耗时：", time.Since(start))
		if err != nil {
			logger.Error(err)
			if strings.Contains(err.Error(), "context canceled") {
				logger.Info("用户取消了请求：", prompt)
				return nil
			}
			if strings.Contains(err.Error(), "no available key") {
				utils.ReplyMessage(ws, "抱歉😔😔😔，系统已经没有可用的 API KEY，请联系管理员！")
				return nil
			}
			utils.ReplyMessage(ws, err.Error())
			return err
		}

		contentType = response.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/event-stream") {
			defer response.Body.Close()
			body, readErr := io.ReadAll(response.Body)
			if readErr != nil {
				utils.ReplyMessage(ws, "请求 OpenAI API 失败："+readErr.Error())
				return fmt.Errorf("error with reading response: %v", readErr)
			}
			var res types.ApiError
			if err := json.Unmarshal(body, &res); err != nil {
				utils.ReplyMessage(ws, "请求 OpenAI API 失败：\n"+"```\n"+string(body)+"```")
				return fmt.Errorf("error with decode response: %v", err)
			}
			if strings.Contains(res.Error.Message, "This key is associated with a deactivated account") {
				utils.ReplyMessage(ws, "请求 OpenAI API 失败：API KEY 所关联的账户被禁用。")
				h.DB.Where("value = ?", apiKey).Delete(&model.ApiKey{})
			} else if strings.Contains(res.Error.Message, "You exceeded your current quota") {
				utils.ReplyMessage(ws, "请求 OpenAI API 失败：API KEY 触发并发限制，请稍后再试。")
			} else if strings.Contains(res.Error.Message, "This model's maximum context length") {
				logger.Error(res.Error.Message)
				utils.ReplyMessage(ws, "当前会话上下文长度超出限制，已为您清空会话上下文！")
				h.App.ChatContexts.Delete(session.ChatId)
				return h.sendMessage(ctx, session, role, prompt, false, ws)
			} else {
				utils.ReplyMessage(ws, "请求 OpenAI API 失败："+res.Error.Message)
			}
			return nil
		}

		replyCreatedAt = time.Now()
		streamRes = h.consumeOpenAIStream(response.Body, ws, userVo, prompt)
		_ = response.Body.Close()

		logger.Info("返回的内容为", streamRes.contents)
		streamRes.contents = cleanOpenAIContents(streamRes.contents)
		logger.Info("处理后的内容为", streamRes.contents)

		if h.openAIStreamHasReply(streamRes) {
			break
		}
		if attempt == openAIEmptyReplyMaxRetry {
			logger.Error("OpenAI 多次返回空内容，model=", req.Model)
			utils.ReplyMessage(ws, ErrorMsg)
			utils.ReplyMessage(ws, ErrImg)
			return nil
		}
	}

	if strings.Contains(contentType, "text/event-stream") {
		contents := streamRes.contents
		message := streamRes.message
		toolCall := streamRes.toolCall
		function := streamRes.function
		arguments := streamRes.arguments

		// 消息发送成功
		if len(contents) > 0 {
			if message.Role == "" {
				message.Role = "assistant"
			}
			message.Content = html.EscapeString(strings.Join(contents, ""))
			useMsg := types.Message{Role: "user", Content: prompt}

			// 更新上下文消息，如果是调用函数则不需要更新上下文
			if h.App.SysConfig.EnableContext && toolCall == false {
				chatCtx = append(chatCtx, useMsg)  // 提问消息
				chatCtx = append(chatCtx, message) // 回复消息
				h.App.ChatContexts.Put(session.ChatId, chatCtx)
			}

			// 追加聊天记录
			useContext := true
			if toolCall {
				useContext = false
			}

			// for prompt
			promptToken, err := utils.CalcTokens(prompt, req.Model)
			if err != nil {
				logger.Error(err)
			}

			escapedPrompt := html.EscapeString(prompt)

			historyUserMsg := model.ChatMessage{
				UserId:     userVo.Id,
				ChatId:     session.ChatId,
				RoleId:     role.Id,
				Type:       types.PromptMsg,
				Icon:       userVo.Avatar,
				Content:    escapedPrompt,
				Tokens:     float64(promptToken),
				UseContext: useContext,
				Model:      req.Model,
			}
			historyUserMsg.CreatedAt = promptCreatedAt
			historyUserMsg.UpdatedAt = promptCreatedAt
			res := h.DB.Save(&historyUserMsg)
			if res.Error != nil {
				logger.Error("failed to save prompt history message: ", res.Error)
			}

			// 计算本次对话消耗的总 token 数量
			var replyTokens = 0
			if toolCall { // prompt + 函数名 + 参数 token
				tokens, _ := utils.CalcTokens(function.Name, req.Model)
				replyTokens += tokens
				tokens, _ = utils.CalcTokens(utils.InterfaceToString(arguments), req.Model)
				replyTokens += tokens
			} else {
				replyTokens, _ = utils.CalcTokens(message.Content, req.Model)
			}
			replyTokens += getTotalTokens(req)

			extractedContent := h.extractImgUrl(message.Content)

			historyReplyMsg := model.ChatMessage{
				UserId:     userVo.Id,
				ChatId:     session.ChatId,
				RoleId:     role.Id,
				Type:       types.ReplyMsg,
				Icon:       role.Icon,
				Content:    extractedContent,
				Tokens:     float64(replyTokens),
				UseContext: useContext,
				Model:      req.Model,
			}
			historyReplyMsg.CreatedAt = replyCreatedAt
			historyReplyMsg.UpdatedAt = replyCreatedAt
			res = h.DB.Create(&historyReplyMsg)
			if res.Error != nil {
				logger.Error("failed to save reply history message: ", res.Error, "content:", message.Content)
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
	}

	return nil
}
