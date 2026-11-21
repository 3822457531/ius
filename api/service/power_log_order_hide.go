package service

import (
	"chatplus/store/model"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

var orderNoInRemarkRe = regexp.MustCompile(`订单号[：:]\s*(\S+)`)

func extractOrderNosFromPowerLogs(logs []model.PowerLog) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, log := range logs {
		for _, m := range orderNoInRemarkRe.FindAllStringSubmatch(log.Remark, -1) {
			if len(m) < 2 {
				continue
			}
			no := strings.TrimSpace(m[1])
			if no == "" {
				continue
			}
			if _, ok := seen[no]; ok {
				continue
			}
			seen[no] = struct{}{}
			out = append(out, no)
		}
	}
	return out
}

func collectPrimaryUserOrderNosToHide(db *gorm.DB, userId uint, after time.Time) ([]string, error) {
	var orderNos []string
	if err := db.Model(&model.Order{}).
		Where("user_id = ? AND created_at > ?", userId, after).
		Pluck("order_no", &orderNos).Error; err != nil {
		return nil, err
	}

	var logs []model.PowerLog
	if err := ActivePowerLogScope(db).
		Where("user_id = ? AND created_at > ?", userId, after).
		Where("remark LIKE ?", "%订单号%").
		Find(&logs).Error; err != nil {
		return nil, err
	}
	orderNos = append(orderNos, extractOrderNosFromPowerLogs(logs)...)

	seen := make(map[string]struct{}, len(orderNos))
	unique := make([]string, 0, len(orderNos))
	for _, no := range orderNos {
		no = strings.TrimSpace(no)
		if no == "" {
			continue
		}
		if _, ok := seen[no]; ok {
			continue
		}
		seen[no] = struct{}{}
		unique = append(unique, no)
	}
	return unique, nil
}

func rollbackPrimaryUserOrders(tx *gorm.DB, userId uint, after time.Time) (int64, error) {
	orderNos, err := collectPrimaryUserOrderNosToHide(tx, userId, after)
	if err != nil {
		return 0, err
	}
	return rollbackPrimaryUserOrdersWithNos(tx, userId, after, orderNos)
}

func rollbackPrimaryUserOrdersWithNos(tx *gorm.DB, userId uint, after time.Time, orderNos []string) (int64, error) {
	var total int64
	n, err := hardDeleteUserOrdersAfterTime(tx, userId, after)
	if err != nil {
		return 0, err
	}
	total += n

	n2, err := hardDeleteOrdersByOrderNos(tx, orderNos)
	total += n2
	return total, err
}

func previewPrimaryUserOrdersToHide(db *gorm.DB, userId uint, after time.Time) (int, error) {
	cnt, err := countUserOrdersAfterTime(db, userId, after)
	if err != nil {
		return 0, err
	}
	orderNos, err := collectPrimaryUserOrderNosToHide(db, userId, after)
	if err != nil {
		return 0, err
	}
	if len(orderNos) > int(cnt) {
		return len(orderNos), nil
	}
	return int(cnt), nil
}
