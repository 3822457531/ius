package mj

import (
	"bytes"
	"chatplus/core/types"
	"chatplus/store/model"
	"chatplus/utils"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/imroc/req/v3"
	"gorm.io/gorm"
)

// NanoClient nano-banana / Gemini 兼容网关客户端（与 banana OpenAIImageProvider 对齐：chat.completions + modalities）

type NanoClient struct {
	client *req.Client
	db     *gorm.DB
}

type nanoParams struct {
	AspectRatio string `json:"aspect_ratio,omitempty"`
	Mode        string `json:"mode,omitempty"`
	Model       string `json:"model,omitempty"`
	Resolution  string `json:"resolution,omitempty"`
}

func NewNanoClient(db *gorm.DB) *NanoClient {
	return &NanoClient{
		client: req.C().SetTimeout(time.Minute * 5).SetUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"),
		db:     db,
	}
}

func parseNanoParams(raw string) nanoParams {
	if raw == "" {
		return nanoParams{}
	}
	var params nanoParams
	_ = json.Unmarshal([]byte(raw), &params)
	return params
}

// buildNanoExtraBody 对齐 services/ai_providers/image/openai_provider.py::_build_extra_body
func buildNanoExtraBody(aspectRatio, resolutionUpper string) map[string]interface{} {
	return map[string]interface{}{
		"aspect_ratio": aspectRatio,
		"resolution":   resolutionUpper,
		"generationConfig": map[string]interface{}{
			"imageConfig": map[string]interface{}{
				"aspectRatio": aspectRatio,
				"imageSize":   resolutionUpper,
			},
		},
	}
}

func jpegDataURLFromImageBytes(raw []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, rgba, &jpeg.Options{Quality: 95}); err != nil {
		return "", err
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func extractImageFromChatCompletion(raw []byte) (string, error) {
	var root map[string]interface{}
	if err := json.Unmarshal(raw, &root); err != nil {
		return "", fmt.Errorf("解析响应 JSON 失败: %w", err)
	}
	choices, ok := root["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("响应中无 choices")
	}
	ch0, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("choices[0] 格式错误")
	}
	msg, ok := ch0["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("无 message 字段")
	}

	if mm, ok := msg["multi_mod_content"].([]interface{}); ok {
		for _, part := range mm {
			p, ok := part.(map[string]interface{})
			if !ok {
				continue
			}
			if idata, ok := p["inline_data"].(map[string]interface{}); ok {
				if data, ok := idata["data"].(string); ok && data != "" {
					mime := "image/png"
					if m, ok := idata["mime_type"].(string); ok && m != "" {
						mime = m
					}
					return fmt.Sprintf("data:%s;base64,%s", mime, data), nil
				}
			}
		}
	}

	mc, ok := msg["content"]
	if ok {
		if arr, ok := mc.([]interface{}); ok {
			for _, part := range arr {
				p, ok := part.(map[string]interface{})
				if !ok {
					continue
				}
				if typ, _ := p["type"].(string); typ == "image_url" {
					iu, _ := p["image_url"].(map[string]interface{})
					if iu == nil {
						continue
					}
					url, _ := iu["url"].(string)
					if url != "" {
						return url, nil
					}
				}
			}
		}
		if s, ok := mc.(string); ok && s != "" {
			reMd := regexp.MustCompile(`!\[[^\]]*\]\((https?://[^\s\)]+)\)`)
			if m := reMd.FindStringSubmatch(s); len(m) > 1 {
				return m[1], nil
			}
			reURL := regexp.MustCompile(`https?://[^\s\)\]]+\.(?:png|jpg|jpeg|gif|webp)(?:\?[^\s\)\]]*)?`)
			if m := reURL.FindString(s); m != "" {
				return m, nil
			}
			reB64 := regexp.MustCompile(`data:image/[^;]+;base64,[A-Za-z0-9+/=]+`)
			if m := reB64.FindString(s); m != "" {
				return m, nil
			}
		}
	}

	return "", fmt.Errorf("无法从 chat 响应中解析出图片，请检查网关是否返回多模态 image 内容")
}

func (c *NanoClient) Imagine(task types.MjTask) (ImageRes, error) {
	params := parseNanoParams(task.Params)
	if params.Model == "" {
		params.Model = "gemini-3-pro-image-preview"
	}
	if params.AspectRatio == "" {
		params.AspectRatio = "1:1"
	}
	if params.Resolution == "" {
		params.Resolution = "2K"
	}
	resUpper := strings.ToUpper(strings.TrimSpace(params.Resolution))

	extra := buildNanoExtraBody(params.AspectRatio, resUpper)
	systemContent := fmt.Sprintf("aspect_ratio=%s, resolution=%s", params.AspectRatio, resUpper)
	if strings.TrimSpace(task.NegPrompt) != "" {
		systemContent += ", negative aspects: " + task.NegPrompt
	}

	userContent := make([]interface{}, 0, len(task.ImgArr)+1)
	for i, imgURL := range task.ImgArr {
		imageData, err := utils.DownloadImage(imgURL, "")
		if err != nil {
			return ImageRes{}, fmt.Errorf("下载参考图 %d 失败: %v", i+1, err)
		}
		dataURL, err := jpegDataURLFromImageBytes(imageData)
		if err != nil {
			return ImageRes{}, fmt.Errorf("处理参考图 %d 失败: %v", i+1, err)
		}
		userContent = append(userContent, map[string]interface{}{
			"type": "image_url",
			"image_url": map[string]interface{}{
				"url": dataURL,
			},
		})
	}
	userContent = append(userContent, map[string]interface{}{
		"type": "text",
		"text": task.Prompt,
	})

	body := map[string]interface{}{
		"model": params.Model,
		"messages": []interface{}{
			map[string]interface{}{"role": "system", "content": systemContent},
			map[string]interface{}{"role": "user", "content": userContent},
		},
		"modalities": []string{"text", "image"},
	}
	for k, v := range extra {
		body[k] = v
	}

	raw, err := c.postJSON("v1/chat/completions", task.ChannelId, body)
	if err != nil {
		return ImageRes{}, err
	}
	urlStr, err := extractImageFromChatCompletion(raw)
	if err != nil {
		return ImageRes{}, err
	}
	return ImageRes{Code: 1, Result: urlStr}, nil
}

func (c *NanoClient) Blend(task types.MjTask) (ImageRes, error) {
	return ImageRes{Code: -1, Description: "当前模型暂不支持融图功能"}, nil
}

func (c *NanoClient) SwapFace(task types.MjTask) (ImageRes, error) {
	return ImageRes{Code: -1, Description: "当前模型暂不支持换脸功能"}, nil
}

func (c *NanoClient) Upscale(task types.MjTask) (ImageRes, error) {
	return ImageRes{Code: -1, Description: "当前模型暂不支持放大功能"}, nil
}

func (c *NanoClient) Variation(task types.MjTask) (ImageRes, error) {
	return ImageRes{Code: -1, Description: "当前模型暂不支持变换功能"}, nil
}

func (c *NanoClient) QueryTask(taskId string) (QueryRes, error) {
	return QueryRes{}, fmt.Errorf("nano 模型不支持查询任务")
}

func (c *NanoClient) postJSON(apiPath string, channel string, body interface{}) ([]byte, error) {
	apiKey, err := c.getApiKey(channel)
	if err != nil {
		return nil, err
	}

	base := apiKey.ApiURL
	if base != "" && base[len(base)-1] == '/' {
		base = base[:len(base)-1]
	}
	path := apiPath
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	apiURL := fmt.Sprintf("%s/%s", base, path)

	resp, err := c.client.R().
		SetHeader("Authorization", "Bearer "+apiKey.Value).
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post(apiURL)
	if err != nil {
		return nil, fmt.Errorf("请求 API 出错：%v", err)
	}

	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("读取响应失败：%v", readErr)
	}

	if resp.IsErrorState() {
		return nil, fmt.Errorf("API 返回错误：%s", string(bodyBytes))
	}

	if err = c.db.Model(&apiKey).Update("last_used_at", time.Now().Unix()).Error; err != nil {
		logger.Error("update api key last used time error: ", err)
	}

	return bodyBytes, nil
}

func (c *NanoClient) getApiKey(channel string) (model.ApiKey, error) {
	apiURL := resolveApiKeyURL(channel)
	session := c.db.Session(&gorm.Session{}).Where("type = ?", "nano").Where("enabled = ?", true)
	if apiURL != "" {
		session = session.Where("api_url = ?", apiURL)
	}
	var apiKey model.ApiKey
	err := session.Order("last_used_at ASC").First(&apiKey).Error
	if err != nil {
		return model.ApiKey{}, fmt.Errorf("no available MidJourney api key: %v", err)
	}
	return apiKey, nil
}
