package websearch

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/imroc/req/v3"
)

const defaultDeepSeekAPIBase = "https://api.deepseek.com"

type deepSeekSearchReq struct {
	Query        string  `json:"query"`
	Count        int     `json:"count,omitempty"`
	OriQuestion  *string `json:"ori_question,omitempty"`
	Localization string  `json:"localization,omitempty"`
	NeedContent  bool    `json:"need_content,omitempty"`
}

type deepSeekSearchResult struct {
	Results []struct {
		Title       string  `json:"title"`
		URL         string  `json:"url"`
		Snippet     string  `json:"snippet"`
		Summary     string  `json:"summary"`
		Content     string  `json:"content"`
		SiteName    *string `json:"site_name"`
		SiteIcon    *string `json:"site_icon"`
		Date        *string `json:"date"`
		Authority   int     `json:"authority"`
		RerankScore float64 `json:"rerank_score"`
	} `json:"results"`
}

type deepSeekAPIError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// SearchHit 结构化搜索结果（供 AI 搜前端 / 聊天引用）
type SearchHit struct {
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Snippet     string  `json:"snippet"`
	Summary     string  `json:"summary"`
	SiteName    string  `json:"site_name"`
	SiteIcon    string  `json:"site_icon,omitempty"`
	Date        string  `json:"date,omitempty"`
	Authority   int     `json:"authority"`
	RerankScore float64 `json:"rerank_score"`
}

// DeepSeekSearchHits 调用 DeepSeek POST /v1/tools/search，返回结构化结果
func DeepSeekSearchHits(apiBase, apiKey, query, oriQuestion string, count int) ([]SearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("搜索关键词为空")
	}
	if len([]rune(query)) > 128 {
		query = string([]rune(query)[:128])
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, fmt.Errorf("未配置联网搜索 API Key")
	}
	base := normalizeHTTPBase(apiBase)
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		return nil, fmt.Errorf("联网搜索 API 地址无效（应为 https://api.deepseek.com），请勿把 API Key 填在地址栏")
	}
	if count <= 0 {
		count = 10
	}
	if count > 50 {
		count = 50
	}

	body := deepSeekSearchReq{
		Query:       query,
		Count:       count,
		NeedContent: false,
	}
	if oq := strings.TrimSpace(oriQuestion); oq != "" {
		if len([]rune(oq)) > 128 {
			oq = string([]rune(oq)[:128])
		}
		body.OriQuestion = &oq
	}

	var ok deepSeekSearchResult
	var errBody deepSeekAPIError
	r, err := req.C().
		SetTimeout(20 * time.Second).
		R().
		SetHeader("Authorization", "Bearer "+apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		SetSuccessResult(&ok).
		SetErrorResult(&errBody).
		Post(base + "/v1/tools/search")
	if err != nil {
		return nil, err
	}
	if r.IsErrorState() {
		msg := errBody.Error.Message
		if msg == "" {
			msg = r.Status
		}
		return nil, fmt.Errorf("搜索失败: %s", msg)
	}

	hits := make([]SearchHit, 0, len(ok.Results))
	for _, item := range ok.Results {
		summary := strings.TrimSpace(item.Summary)
		snippet := strings.TrimSpace(item.Snippet)
		if summary == "" {
			summary = snippet
		}
		site := ""
		if item.SiteName != nil {
			site = *item.SiteName
		}
		icon := ""
		if item.SiteIcon != nil {
			icon = strings.TrimSpace(*item.SiteIcon)
		}
		date := ""
		if item.Date != nil {
			date = *item.Date
		}
		hits = append(hits, SearchHit{
			Title:       item.Title,
			URL:         item.URL,
			Snippet:     snippet,
			Summary:     summary,
			SiteName:    site,
			SiteIcon:    icon,
			Date:        date,
			Authority:   item.Authority,
			RerankScore: item.RerankScore,
		})
	}
	return hits, nil
}

// DeepSeekSearch 调用 DeepSeek POST /v1/tools/search（聊天上下文文本格式）
func DeepSeekSearch(apiBase, apiKey, query, oriQuestion string) (string, error) {
	hits, err := DeepSeekSearchHits(apiBase, apiKey, query, oriQuestion, 10)
	if err != nil {
		return "", err
	}
	if len(hits) == 0 {
		return "未找到相关网页结果，请换个关键词。", nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("关键词「%s」联网检索结果（供回答参考）：\n\n", query))
	for i, item := range hits {
		if i >= 10 {
			break
		}
		b.WriteString(fmt.Sprintf("%d. **%s**", i+1, item.Title))
		if item.SiteName != "" {
			b.WriteString(fmt.Sprintf("（%s）", item.SiteName))
		}
		if item.Date != "" {
			b.WriteString(fmt.Sprintf(" [%s]", item.Date))
		}
		b.WriteString(fmt.Sprintf("\n   %s\n   %s\n\n", item.Summary, item.URL))
	}
	return strings.TrimSpace(b.String()), nil
}

// DeepSeekFetch POST /v1/tools/fetch（按需抓取单页）
func DeepSeekFetch(apiBase, apiKey, pageURL string) (string, error) {
	pageURL = strings.TrimSpace(pageURL)
	if pageURL == "" {
		return "", fmt.Errorf("url 为空")
	}
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", fmt.Errorf("未配置联网搜索 API Key")
	}
	base := strings.TrimRight(strings.TrimSpace(apiBase), "/")
	if base == "" {
		base = defaultDeepSeekAPIBase
	}
	var ok struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	var errBody deepSeekAPIError
	r, err := req.C().R().
		SetHeader("Authorization", "Bearer "+apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]string{"url": pageURL}).
		SetSuccessResult(&ok).
		SetErrorResult(&errBody).
		Post(base + "/v1/tools/fetch")
	if err != nil {
		return "", err
	}
	if r.IsErrorState() {
		msg := errBody.Error.Message
		if msg == "" {
			msg = r.Status
		}
		return "", fmt.Errorf("抓取失败: %s", msg)
	}
	if strings.TrimSpace(ok.Content) == "" {
		return "", fmt.Errorf("页面正文为空")
	}
	title := strings.TrimSpace(ok.Title)
	if title == "" {
		title = pageURL
	}
	content := ok.Content
	if len(content) > 12000 {
		content = content[:12000] + "\n...(已截断)"
	}
	return fmt.Sprintf("# %s\n\n%s", title, content), nil
}

func looksLikeAPIKey(s string) bool {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "Bearer ")
	return strings.HasPrefix(s, "sk-") || (len(s) > 20 && !strings.Contains(s, "://") && !strings.Contains(s, "/"))
}

func normalizeHTTPBase(base string) string {
	base = strings.TrimSpace(base)
	base = strings.TrimPrefix(base, "Bearer ")
	base = strings.TrimRight(base, "/")
	if base == "" {
		return defaultDeepSeekAPIBase
	}
	if looksLikeAPIKey(base) {
		return defaultDeepSeekAPIBase
	}
	if i := strings.Index(base, "/v1/"); i > 0 {
		base = base[:i]
	}
	if strings.HasPrefix(base, "http://") || strings.HasPrefix(base, "https://") {
		return base
	}
	if strings.Contains(base, ".") {
		return "https://" + strings.TrimLeft(base, "/")
	}
	return defaultDeepSeekAPIBase
}

// ResolveWebSearchEndpoint 纠正「把 Key 填进地址栏」等配置错误
func ResolveWebSearchEndpoint(configBase, configKey string) (base, key string) {
	base = strings.TrimSpace(configBase)
	key = strings.TrimSpace(configKey)
	key = strings.TrimPrefix(key, "Bearer ")
	if looksLikeAPIKey(base) {
		if key == "" {
			key = strings.TrimPrefix(base, "Bearer ")
		}
		base = ""
	}
	base = normalizeHTTPBase(base)
	return base, key
}

// BaseFromChatAPIURL 从 DeepSeek 聊天 api_url 提取网关根地址
func BaseFromChatAPIURL(apiURL string) string {
	u := strings.TrimSpace(apiURL)
	u = strings.TrimSuffix(u, "/")
	u = strings.TrimSuffix(u, "/v1/chat/completions")
	u = strings.TrimSuffix(u, "/chat/completions")
	return normalizeHTTPBase(u)
}

func NormalizeAPIBase(base string) string {
	return normalizeHTTPBase(base)
}

func ParseAPIError(body []byte) string {
	var e deepSeekAPIError
	if json.Unmarshal(body, &e) == nil && e.Error.Message != "" {
		return e.Error.Message
	}
	return string(body)
}
