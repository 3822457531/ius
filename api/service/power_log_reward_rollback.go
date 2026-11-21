package service

import (
	"chatplus/core/types"
	"chatplus/store/model"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

const downstreamRewardPairWindow = 10 * time.Minute

var downstreamRewardModels = []string{
	"推荐奖励释放",
	"团队奖励释放",
	"推荐静态奖励释放",
	"推荐奖静态释放",
	"管理奖静态释放",
}

// isDownstreamTriggerLog 下级投入/质押类流水，可能触发上级推荐奖或团队奖
func isDownstreamTriggerLog(log model.PowerLog) bool {
	if log.Mark != types.PowerSub || log.Amount <= 0 {
		return false
	}
	modelName := strings.TrimSpace(log.Model)
	remark := strings.TrimSpace(log.Remark)
	if strings.Contains(modelName, "购买算力产品") || strings.Contains(remark, "用户质押") {
		return true
	}
	switch log.Type {
	case types.PowerRecharge, types.PowerZy:
		return true
	default:
		return false
	}
}

// findDownstreamRewardPairedLogs 查找因下级用户 triggerLog 触发的上级推荐奖/团队奖流水
func findDownstreamRewardPairedLogs(db *gorm.DB, triggerLog model.PowerLog, downlineUsername string) ([]model.PowerLog, error) {
	downlineUsername = strings.TrimSpace(downlineUsername)
	if downlineUsername == "" {
		return nil, nil
	}

	windowEnd := triggerLog.CreatedAt.Add(downstreamRewardPairWindow)
	usernameLike := fmt.Sprintf("%%用户名：%s%%", downlineUsername)
	fromUserLike := fmt.Sprintf("%%来自用户 %s%%", downlineUsername)
	nameLike := fmt.Sprintf("%%%s%%", downlineUsername)

	var logs []model.PowerLog
	err := ActivePowerLogScope(db).
		Where("mark = ? AND amount > 0", types.PowerAdd).
		Where("user_id <> ?", triggerLog.UserId).
		Where("created_at >= ? AND created_at <= ?", triggerLog.CreatedAt, windowEnd).
		Where("(remark LIKE ? OR remark LIKE ? OR remark LIKE ?)", usernameLike, fromUserLike, nameLike).
		Where("(model IN ? OR model LIKE ? OR model LIKE ? OR model LIKE ?)",
			downstreamRewardModels, "%推荐%", "%团队%", "%释放%").
		Order("created_at ASC, id ASC").
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func attachDownstreamRewardPairs(db *gorm.DB, plan *powerRollbackPlan, userId uint, rollbackAt time.Time, addPaired func(model.PowerLog)) error {
	var primaryUser model.User
	if err := db.Where("id = ?", userId).First(&primaryUser).Error; err != nil {
		return fmt.Errorf("用户不存在")
	}

	var triggerLogs []model.PowerLog
	if err := ActivePowerLogScope(db).
		Where("user_id = ? AND created_at > ?", userId, rollbackAt).
		Where("mark = ?", types.PowerSub).
		Where("model LIKE ? OR remark LIKE ?", "%购买算力产品%", "%用户质押%").
		Order("created_at ASC, id ASC").
		Find(&triggerLogs).Error; err != nil {
		return err
	}

	for _, log := range triggerLogs {
		if !isDownstreamTriggerLog(log) {
			continue
		}
		rewards, err := findDownstreamRewardPairedLogs(db, log, primaryUser.Username)
		if err != nil {
			return err
		}
		for _, reward := range rewards {
			addPaired(reward)
		}
	}
	return nil
}
