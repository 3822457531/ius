package service

import (
	"chatplus/core/types"
	"chatplus/store/model"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type MemberTier int

const (
	MemberTierInactive MemberTier = iota
	MemberTierNormal
	MemberTierVip
	MemberTierLegacyVip
)

func (t MemberTier) String() string {
	switch t {
	case MemberTierNormal:
		return "normal"
	case MemberTierVip:
		return "vip"
	case MemberTierLegacyVip:
		return "legacy_vip"
	default:
		return "inactive"
	}
}

func MemberTierLabel(t MemberTier) string {
	switch t {
	case MemberTierNormal:
		return "普通会员"
	case MemberTierVip, MemberTierLegacyVip:
		return "VIP会员"
	default:
		return "未激活"
	}
}

func memberTierDefaults(cfg *types.SystemConfig) (normalMin, normalMax, vipMin, legacyMin float64) {
	normalMin, normalMax, vipMin, legacyMin = 1000, 9999, 10000, 5000
	if cfg == nil {
		return
	}
	if cfg.MemberNormalMinPower > 0 {
		normalMin = cfg.MemberNormalMinPower
	}
	if cfg.MemberNormalMaxPower > 0 {
		normalMax = cfg.MemberNormalMaxPower
	}
	if cfg.MemberVipMinPower > 0 {
		vipMin = cfg.MemberVipMinPower
	}
	if cfg.LegacyVipMinPower > 0 {
		legacyMin = cfg.LegacyVipMinPower
	}
	return
}

// MemberTierThresholds 导出档位阈值（供 handler 复用）
func MemberTierThresholds(cfg *types.SystemConfig) (normalMin, normalMax, vipMin, legacyMin float64) {
	return memberTierDefaults(cfg)
}

// ResolveMemberTier 在已知 maxSales 时解析档位，避免重复扫订单；legacyCheck 仅在需要时查询
func ResolveMemberTier(user *model.User, cfg *types.SystemConfig, maxSales float64, legacyCheck func() bool) MemberTier {
	if user == nil || user.IsJh != 1 {
		return MemberTierInactive
	}
	normalMin, normalMax, vipMin, legacyMin := memberTierDefaults(cfg)

	_, cutoffOK := ParseTeamRewardNormalCutoff("")
	if cfg != nil {
		_, cutoffOK = ParseTeamRewardNormalCutoff(cfg.TeamRewardNormalCutoff)
	}

	if cutoffOK {
		if maxSales >= vipMin {
			return MemberTierVip
		}
		if maxSales >= legacyMin && maxSales < vipMin {
			if legacyCheck != nil && legacyCheck() {
				return MemberTierLegacyVip
			}
		}
	} else {
		if maxSales >= legacyMin && maxSales < vipMin {
			return MemberTierLegacyVip
		}
	}

	if maxSales >= vipMin {
		return MemberTierVip
	}
	if maxSales >= normalMin && maxSales <= normalMax {
		return MemberTierNormal
	}
	if maxSales > 0 {
		return MemberTierNormal
	}
	return MemberTierNormal
}

// ParseTeamRewardNormalCutoff 解析「普通会员小区奖励截止时间」
func ParseTeamRewardNormalCutoff(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, time.Local); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// GetMaxActiveOrderSales 用户在投订单（status=8）中单笔最大 sales
func GetMaxActiveOrderSales(db *gorm.DB, userID int) float64 {
	var maxSales float64
	db.Model(&model.Order{}).
		Where("user_id = ? AND status = ?", userID, types.OrderZyz).
		Select("COALESCE(MAX(sales), 0)").
		Scan(&maxSales)
	return maxSales
}

// HasLegacyVipOrderBeforeCutoff 是否存在 sales>=legacyMin 且 created_at 严格早于截止的订单
func HasLegacyVipOrderBeforeCutoff(db *gorm.DB, userID int, legacyMin float64, cutoff time.Time) bool {
	var count int64
	db.Unscoped().Model(&model.Order{}).
		Where("user_id = ? AND sales >= ? AND created_at < ?", userID, legacyMin, cutoff).
		Count(&count)
	return count > 0
}

// GetMemberTier 计算用户会员档位
func GetMemberTier(db *gorm.DB, user *model.User, cfg *types.SystemConfig) (MemberTier, float64) {
	if user == nil || user.IsJh != 1 {
		return MemberTierInactive, 0
	}
	_, _, _, legacyMin := memberTierDefaults(cfg)
	maxSales := GetMaxActiveOrderSales(db, int(user.Id))
	tier := ResolveMemberTier(user, cfg, maxSales, func() bool {
		cutoff, ok := ParseTeamRewardNormalCutoff("")
		if cfg != nil {
			cutoff, ok = ParseTeamRewardNormalCutoff(cfg.TeamRewardNormalCutoff)
		}
		if !ok {
			return false
		}
		return HasLegacyVipOrderBeforeCutoff(db, int(user.Id), legacyMin, cutoff)
	})
	return tier, maxSales
}

// TeamRewardEligible 收奖人是否可领取小区团队奖
func TeamRewardEligible(db *gorm.DB, user *model.User, cfg *types.SystemConfig) bool {
	if user == nil || user.IsJh != 1 {
		return false
	}
	tier, _ := GetMemberTier(db, user, cfg)
	switch tier {
	case MemberTierVip, MemberTierLegacyVip:
		return true
	case MemberTierNormal:
		if cfg == nil || strings.TrimSpace(cfg.TeamRewardNormalCutoff) == "" {
			return false
		}
		cutoff, ok := ParseTeamRewardNormalCutoff(cfg.TeamRewardNormalCutoff)
		if !ok {
			return false
		}
		_, _, _, legacyMin := memberTierDefaults(cfg)
		return HasLegacyVipOrderBeforeCutoff(db, int(user.Id), legacyMin, cutoff)
	default:
		return false
	}
}

// TeamRewardScaleForReceiver 收奖人团队奖励比例；-1 表示不发。
// defaultScale 由调用方传入（通常来自 team_reward_pro）；0 表示关闭，不再回退成 0.2。
func TeamRewardScaleForReceiver(db *gorm.DB, user *model.User, cfg *types.SystemConfig, defaultScale float64) float64 {
	if !TeamRewardEligible(db, user, cfg) {
		return -1
	}
	tier, _ := GetMemberTier(db, user, cfg)
	if tier == MemberTierLegacyVip {
		if cfg != nil && cfg.LegacyVipTeamRewardPro > 0 {
			return cfg.LegacyVipTeamRewardPro
		}
		return 0.1
	}
	if defaultScale < 0 {
		return 0.2
	}
	return defaultScale
}

// ValidateInvestPower 校验 confirmJihuoEx 单笔投入（只卡最低额，不再设中间禁投空隙）
func ValidateInvestPower(power float64, cfg *types.SystemConfig) error {
	normalMin, _, _, _ := memberTierDefaults(cfg)
	minPower := normalMin
	if cfg != nil && cfg.JihuoExMinPower > 0 {
		minPower = cfg.JihuoExMinPower
	}
	if power < minPower {
		return fmt.Errorf("最低投入算力为%g", minPower)
	}
	return nil
}

// EnableMintPurchaseForUser 开通用户算力机抢购（写库）
func EnableMintPurchaseForUser(db *gorm.DB, userID uint) {
	if db == nil || userID == 0 {
		return
	}
	_ = db.Model(&model.User{}).Where("id = ? AND allow_mint = ?", userID, false).Update("allow_mint", true).Error
}
