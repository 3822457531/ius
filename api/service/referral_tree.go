package service

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// 推荐伞下关系判定
// 推荐树由 chatgpt_users.zj_fid（直接上级）串成长链。
// 与安置树（chatgpt_placement_nodes 左右区）是两棵不同的树：
// 滑落会导致「安置子树里的人不在推荐伞下」，旁部门成员属于这种情况。

// maxReferralDepth 向上追溯的深度上限，仅用于防御脏数据成环，不表达业务层级限制
const maxReferralDepth = 1024

type referralLink struct {
	Id    uint
	ZjFid int
}

// IsInReferralSubtree 判断 userID 是否位于 ancestorID 的推荐伞下（zj_fid 链上的任意深度后代）。
// 不含本人；ancestorID == userID 返回 false。
// 向上追溯只需走一条链，比向下展开整棵树便宜，且天然覆盖任意深度。
func IsInReferralSubtree(db *gorm.DB, ancestorID, userID uint) (bool, error) {
	if db == nil {
		return false, fmt.Errorf("参数错误")
	}
	if ancestorID == 0 || userID == 0 || ancestorID == userID {
		return false, nil
	}

	current := userID
	for depth := 0; depth < maxReferralDepth; depth++ {
		var link referralLink
		tx := db.Table("chatgpt_users").
			Select("id, zj_fid").
			Where("id = ?", current).
			Take(&link)
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		if tx.Error != nil {
			return false, tx.Error
		}
		if link.ZjFid <= 0 {
			return false, nil
		}
		if uint(link.ZjFid) == ancestorID {
			return true, nil
		}
		current = uint(link.ZjFid)
	}
	return false, nil
}