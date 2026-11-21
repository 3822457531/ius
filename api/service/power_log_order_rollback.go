package service

import (
	"chatplus/core/types"
	"chatplus/store/model"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// isOrderAcceleratedReleaseLog 是否会同步增加 js_power、减少 now_power 的加速释放流水
func isOrderAcceleratedReleaseLog(log model.PowerLog) bool {
	if log.Mark != types.PowerAdd || log.Amount <= 0 {
		return false
	}
	orderNo := strings.TrimSpace(log.OrderNo)
	if orderNo == "" {
		return false
	}
	modelName := strings.TrimSpace(log.Model)
	switch modelName {
	case "用户领取每日福利",
		"静态收益释放",
		"推荐静态奖励释放",
		"团队奖励释放",
		"推荐奖励释放",
		"等级分红奖励":
		return true
	}
	switch log.Type {
	case types.PowerSf, types.PowerJT:
		return true
	default:
		return false
	}
}

func collectOrderNosFromReleaseLogs(logs []model.PowerLog) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, log := range logs {
		if !isOrderAcceleratedReleaseLog(log) {
			continue
		}
		orderNo := strings.TrimSpace(log.OrderNo)
		if orderNo == "" {
			continue
		}
		if _, ok := seen[orderNo]; ok {
			continue
		}
		seen[orderNo] = struct{}{}
		out = append(out, orderNo)
	}
	return out
}

func countOrderReleaseLogs(logs []model.PowerLog) int {
	n := 0
	for _, log := range logs {
		if isOrderAcceleratedReleaseLog(log) {
			n++
		}
	}
	return n
}

func loadPowerLogsByIds(db *gorm.DB, ids []uint) ([]model.PowerLog, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var logs []model.PowerLog
	err := db.Where("id IN ?", ids).Find(&logs).Error
	return logs, err
}

// RecalculateOrderReleaseFromActiveLogs 按有效加速释放流水重算订单 js_power / now_power / status
func RecalculateOrderReleaseFromActiveLogs(tx *gorm.DB, orderNo string) error {
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return nil
	}

	var order model.Order
	if err := tx.Where("order_no = ?", orderNo).First(&order).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	var jsPower float64
	err := ActivePowerLogScope(tx).Model(&model.PowerLog{}).
		Where("order_no = ? AND mark = ? AND amount > 0", orderNo, types.PowerAdd).
		Where(
			"model IN ? OR type IN ?",
			orderAcceleratedReleaseModels,
			[]types.PowerType{types.PowerSf, types.PowerJT},
		).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&jsPower).Error
	if err != nil {
		return err
	}
	if jsPower > order.Power {
		jsPower = order.Power
	}
	if jsPower < 0 {
		jsPower = 0
	}

	nowPower := order.Power - jsPower
	if nowPower < 0 {
		nowPower = 0
	}

	updates := map[string]interface{}{
		"js_power":  jsPower,
		"now_power": nowPower,
	}
	if order.Status == types.OrderZyz || order.Status == types.OrderYjs {
		if nowPower <= 0 || jsPower >= order.Power {
			updates["status"] = types.OrderYjs
		} else {
			updates["status"] = types.OrderZyz
		}
	}

	return tx.Model(&model.Order{}).Where("order_no = ?", orderNo).Updates(updates).Error
}

func recalculateOrdersFromHiddenLogs(tx *gorm.DB, hideLogIds []uint) (int, error) {
	logs, err := loadPowerLogsByIds(tx, hideLogIds)
	if err != nil {
		return 0, err
	}
	orderNos := collectOrderNosFromReleaseLogs(logs)
	for _, orderNo := range orderNos {
		if err := RecalculateOrderReleaseFromActiveLogs(tx, orderNo); err != nil {
			return 0, fmt.Errorf("重算订单 %s 失败: %w", orderNo, err)
		}
	}
	return len(orderNos), nil
}

func recalculateOrdersFromRollbackBatch(tx *gorm.DB, batchId string) (int, error) {
	var orderNos []string
	if err := tx.Model(&model.PowerLog{}).
		Where("rollback_batch_id = ? AND order_no <> ''", batchId).
		Distinct().
		Pluck("order_no", &orderNos).Error; err != nil {
		return 0, err
	}
	for _, orderNo := range orderNos {
		if err := RecalculateOrderReleaseFromActiveLogs(tx, orderNo); err != nil {
			return 0, fmt.Errorf("重算订单 %s 失败: %w", orderNo, err)
		}
	}
	return len(orderNos), nil
}

func previewOrderRollbackImpactForPlan(db *gorm.DB, userId uint, rollbackAt time.Time, pairedIds []uint) (orderCount int, releaseLogCount int, err error) {
	type orderRow struct {
		OrderNo string
		Cnt     int
	}
	var rows []orderRow

	q := ActivePowerLogScope(db).Model(&model.PowerLog{}).
		Select("order_no, COUNT(*) as cnt").
		Where("order_no <> '' AND mark = ?", types.PowerAdd).
		Where(
			"(model IN ? OR type IN ?)",
			orderAcceleratedReleaseModels,
			[]types.PowerType{types.PowerSf, types.PowerJT, types.PowerGift},
		)
	if len(pairedIds) > 0 {
		q = q.Where("(user_id = ? AND created_at > ?) OR id IN ?", userId, rollbackAt, pairedIds)
	} else {
		q = q.Where("user_id = ? AND created_at > ?", userId, rollbackAt)
	}

	if err := q.Group("order_no").Scan(&rows).Error; err != nil {
		return 0, 0, err
	}
	releaseLogCount = 0
	for _, row := range rows {
		if strings.TrimSpace(row.OrderNo) != "" {
			orderCount++
			releaseLogCount += row.Cnt
		}
	}
	return orderCount, releaseLogCount, nil
}

func previewOrderRollbackImpact(db *gorm.DB, hideLogIds []uint) (orderCount int, releaseLogCount int, err error) {
	logs, err := loadPowerLogsByIds(db, hideLogIds)
	if err != nil {
		return 0, 0, err
	}
	return len(collectOrderNosFromReleaseLogs(logs)), countOrderReleaseLogs(logs), nil
}
