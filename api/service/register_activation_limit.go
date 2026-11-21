package service

import (
	"chatplus/core/types"
	"chatplus/store/model"
	"fmt"

	"gorm.io/gorm"
)

// RegisterIPLimit 同 IP 注册上限；返回 0 表示不限制
func RegisterIPLimit(cfg *types.SystemConfig) int {
	if cfg == nil || !cfg.EnabledRegisterIpLimit {
		return 0
	}
	if cfg.RegisterIpLimit > 0 {
		return cfg.RegisterIpLimit
	}
	// 显式填 0 表示不限制
	return 0
}

// CheckRegisterIPLimit 检查 IP 是否已达注册上限
func CheckRegisterIPLimit(db *gorm.DB, ip string, cfg *types.SystemConfig) error {
	limit := RegisterIPLimit(cfg)
	if limit <= 0 || ip == "" {
		return nil
	}
	var count int64
	if err := db.Model(&model.User{}).Where("login_ip = ?", ip).Count(&count).Error; err != nil {
		return err
	}
	if count >= int64(limit) {
		return fmt.Errorf("该IP地址注册账号已达上限（最多%d个）", limit)
	}
	return nil
}

// ActivationEnabled 是否开放质押/激活
func ActivationEnabled(cfg *types.SystemConfig) bool {
	if cfg == nil {
		return true
	}
	return cfg.EnabledActivation
}

// ActivationUserLimit 单用户复投上限；返回 0 表示不限制
func ActivationUserLimit(cfg *types.SystemConfig) int {
	if cfg == nil || !cfg.EnabledActivationUserLimit {
		return 0
	}
	if cfg.ActivationUserLimit > 0 {
		return cfg.ActivationUserLimit
	}
	return 0
}

// CheckActivationUserLimit 已激活用户 confirmJihuoEx 复投次数校验
func CheckActivationUserLimit(isJh int, reinvestCount int, cfg *types.SystemConfig) error {
	if isJh != 1 {
		return nil
	}
	limit := ActivationUserLimit(cfg)
	if limit <= 0 {
		return nil
	}
	if reinvestCount >= limit {
		return fmt.Errorf("激活次数已达上限，最多复投%d次", limit)
	}
	return nil
}
