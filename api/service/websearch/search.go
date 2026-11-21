package websearch

import (
	"chatplus/core/types"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/imroc/req/v3"
)

// FetchZaoBaoViaAPI 通过 ChatPlus 函数服务拉取早报（需 ApiConfig）
func FetchZaoBaoViaAPI(cfg types.ChatPlusApiConfig) (string, error) {
	if cfg.Token == "" || cfg.ApiURL == "" {
		return "", fmt.Errorf("未配置 ApiConfig")
	}
	url := strings.TrimRight(cfg.ApiURL, "/") + "/api/zaobao/fetch"
	var res struct {
		Code    types.BizCode `json:"code"`
		Message string        `json:"message"`
		Data    struct {
			Title     string `json:"title"`
			UpdatedAt string `json:"updated_at"`
			Items     []struct {
				Title string `json:"title"`
			} `json:"items"`
		} `json:"data"`
	}
	r, err := req.C().R().
		SetHeader("AppId", cfg.AppId).
		SetHeader("Authorization", "Bearer "+cfg.Token).
		SetSuccessResult(&res).Get(url)
	if err != nil || r.IsErrorState() {
		return "", fmt.Errorf("早报服务请求失败")
	}
	if res.Code != types.Success {
		return "", fmt.Errorf(res.Message)
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("**%s 早报**（更新：%s）\n\n", res.Data.UpdatedAt, res.Data.Title))
	for i, v := range res.Data.Items {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, v.Title))
	}
	return b.String(), nil
}

func LooksLikeNewsQuery(q string) bool {
	q = strings.ToLower(q)
	for _, kw := range []string{"新闻", "早报", "热点", "热搜", "今日", "今天", "头条"} {
		if strings.Contains(q, kw) {
			return true
		}
	}
	return false
}

func isInvalidSearchQuery(q string) bool {
	q = strings.TrimSpace(strings.ToLower(q))
	switch q {
	case "", "null", "nil", "none", "undefined", "{}", "[]":
		return true
	}
	return false
}

// EffectiveSearchQuery 工具参数缺失时回退为用户原问题
func EffectiveSearchQuery(toolQuery, userPrompt string) string {
	q := strings.TrimSpace(toolQuery)
	if isInvalidSearchQuery(q) {
		q = strings.TrimSpace(userPrompt)
	}
	if isInvalidSearchQuery(q) {
		return "今日新闻"
	}
	if LooksLikeNewsQuery(q) && len([]rune(q)) <= 8 {
		return q + " 最新 热点"
	}
	if len([]rune(q)) > 128 {
		q = string([]rune(q)[:128])
	}
	return q
}

func QueryFromParams(params map[string]interface{}) string {
	if params == nil {
		return ""
	}
	if v, ok := params["query"]; ok && v != nil {
		switch t := v.(type) {
		case string:
			return strings.TrimSpace(t)
		default:
			s := strings.TrimSpace(fmt.Sprint(v))
			if !isInvalidSearchQuery(s) {
				return s
			}
		}
	}
	raw, _ := json.Marshal(params)
	return strings.TrimSpace(string(raw))
}
