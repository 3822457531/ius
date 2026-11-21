package chatimpl

import (
	"chatplus/core/types"
	"chatplus/service/websearch"
	"chatplus/store/model"
	"fmt"
	"strings"
)

func (h *ChatHandler) resolveWebSearchCredentials() (base, apiKey string, err error) {
	cfgBase, cfgKey := "", ""
	if h.App.SysConfig != nil {
		cfgBase = h.App.SysConfig.WebSearchApiBase
		cfgKey = h.App.SysConfig.WebSearchApiKey
	}
	base, apiKey = websearch.ResolveWebSearchEndpoint(cfgBase, cfgKey)
	if apiKey != "" {
		return base, apiKey, nil
	}

	var key model.ApiKey
	res := h.DB.Where("platform = ?", types.DeepSeek).Where("type = ?", "chat").
		Where("enabled = ?", true).Order("last_used_at ASC").First(&key)
	if res.Error != nil {
		return base, "", fmt.Errorf("请配置 DeepSeek 聊天 API KEY，或在系统配置「联网搜索 API Key」中填写")
	}
	if strings.TrimSpace(cfgBase) == "" && strings.TrimSpace(key.ApiURL) != "" {
		base = websearch.BaseFromChatAPIURL(key.ApiURL)
	}
	return base, key.Value, nil
}

func (h *ChatHandler) runWebSearch(query, oriQuestion string) (string, error) {
	query = websearch.EffectiveSearchQuery(query, oriQuestion)
	base, apiKey, err := h.resolveWebSearchCredentials()
	if err != nil {
		if websearch.LooksLikeNewsQuery(query) {
			if text, err2 := websearch.FetchZaoBaoViaAPI(h.App.Config.ApiConfig); err2 == nil {
				return text, nil
			}
		}
		return "", err
	}
	text, err := websearch.DeepSeekSearch(base, apiKey, query, oriQuestion)
	if err == nil {
		return text, nil
	}
	if websearch.LooksLikeNewsQuery(query) {
		if text, err2 := websearch.FetchZaoBaoViaAPI(h.App.Config.ApiConfig); err2 == nil {
			return text, nil
		}
	}
	return "", err
}
