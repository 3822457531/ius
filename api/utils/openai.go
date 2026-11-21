package utils

import (
	"chatplus/core/types"
	"chatplus/store/model"
	"encoding/json"
	"fmt"
	"io"

	"time"

	"github.com/imroc/req/v3"
	"github.com/pkoukk/tiktoken-go"
	"gorm.io/gorm"
)

func CalcTokens(text string, model string) (int, error) {
	encoding, ok := tiktoken.MODEL_TO_ENCODING[model]
	if !ok {
		encoding = "cl100k_base"
	}
	tke, err := tiktoken.GetEncoding(encoding)
	if err != nil {
		return 0, fmt.Errorf("getEncoding: %v", err)
	}

	token := tke.Encode(text, nil, nil)
	return len(token), nil
}

type apiRes struct {
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type apiErrRes struct {
	Error struct {
		Code    interface{} `json:"code"`
		Message string      `json:"message"`
		Param   interface{} `json:"param"`
		Type    string      `json:"type"`
	} `json:"error"`
}

func OpenAIRequest(db *gorm.DB, prompt string) (string, error) {
	var apiKey model.ApiKey
	res := db.Where("platform = ?", types.OpenAI).Where("type = ?", "chat").Where("enabled = ?", true).First(&apiKey)
	if res.Error != nil {
		return "", fmt.Errorf("error with fetch OpenAI API KEY：%v", res.Error)
	}

	messages := make([]interface{}, 1)
	messages[0] = types.Message{
		Role:    "user",
		Content: prompt,
	}

	var response apiRes
	var errRes apiErrRes
	client := req.C()
	if len(apiKey.ProxyURL) > 5 {
		client.SetProxyURL(apiKey.ApiURL)
	}
	r, err := client.R().SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+apiKey.Value).
		SetBody(types.ApiRequest{
			Model:       "gpt-3.5-turbo-0125",
			Temperature: 0.9,
			MaxTokens:   1024,
			Stream:      false,
			Messages:    messages,
		}).
		SetErrorResult(&errRes).
		SetSuccessResult(&response).Post(apiKey.ApiURL)
	if err != nil || r.IsErrorState() {
		return "", fmt.Errorf("error with http request: %v%v%s", err, r.Err, errRes.Error.Message)
	}

	// 更新 API KEY 的最后使用时间
	db.Model(&apiKey).UpdateColumn("last_used_at", time.Now().Unix())

	return response.Choices[0].Message.Content, nil
}

// sendTeachingChatCompletionOnce 短剧/教学文案模型非流式对话
func sendTeachingChatCompletionOnce(db *gorm.DB, modelId int, messages []interface{}, minResponseTokens int) (string, error) {
	if modelId <= 0 {
		return "", fmt.Errorf("未配置教学文案模型，请在系统配置中设置 teaching_script_model_id")
	}
	var chatModel model.ChatModel
	if err := db.Where("id", modelId).First(&chatModel).Error; err != nil {
		return "", fmt.Errorf("教学模型 id=%d 不存在: %w", modelId, err)
	}
	if chatModel.Value == "" {
		chatModel.Value = "gpt-4o"
	}
	var apiKey model.ApiKey
	q := db.Where("type", "dy").Where("enabled", true)
	if chatModel.Platform != "" {
		q = q.Where("platform = ?", chatModel.Platform)
	}
	if err := q.Order("last_used_at ASC").First(&apiKey).Error; err != nil {
		return "", fmt.Errorf("获取 Chat API KEY 失败：%v", err)
	}

	temp := float32(0.7)
	if chatModel.Temperature > 0 {
		temp = chatModel.Temperature
	}
	maxTok := chatModel.MaxTokens
	if maxTok <= 0 {
		maxTok = 4096
	}
	if minResponseTokens > 0 && maxTok < minResponseTokens {
		maxTok = minResponseTokens
	}
	if maxTok > 16384 {
		maxTok = 16384
	}

	var response apiRes
	client := req.C()
	if len(apiKey.ProxyURL) > 5 {
		client.SetProxyURL(apiKey.ProxyURL)
	}
	apiURL := fmt.Sprintf("%s/v1/chat/completions", apiKey.ApiURL)
	r, err := client.R().SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+apiKey.Value).
		SetBody(types.ApiRequest{
			Model:       chatModel.Value,
			Temperature: temp,
			MaxTokens:   maxTok,
			Stream:      false,
			Messages:    messages,
		}).Post(apiURL)
	if err != nil {
		return "", fmt.Errorf("请求 OpenAI API失败：%v", err)
	}
	if r.IsErrorState() {
		body, _ := io.ReadAll(r.Body)
		return "", fmt.Errorf("请求 OpenAI API失败：%v, %s", r.Status, string(body))
	}
	body, _ := io.ReadAll(r.Body)
	if err = json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("解析API数据失败：%v, %s", err, string(body))
	}
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("模型返回空 choices：%s", string(body))
	}
	db.Model(&apiKey).UpdateColumn("last_used_at", time.Now().Unix())
	return response.Choices[0].Message.Content, nil
}

func SendTeachingChatCompletion(db *gorm.DB, modelId int, messages []interface{}) (string, error) {
	return sendTeachingChatCompletionOnce(db, modelId, messages, 0)
}

func SendTeachingChatCompletionLongJSON(db *gorm.DB, modelId int, messages []interface{}, minResponseTokens int) (string, error) {
	if minResponseTokens < 4096 {
		minResponseTokens = 4096
	}
	return sendTeachingChatCompletionOnce(db, modelId, messages, minResponseTokens)
}
