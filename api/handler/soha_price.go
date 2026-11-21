package handler

import (
	"chatplus/core/types"
	"math"
	"math/rand"
	"time"
)

const (
	sohaDailyChangeMin = 0.10 // 每日最小涨幅 %
	sohaDailyChangeMax = 1.99// 每日最大涨幅 %
	sohaPriceDecimals    = 6   // 与 chatgpt_price_histories.price decimal(20,6) 一致
)

// SohaGuideBasePrice 首日 SOHA 指导价基准（默认 0.1）
func SohaGuideBasePrice(cfg *types.SystemConfig) float64 {
	if cfg != nil && cfg.SohaGuideBasePrice > 0 {
		return cfg.SohaGuideBasePrice
	}
	return 0.1
}

// RandomSohaDailyChangePercent 每日随机涨幅 [0.10%, 0.50%]，保留两位小数
func RandomSohaDailyChangePercent() float64 {
	p := sohaDailyChangeMin + rand.Float64()*(sohaDailyChangeMax-sohaDailyChangeMin)
	return math.Round(p*100) / 100
}

// RoundSohaPrice 与数据库 decimal(20,6) 对齐
func RoundSohaPrice(price float64) float64 {
	scale := math.Pow(10, float64(sohaPriceDecimals))
	return math.Round(price*scale) / scale
}

// SohaPriceMinTick 最小价格变动单位
func SohaPriceMinTick() float64 {
	return 1 / math.Pow(10, float64(sohaPriceDecimals))
}

// IsLegacySohaPrice 历史数据曾被缩放为 0.0001 等极小价，与当前配置基准价差距过大
func IsLegacySohaPrice(price float64, cfg *types.SystemConfig) bool {
	base := SohaGuideBasePrice(cfg)
	if base <= 0 || price <= 0 {
		return false
	}
	return price < base*0.01
}

// EffectiveSohaPreviousPrice 复利计算用的有效前一日收盘价（legacy 价改用配置基准价）
func EffectiveSohaPreviousPrice(lastPrice float64, cfg *types.SystemConfig) float64 {
	if IsLegacySohaPrice(lastPrice, cfg) {
		return SohaGuideBasePrice(cfg)
	}
	return lastPrice
}

// ApplySohaDailyChange 在前价基础上按涨幅计算新价；保证涨幅>0 时 price 字段必定变大
func ApplySohaDailyChange(previousPrice float64, changePercent float64) float64 {
	if previousPrice <= 0 {
		return RoundSohaPrice(SohaGuideBasePrice(nil))
	}
	if changePercent <= 0 {
		return RoundSohaPrice(previousPrice)
	}
	next := previousPrice * (1 + changePercent/100)
	next = RoundSohaPrice(next)
	tick := SohaPriceMinTick()
	if next <= previousPrice {
		next = RoundSohaPrice(previousPrice + tick)
	}
	return next
}

// SohaChangeRateFromPrices 相对前一日收盘价的实际涨跌幅（%），保留两位小数
func SohaChangeRateFromPrices(previousPrice, currentPrice float64) float64 {
	if previousPrice <= 0 || currentPrice <= 0 {
		return 0
	}
	rate := (currentPrice - previousPrice) / previousPrice * 100
	return math.Round(rate*100) / 100
}

// TodayDateLocal 当日 0 点（本地时区）
func TodayDateLocal() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}
