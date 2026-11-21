package chatimpl

import (
	"chatplus/core/types"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

type xunFeiResp struct {
	Header struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Sid     string `json:"sid"`
		Status  int    `json:"status"`
	} `json:"header"`
	Payload struct {
		Choices struct {
			Status int `json:"status"`
			Seq    int `json:"seq"`
			Text   []struct {
				Content string `json:"content"`
				Role    string `json:"role"`
				Index   int    `json:"index"`
			} `json:"text"`
		} `json:"choices"`
		Usage struct {
			Text struct {
				QuestionTokens   int `json:"question_tokens"`
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"text"`
		} `json:"usage"`
	} `json:"payload"`
}

var Model2URL = map[string]string{
	"general":     "v1.1",
	"generalv2":   "v2.1",
	"generalv3":   "v3.1",
	"generalv3.5": "v3.5",
}

// 科大讯飞消息发送实现

func (h *ChatHandler) sendXunFeiMessage(
	chatCtx []types.Message,
	req types.ApiRequest,
	userVo vo.User,
	ctx context.Context,
	session *types.ChatSession,
	role model.ChatRole,
	prompt string,
	ws *types.WsClient) error {
	promptCreatedAt := time.Now() // 记录提问时间
	var apiKey model.ApiKey
	res := h.DB.Where("platform = ?", session.Model.Platform).Where("type = ?", "chat").Where("enabled = ?", true).Order("last_used_at ASC").First(&apiKey)
	if res.Error != nil {
		utils.ReplyMessage(ws, "抱歉😔😔😔，系统已经没有可用的 API KEY，请联系管理员！")
		return nil
	}
	// 更新 API KEY 的最后使用时间
	h.DB.Model(&apiKey).UpdateColumn("last_used_at", time.Now().Unix())

	d := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	key := strings.Split(apiKey.Value, "|")
	if len(key) != 3 {
		utils.ReplyMessage(ws, "非法的 API KEY！")
		return nil
	}

	apiURL := strings.Replace(apiKey.ApiURL, "{version}", Model2URL[req.Model], 1)
	logger.Debugf("Sending %s request, ApiURL:%s, API KEY:%s, PROXY: %s, Model: %s", session.Model.Platform, apiURL, apiKey.Value, apiKey.ProxyURL, req.Model)
	wsURL, err := assembleAuthUrl(apiURL, key[1], key[2])
	//握手并建立websocket 连接
	conn, resp, err := d.Dial(wsURL, nil)
	if err != nil {
		logger.Error(readResp(resp) + err.Error())
		utils.ReplyMessage(ws, "请求讯飞星火模型 API 失败："+readResp(resp)+err.Error())
		return nil
	} else if resp.StatusCode != 101 {
		utils.ReplyMessage(ws, "请求讯飞星火模型 API 失败："+readResp(resp)+err.Error())
		return nil
	}

	data := buildRequest(key[0], req)
	fmt.Printf("%+v", data)
	fmt.Println(apiURL)
	err = conn.WriteJSON(data)
	if err != nil {
		utils.ReplyMessage(ws, "发送消息失败："+err.Error())
		return nil
	}

	replyCreatedAt := time.Now() // 记录回复时间
	// 循环读取 Chunk 消息
	var message = types.Message{}
	var contents = make([]string, 0)
	var content string
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			logger.Error("error with read message:", err)
			utils.ReplyMessage(ws, fmt.Sprintf("**数据读取失败：%s**", err))
			break
		}

		// 解析数据
		var result xunFeiResp
		err = json.Unmarshal(msg, &result)
		if err != nil {
			logger.Error("error with parsing JSON:", err)
			utils.ReplyMessage(ws, fmt.Sprintf("**解析数据行失败：%s**", err))
			return nil
		}

		if result.Header.Code != 0 {
			utils.ReplyMessage(ws, fmt.Sprintf("**请求 API 返回错误：%s**", result.Header.Message))
			return nil
		}

		content = result.Payload.Choices.Text[0].Content
		// 处理代码换行
		if len(content) == 0 {
			content = "\n"
		}
		contents = append(contents, content)
		// 第一个结果
		if result.Payload.Choices.Status == 0 {
			utils.ReplyChunkMessage(ws, types.WsMessage{Type: types.WsStart})
		}
		utils.ReplyChunkMessage(ws, types.WsMessage{
			Type:    types.WsMiddle,
			Content: utils.InterfaceToString(content),
		})

		if result.Payload.Choices.Status == 2 { // 最终结果
			_ = conn.Close() // 关闭连接
			break
		}

		select {
		case <-ctx.Done():
			utils.ReplyMessage(ws, "**用户取消了生成指令！**")
			return nil
		default:
			continue
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

		// for reply
		// 计算本次对话消耗的总 token 数量
		replyTokens, _ := utils.CalcTokens(message.Content, req.Model)
		totalTokens := replyTokens + getTotalTokens(req)
		historyReplyMsg := model.ChatMessage{
			UserId:     userVo.Id,
			ChatId:     session.ChatId,
			RoleId:     role.Id,
			Type:       types.ReplyMsg,
			Icon:       role.Icon,
			Content:    message.Content,
			Tokens:     float64(totalTokens),
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

	return nil
}

// 构建 websocket 请求实体
func buildRequest(appid string, req types.ApiRequest) map[string]interface{} {
	return map[string]interface{}{
		"header": map[string]interface{}{
			"app_id": appid,
		},
		"parameter": map[string]interface{}{
			"chat": map[string]interface{}{
				"domain":      req.Model,
				"temperature": req.Temperature,
				"top_k":       int64(6),
				"max_tokens":  int64(req.MaxTokens),
				"auditing":    "default",
			},
		},
		"payload": map[string]interface{}{
			"message": map[string]interface{}{
				"text": req.Messages,
			},
		},
	}
}

// 创建鉴权 URL
func assembleAuthUrl(hostURL string, apiKey, apiSecret string) (string, error) {
	ul, err := url.Parse(hostURL)
	if err != nil {
		return "", err
	}

	date := time.Now().UTC().Format(time.RFC1123)
	signString := []string{"host: " + ul.Host, "date: " + date, "GET " + ul.Path + " HTTP/1.1"}
	//拼接签名字符串
	signStr := strings.Join(signString, "\n")
	sha := hmacWithSha256(signStr, apiSecret)

	authUrl := fmt.Sprintf("hmac username=\"%s\", algorithm=\"%s\", headers=\"%s\", signature=\"%s\"", apiKey,
		"hmac-sha256", "host date request-line", sha)
	//将请求参数使用base64编码
	authorization := base64.StdEncoding.EncodeToString([]byte(authUrl))
	v := url.Values{}
	v.Add("host", ul.Host)
	v.Add("date", date)
	v.Add("authorization", authorization)
	//将编码后的字符串url encode后添加到url后面
	return hostURL + "?" + v.Encode(), nil
}

// 使用 sha256 签名
func hmacWithSha256(data, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	encodeData := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(encodeData)
}

// 读取响应
func readResp(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("code=%d,body=%s", resp.StatusCode, string(b))
}
