package mj

import (
	"bytes"
	"chatplus/core/types"
	"chatplus/store/model"
	"chatplus/utils"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/imroc/req/v3"
	"gorm.io/gorm"
)

// GptImage2Client OpenAI 兼容图像接口
// - Apilio：POST /v1/images/generations，文生图 JSON；图生图同接口传 image:[公网URL]
// - 其它网关（如 weda）：文生图 generations；图生图走 /v1/images/edits multipart

const defaultGptImage2APIBase = "https://api.apilio.ai"
const defaultGptImage2Model = "gpt-image-2"

// 同一用户同一次生图请求最多尝试次数（含首次）；连续均失败才判定失败
const gptImage2MaxAttempts = 3

type GptImage2Client struct {
	client *req.Client
	db     *gorm.DB
}

type gptImage2Params struct {
	AspectRatio string `json:"aspect_ratio,omitempty"`
	Mode        string `json:"mode,omitempty"`
	Model       string `json:"model,omitempty"`
	Quality     string `json:"quality,omitempty"`
	Size        string `json:"size,omitempty"`
}

type gptImage2Response struct {
	Data []struct {
		B64JSON       string `json:"b64_json"`
		URL           string `json:"url"`
		RevisedPrompt string `json:"revised_prompt"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func NewGptImage2Client(db *gorm.DB) *GptImage2Client {
	return &GptImage2Client{
		client: req.C().SetTimeout(5 * time.Minute).SetUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"),
		db:     db,
	}
}

func parseGptImage2Params(raw string) gptImage2Params {
	if raw == "" {
		return gptImage2Params{}
	}
	var params gptImage2Params
	_ = json.Unmarshal([]byte(raw), &params)
	return params
}

func gptImage2ModelName(m string) string {
	m = strings.TrimSpace(m)
	if m == "" || strings.EqualFold(m, "gpt-image-2-all") {
		return defaultGptImage2Model
	}
	return m
}

func gptImage2SizeFromAspect(rate, explicitSize string) string {
	if s := strings.TrimSpace(explicitSize); s != "" {
		return s
	}
	switch strings.TrimSpace(rate) {
	case "16:9":
		return "1536x1024"
	case "9:16":
		return "1024x1536"
	default:
		return "1024x1024"
	}
}

func gptImage2GenQuality(q string) string {
	q = strings.ToLower(strings.TrimSpace(q))
	switch q {
	case "low", "medium", "high", "auto":
		return q
	default:
		return "low"
	}
}

func gptImage2EditQuality(q string) string {
	q = strings.ToLower(strings.TrimSpace(q))
	switch q {
	case "low", "medium", "high", "auto":
		return q
	default:
		return "auto"
	}
}

// gptImage2ShouldRetry 上游失败、无图、安全策略拒绝等可重试
func gptImage2ShouldRetry(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// 配置/参数类错误不重试
	if strings.Contains(msg, "no available gpt-image-2 api key") ||
		strings.Contains(msg, "unknown parameter") ||
		strings.Contains(msg, "参考图必须是公网") ||
		strings.Contains(msg, "image url") {
		return false
	}
	retryHints := []string{
		"请求 api 出错",
		"api 返回错误",
		"未返回图片数据",
		"未返回 url",
		"解析响应失败",
		"safety system",
		"rejected by the safety",
		"content_policy",
		"content policy",
		"moderation",
		"rate limit",
		"too many requests",
		"timeout",
		"timed out",
		"connection reset",
		"connection refused",
		"temporarily unavailable",
		"bad gateway",
		"service unavailable",
		"internal server error",
		"server error",
		"overloaded",
		"try again",
		"upstream",
		"空响应",
		"下载参考图失败",
	}
	for _, h := range retryHints {
		if strings.Contains(msg, h) {
			return true
		}
	}
	// 默认：未知上游错误也重试，提升成功率
	return true
}

func (c *GptImage2Client) Imagine(task types.MjTask) (ImageRes, error) {
	params := parseGptImage2Params(task.Params)
	modelName := gptImage2ModelName(params.Model)

	var lastErr error
	var tried int
	for attempt := 1; attempt <= gptImage2MaxAttempts; attempt++ {
		tried = attempt
		payload, err := c.imagineOnce(task, params, modelName)
		if err == nil && strings.TrimSpace(payload) != "" {
			if attempt > 1 {
				logger.Infof("gpt-image-2 第 %d/%d 次尝试成功 taskId=%s", attempt, gptImage2MaxAttempts, task.TaskId)
			}
			return ImageRes{Code: 1, Result: payload}, nil
		}
		if err == nil {
			err = fmt.Errorf("gpt-image-2 未返回图片数据")
		}
		lastErr = err
		logger.Warnf("gpt-image-2 第 %d/%d 次失败 taskId=%s: %v", attempt, gptImage2MaxAttempts, task.TaskId, err)

		if attempt >= gptImage2MaxAttempts || !gptImage2ShouldRetry(err) {
			break
		}
		// 退避：1s、2s
		time.Sleep(time.Duration(attempt) * time.Second)
	}
	return ImageRes{}, fmt.Errorf("生图失败（已尝试 %d/%d 次）: %v", tried, gptImage2MaxAttempts, lastErr)
}

func extractGptImage2Payload(res gptImage2Response) (string, error) {
	if res.Error != nil {
		msg := strings.TrimSpace(res.Error.Message)
		if msg == "" {
			msg = strings.TrimSpace(res.Error.Code)
		}
		if msg == "" {
			msg = "上游返回 error"
		}
		return "", fmt.Errorf("API 返回错误：%s", msg)
	}
	if len(res.Data) == 0 {
		return "", fmt.Errorf("gpt-image-2 未返回图片数据")
	}
	item := res.Data[0]
	if item.URL != "" {
		return item.URL, nil
	}
	b64 := strings.TrimSpace(item.B64JSON)
	if b64 == "" {
		return "", fmt.Errorf("gpt-image-2 未返回 url 或 b64_json")
	}
	if strings.HasPrefix(b64, "data:image") {
		return b64, nil
	}
	return "data:image/jpeg;base64," + b64, nil
}

func gptImage2FileName(imgURL string, index int) string {
	name := filepath.Base(imgURL)
	ext := filepath.Ext(name)
	if ext == "" {
		ext = ".jpeg"
	}
	if name == "." || name == "/" || name == "" || name == ext {
		return fmt.Sprintf("image_%d%s", index, ext)
	}
	return name
}

func isApilioImageAPI(apiURL, platform string) bool {
	p := strings.ToLower(strings.TrimSpace(platform))
	if p == "apilio" {
		return true
	}
	u := strings.ToLower(strings.TrimSpace(apiURL))
	return strings.Contains(u, "apilio.ai") || strings.Contains(u, "aiproxy.vip")
}

func isPublicReachableImageURL(u string) bool {
	u = strings.TrimSpace(u)
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		return false
	}
	low := strings.ToLower(u)
	blocked := []string{"localhost", "127.0.0.1", "0.0.0.0", "::1", "host.docker.internal"}
	for _, b := range blocked {
		if strings.Contains(low, b) {
			return false
		}
	}
	return true
}

// joinPublicImageURL 用 KEY 上配置的公网域名拼接相对路径（如 /static/upload/...）
func joinPublicImageURL(publicBase, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	// 纠正误拼的 /static/static/
	for strings.Contains(path, "/static/static/") {
		path = strings.ReplaceAll(path, "/static/static/", "/static/")
	}
	base := strings.TrimRight(strings.TrimSpace(publicBase), "/")
	if base == "" {
		return path
	}
	// base 已含 /static 且 path 也以 /static 开头时，避免重复
	if strings.HasSuffix(strings.ToLower(base), "/static") && strings.HasPrefix(strings.ToLower(path), "/static/") {
		path = path[len("/static"):]
	}
	return base + path
}

func (c *GptImage2Client) imagineOnce(task types.MjTask, params gptImage2Params, modelName string) (string, error) {
	size := gptImage2SizeFromAspect(params.AspectRatio, params.Size)
	var resBody gptImage2Response

	apiKey, err := c.getApiKey(task.ChannelId)
	if err != nil {
		return "", err
	}
	useApilio := isApilioImageAPI(apiKey.ApiURL, apiKey.Platform)

	if useApilio {
		// Apilio：文生图 / 图生图均走 /v1/images/generations；图生图传公网 URL 数组
		body := map[string]interface{}{
			"model":  modelName,
			"prompt": task.Prompt,
			"size":   size,
		}
		if len(task.ImgArr) > 0 {
			urls := make([]string, 0, len(task.ImgArr))
			for _, u := range task.ImgArr {
				u = joinPublicImageURL(apiKey.PublicBase, u)
				if u == "" {
					continue
				}
				if !isPublicReachableImageURL(u) {
					if strings.TrimSpace(apiKey.PublicBase) == "" {
						return "", fmt.Errorf("参考图为相对路径，请在后台该 API KEY 填写「公网域名」后拼接，当前路径: %s", u)
					}
					return "", fmt.Errorf("参考图必须是公网可访问的图片 URL（Apilio 不支持本地/内网地址）: %s", u)
				}
				urls = append(urls, u)
			}
			if len(urls) == 0 {
				return "", fmt.Errorf("参考图必须是公网可访问的图片 URL")
			}
			body["image"] = urls
		}
		if err := c.postJSONWithKey(apiKey, "v1/images/generations", body, &resBody); err != nil {
			return "", err
		}
		return extractGptImage2Payload(resBody)
	}

	// 旧网关：文生图 generations；图生图 edits multipart
	if len(task.ImgArr) > 0 {
		quality := gptImage2EditQuality(params.Quality)
		if err := c.postEdits(task, modelName, size, quality, &resBody); err != nil {
			return "", err
		}
	} else {
		quality := gptImage2GenQuality(params.Quality)
		body := c.buildGenerationBody(modelName, task.Prompt, size, quality)
		if err := c.postJSONWithKey(apiKey, "v1/images/generations", body, &resBody); err != nil {
			return "", err
		}
	}
	return extractGptImage2Payload(resBody)
}

// buildGenerationBody 兼容要求 response_format=b64_json 的旧网关
func (c *GptImage2Client) buildGenerationBody(modelName, prompt, size, quality string) map[string]interface{} {
	return map[string]interface{}{
		"model":           modelName,
		"prompt":          prompt,
		"n":               1,
		"size":            size,
		"quality":         quality,
		"response_format": "b64_json",
	}
}

func (c *GptImage2Client) postEdits(task types.MjTask, modelName, size, quality string, result *gptImage2Response) error {
	var bodyBuffer bytes.Buffer
	writer := multipart.NewWriter(&bodyBuffer)
	_ = writer.WriteField("model", modelName)
	_ = writer.WriteField("prompt", task.Prompt)
	_ = writer.WriteField("n", "1")
	_ = writer.WriteField("size", size)
	_ = writer.WriteField("quality", quality)
	// 勿传 format/background/moderation：当前上游 edits 不识别会报 Unknown parameter

	for i, imgURL := range task.ImgArr {
		imageData, err := utils.DownloadImage(imgURL, "")
		if err != nil {
			_ = writer.Close()
			return fmt.Errorf("下载参考图失败: %v", err)
		}
		fileName := gptImage2FileName(imgURL, i)
		part, err := writer.CreateFormFile("image", fileName)
		if err != nil {
			_ = writer.Close()
			return err
		}
		if _, err = io.Copy(part, bytes.NewReader(imageData)); err != nil {
			_ = writer.Close()
			return err
		}
	}

	contentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		return err
	}
	return c.postRaw("v1/images/edits", task.ChannelId, contentType, bodyBuffer.Bytes(), result)
}

func (c *GptImage2Client) Blend(task types.MjTask) (ImageRes, error) {
	return ImageRes{Code: -1, Description: "gpt-image-2 暂不支持融图"}, nil
}

func (c *GptImage2Client) SwapFace(task types.MjTask) (ImageRes, error) {
	return ImageRes{Code: -1, Description: "gpt-image-2 暂不支持换脸"}, nil
}

func (c *GptImage2Client) Upscale(task types.MjTask) (ImageRes, error) {
	return ImageRes{Code: -1, Description: "gpt-image-2 暂不支持放大"}, nil
}

func (c *GptImage2Client) Variation(task types.MjTask) (ImageRes, error) {
	return ImageRes{Code: -1, Description: "gpt-image-2 暂不支持变换"}, nil
}

func (c *GptImage2Client) QueryTask(taskId string) (QueryRes, error) {
	return QueryRes{}, fmt.Errorf("gpt-image-2 不支持查询任务")
}

func gptImage2ParseAPIError(raw []byte, httpErr bool) error {
	msg := strings.TrimSpace(string(raw))
	if msg == "" {
		if httpErr {
			return fmt.Errorf("API 返回错误：空响应")
		}
		return nil
	}
	var envelope struct {
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &envelope) == nil {
		if envelope.Error != nil {
			m := strings.TrimSpace(envelope.Error.Message)
			if m == "" {
				m = strings.TrimSpace(envelope.Error.Code)
			}
			if m != "" {
				return fmt.Errorf("API 返回错误：%s", m)
			}
		}
		if m := strings.TrimSpace(envelope.Message); m != "" && httpErr {
			return fmt.Errorf("API 返回错误：%s", m)
		}
	}
	if httpErr {
		// 截断过长原文，便于日志与重试匹配（含 safety system 文案）
		if len(msg) > 500 {
			msg = msg[:500] + "..."
		}
		return fmt.Errorf("API 返回错误：%s", msg)
	}
	return nil
}

func (c *GptImage2Client) postJSON(apiPath string, channel string, body interface{}, result interface{}) error {
	apiKey, err := c.getApiKey(channel)
	if err != nil {
		return err
	}
	return c.postJSONWithKey(apiKey, apiPath, body, result)
}

func (c *GptImage2Client) postJSONWithKey(apiKey model.ApiKey, apiPath string, body interface{}, result interface{}) error {
	apiURL := joinAPIURL(apiKey.ApiURL, apiPath)

	resp, err := c.client.R().
		SetHeader("Authorization", "Bearer "+apiKey.Value).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetBody(body).
		Post(apiURL)
	if err != nil {
		return fmt.Errorf("请求 API 出错：%v", err)
	}
	raw, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return readErr
	}
	if resp.IsErrorState() {
		return gptImage2ParseAPIError(raw, true)
	}
	// 部分网关 HTTP 200 但 body 含 error / 无 data
	if err := gptImage2ParseAPIError(raw, false); err != nil {
		return err
	}
	if result != nil {
		if err = json.Unmarshal(raw, result); err != nil {
			return fmt.Errorf("解析响应失败：%w", err)
		}
	}
	_ = c.db.Model(&apiKey).Update("last_used_at", time.Now().Unix()).Error
	return nil
}

func (c *GptImage2Client) postRaw(apiPath, channel, contentType string, body []byte, result interface{}) error {
	apiKey, err := c.getApiKey(channel)
	if err != nil {
		return err
	}
	apiURL := joinAPIURL(apiKey.ApiURL, apiPath)

	resp, err := c.client.R().
		SetHeader("Authorization", "Bearer "+apiKey.Value).
		SetHeader("Content-Type", contentType).
		SetHeader("Accept", "application/json").
		SetBody(body).
		Post(apiURL)
	if err != nil {
		return fmt.Errorf("请求 API 出错：%v", err)
	}
	raw, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return readErr
	}
	if resp.IsErrorState() {
		return gptImage2ParseAPIError(raw, true)
	}
	if err := gptImage2ParseAPIError(raw, false); err != nil {
		return err
	}
	if result != nil {
		if err = json.Unmarshal(raw, result); err != nil {
			return fmt.Errorf("解析响应失败：%w", err)
		}
	}
	_ = c.db.Model(&apiKey).Update("last_used_at", time.Now().Unix()).Error
	return nil
}

func joinAPIURL(base, apiPath string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		base = defaultGptImage2APIBase
	}
	path := strings.TrimPrefix(apiPath, "/")
	return fmt.Sprintf("%s/%s", base, path)
}

func (c *GptImage2Client) getApiKey(channel string) (model.ApiKey, error) {
	apiURL := resolveApiKeyURL(channel)
	typesTry := []string{"image2", "img", "nano"}
	for _, typ := range typesTry {
		session := c.db.Session(&gorm.Session{}).Where("type = ?", typ).Where("enabled = ?", true)
		if apiURL != "" {
			session = session.Where("api_url = ?", apiURL)
		}
		var apiKey model.ApiKey
		// 优先级越高越优先；同优先级按最少使用轮询
		err := session.Order("priority DESC").Order("last_used_at ASC").First(&apiKey).Error
		if err == nil {
			if strings.TrimSpace(apiKey.ApiURL) == "" {
				apiKey.ApiURL = defaultGptImage2APIBase
			}
			return apiKey, nil
		}
	}
	return model.ApiKey{}, fmt.Errorf("no available gpt-image-2 api key (type=image2/img/nano), 请在后台配置 Apilio 等生图 KEY，api_url=%s", defaultGptImage2APIBase)
}
