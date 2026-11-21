package service

import (
	"chatplus/core/types"
	"chatplus/store/model"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

var orderAcceleratedReleaseModels = []string{
	"用户领取每日福利",
	"静态收益释放",
	"推荐静态奖励释放",
	"团队奖励释放",
	"推荐奖励释放",
	"等级分红奖励",
}

func hideSetFromIds(ids []uint) map[uint]struct{} {
	set := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

func hideIdsFromSet(set map[uint]struct{}) []uint {
	ids := make([]uint, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	return ids
}

// powerBalanceAtRollbackPoint 回滚时刻最后一条有效流水的 balance
func powerBalanceAtRollbackPoint(db *gorm.DB, userId uint, rollbackAt interface{}) (float64, error) {
	var log model.PowerLog
	err := ActivePowerLogScope(db).
		Where("user_id = ? AND created_at <= ?", userId, rollbackAt).
		Order("created_at DESC, id DESC").
		First(&log).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return log.Balance, nil
}

// powerBalanceBeforeStrict 指定时刻 strictly 之前的最后一条有效流水 balance
func powerBalanceBeforeStrict(db *gorm.DB, userId uint, before time.Time) (float64, error) {
	var log model.PowerLog
	err := ActivePowerLogScope(db).
		Where("user_id = ? AND created_at < ?", userId, before).
		Order("created_at DESC, id DESC").
		First(&log).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return log.Balance, nil
}

// isTailOnlyHide 被隐藏的流水是否都在时间线末尾（之前流水完整保留）
func isTailOnlyHide(db *gorm.DB, userId uint, hideSet map[uint]struct{}) (bool, error) {
	if len(hideSet) == 0 {
		return true, nil
	}
	hideIds := hideIdsFromSet(hideSet)
	var earliest model.PowerLog
	if err := db.Where("id IN ?", hideIds).Order("created_at ASC, id ASC").First(&earliest).Error; err != nil {
		return false, err
	}
	var cnt int64
	err := ActivePowerLogScope(db).Model(&model.PowerLog{}).
		Where("user_id = ?", userId).
		Where("id NOT IN ?", hideIds).
		Where("(created_at > ? OR (created_at = ? AND id >= ?))",
			earliest.CreatedAt, earliest.CreatedAt, earliest.Id).
		Count(&cnt).Error
	return cnt == 0, err
}

func restoredPowerForTailHide(db *gorm.DB, userId uint, hideSet map[uint]struct{}) (float64, error) {
	q := ActivePowerLogScope(db).Where("user_id = ?", userId)
	if len(hideSet) > 0 {
		q = q.Where("id NOT IN ?", hideIdsFromSet(hideSet))
	}
	var log model.PowerLog
	err := q.Order("created_at DESC, id DESC").First(&log).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return log.Balance, nil
}

func openingBalanceBeforeLog(log model.PowerLog) float64 {
	balance := log.Balance
	if log.Amount == 0 {
		return balance
	}
	if log.Mark == types.PowerAdd {
		return balance - log.Amount
	}
	return balance + log.Amount
}

func replayPowerFromEarliestHidden(db *gorm.DB, userId uint, hideSet map[uint]struct{}) (float64, error) {
	hideIds := hideIdsFromSet(hideSet)
	var earliest model.PowerLog
	if err := db.Where("id IN ?", hideIds).Order("created_at ASC, id ASC").First(&earliest).Error; err != nil {
		return 0, err
	}

	balance := 0.0
	var prev model.PowerLog
	err := ActivePowerLogScope(db).
		Where("user_id = ? AND (created_at < ? OR (created_at = ? AND id < ?))",
			userId, earliest.CreatedAt, earliest.CreatedAt, earliest.Id).
		Order("created_at DESC, id DESC").
		First(&prev).Error
	switch err {
	case nil:
		balance = prev.Balance
	case gorm.ErrRecordNotFound:
		balance = openingBalanceBeforeLog(earliest)
	default:
		return 0, err
	}

	var logs []model.PowerLog
	if err := ActivePowerLogScope(db).
		Where("user_id = ? AND (created_at > ? OR (created_at = ? AND id >= ?))",
			userId, earliest.CreatedAt, earliest.CreatedAt, earliest.Id).
		Where("id NOT IN ?", hideIds).
		Order("created_at ASC, id ASC").
		Find(&logs).Error; err != nil {
		return 0, err
	}

	for i := range logs {
		if logs[i].Amount == 0 {
			logs[i].Balance = balance
		} else if logs[i].Mark == types.PowerAdd {
			balance += logs[i].Amount
			logs[i].Balance = balance
		} else {
			balance -= logs[i].Amount
			logs[i].Balance = balance
		}
	}
	return balance, nil
}

func simulatePowerAfterHidingLogsFast(db *gorm.DB, userId uint, hideSet map[uint]struct{}) (float64, error) {
	if len(hideSet) == 0 {
		var user model.User
		if err := db.Where("id = ?", userId).First(&user).Error; err != nil {
			return 0, err
		}
		return user.Power, nil
	}
	tail, err := isTailOnlyHide(db, userId, hideSet)
	if err != nil {
		return 0, err
	}
	if tail {
		return restoredPowerForTailHide(db, userId, hideSet)
	}
	return replayPowerFromEarliestHidden(db, userId, hideSet)
}

func batchUpdatePowerLogBalances(tx *gorm.DB, logs []model.PowerLog) error {
	const batchSize = 500
	for i := 0; i < len(logs); i += batchSize {
		end := i + batchSize
		if end > len(logs) {
			end = len(logs)
		}
		batch := logs[i:end]
		caseSQL := strings.Builder{}
		args := make([]interface{}, 0, len(batch)*2+len(batch))
		ids := make([]uint, 0, len(batch))
		for j, log := range batch {
			if j > 0 {
				caseSQL.WriteString(" ")
			}
			caseSQL.WriteString("WHEN ? THEN ?")
			args = append(args, log.Id, log.Balance)
			ids = append(ids, log.Id)
		}
		inPlaceholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
		for _, id := range ids {
			args = append(args, id)
		}
		sql := fmt.Sprintf(
			"UPDATE chatgpt_power_logs SET balance = CASE id %s END WHERE id IN (%s)",
			caseSQL.String(), inPlaceholders,
		)
		if err := tx.Exec(sql, args...).Error; err != nil {
			return err
		}
	}
	return nil
}

// RecalculateUserPowerAfterRollback 回滚后重算用户算力（优先尾部删除快路径，避免全表扫描）
func RecalculateUserPowerAfterRollback(tx *gorm.DB, userId uint, hideIdsForUser []uint) (float64, error) {
	hideSet := hideSetFromIds(hideIdsForUser)
	tail, err := isTailOnlyHide(tx, userId, hideSet)
	if err != nil {
		return 0, err
	}
	if tail {
		return restoredPowerForTailHide(tx, userId, hideSet)
	}

	hideIds := hideIdsFromSet(hideSet)
	var earliest model.PowerLog
	if err := tx.Where("id IN ?", hideIds).Order("created_at ASC, id ASC").First(&earliest).Error; err != nil {
		return 0, err
	}

	balance := 0.0
	var prev model.PowerLog
	err = ActivePowerLogScope(tx).
		Where("user_id = ? AND (created_at < ? OR (created_at = ? AND id < ?))",
			userId, earliest.CreatedAt, earliest.CreatedAt, earliest.Id).
		Order("created_at DESC, id DESC").
		First(&prev).Error
	switch err {
	case nil:
		balance = prev.Balance
	case gorm.ErrRecordNotFound:
		balance = openingBalanceBeforeLog(earliest)
	default:
		return 0, err
	}

	var logs []model.PowerLog
	if err := ActivePowerLogScope(tx).
		Where("user_id = ? AND (created_at > ? OR (created_at = ? AND id >= ?))",
			userId, earliest.CreatedAt, earliest.CreatedAt, earliest.Id).
		Where("id NOT IN ?", hideIds).
		Order("created_at ASC, id ASC").
		Find(&logs).Error; err != nil {
		return 0, err
	}

	for i := range logs {
		if logs[i].Amount == 0 {
			logs[i].Balance = balance
		} else if logs[i].Mark == types.PowerAdd {
			balance += logs[i].Amount
			logs[i].Balance = balance
		} else {
			balance -= logs[i].Amount
			logs[i].Balance = balance
		}
	}
	if len(logs) > 0 {
		if err := batchUpdatePowerLogBalances(tx, logs); err != nil {
			return 0, err
		}
	}
	return balance, nil
}
