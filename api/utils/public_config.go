package utils

import (
	"strings"
)

// 公开配置允许下发的字段白名单（默认拒绝）
var publicSystemConfigFields = map[string]bool{
	"title": true, "site_name": true, "logo": true, "favicon": true,
	"notice": true, "register_ways": true, "enabled_register": true,
	"enabled_reward": true, "enabled_verify": true, "enabled_msg": true,
	"init_power": true, "invite_power": true, "vip_month_power": true,
	"order_pay_timeout": true, "mj_power": true, "mj_action_power": true,
	"sd_power": true, "dall_power": true, "prompt_power": true,
	"wechat_card_url": true, "enable_context": true, "context_deep": true,
	"sd_neg_prompt": true, "index_bg_url": true,
	"enabled_app_image": true, "enabled_app_music": true, "enabled_app_video": true,
	"enabled_app_ppt": true, "enabled_app_logo": true, "enabled_app_drama": true,
	"enabled_soha_market": true, "enabled_soha_kline": true,
	"enabled_register_ip_limit": true, "register_ip_limit": true,
	"enabled_activation": true, "enabled_activation_user_limit": true,
	"activation_user_limit": true, "jihuo_ex_min_power": true,
	"member_normal_min_power": true, "member_normal_max_power": true,
	"member_vip_min_power": true, "legacy_vip_min_power": true,
	"swap_power_price": true, "soha_guide_base_price": true,
	"ai_search_search_power": true, "ai_search_detail_power": true,
	"music_power": true, "teaching_script_power": true,
	"teaching_tts_power_per_segment": true,
	"transfer_fee_percent": true, "enabled_transfer_fee": true,
	"transfer_fee_to_pool_percent": true,
	"team_reward_measure_layer_depth": true, "team_reward_layer4_width_limit": true,
	"team_reward_pro": true, "team_reward_weekly_cap": true,
	"legacy_vip_team_reward_pro": true,
	"kj_zj_pro": true, "kj_team_reward_pro": true,
	"zj_static_pro": true, "mgmt_static_pro": true,
	"invest_multiplier": true, "power_price": true, "currency": true,
	"image_hub_custom_apps": true,
}

var sensitiveKeySubstr = []string{
	"api_key", "apikey", "secret", "password", "token", "access_key",
	"private_key", "smtp_pass", "web_search_api",
}

// FilterPublicConfig 对公开 /api/config/get 做字段白名单 + 敏感键剥离
func FilterPublicConfig(marker string, value map[string]interface{}) map[string]interface{} {
	if value == nil {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(value))
	for k, v := range value {
		lk := strings.ToLower(k)
		if isSensitiveConfigKey(lk) {
			continue
		}
		if marker == "system" {
			if !publicSystemConfigFields[lk] && !publicSystemConfigFields[k] {
				// 非白名单：仅放行明显非敏感的布尔/开关类短字段，其余丢弃
				if !looksSafePublicField(lk, v) {
					continue
				}
			}
		}
		out[k] = scrubPublicValue(k, v)
	}
	return out
}

func isSensitiveConfigKey(lk string) bool {
	for _, s := range sensitiveKeySubstr {
		if strings.Contains(lk, s) {
			return true
		}
	}
	return false
}

func looksSafePublicField(lk string, v interface{}) bool {
	if strings.HasPrefix(lk, "enabled_") || strings.HasSuffix(lk, "_power") ||
		strings.HasSuffix(lk, "_pro") || strings.HasSuffix(lk, "_percent") ||
		strings.HasSuffix(lk, "_limit") || strings.HasSuffix(lk, "_price") {
		return true
	}
	switch v.(type) {
	case bool, float64, int, int64:
		return true
	default:
		return false
	}
}

func scrubPublicValue(key string, v interface{}) interface{} {
	s, ok := v.(string)
	if !ok {
		return v
	}
	lk := strings.ToLower(key)
	if lk == "logo" || lk == "reward_img" || lk == "index_bg_url" || strings.HasSuffix(lk, "_url") {
		// 去掉内网/源站直连痕迹
		if strings.Contains(s, "localhost") || strings.Contains(s, "127.0.0.1") {
			return ""
		}
	}
	return v
}

// MaskSecret 仅返回是否配置 + 末 4 位
func MaskSecret(raw string) map[string]interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]interface{}{"configured": false, "hint": ""}
	}
	hint := raw
	if len(hint) > 4 {
		hint = "****" + hint[len(hint)-4:]
	} else {
		hint = "****"
	}
	return map[string]interface{}{"configured": true, "hint": hint}
}
