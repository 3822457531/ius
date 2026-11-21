package service

import (
	"chatplus/core/types"
	"chatplus/store/model"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

const musicFailedRefundRemarkPrefix = "音乐生成任务失败，退回算力"

var musicTaskIDRe = regexp.MustCompile(`任务ID[：:]\s*(\S+)`)

// MusicRefundDuplicateGroup 同一任务 ID 的重复退款
type MusicRefundDuplicateGroup struct {
	TaskId         string  `json:"task_id"`
	RefundCount    int     `json:"refund_count"`
	KeepLogId      uint    `json:"keep_log_id"`
	RemoveLogCount int     `json:"remove_log_count"`
	ExcessAmount   float64 `json:"excess_amount"`
}

// MusicRefundRollbackPreview 音乐失败重复退款回滚预览
type MusicRefundRollbackPreview struct {
	UserId            uint                        `json:"user_id"`
	Username          string                      `json:"username"`
	CurrentPower      float64                     `json:"current_power"`
	ExcessPower       float64                     `json:"excess_power"`
	CorrectedPower    float64                     `json:"corrected_power"`
	DuplicateLogCount int                         `json:"duplicate_log_count"`
	TaskGroups        []MusicRefundDuplicateGroup `json:"task_groups"`
}

// FindMusicRefundDuplicates 找出音乐失败重复退款日志（每个任务 ID 保留最早一条）
func FindMusicRefundDuplicates(db *gorm.DB, userId uint) (*MusicRefundRollbackPreview, []uint, error) {
	var user model.User
	if err := db.Where("id = ?", userId).First(&user).Error; err != nil {
		return nil, nil, fmt.Errorf("用户不存在")
	}

	var logs []model.PowerLog
	err := db.Where("user_id = ?", userId).
		Where("remark LIKE ?", musicFailedRefundRemarkPrefix+"%").
		Where("mark = ?", types.PowerAdd).
		Where("deleted_at IS NULL").
		Order("id ASC").
		Find(&logs).Error
	if err != nil {
		return nil, nil, err
	}

	grouped := make(map[string][]model.PowerLog)
	order := make([]string, 0)
	for _, log := range logs {
		taskId := parseMusicRefundTaskID(log.Remark)
		if taskId == "" {
			continue
		}
		if _, ok := grouped[taskId]; !ok {
			order = append(order, taskId)
		}
		grouped[taskId] = append(grouped[taskId], log)
	}

	preview := &MusicRefundRollbackPreview{
		UserId:         userId,
		Username:       user.Username,
		CurrentPower:   user.Power,
		CorrectedPower: user.Power,
		TaskGroups:     make([]MusicRefundDuplicateGroup, 0),
	}
	removeIds := make([]uint, 0)

	for _, taskId := range order {
		items := grouped[taskId]
		if len(items) <= 1 {
			continue
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Id < items[j].Id })

		var excess float64
		taskRemoveIds := make([]uint, 0, len(items)-1)
		for i := 1; i < len(items); i++ {
			excess += items[i].Amount
			taskRemoveIds = append(taskRemoveIds, items[i].Id)
		}
		removeIds = append(removeIds, taskRemoveIds...)

		preview.TaskGroups = append(preview.TaskGroups, MusicRefundDuplicateGroup{
			TaskId:         taskId,
			RefundCount:    len(items),
			KeepLogId:      items[0].Id,
			RemoveLogCount: len(taskRemoveIds),
			ExcessAmount:   excess,
		})
		preview.ExcessPower += excess
		preview.DuplicateLogCount += len(taskRemoveIds)
	}

	if len(removeIds) > 0 {
		corrected, err := simulatePowerAfterRemovingLogs(db, userId, removeIds)
		if err != nil {
			return nil, nil, err
		}
		preview.CorrectedPower = corrected
	}

	return preview, removeIds, nil
}

// RollbackMusicRefundDuplicates 删除重复退款日志并重算余额
func RollbackMusicRefundDuplicates(db *gorm.DB, userId uint, adminId uint) (*MusicRefundRollbackPreview, error) {
	preview, removeIds, err := FindMusicRefundDuplicates(db, userId)
	if err != nil {
		return nil, err
	}
	if len(removeIds) == 0 {
		return preview, fmt.Errorf("该用户没有可回滚的重复退款记录")
	}

	beforePower := preview.CurrentPower
	batchId := fmt.Sprintf("music-refund-%d", time.Now().UnixNano())
	now := time.Now()

	tx := db.Begin()
	if err := softDeletePowerLogsByIds(tx, removeIds, batchId, now); err != nil {
		tx.Rollback()
		return nil, err
	}

	finalBalance, err := RecalculateUserPowerLogBalances(tx, userId)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Model(&model.User{}).Where("id = ?", userId).UpdateColumn("power", finalBalance).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	var user model.User
	if err := tx.Where("id = ?", userId).First(&user).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	auditRemark := fmt.Sprintf(
		"管理员回滚音乐失败重复退款：删除%d条重复日志，扣回算力%.2f，回滚前%.2f，回滚后%.2f，管理员ID：%d",
		len(removeIds), preview.ExcessPower, beforePower, finalBalance, adminId,
	)
	if err := tx.Create(&model.PowerLog{
		UserId:    user.Id,
		Username:  user.Username,
		Type:      types.PowerGift,
		Amount:    0,
		Balance:   finalBalance,
		Mark:      types.PowerAdd,
		Model:     "管理员",
		Remark:    auditRemark,
	}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	preview.CorrectedPower = finalBalance
	return preview, nil
}

func simulatePowerAfterRemovingLogs(db *gorm.DB, userId uint, removeIds []uint) (float64, error) {
	return simulatePowerAfterHidingLogs(db, userId, idsToSet(removeIds))
}

func idsToSet(ids []uint) map[uint]struct{} {
	set := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return set
}

// RecalculateUserPowerLogBalances 按时间顺序重算用户有效算力流水余额
func RecalculateUserPowerLogBalances(db *gorm.DB, userId uint) (float64, error) {
	logs, err := loadActivePowerLogs(db, userId)
	if err != nil {
		return 0, err
	}
	if len(logs) == 0 {
		var user model.User
		if err := db.Where("id = ?", userId).First(&user).Error; err != nil {
			return 0, err
		}
		return user.Power, nil
	}

	balance, err := replayPowerLogBalances(logs)
	if err != nil {
		return 0, err
	}

	if len(logs) > 0 {
		if err := batchUpdatePowerLogBalances(db, logs); err != nil {
			return 0, err
		}
	}
	return balance, nil
}

func replayPowerLogBalances(logs []model.PowerLog) (float64, error) {
	if len(logs) == 0 {
		return 0, nil
	}

	balance := logs[0].Balance
	if logs[0].Mark == types.PowerAdd {
		balance -= logs[0].Amount
	} else {
		balance += logs[0].Amount
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

func parseMusicRefundTaskID(remark string) string {
	m := musicTaskIDRe.FindStringSubmatch(remark)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}
