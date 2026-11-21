package service

import (
	"chatplus/core/types"
	"chatplus/store/model"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	transferReceiverRe = regexp.MustCompile(`转账(?:手续费)?:收款人\(([^)]+)\)`)
	transferSenderRe   = regexp.MustCompile(`收款:转账人\(([^)]+)\)`)
)

// PowerPointRollbackAffectedUser 回滚影响的单个用户
type PowerPointRollbackAffectedUser struct {
	UserId        uint    `json:"user_id"`
	Username      string  `json:"username"`
	CurrentPower  float64 `json:"current_power"`
	RestoredPower float64 `json:"restored_power"`
	HiddenLogCount int    `json:"hidden_log_count"`
}

// PowerPointRollbackPreview 按时间点回滚预览
type PowerPointRollbackPreview struct {
	UserId              uint                             `json:"user_id"`
	Username            string                           `json:"username"`
	RollbackAt          int64                            `json:"rollback_at"`
	CurrentPower        float64                          `json:"current_power"`
	RestoredPower       float64                          `json:"restored_power"`
	PrimaryHiddenCount  int                              `json:"primary_hidden_count"`
	PairedHiddenCount   int                              `json:"paired_hidden_count"`
	TotalHiddenCount    int                              `json:"total_hidden_count"`
	CategorySummary     map[string]int                   `json:"category_summary"`
	AffectedUsers       []PowerPointRollbackAffectedUser `json:"affected_users"`
	AffectedOrderCount  int                              `json:"affected_order_count"`
	OrderReleaseLogCount int                             `json:"order_release_log_count"`
	CascadeOrderCount   int                              `json:"cascade_order_count"`
}

type powerRollbackPlan struct {
	RollbackAt           time.Time
	BatchId              string
	PrimaryUserId        uint
	PrimaryHiddenCount   int
	PairedHideIds        []uint
	RecipientCascades    []recipientCascade
	HideLogIds           []uint // 兼容：仅含 PairedHideIds
	AffectedUIDs         []uint
	ByUserHide           map[uint]int
	ByUserHideIds        map[uint][]uint
	Categories           map[string]int
}

type recipientCascade struct {
	UserId   uint
	FromTime time.Time
}

func (p *powerRollbackPlan) cascadeUserIds() map[uint]struct{} {
	set := make(map[uint]struct{}, len(p.RecipientCascades))
	for _, c := range p.RecipientCascades {
		set[c.UserId] = struct{}{}
	}
	return set
}

func (p *powerRollbackPlan) cascadeFromTime(userId uint) (time.Time, bool) {
	var t time.Time
	found := false
	for _, c := range p.RecipientCascades {
		if c.UserId != userId {
			continue
		}
		if !found || c.FromTime.Before(t) {
			t = c.FromTime
			found = true
		}
	}
	return t, found
}

func (p *powerRollbackPlan) totalHiddenCount() int {
	total := p.PrimaryHiddenCount + len(p.PairedHideIds)
	for _, c := range p.RecipientCascades {
		total += p.ByUserHide[c.UserId]
	}
	return total
}

func (p *powerRollbackPlan) pairedAndCascadeHiddenCount() int {
	total := len(p.PairedHideIds)
	for _, c := range p.RecipientCascades {
		total += p.ByUserHide[c.UserId]
	}
	return total
}

// PreviewPowerPointRollback 预览：软隐藏 rollbackAt 之后的流水并重算余额
func PreviewPowerPointRollback(db *gorm.DB, userId uint, rollbackAt time.Time) (*PowerPointRollbackPreview, error) {
	var user model.User
	if err := db.Where("id = ?", userId).First(&user).Error; err != nil {
		return nil, fmt.Errorf("用户不存在")
	}

	plan, err := buildPowerRollbackPlan(db, userId, rollbackAt)
	if err != nil {
		return nil, err
	}

	preview := &PowerPointRollbackPreview{
		UserId:             userId,
		Username:           user.Username,
		RollbackAt:         rollbackAt.Unix(),
		CurrentPower:       user.Power,
		CategorySummary:    plan.Categories,
		PrimaryHiddenCount: plan.ByUserHide[userId],
		PairedHiddenCount:  plan.pairedAndCascadeHiddenCount(),
		TotalHiddenCount:   plan.totalHiddenCount(),
		AffectedUsers:      make([]PowerPointRollbackAffectedUser, 0),
	}

	for _, uid := range plan.AffectedUIDs {
		var u model.User
		if err := db.Where("id = ?", uid).First(&u).Error; err != nil {
			continue
		}
		var restored float64
		var err error
		if uid == userId {
			restored, err = powerBalanceAtRollbackPoint(db, userId, rollbackAt)
		} else if fromTime, ok := plan.cascadeFromTime(uid); ok {
			restored, err = powerBalanceBeforeStrict(db, uid, fromTime)
		} else {
			restored, err = simulatePowerAfterHidingLogsFast(db, uid, hideSetFromIds(plan.ByUserHideIds[uid]))
		}
		if err != nil {
			return nil, err
		}
		preview.AffectedUsers = append(preview.AffectedUsers, PowerPointRollbackAffectedUser{
			UserId:         uid,
			Username:       u.Username,
			CurrentPower:   u.Power,
			RestoredPower:  restored,
			HiddenLogCount: plan.ByUserHide[uid],
		})
		if uid == userId {
			preview.RestoredPower = restored
		}
	}

	orderCount, releaseLogCount, err := previewOrderRollbackImpactForPlan(db, userId, rollbackAt, plan.PairedHideIds)
	if err != nil {
		return nil, err
	}
	preview.AffectedOrderCount = orderCount
	preview.OrderReleaseLogCount = releaseLogCount

	var cascadeOrderTotal int64
	for _, c := range plan.RecipientCascades {
		var cnt int64
		db.Model(&model.Order{}).Where("user_id = ? AND created_at >= ?", c.UserId, c.FromTime).Count(&cnt)
		cascadeOrderTotal += cnt
	}
	primaryOrderCnt, err := previewPrimaryUserOrdersToHide(db, userId, rollbackAt)
	if err != nil {
		return nil, err
	}
	preview.CascadeOrderCount = int(cascadeOrderTotal) + primaryOrderCnt

	return preview, nil
}

// ExecutePowerPointRollback 执行按时间点回滚（软删除流水 + 重算余额）
func ExecutePowerPointRollback(db *gorm.DB, userId uint, rollbackAt time.Time, adminId uint) (*PowerPointRollbackPreview, error) {
	plan, err := buildPowerRollbackPlan(db, userId, rollbackAt)
	if err != nil {
		return nil, err
	}
	if plan.totalHiddenCount() == 0 {
		preview, _ := PreviewPowerPointRollback(db, userId, rollbackAt)
		if preview == nil {
			preview = &PowerPointRollbackPreview{UserId: userId, RollbackAt: rollbackAt.Unix()}
		}
		return preview, fmt.Errorf("该时间点之后没有可回滚的算力流水")
	}

	preview, err := PreviewPowerPointRollback(db, userId, rollbackAt)
	if err != nil {
		return nil, err
	}

	plan.BatchId = uuid.NewString()
	now := time.Now()

	primaryOrderNos, err := collectPrimaryUserOrderNosToHide(db, userId, rollbackAt)
	if err != nil {
		return nil, err
	}

	tx := db.Begin()
	if _, err := softDeleteUserLogsAfterTime(tx, userId, rollbackAt, plan.BatchId, now); err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := softDeletePowerLogsByIds(tx, plan.PairedHideIds, plan.BatchId, now); err != nil {
		tx.Rollback()
		return nil, err
	}
	for _, cascade := range plan.RecipientCascades {
		if _, err := softDeleteUserLogsFromTime(tx, cascade.UserId, cascade.FromTime, plan.BatchId, now); err != nil {
			tx.Rollback()
			return nil, err
		}
		if _, err := hardDeleteUserOrdersFromTime(tx, cascade.UserId, cascade.FromTime); err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	if _, err := rollbackPrimaryUserOrdersWithNos(tx, userId, rollbackAt, primaryOrderNos); err != nil {
		tx.Rollback()
		return nil, err
	}

	affectedOrderCount, err := recalculateOrdersFromRollbackBatch(tx, plan.BatchId)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	preview.AffectedOrderCount = affectedOrderCount

	cascadeUsers := plan.cascadeUserIds()
	for _, uid := range plan.AffectedUIDs {
		var finalBalance float64
		var err error
		if uid == userId {
			finalBalance, err = powerBalanceAtRollbackPoint(tx, userId, rollbackAt)
		} else if _, ok := cascadeUsers[uid]; ok {
			fromTime, _ := plan.cascadeFromTime(uid)
			finalBalance, err = powerBalanceBeforeStrict(tx, uid, fromTime)
		} else {
			finalBalance, err = RecalculateUserPowerAfterRollback(tx, uid, plan.ByUserHideIds[uid])
		}
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		if err := tx.Model(&model.User{}).Where("id = ?", uid).UpdateColumn("power", finalBalance).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		for i := range preview.AffectedUsers {
			if preview.AffectedUsers[i].UserId == uid {
				preview.AffectedUsers[i].RestoredPower = finalBalance
			}
		}
		if uid == userId {
			preview.RestoredPower = finalBalance
			if err := resetSignInStateAfterRollback(tx, uid, rollbackAt); err != nil {
				tx.Rollback()
				return nil, err
			}
		}
	}

	var user model.User
	if err := tx.Where("id = ?", userId).First(&user).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	auditRemark := fmt.Sprintf(
		"全球算力智能检测异常,算力时间点回滚：回滚至 %s，软隐藏 %d 条流水，同步重算 %d 笔订单释放字段，本用户算力 %.2f → %.2f，批次 %s",
		rollbackAt.Format("2006-01-02 15:04:05"),
		preview.TotalHiddenCount,
		preview.AffectedOrderCount,
		preview.CurrentPower,
		preview.RestoredPower,
		plan.BatchId,

	)
	if err := tx.Create(&model.PowerLog{
		UserId:    user.Id,
		Username:  user.Username,
		Type:      types.PowerGift,
		Amount:    0,
		Balance:   preview.RestoredPower,
		Mark:      types.PowerAdd,
		Model:     "管理员",
		Remark:    auditRemark,
		CreatedAt: now,
	}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return preview, nil
}

func buildPowerRollbackPlan(db *gorm.DB, userId uint, rollbackAt time.Time) (*powerRollbackPlan, error) {
	primaryCount, err := countUserLogsAfterTime(db, userId, rollbackAt)
	if err != nil {
		return nil, err
	}

	plan := &powerRollbackPlan{
		RollbackAt:         rollbackAt,
		PrimaryUserId:      userId,
		PrimaryHiddenCount: int(primaryCount),
		PairedHideIds:      make([]uint, 0),
		RecipientCascades:  make([]recipientCascade, 0),
		ByUserHide:         make(map[uint]int),
		ByUserHideIds:      make(map[uint][]uint),
		Categories:         make(map[string]int),
	}
	if primaryCount == 0 {
		return plan, nil
	}
	plan.ByUserHide[userId] = int(primaryCount)

	type modelCountRow struct {
		Model string
		Cnt   int
	}
	var modelRows []modelCountRow
	if err := ActivePowerLogScope(db).Model(&model.PowerLog{}).
		Select("model, COUNT(*) as cnt").
		Where("user_id = ? AND created_at > ?", userId, rollbackAt).
		Group("model").
		Scan(&modelRows).Error; err != nil {
		return nil, err
	}
	for _, row := range modelRows {
		plan.Categories[categorizePowerLogModel(row.Model)] += row.Cnt
	}

	// 仅加载可能涉及转账配对的流水（通常很少）
	var transferLogs []model.PowerLog
	if err := ActivePowerLogScope(db).
		Where("user_id = ? AND created_at > ?", userId, rollbackAt).
		Where("type IN ? AND mark IN ?",
			[]types.PowerType{types.PowerFk, types.PowerSk},
			[]types.PowerMark{types.PowerSub, types.PowerAdd}).
		Find(&transferLogs).Error; err != nil {
		return nil, err
	}

	pairedSet := make(map[uint]struct{})
	cascadeEarliest := make(map[uint]time.Time)

	addPaired := func(log model.PowerLog) {
		if _, ok := pairedSet[log.Id]; ok {
			return
		}
		if log.UserId == userId && log.CreatedAt.After(rollbackAt) {
			return
		}
		pairedSet[log.Id] = struct{}{}
		plan.PairedHideIds = append(plan.PairedHideIds, log.Id)
		plan.ByUserHide[log.UserId]++
		plan.ByUserHideIds[log.UserId] = append(plan.ByUserHideIds[log.UserId], log.Id)
		plan.Categories[categorizePowerLog(log)]++
	}

	addRecipientCascade := func(recvLog model.PowerLog) {
		if t, ok := cascadeEarliest[recvLog.UserId]; ok {
			if recvLog.CreatedAt.Before(t) {
				cascadeEarliest[recvLog.UserId] = recvLog.CreatedAt
			}
			return
		}
		cascadeEarliest[recvLog.UserId] = recvLog.CreatedAt
	}

	for _, log := range transferLogs {
		pairs, err := findTransferPairedLogs(db, log)
		if err != nil {
			return nil, err
		}
		for _, p := range pairs {
			if isTransferReceiveLog(p) {
				addRecipientCascade(p)
			} else {
				addPaired(p)
			}
		}
	}

	for uid, fromTime := range cascadeEarliest {
		cnt, err := countUserLogsFromTime(db, uid, fromTime)
		if err != nil {
			return nil, err
		}
		plan.RecipientCascades = append(plan.RecipientCascades, recipientCascade{
			UserId:   uid,
			FromTime: fromTime,
		})
		plan.ByUserHide[uid] = int(cnt)
		delete(plan.ByUserHideIds, uid)
	}

	if err := attachDownstreamRewardPairs(db, plan, userId, rollbackAt, addPaired); err != nil {
		return nil, err
	}

	plan.HideLogIds = plan.PairedHideIds
	affected := make(map[uint]struct{})
	for uid := range plan.ByUserHide {
		affected[uid] = struct{}{}
	}
	for uid := range affected {
		plan.AffectedUIDs = append(plan.AffectedUIDs, uid)
	}
	return plan, nil
}

func isTransferReceiveLog(log model.PowerLog) bool {
	return log.Type == types.PowerSk &&
		log.Mark == types.PowerAdd &&
		strings.Contains(log.Model, "收款:转账人")
}

func categorizePowerLogModel(modelName string) string {
	return categorizePowerLog(model.PowerLog{Model: modelName})
}

func (p *powerRollbackPlan) hideSet() map[uint]struct{} {
	set := make(map[uint]struct{}, len(p.HideLogIds))
	for _, id := range p.HideLogIds {
		set[id] = struct{}{}
	}
	return set
}

func categorizePowerLog(log model.PowerLog) string {
	modelName := strings.TrimSpace(log.Model)
	switch {
	case strings.Contains(modelName, "转账"):
		return "转账"
	case strings.Contains(modelName, "收款"):
		return "收款"
	case strings.Contains(modelName, "用户领取每日福利"):
		return "签到福利"
	case strings.Contains(modelName, "音乐生成") || strings.Contains(modelName, "AI音乐"):
		return "AI音乐"
	case strings.Contains(modelName, "团队"):
		return "团队奖励"
	case strings.Contains(modelName, "推荐") || strings.Contains(modelName, "释放"):
		return "推荐/释放"
	case strings.Contains(modelName, "算力机") || strings.Contains(modelName, "矿机"):
		return "算力机/投入"
	default:
		if modelName != "" {
			return modelName
		}
		return "其他"
	}
}

func findTransferPairedLogs(db *gorm.DB, log model.PowerLog) ([]model.PowerLog, error) {
	out := make([]model.PowerLog, 0)
	windowStart := log.CreatedAt.Add(-3 * time.Second)
	windowEnd := log.CreatedAt.Add(3 * time.Second)

	if log.Type == types.PowerFk && log.Mark == types.PowerSub {
		m := transferReceiverRe.FindStringSubmatch(log.Model)
		if len(m) < 2 {
			return out, nil
		}
		var receiver model.User
		if err := db.Where("username = ?", strings.TrimSpace(m[1])).First(&receiver).Error; err != nil {
			return out, nil
		}
		var recv model.PowerLog
		err := ActivePowerLogScope(db).
			Where("user_id = ? AND type = ? AND mark = ? AND amount = ? AND created_at BETWEEN ? AND ?",
				receiver.Id, types.PowerSk, types.PowerAdd, log.Amount, windowStart, windowEnd).
			First(&recv).Error
		if err == nil {
			out = append(out, recv)
		}
		// 同批手续费（若有）
		var feeLogs []model.PowerLog
		ActivePowerLogScope(db).
			Where("user_id = ? AND model LIKE ? AND created_at BETWEEN ? AND ?",
				log.UserId, "转账手续费:%", windowStart, windowEnd).
			Find(&feeLogs)
		out = append(out, feeLogs...)
		return out, nil
	}

	if log.Type == types.PowerSk && log.Mark == types.PowerAdd {
		m := transferSenderRe.FindStringSubmatch(log.Model)
		if len(m) < 2 {
			return out, nil
		}
		var sender model.User
		if err := db.Where("username = ?", strings.TrimSpace(m[1])).First(&sender).Error; err != nil {
			return out, nil
		}
		var send model.PowerLog
		err := ActivePowerLogScope(db).
			Where("user_id = ? AND type = ? AND mark = ? AND amount = ? AND created_at BETWEEN ? AND ?",
				sender.Id, types.PowerFk, types.PowerSub, log.Amount, windowStart, windowEnd).
			First(&send).Error
		if err == nil {
			out = append(out, send)
			var feeLogs []model.PowerLog
			ActivePowerLogScope(db).
				Where("user_id = ? AND model LIKE ? AND created_at BETWEEN ? AND ?",
					sender.Id, "转账手续费:%", windowStart, windowEnd).
				Find(&feeLogs)
			out = append(out, feeLogs...)
		}
	}
	return out, nil
}

func simulatePowerAfterHidingLogs(db *gorm.DB, userId uint, hideSet map[uint]struct{}) (float64, error) {
	return simulatePowerAfterHidingLogsFast(db, userId, hideSet)
}

func resetSignInStateAfterRollback(tx *gorm.DB, userId uint, rollbackAt time.Time) error {
	// 回滚后若今日无有效签到流水，允许重新签到
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if rollbackAt.Before(dayStart) {
		var cnt int64
		ActivePowerLogScope(tx).Model(&model.PowerLog{}).
			Where("user_id = ? AND model = ? AND created_at >= ?", userId, "用户领取每日福利", dayStart).
			Count(&cnt)
		if cnt == 0 {
			return tx.Model(&model.User{}).Where("id = ?", userId).Updates(map[string]interface{}{
				"is_partner":        0,
				"sign_in_streak":    0,
				"last_sign_in_date": nil,
			}).Error
		}
	}
	return nil
}

func ParseRollbackTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("请指定回滚时间")
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, raw, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("时间格式无效，请使用 2006-01-02 15:04:05")
}
