package handler

import (
	"chatplus/core/types"
	"chatplus/store/model"

	"gorm.io/gorm"
)

// SohaMarketGloballyEnabled 全局 SOHA 交易市场是否开放
func SohaMarketGloballyEnabled(cfg *types.SystemConfig) bool {
	return cfg != nil && cfg.EnabledSohaMarket
}

// SohaKlineDisplayEnabled 是否向用户展示 K 线（默认开启）
func SohaKlineDisplayEnabled(cfg *types.SystemConfig) bool {
	if cfg == nil {
		return true
	}
	return cfg.EnabledSohaKline
}

// CanOperateSohaMarket 全局开放且用户被授权时方可操作（Swap、C2C 等；算力机另见 CanBuyMint）
func CanOperateSohaMarket(cfg *types.SystemConfig, user *model.User) (ok bool, reason string) {
	if !SohaMarketGloballyEnabled(cfg) {
		return false, "SOHA交易市场暂未开放"
	}
	if user == nil || !user.AllowQ2 {
		return false, "无SOHA交易权限"
	}
	return true, ""
}

// CanBuyMint 是否允许购买/激活算力机（以 allow_mint 为准；管理员可关闭为「已售罄」；不依赖 allow_q2）
func CanBuyMint(db *gorm.DB, cfg *types.SystemConfig, user *model.User) (ok bool, reason string) {
	if user == nil {
		return false, "无效用户"
	}
	if !user.AllowMint {
		return false, "已抢购完毕"
	}
	return true, ""
}
