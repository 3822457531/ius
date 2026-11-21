package admin

import (
	"bytes"
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/handler"
	"chatplus/store/model"
	"chatplus/utils"
	"chatplus/utils/resp"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ConfigHandler struct {
	handler.BaseHandler
}

func NewConfigHandler(app *core.AppServer, db *gorm.DB) *ConfigHandler {
	return &ConfigHandler{BaseHandler: handler.BaseHandler{App: app, DB: db}}
}

func (h *ConfigHandler) Update(c *gin.Context) {
	var data struct {
		Key    string `json:"key"`
		Config struct {
			types.SystemConfig
			types.PptAppSettings
			types.FenHongConfig
			Content string `json:"content,omitempty"`
			Updated bool   `json:"updated,omitempty"`
		} `json:"config"`
	}

	oldSystemAddress := normalizeBscAddress(h.App.SysConfig.Address)

	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	value := utils.JsonEncode(&data.Config)
	config := model.Config{Key: data.Key, Config: value}
	res := h.DB.FirstOrCreate(&config, model.Config{Key: data.Key})
	if res.Error != nil {
		resp.ERROR(c, res.Error.Error())
		return
	}

	if config.Id > 0 {
		config.Config = value
		res := h.DB.Updates(&config)
		if res.Error != nil {
			resp.ERROR(c, res.Error.Error())
			return
		}

		// update config cache for AppServer
		var cfg model.Config
		h.DB.Where("marker", data.Key).First(&cfg)
		var err error
		if data.Key == "system" {
			fresh := &types.SystemConfig{}
			err = utils.JsonDecode(cfg.Config, fresh)
			if err == nil {
				h.App.SysConfig = fresh
			}
		}
		if err != nil {
			resp.ERROR(c, "Failed to update config cache: "+err.Error())
			return
		}

		// 如果是 system 配置，且收款地址有变更，则同步到 Moralis Streams
		if data.Key == "system" {
			newSystemAddress := normalizeBscAddress(h.App.SysConfig.Address)
			if newSystemAddress != "" && !strings.EqualFold(newSystemAddress, oldSystemAddress) {
				_, _, err := h.updateMoralisStreamAddress(newSystemAddress, "")
				if err != nil {
					resp.ERROR(c, "更新 Moralis 监听地址失败: "+err.Error())
					return
				}
			}
		}
		logger.Infof("Update AppServer's config successfully: %v", config.Config)
	}

	resp.SUCCESS(c, config)
}

// UpdateAddress 调用 Moralis Streams API 更新“监听钱包地址”
// 请求示例（你提供的 curl）：
// PATCH https://api.moralis-streams.com/streams/evm/{streamId}/address
// Headers: x-api-key, Content-Type: application/json
// Body: { "address": "0x..." }
func (h *ConfigHandler) UpdateAddress(c *gin.Context) {
	var req struct {
		Address  string `json:"address"`
		StreamId string `json:"streamId,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	data, _, err := h.updateMoralisStreamAddress(req.Address, req.StreamId)
	if err != nil {
		resp.ERROR(c, err.Error())
		return
	}
	resp.SUCCESS(c, data)
}

func normalizeBscAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(addr), "0x") {
		addr = "0x" + strings.TrimPrefix(addr, "0X")
	}
	// BSC BEP20 地址为 42 位：0x + 40 hex
	if len(addr) != 42 {
		return ""
	}
	return strings.ToLower(addr)
}

// updateMoralisStreamAddress 更新 Moralis Streams 的监听地址
// 返回 ignored=true 表示被忽略（例如当前系统缓存地址和请求地址不一致）。
func (h *ConfigHandler) updateMoralisStreamAddress(address string, streamId string) (gin.H, bool, error) {
	newAddress := normalizeBscAddress(address)
	if newAddress == "" {
		return nil, false, fmt.Errorf("address 不能为空或不合法")
	}

	streamId = strings.TrimSpace(streamId)
	if streamId == "" {
		// 默认 streamId（使用你提供的 Moralis PATCH 示例）
		streamId = "75ed431f-bac8-4fb3-ab74-5cc54c4d7b15"
	}

	// 如果当前系统配置地址并未同步到新地址，则直接忽略，避免次序不一致导致错误提交
	current := normalizeBscAddress(h.App.SysConfig.Address)
	if current != "" && !strings.EqualFold(current, newAddress) {
		return gin.H{
			"message":   "系统配置地址未同步到新地址，已忽略本次 Moralis 更新",
			"current":   current,
			"requested": newAddress,
			"stream_id": streamId,
			"ignored":   true,
		}, true, nil
	}

	apiKey := strings.TrimSpace(h.App.Config.Bsc.MoralisApiKey)
	if apiKey == "" {
		return nil, false, fmt.Errorf("BscConfig.moralis_api_key 未配置")
	}

	url := fmt.Sprintf("https://api.moralis-streams.com/streams/evm/%s/address", streamId)
	payload, err := json.Marshal(gin.H{"address": newAddress})
	if err != nil {
		return nil, false, fmt.Errorf("payload 组装失败: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(payload))
	if err != nil {
		return nil, false, fmt.Errorf("创建 Moralis 请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	respMoralis, err := client.Do(httpReq)
	if err != nil {
		return nil, false, fmt.Errorf("请求 Moralis 失败: %w", err)
	}
	defer respMoralis.Body.Close()

	bodyBytes, _ := io.ReadAll(respMoralis.Body)
	bodyStr := string(bodyBytes)
	logger.Infof("Moralis updateMoralisStreamAddress resp: status=%d, stream_id=%s, body=%s", respMoralis.StatusCode, streamId, bodyStr)

	if respMoralis.StatusCode < 200 || respMoralis.StatusCode >= 300 {
		return nil, false, fmt.Errorf("Moralis 更新地址失败，status=%d, body=%s", respMoralis.StatusCode, bodyStr)
	}

	return gin.H{
		"message":   "Moralis 地址更新成功",
		"stream_id": streamId,
		"address":   newAddress,
		"ignored":   false,
	}, false, nil
}

// Get 获取指定的系统配置
func (h *ConfigHandler) Get(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		resp.NotPermission(c)
		return
	}

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

	resp.SUCCESS(c, value)
}

var adminConfigSecretFields = map[string][]string{
	"ppt": {
		"ppt_text_api_key",
		"ppt_image_api_key",
	},
	"system": {
		"teaching_image_api_key",
		"tts_api_key",
		"tts_volc_secret_key",
		"web_search_api_key",
	},
}

// GetSecretField 读取配置中的敏感字段明文（仅管理员）
func (h *ConfigHandler) GetSecretField(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		resp.NotPermission(c)
		return
	}

	key := strings.TrimSpace(c.Query("key"))
	field := strings.TrimSpace(c.Query("field"))
	allowed, ok := adminConfigSecretFields[key]
	if !ok {
		resp.ERROR(c, "不支持的配置项")
		return
	}
	fieldAllowed := false
	for _, item := range allowed {
		if item == field {
			fieldAllowed = true
			break
		}
	}
	if !fieldAllowed {
		resp.ERROR(c, "不支持的字段")
		return
	}

	var config model.Config
	res := h.DB.Where("marker", key).First(&config)
	if res.Error != nil {
		resp.ERROR(c, res.Error.Error())
		return
	}

	var value map[string]interface{}
	if err := utils.JsonDecode(config.Config, &value); err != nil {
		resp.ERROR(c, err.Error())
		return
	}

	raw, ok := value[field]
	if !ok {
		m := utils.MaskSecret("")
		m["value"] = ""
		resp.SUCCESS(c, m)
		return
	}
	// 永不回明文：仅掩码 hint；value 同步为 hint 以兼容旧前端 SecretKeyInput
	m := utils.MaskSecret(utils.InterfaceToString(raw))
	m["value"] = m["hint"]
	resp.SUCCESS(c, m)
}
