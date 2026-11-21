package service

import (
	"chatplus/store/model"
	"fmt"
	"math"

	"gorm.io/gorm"
)

// 复投档位（硬编码）：按单笔投入算力区间限制该档订单次数
// 1000～4999 / 5000～9999 / 1万～49999 / 5万～99999：各最多 10 次
// 10 万及以上：不限次数

type reinvestTier struct {
	Min   float64
	Max   float64 // 含上界；Max < 0 表示无上界
	Limit int     // 0 = 不限
	Label string
}

var reinvestTiers = []reinvestTier{
	{Min: 1000, Max: 4999, Limit: 10, Label: "1000～4999"},
	{Min: 5000, Max: 9999, Limit: 10, Label: "5000～9999"},
	{Min: 10000, Max: 49999, Limit: 10, Label: "1万～49999"},
	{Min: 50000, Max: 99999, Limit: 10, Label: "5万～99999"},
	{Min: 100000, Max: -1, Limit: 0, Label: "10万及以上"},
}

func matchReinvestTier(amount float64) (reinvestTier, bool) {
	for _, t := range reinvestTiers {
		if amount < t.Min {
			continue
		}
		if t.Max < 0 || amount <= t.Max {
			return t, true
		}
	}
	return reinvestTier{}, false
}

// CheckReinvestTierLimit 按投入档位校验复投次数（已有同档订单数 ≥ 上限则拒绝）
func CheckReinvestTierLimit(db *gorm.DB, userID uint, amount float64) error {
	if db == nil || userID == 0 {
		return fmt.Errorf("参数错误")
	}
	amount = math.Round(amount*1e6) / 1e6
	tier, ok := matchReinvestTier(amount)
	if !ok {
		return fmt.Errorf("投入算力不在可复投档位（最低1000）")
	}
	if tier.Limit <= 0 {
		return nil
	}

	q := db.Model(&model.Order{}).Where("user_id = ? AND sales >= ?", userID, tier.Min)
	if tier.Max >= 0 {
		q = q.Where("sales <= ?", tier.Max)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return err
	}
	if count >= int64(tier.Limit) {
		return fmt.Errorf("该档位（%s算力）复投已达上限，最多%d次", tier.Label, tier.Limit)
	}
	return nil
}
