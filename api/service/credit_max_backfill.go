package service

import (
	"chatplus/core/types"
	"chatplus/store/model"
	"chatplus/utils"
)

// 授信保底回填（一次性维护任务，幂等）
//
// 背景：新规则把「单笔 < 1万」的历史投入从基数里剔除，上线那一次重算把一批老用户的
// credit_max 直接写成 0（流水里就是「最大额度减少 50000.000000 → 0.000000」）。
// 保底机制（CreditMaxFloor）本身只在 RecalcMax 时生效，而这些用户往往不再下单，
// 于是会长期停在 0 —— 除非有人主动替他们触发一次重算。
//
// 这里做的就是「主动触发一次」：
//
//	候选集 = credit_max_floor = 0 且 有授信历史痕迹（旧 Max / 可用 / 待还 / 有过 MaxDown 流水）
//	动作   = 逐个走 RecalcMax → 首次 bootstrap 写入旧口径 Floor，并把 Max 恢复回去
//
// 幂等：处理过的用户 floor > 0，下次不会再命中；全新用户不在候选集里，不会凭空拿到额度。
const creditFloorBackfillBatch = 200

// BackfillCreditMaxFloor 扫描并补写历史保底，返回实际重算的用户数。
// cfg 传 nil 时从 configs 表读取当前 system 配置（与定时任务同一读法）。
func (s *CreditService) BackfillCreditMaxFloor(cfg *types.SystemConfig) (int, error) {
	if cfg == nil {
		cfg = s.systemConfig()
	}

	done := 0
	var cursor uint
	for {
		ids, err := s.pendingFloorUserIds(cursor, creditFloorBackfillBatch)
		if err != nil {
			return done, err
		}
		if len(ids) == 0 {
			break
		}
		for _, id := range ids {
			// 单个用户失败不阻断整体：脏订单/脏流水只影响他自己那一行
			if err := s.RecalcMax(nil, id, cfg); err != nil {
				logger.Errorf("授信保底回填失败 user_id=%d: %v", id, err)
				continue
			}
			done++
		}
		cursor = ids[len(ids)-1]
		if len(ids) < creditFloorBackfillBatch {
			break
		}
	}
	if done > 0 {
		logger.Infof("授信保底回填完成，重算 %d 个用户的历史额度", done)
	}
	return done, nil
}

// pendingFloorUserIds 取下一批「有授信历史但还没写 Floor」的 user_id（按 id 递增游标分页）。
func (s *CreditService) pendingFloorUserIds(afterId uint, limit int) ([]uint, error) {
	// 出现过「最大额度减少」流水的用户：被新规则打成 0 的现场证据
	downUsers := s.db.Model(&model.PowerLog{}).
		Select("user_id").
		Where("model = ? AND deleted_at IS NULL", CreditLogModelMaxDown)

	var ids []uint
	err := s.db.Model(&model.User{}).
		Where("id > ?", afterId).
		Where("credit_max_floor = 0").
		Where(
			s.db.Where("credit_max > 0").
				Or("credit_available > 0").
				Or("credit_debt > 0").
				Or("id IN (?)", downUsers),
		).
		Order("id").
		Limit(limit).
		Pluck("id", &ids).Error
	return ids, err
}

// systemConfig 读取 system 配置；缺行或坏 JSON 时返回空配置，由各 xxxOrDefault 兜底默认值。
func (s *CreditService) systemConfig() *types.SystemConfig {
	var row model.Config
	if err := s.db.Where("marker", "system").First(&row).Error; err != nil {
		return &types.SystemConfig{}
	}
	var cfg types.SystemConfig
	if err := utils.JsonDecode(row.Config, &cfg); err != nil {
		return &types.SystemConfig{}
	}
	return &cfg
}