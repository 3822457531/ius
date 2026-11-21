package handler

import (
	"chatplus/core"
	"chatplus/store/model"
	"chatplus/utils"
	"chatplus/utils/resp"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ConfigHandler struct {
	BaseHandler
}

func NewConfigHandler(app *core.AppServer, db *gorm.DB) *ConfigHandler {
	return &ConfigHandler{BaseHandler: BaseHandler{App: app, DB: db}}
}

// Get 获取指定的系统配置（公开接口，敏感字段不下发）
func (h *ConfigHandler) Get(c *gin.Context) {
	key := c.Query("key")
	var config model.Config
	res := h.DB.Where("marker", key).First(&config)
	if res.Error != nil {
		resp.ERROR(c, res.Error.Error())
		return
	}

	var value map[string]interface{}
	err := utils.JsonDecode(config.Config, &value)
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}
	resp.SUCCESS(c, utils.FilterPublicConfig(key, value))
}

// Locale 根据客户端 IP 推荐界面语言（中/英/日/韩）
// 优先读 Cloudflare CF-IPCountry，否则走在线 Geo（ip-api / geojs）。
func (h *ConfigHandler) Locale(c *gin.Context) {
	hint := strings.TrimSpace(c.GetHeader("CF-IPCountry"))
	if hint == "" {
		hint = strings.TrimSpace(c.GetHeader("X-Country-Code"))
	}
	country := hint
	if country == "" || strings.EqualFold(country, "XX") || strings.EqualFold(country, "T1") {
		// 不回显客户端 IP，仅用于语言推断
		country = utils.LookupCountryCode(c.ClientIP())
	}
	locale := utils.CountryToLocale(country)
	resp.SUCCESS(c, gin.H{
		"locale":  locale,
		"country": country,
	})
}
