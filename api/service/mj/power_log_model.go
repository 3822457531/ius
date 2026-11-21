package mj

import (
	"chatplus/service"
	"strings"
)

const LogoPromptMarker = "[AI-LOGO]"

// LogoPowerLogModel Logo 生图在算力流水中的 model 名
const LogoPowerLogModel = service.PowerLogModelLogo

// ResolveImagePowerLogModel 根据提示词区分 Logo 与普通生图，算力日志统一展示名。
func ResolveImagePowerLogModel(prompt, model string) string {
	_ = model
	if strings.Contains(prompt, LogoPromptMarker) {
		return LogoPowerLogModel
	}
	return service.PowerLogModelImage
}
