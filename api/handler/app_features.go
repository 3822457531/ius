package handler

import "chatplus/core/types"

const AppMaintenanceMsg = "该应用暂时维护中"

type AppFeature string

const (
	AppFeatureImage AppFeature = "image"
	AppFeatureMusic AppFeature = "music"
	AppFeatureVideo AppFeature = "video"
	AppFeaturePpt   AppFeature = "ppt"
	AppFeatureLogo  AppFeature = "logo"
	AppFeatureDrama AppFeature = "drama"
)

func defaultAppFeatureEnabled(feature AppFeature) bool {
	switch feature {
	case AppFeatureVideo:
		return false
	default:
		return true
	}
}

func AppFeatureEnabled(cfg *types.SystemConfig, feature AppFeature) bool {
	if cfg == nil {
		return defaultAppFeatureEnabled(feature)
	}
	switch feature {
	case AppFeatureImage:
		return cfg.EnabledAppImage
	case AppFeatureMusic:
		return cfg.EnabledAppMusic
	case AppFeatureVideo:
		return cfg.EnabledAppVideo
	case AppFeaturePpt:
		return cfg.EnabledAppPpt
	case AppFeatureLogo:
		return cfg.EnabledAppLogo
	case AppFeatureDrama:
		return cfg.EnabledAppDrama
	default:
		return true
	}
}

func DenyIfAppDisabled(cfg *types.SystemConfig, feature AppFeature) (deny bool, msg string) {
	if AppFeatureEnabled(cfg, feature) {
		return false, ""
	}
	return true, AppMaintenanceMsg
}
