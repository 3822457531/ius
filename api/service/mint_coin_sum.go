package service

import (
	"chatplus/core/types"
	"chatplus/store/model"

	"gorm.io/gorm"
)

func sumCoinAmount(db *gorm.DB, sql string, args ...interface{}) float64 {
	var sum float64
	_ = db.Raw(sql, args...).Scan(&sum).Error
	return sum
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// SumMintProducedCoin 按产出流水汇总矿机已产出 SOHA（与明细页同源）
// 取 coin_logs / q2_logs（含历史无 order_no）合计的较大值，避免部分回填导致累计偏小
func SumMintProducedCoin(db *gorm.DB, userId uint, orderNo string) float64 {
	if db == nil || userId == 0 || orderNo == "" {
		return 0
	}

	sum := sumCoinAmount(db, `
		SELECT COALESCE(SUM(amount), 0) FROM chatgpt_coin_logs
		WHERE user_id = ? AND order_no = ? AND mark = ?
	`, userId, orderNo, types.PowerAdd)

	sum = maxFloat(sum, sumCoinAmount(db, `
		SELECT COALESCE(SUM(amount), 0) FROM chatgpt_q2_logs
		WHERE user_id = ? AND order_no = ? AND mark = ?
	`, userId, orderNo, types.PowerAdd))

	var runningCount int64
	db.Model(&model.MintOrder{}).
		Where("user_id = ? AND status = ?", userId, types.OrderZyz).
		Count(&runningCount)
	if runningCount == 1 {
		sum = maxFloat(sum, sumCoinAmount(db, `
			SELECT COALESCE(SUM(amount), 0) FROM chatgpt_q2_logs
			WHERE user_id = ? AND mark = ? AND model = 'SOHA产出'
			  AND (order_no = ? OR order_no = '' OR order_no IS NULL)
		`, userId, types.PowerAdd, orderNo))
	}

	return sum
}

// SyncMintOrderJsCoin 用流水合计校正 mint_order.js_coin（解决历史漏记导致列表「累计产出」偏小）
func SyncMintOrderJsCoin(db *gorm.DB, userId uint, orderNo string, current float64) float64 {
	sum := SumMintProducedCoin(db, userId, orderNo)
	if sum <= 0 {
		return current
	}
	if sum > current+0.0001 {
		_ = db.Model(&model.MintOrder{}).
			Where("user_id = ? AND order_no = ?", userId, orderNo).
			UpdateColumn("js_coin", sum).Error
		return sum
	}
	if current > 0 {
		return current
	}
	return sum
}
