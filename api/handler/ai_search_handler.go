package handler

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/service"
	"chatplus/service/websearch"
	"chatplus/store/model"
	"chatplus/utils/resp"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AiSearchHandler 手机端 AI搜：DeepSeek Search 网页检索
type AiSearchHandler struct {
	*BaseHandler
	userService *service.UserService
}

func NewAiSearchHandler(app *core.AppServer, config *types.AppConfig, db *gorm.DB, userService *service.UserService) *AiSearchHandler {
	return &AiSearchHandler{
		BaseHandler: &BaseHandler{App: app, DB: db},
		userService: userService,
	}
}

func (h *AiSearchHandler) resolveCredentials() (base, apiKey string, err error) {
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
		return base, "", fmt.Errorf("请配置联网搜索 API Key（系统配置或 API KEY 管理）")
	}
	if strings.TrimSpace(cfgBase) == "" && strings.TrimSpace(key.ApiURL) != "" {
		base = websearch.BaseFromChatAPIURL(key.ApiURL)
	}
	return base, key.Value, nil
}

func (h *AiSearchHandler) searchPowerCost() int {
	if h.App.SysConfig == nil {
		return 1
	}
	cost := h.App.SysConfig.AiSearchSearchPower
	if cost < 0 {
		return 0
	}
	return cost
}

func (h *AiSearchHandler) detailPowerCost() int {
	if h.App.SysConfig == nil {
		return 10
	}
	cost := h.App.SysConfig.AiSearchDetailPower
	if cost < 0 {
		return 0
	}
	return cost
}

func (h *AiSearchHandler) ensurePower(c *gin.Context, cost int) bool {
	if cost <= 0 {
		return true
	}
	user, err := h.GetLoginUser(c)
	if err != nil {
		resp.NotAuth(c)
		return false
	}
	if user.Power < float64(cost) {
		resp.ERROR(c, fmt.Sprintf("算力不足，本次操作需要 %d 算力", cost))
		return false
	}
	return true
}

func (h *AiSearchHandler) charge(c *gin.Context, cost int, remark string) bool {
	if cost <= 0 {
		return true
	}
	uid := int(h.GetLoginUserId(c))
	if err := h.userService.DecreasePower(uid, cost, model.PowerLog{
		Type:   types.PowerConsume,
		Model:  service.PowerLogModelAiSearch,
		Remark: remark,
	}); err != nil {
		resp.ERROR(c, err.Error())
		return false
	}
	notifyKey := fmt.Sprintf("ai-search:%d:%d", uid, time.Now().UnixNano())
	service.NotifyAppPlazaPowerConsumed(h.DB, uint(uid), float64(cost), "AI搜", notifyKey)
	return true
}

// Search POST /api/ai-search/search  — 网页搜索（上游无 offset，一次多取供前端分页/筛选）
func (h *AiSearchHandler) Search(c *gin.Context) {
	if h.GetLoginUserId(c) == 0 {
		resp.NotAuth(c)
		return
	}
	var body struct {
		Query string `json:"query"`
		Count int    `json:"count"` // 1~50，供前端本地分页/筛选
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	query := strings.TrimSpace(body.Query)
	if query == "" {
		resp.ERROR(c, "请输入搜索关键词")
		return
	}
	cost := h.searchPowerCost()
	if !h.ensurePower(c, cost) {
		return
	}
	base, apiKey, err := h.resolveCredentials()
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}
	fetchCount := body.Count
	if fetchCount <= 0 {
		fetchCount = 40
	}
	if fetchCount > 50 {
		fetchCount = 50
	}
	hits, err := websearch.DeepSeekSearchHits(base, apiKey, query, query, fetchCount)
	if err != nil {
		resp.ERROR(c, sanitizeSearchErr(err))
		return
	}
	if !h.charge(c, cost, fmt.Sprintf("AI搜关键词检索，消耗 %d 算力", cost)) {
		return
	}
	resp.SUCCESS(c, gin.H{
		"ok":      true,
		"query":   query,
		"results": hits,
		"count":   len(hits),
		"total":   len(hits),
	})
}

// Fetch POST /api/ai-search/fetch — 抓取单页正文
func (h *AiSearchHandler) Fetch(c *gin.Context) {
	if h.GetLoginUserId(c) == 0 {
		resp.NotAuth(c)
		return
	}
	var body struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	pageURL := strings.TrimSpace(body.URL)
	if pageURL == "" || (!strings.HasPrefix(pageURL, "http://") && !strings.HasPrefix(pageURL, "https://")) {
		resp.ERROR(c, "无效的链接")
		return
	}
	cost := h.detailPowerCost()
	if !h.ensurePower(c, cost) {
		return
	}
	base, apiKey, err := h.resolveCredentials()
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}
	text, err := websearch.DeepSeekFetch(base, apiKey, pageURL)
	if err != nil {
		resp.ERROR(c, sanitizeSearchErr(err))
		return
	}
	if !h.charge(c, cost, fmt.Sprintf("AI搜查看网页详情，消耗 %d 算力", cost)) {
		return
	}
	title := pageURL
	content := text
	if strings.HasPrefix(text, "# ") {
		if i := strings.Index(text, "\n"); i > 0 {
			title = strings.TrimSpace(strings.TrimPrefix(text[:i], "# "))
			content = strings.TrimSpace(text[i+1:])
		}
	}
	resp.SUCCESS(c, gin.H{
		"ok":      true,
		"url":     pageURL,
		"title":   title,
		"content": content,
	})
}

func sanitizeSearchErr(err error) string {
	if err == nil {
		return "搜索失败"
	}
	msg := err.Error()
	msg = strings.ReplaceAll(msg, "DeepSeek ", "")
	msg = strings.ReplaceAll(msg, "DeepSeek", "")
	msg = strings.TrimSpace(msg)
	if msg == "" || msg == "搜索失败:" || msg == "抓取失败:" {
		return "搜索失败，请稍后重试"
	}
	return msg
}
