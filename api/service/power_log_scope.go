package service

import (
	"chatplus/store/model"
	"time"

	"gorm.io/gorm"
)

// ActivePowerLogScope 仅查询未软删除的算力流水
func ActivePowerLogScope(db *gorm.DB) *gorm.DB {
	return db.Where("deleted_at IS NULL")
}

func activePowerLogsQuery(db *gorm.DB, userId uint) *gorm.DB {
	q := ActivePowerLogScope(db).Where("user_id = ?", userId)
	return q.Order("created_at ASC, id ASC")
}

func loadActivePowerLogs(db *gorm.DB, userId uint) ([]model.PowerLog, error) {
	var logs []model.PowerLog
	err := activePowerLogsQuery(db, userId).Find(&logs).Error
	return logs, err
}

func softDeletePowerLogsByIds(tx *gorm.DB, ids []uint, batchId string, now time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	const batchSize = 500
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		if err := tx.Model(&model.PowerLog{}).
			Where("id IN ? AND deleted_at IS NULL", ids[i:end]).
			Updates(map[string]interface{}{
				"deleted_at":        now,
				"rollback_batch_id": batchId,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

// softDeleteUserLogsAfterTime 批量软隐藏某用户指定时间点之后的流水（避免百万条载入内存）
func softDeleteUserLogsAfterTime(tx *gorm.DB, userId uint, after time.Time, batchId string, now time.Time) (int64, error) {
	res := ActivePowerLogScope(tx).Model(&model.PowerLog{}).
		Where("user_id = ? AND created_at > ?", userId, after).
		Updates(map[string]interface{}{
			"deleted_at":        now,
			"rollback_batch_id": batchId,
		})
	return res.RowsAffected, res.Error
}

func countUserLogsAfterTime(db *gorm.DB, userId uint, after time.Time) (int64, error) {
	var cnt int64
	err := ActivePowerLogScope(db).Model(&model.PowerLog{}).
		Where("user_id = ? AND created_at > ?", userId, after).
		Count(&cnt).Error
	return cnt, err
}

func countUserLogsFromTime(db *gorm.DB, userId uint, from time.Time) (int64, error) {
	var cnt int64
	err := ActivePowerLogScope(db).Model(&model.PowerLog{}).
		Where("user_id = ? AND created_at >= ?", userId, from).
		Count(&cnt).Error
	return cnt, err
}

// softDeleteUserLogsFromTime 软隐藏某用户从指定时刻起（含）的全部流水
func softDeleteUserLogsFromTime(tx *gorm.DB, userId uint, from time.Time, batchId string, now time.Time) (int64, error) {
	res := ActivePowerLogScope(tx).Model(&model.PowerLog{}).
		Where("user_id = ? AND created_at >= ?", userId, from).
		Updates(map[string]interface{}{
			"deleted_at":        now,
			"rollback_batch_id": batchId,
		})
	return res.RowsAffected, res.Error
}

// hardDeleteUserOrdersFromTime 物理删除某用户从指定时刻起（含）创建的质押订单
func hardDeleteUserOrdersFromTime(tx *gorm.DB, userId uint, from time.Time) (int64, error) {
	res := tx.Unscoped().Where("user_id = ? AND created_at >= ?", userId, from).Delete(&model.Order{})
	return res.RowsAffected, res.Error
}

// hardDeleteUserOrdersAfterTime 物理删除某用户指定时刻之后创建的订单（与主流水回滚边界一致）
func hardDeleteUserOrdersAfterTime(tx *gorm.DB, userId uint, after time.Time) (int64, error) {
	res := tx.Unscoped().Where("user_id = ? AND created_at > ?", userId, after).Delete(&model.Order{})
	return res.RowsAffected, res.Error
}

// hardDeleteOrdersByOrderNos 按订单号物理删除
func hardDeleteOrdersByOrderNos(tx *gorm.DB, orderNos []string) (int64, error) {
	if len(orderNos) == 0 {
		return 0, nil
	}
	res := tx.Unscoped().Where("order_no IN ?", orderNos).Delete(&model.Order{})
	return res.RowsAffected, res.Error
}

func countUserOrdersAfterTime(db *gorm.DB, userId uint, after time.Time) (int64, error) {
	var cnt int64
	err := db.Model(&model.Order{}).Where("user_id = ? AND created_at > ?", userId, after).Count(&cnt).Error
	return cnt, err
}
