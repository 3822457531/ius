package service

import (
	"chatplus/store/model"
	"strings"

	"gorm.io/gorm"
)

// AppConsumeRewardHandler 应用广场算力成功消耗后的团队分红（逻辑同 confirmJihuoEx）
type AppConsumeRewardHandler interface {
	OnAppPlazaPowerConsumed(userId uint, power float64, sourceName string)
}

var appConsumeRewardHandler AppConsumeRewardHandler

func RegisterAppConsumeRewardHandler(h AppConsumeRewardHandler) {
	appConsumeRewardHandler = h
}

// NotifyAppPlazaPowerConsumed 在算力已确定消耗、不会退回时触发分红（应用广场 / 对话等，逻辑同 confirmJihuoEx）
func NotifyAppPlazaPowerConsumed(db *gorm.DB, userId uint, power float64, sourceName, sourceKey string) {
	if db == nil || appConsumeRewardHandler == nil {
		return
	}
	if userId == 0 || power <= 0 {
		return
	}
	sourceKey = strings.TrimSpace(sourceKey)
	if sourceKey == "" {
		return
	}
	sourceName = strings.TrimSpace(sourceName)
	if sourceName == "" {
		sourceName = "应用消费"
	}

	row := model.ConsumeRewardLog{
		SourceKey:  sourceKey,
		UserId:     userId,
		Power:      power,
		SourceName: sourceName,
	}
	if err := db.Create(&row).Error; err != nil {
		return
	}
	go appConsumeRewardHandler.OnAppPlazaPowerConsumed(userId, power, sourceName)
}
