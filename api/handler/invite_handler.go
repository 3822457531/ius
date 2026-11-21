package handler

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/service"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils"
	"chatplus/utils/resp"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InviteHandler 用户邀请
type InviteHandler struct {
	BaseHandler
}

func NewInviteHandler(app *core.AppServer, db *gorm.DB) *InviteHandler {
	return &InviteHandler{BaseHandler: BaseHandler{App: app, DB: db}}
}

// Code 获取当前团队
func (h *InviteHandler) Code(c *gin.Context) {
	userId := h.GetLoginUserId(c)
	var user model.User
	_ = h.DB.Select("id", "is_jh").First(&user, userId)

	var inviteCode model.InviteCode
	res := h.DB.Where("user_id = ?", userId).First(&inviteCode)
	// 如果邀请码不存在，则创建一个
	if res.Error != nil {
		code := strings.ToUpper(utils.RandString(8))
		for {
			res = h.DB.Where("code = ?", code).First(&inviteCode)
			if res.Error != nil { // 不存在相同的邀请码则退出
				break
			}
			code = strings.ToUpper(utils.RandString(8))
		}
		inviteCode.UserId = userId
		inviteCode.Code = code
		h.DB.Create(&inviteCode)
	}

	var codeVo vo.InviteCode
	err := utils.CopyObject(inviteCode, &codeVo)
	if err != nil {
		resp.ERROR(c, "拷贝对象失败")
		return
	}
	codeVo.IsJh = user.IsJh

	//获取间接推广的人数
	//;
	// var total int64
	//h.DB.Raw("SELECT * FROM chatgpt_users WHERE zj_fid=? OR jj_fid=? OR san_fid=? OR si_fid=? OR fids_json LIKE ?", userId, userId, userId, userId, fmt.Sprintf("%%%d%%", userId)).Count(&total)
	//
	////	h.DB.Raw("SELECT * FROM chatgpt_users WHERE zj_fid=? OR jj_fid=? OR san_fid=? OR si_fid=? OR fids_json LIKE ?", userId, userId, userId, userId, fmt.Sprintf("%%%s%%", userId)).Count(&total)
	//logger.Info("数量为", total)

	// 使用 COUNT(*) 来减少不必要的数据处理
	//	query := `
	//    SELECT COUNT(*)
	//    FROM chatgpt_users
	//    WHERE zj_fid = ?
	//    OR jj_fid = ?
	//    OR san_fid = ?
	//    OR si_fid = ?
	//    OR fids_json LIKE ?
	//`
	//
	//	// 确保 userId 被格式化为字符串，以适应 LIKE 查询
	//	likeQuery := fmt.Sprintf("%%%d%%", userId)
	//
	//	// 执行查询并直接计数
	//	h.DB.Raw(query, userId, userId, userId, userId, likeQuery).Scan(&total)
	//
	//	logger.Info("数量为", total)

	// likeQuery := fmt.Sprintf("[%%,%d,%%]", userId)   // 匹配逗号分隔的 ID
	// likeQueryStart := fmt.Sprintf("[%%,%d]", userId) // 匹配开头的 ID
	// likeQueryEnd := fmt.Sprintf("[%d,%%]", userId)   // 匹配结尾的 ID
	// exactMatch := fmt.Sprintf("[%d]", userId)        // 精确匹配唯一 ID

	// 	query := `
	//     SELECT COUNT(*)
	//     FROM chatgpt_users
	//     WHERE zj_fid = ?
	//     OR jj_fid = ?
	//     OR san_fid = ?
	//     OR si_fid = ?
	//     OR fids_json LIKE ?
	//     OR fids_json LIKE ?
	//     OR fids_json LIKE ?
	//     OR fids_json = ?
	// `
	// 	// 执行查询并获取计数
	// 	h.DB.Raw(query, userId, userId, userId, userId, likeQuery, likeQueryStart, likeQueryEnd, exactMatch).Scan(&total)
	// 	logger.Info("数量为", total)

	// === 新团队逻辑：基于安置树 ===
	teamIDs, err := h.getPlacementTeamMemberIDs(int(userId))
	if err != nil {
		log.Println("获取安置树团队成员失败:", err)
		resp.ERROR(c, "获取团队数据失败")
		return
	}

	//获取激活总数
	var jhTotal int64
	result := h.DB.Raw("SELECT COUNT(*) FROM chatgpt_users WHERE zj_fid = ? AND is_jh = 1", userId).Scan(&jhTotal)

	if result.Error != nil {
		log.Println("Error:", result.Error)
	} else {
		log.Println("激活总数:", jhTotal)
	}

	// 团队总人数（不包含自己）
	codeVo.Hits = len(teamIDs)

	//获取团队人数
	//codeVo.Hits = int(total)
	codeVo.JhNum = jhTotal
	resp.SUCCESS(c, codeVo)

}

// List Log 用户邀请记录
func (h *InviteHandler) List(c *gin.Context) {

	var data struct {
		Page         int    `json:"page"`
		PageSize     int    `json:"page_size"`
		SortBy       string `json:"sort_by"`        // 排序字段: "yeji" 或空字符串（默认）
		ParentUserId uint   `json:"parent_user_id"` // 查询某个直推用户的直推列表
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	userId := h.GetLoginUserId(c)
	targetInviterId := userId
	if data.ParentUserId > 0 {
		var relCount int64
		h.DB.Model(&model.InviteLog{}).
			Where("inviter_id = ? AND user_id = ?", userId, data.ParentUserId).
			Count(&relCount)
		if relCount == 0 {
			resp.ERROR(c, "无权查看该用户下级")
			return
		}
		targetInviterId = data.ParentUserId
	}
	session := h.DB.Session(&gorm.Session{}).Where("inviter_id = ?", targetInviterId)

	// 获取用户信息用于预检查
	user, err := h.GetLoginUser(c)
	if err != nil {
		logger.Info("用户未登录")
		c.Abort()
		return
	}
	logger.Info("zjfid", user.ZjFid)

	//往上查上级用户测试用
	upUserIds := getUpIds(h.DB, uint(user.ZjFid))
	logger.Info(fmt.Sprintf("寻根问祖的id为(len=%d): %+v", len(upUserIds), upUserIds))

	var total int64
	session.Model(&model.InviteLog{}).Count(&total)
	var items []model.InviteLog
	var list = make([]vo.InviteLog, 0)

	// 如果需要按业绩排序，需要先获取所有数据，计算业绩后再排序分页
	if data.SortBy == "yeji" {
		// 获取所有数据
		res := session.Order("id DESC").Find(&items)
		if res.Error == nil {
			// 创建钻石等级缓存，避免重复计算
			diamondLevelCache := make(map[int]int)

			// 批量预加载所有用户的 PlacementNode 数据
			userIds := make([]int, 0, len(items))
			for _, item := range items {
				userIds = append(userIds, int(item.UserId))
			}

			// 预加载所有相关的 PlacementNode
			var nodes []PlacementNode
			if len(userIds) > 0 {
				h.DB.Where("user_id IN ?", userIds).Find(&nodes)
			}
			nodeMap := make(map[int]PlacementNode)
			for _, node := range nodes {
				nodeMap[node.UserID] = node
			}

			for _, item := range items {
				var v vo.InviteLog
				err := utils.CopyObject(item, &v)
				if err == nil {
					var nextModel model.InviteCode
					h.DB.Model(&model.InviteCode{}).Where("user_id = ?", item.UserId).Find(&nextModel)

					var userModel model.User
					h.DB.Model(&model.User{}).Where("id = ?", item.UserId).Find(&userModel)

					v.Id = item.Id
					v.CreatedAt = item.CreatedAt.Unix()
					v.NextCode = nextModel.Code
					v.IsJh = userModel.IsJh //激活状态

					// 获取钻石等级（使用缓存）
					v.DiamondLevel = h.getDiamondLevelWithCache(int(item.UserId), diamondLevelCache, nodeMap)
					tier, _ := service.GetMemberTier(h.DB, &userModel, h.App.SysConfig)
					v.MemberTier = tier.String()
					v.MemberTierLabel = service.MemberTierLabel(tier)

					// 业绩统计
					subUserIds := getTenLevelUserIds(h.DB, item.UserId)
					// 移除日志输出以减少性能开销
					// logger.Info(fmt.Sprintf("subUserIds(len=%d): %+v", len(subUserIds), subUserIds))

					var subSalesSum float64
					var subPowerSum float64
					if len(subUserIds) > 0 {
						h.DB.Table("chatgpt_orders").Where("user_id IN ?", subUserIds).Select("IFNULL(SUM(sales),0)").Scan(&subSalesSum)
						h.DB.Table("chatgpt_mint_orders").Where("user_id IN ?", subUserIds).Select("IFNULL(SUM(power),0)").Scan(&subPowerSum)
					}
					// 直推用户自己的套餐一和套餐二
					var selfSales float64
					h.DB.Table("chatgpt_orders").Where("user_id = ?", item.UserId).Select("IFNULL(SUM(sales),0)").Scan(&selfSales)
					var selfPower float64
					h.DB.Table("chatgpt_mint_orders").Where("user_id = ?", item.UserId).Select("IFNULL(SUM(power),0)").Scan(&selfPower)

					v.Yeji = subSalesSum + subPowerSum + selfSales + selfPower

					list = append(list, v)
				} else {
					logger.Error(err)
				}
			}

			// 按业绩排序（降序）
			sort.Slice(list, func(i, j int) bool {
				return list[i].Yeji > list[j].Yeji
			})

			// 分页处理
			offset := (data.Page - 1) * data.PageSize
			end := offset + data.PageSize
			if end > len(list) {
				end = len(list)
			}
			if offset < len(list) {
				list = list[offset:end]
			} else {
				list = make([]vo.InviteLog, 0)
			}
		}
	} else {
		// 默认排序：按 id DESC 排序并分页
		offset := (data.Page - 1) * data.PageSize
		res := session.Order("id DESC").Offset(offset).Limit(data.PageSize).Find(&items)
		if res.Error == nil {
			// 创建钻石等级缓存，避免重复计算
			diamondLevelCache := make(map[int]int)

			// 批量预加载所有用户的 PlacementNode 数据
			userIds := make([]int, 0, len(items))
			for _, item := range items {
				userIds = append(userIds, int(item.UserId))
			}

			// 预加载所有相关的 PlacementNode
			var nodes []PlacementNode
			if len(userIds) > 0 {
				h.DB.Where("user_id IN ?", userIds).Find(&nodes)
			}
			nodeMap := make(map[int]PlacementNode)
			for _, node := range nodes {
				nodeMap[node.UserID] = node
			}

			for _, item := range items {
				var v vo.InviteLog
				err := utils.CopyObject(item, &v)
				if err == nil {
					var nextModel model.InviteCode
					h.DB.Model(&model.InviteCode{}).Where("user_id = ?", item.UserId).Find(&nextModel)

					var userModel model.User
					h.DB.Model(&model.User{}).Where("id = ?", item.UserId).Find(&userModel)

					v.Id = item.Id
					v.CreatedAt = item.CreatedAt.Unix()
					v.NextCode = nextModel.Code
					v.IsJh = userModel.IsJh //激活状态

					// 获取钻石等级（使用缓存）
					v.DiamondLevel = h.getDiamondLevelWithCache(int(item.UserId), diamondLevelCache, nodeMap)
					tier, _ := service.GetMemberTier(h.DB, &userModel, h.App.SysConfig)
					v.MemberTier = tier.String()
					v.MemberTierLabel = service.MemberTierLabel(tier)

					// 业绩统计
					subUserIds := getTenLevelUserIds(h.DB, item.UserId)
					// 移除日志输出以减少性能开销
					// logger.Info(fmt.Sprintf("subUserIds(len=%d): %+v", len(subUserIds), subUserIds))

					var subSalesSum float64
					var subPowerSum float64
					if len(subUserIds) > 0 {
						h.DB.Table("chatgpt_orders").Where("user_id IN ?", subUserIds).Select("IFNULL(SUM(sales),0)").Scan(&subSalesSum)
						h.DB.Table("chatgpt_mint_orders").Where("user_id IN ?", subUserIds).Select("IFNULL(SUM(power),0)").Scan(&subPowerSum)
					}
					// 直推用户自己的套餐一和套餐二
					var selfSales float64
					h.DB.Table("chatgpt_orders").Where("user_id = ?", item.UserId).Select("IFNULL(SUM(sales),0)").Scan(&selfSales)
					var selfPower float64
					h.DB.Table("chatgpt_mint_orders").Where("user_id = ?", item.UserId).Select("IFNULL(SUM(power),0)").Scan(&selfPower)

					v.Yeji = subSalesSum + subPowerSum + selfSales + selfPower

					list = append(list, v)
				} else {
					logger.Error(err)
				}
			}
		}
	}
	resp.SUCCESS(c, vo.NewPage(total, data.Page, data.PageSize, list))
}

// Hits 访问邀请码
func (h *InviteHandler) Hits(c *gin.Context) {
	code := c.Query("code")
	h.DB.Model(&model.InviteCode{}).Where("code = ?", code).UpdateColumn("hits", gorm.Expr("hits + ?", 1))
	resp.SUCCESS(c)
}

// getDiamondLevel 获取用户的钻石等级（0表示未激活，1-9表示1-9钻）
func (h *InviteHandler) getDiamondLevel(userID int) int {
	// 使用空缓存调用带缓存的方法
	return h.getDiamondLevelWithCache(userID, make(map[int]int), make(map[int]PlacementNode))
}

// getDiamondLevelWithCache 获取用户的钻石等级（带缓存优化）
func (h *InviteHandler) getDiamondLevelWithCache(userID int, cache map[int]int, nodeMap map[int]PlacementNode) int {
	// 检查缓存
	if level, ok := cache[userID]; ok {
		return level
	}

	// 按照优先级从高到低判断：9钻 -> 8钻 -> ... -> 1钻
	var level int
	if h.isDiamond9WithCache(userID, cache, nodeMap) {
		level = 9
	} else if h.isDiamond8WithCache(userID, cache, nodeMap) {
		level = 8
	} else if h.isDiamond7WithCache(userID, cache, nodeMap) {
		level = 7
	} else if h.isDiamond6WithCache(userID, cache, nodeMap) {
		level = 6
	} else if h.isDiamond5WithCache(userID, cache, nodeMap) {
		level = 5
	} else if h.isDiamond4WithCache(userID, cache, nodeMap) {
		level = 4
	} else if h.isDiamond3WithCache(userID, cache, nodeMap) {
		level = 3
	} else if h.isDiamond2WithCache(userID, cache, nodeMap) {
		level = 2
	} else if h.isDiamond1WithCache(userID, nodeMap) {
		level = 1
	} else {
		level = 0
	}

	// 存入缓存
	cache[userID] = level
	return level
}

// isDiamond1 判断用户是否是1钻（小区业绩 >= 100万算力）
func (h *InviteHandler) isDiamond1(userID int) bool {
	return h.isDiamond1WithCache(userID, make(map[int]PlacementNode))
}

// isDiamond1WithCache 判断用户是否是1钻（带缓存）
func (h *InviteHandler) isDiamond1WithCache(userID int, nodeMap map[int]PlacementNode) bool {
	var node PlacementNode
	var ok bool
	if node, ok = nodeMap[userID]; !ok {
		if err := h.DB.Where("user_id = ?", userID).First(&node).Error; err != nil {
			return false
		}
		nodeMap[userID] = node
	}

	// 计算小区业绩（左右区中较小的那个）
	minPerf := node.LeftPerf
	if node.RightPerf < minPerf {
		minPerf = node.RightPerf
	}

	// 1钻条件：小区业绩 >= 100万算力
	return minPerf >= 1000000.0
}

// isDiamond2 判断用户是否是2钻（有2个1钻直接推下级）
func (h *InviteHandler) isDiamond2(userID int) bool {
	return h.isDiamond2WithCache(userID, make(map[int]int), make(map[int]PlacementNode))
}

// isDiamond2WithCache 判断用户是否是2钻（带缓存）
func (h *InviteHandler) isDiamond2WithCache(userID int, cache map[int]int, nodeMap map[int]PlacementNode) bool {
	// 查询所有直推下级（zj_fid = userID）
	var directReferrals []model.User
	if err := h.DB.Where("zj_fid = ?", userID).Find(&directReferrals).Error; err != nil {
		return false
	}

	// 统计直推下级中1钻的数量
	diamond1Count := 0
	for _, referral := range directReferrals {
		if h.isDiamond1WithCache(int(referral.Id), nodeMap) {
			diamond1Count++
		}
	}

	// 2钻条件：有2个1钻直推下级
	return diamond1Count >= 2
}

// isDiamond3 判断用户是否是3钻（有2个2钻直接推下级）
func (h *InviteHandler) isDiamond3(userID int) bool {
	return h.isDiamond3WithCache(userID, make(map[int]int), make(map[int]PlacementNode))
}

// isDiamond3WithCache 判断用户是否是3钻（带缓存）
func (h *InviteHandler) isDiamond3WithCache(userID int, cache map[int]int, nodeMap map[int]PlacementNode) bool {
	// 查询所有直推下级（zj_fid = userID）
	var directReferrals []model.User
	if err := h.DB.Where("zj_fid = ?", userID).Find(&directReferrals).Error; err != nil {
		return false
	}

	// 统计直推下级中2钻的数量
	diamond2Count := 0
	for _, referral := range directReferrals {
		if h.isDiamond2WithCache(int(referral.Id), cache, nodeMap) {
			diamond2Count++
		}
	}

	// 3钻条件：有2个2钻直推下级
	return diamond2Count >= 2
}

// isDiamond4 判断用户是否是4钻（有3个3钻直推下级）
func (h *InviteHandler) isDiamond4(userID int) bool {
	return h.isDiamond4WithCache(userID, make(map[int]int), make(map[int]PlacementNode))
}

// isDiamond4WithCache 判断用户是否是4钻（带缓存）
func (h *InviteHandler) isDiamond4WithCache(userID int, cache map[int]int, nodeMap map[int]PlacementNode) bool {
	// 查询所有直推下级（zj_fid = userID）
	var directReferrals []model.User
	if err := h.DB.Where("zj_fid = ?", userID).Find(&directReferrals).Error; err != nil {
		return false
	}

	// 统计直推下级中3钻的数量
	diamond3Count := 0
	for _, referral := range directReferrals {
		if h.isDiamond3WithCache(int(referral.Id), cache, nodeMap) {
			diamond3Count++
		}
	}

	// 4钻条件：有3个3钻直推下级
	return diamond3Count >= 3
}

// isDiamond5 判断用户是否是5钻（有3个4钻直推下级）
func (h *InviteHandler) isDiamond5(userID int) bool {
	return h.isDiamond5WithCache(userID, make(map[int]int), make(map[int]PlacementNode))
}

// isDiamond5WithCache 判断用户是否是5钻（带缓存）
func (h *InviteHandler) isDiamond5WithCache(userID int, cache map[int]int, nodeMap map[int]PlacementNode) bool {
	// 查询所有直推下级（zj_fid = userID）
	var directReferrals []model.User
	if err := h.DB.Where("zj_fid = ?", userID).Find(&directReferrals).Error; err != nil {
		return false
	}

	// 统计直推下级中4钻的数量
	diamond4Count := 0
	for _, referral := range directReferrals {
		if h.isDiamond4WithCache(int(referral.Id), cache, nodeMap) {
			diamond4Count++
		}
	}

	// 5钻条件：有3个4钻直推下级
	return diamond4Count >= 3
}

// isDiamond6 判断用户是否是6钻（有4个5钻直推下级）
func (h *InviteHandler) isDiamond6(userID int) bool {
	return h.isDiamond6WithCache(userID, make(map[int]int), make(map[int]PlacementNode))
}

// isDiamond6WithCache 判断用户是否是6钻（带缓存）
func (h *InviteHandler) isDiamond6WithCache(userID int, cache map[int]int, nodeMap map[int]PlacementNode) bool {
	// 查询所有直推下级（zj_fid = userID）
	var directReferrals []model.User
	if err := h.DB.Where("zj_fid = ?", userID).Find(&directReferrals).Error; err != nil {
		return false
	}

	// 统计直推下级中5钻的数量
	diamond5Count := 0
	for _, referral := range directReferrals {
		if h.isDiamond5WithCache(int(referral.Id), cache, nodeMap) {
			diamond5Count++
		}
	}

	// 6钻条件：有4个5钻直推下级
	return diamond5Count >= 4
}

// isDiamond7 判断用户是否是7钻（有4个6钻直推下级）
func (h *InviteHandler) isDiamond7(userID int) bool {
	return h.isDiamond7WithCache(userID, make(map[int]int), make(map[int]PlacementNode))
}

// isDiamond7WithCache 判断用户是否是7钻（带缓存）
func (h *InviteHandler) isDiamond7WithCache(userID int, cache map[int]int, nodeMap map[int]PlacementNode) bool {
	// 查询所有直推下级（zj_fid = userID）
	var directReferrals []model.User
	if err := h.DB.Where("zj_fid = ?", userID).Find(&directReferrals).Error; err != nil {
		return false
	}

	// 统计直推下级中6钻的数量
	diamond6Count := 0
	for _, referral := range directReferrals {
		if h.isDiamond6WithCache(int(referral.Id), cache, nodeMap) {
			diamond6Count++
		}
	}

	// 7钻条件：有4个6钻直推下级
	return diamond6Count >= 4
}

// isDiamond8 判断用户是否是8钻（有5个7钻直推下级）
func (h *InviteHandler) isDiamond8(userID int) bool {
	return h.isDiamond8WithCache(userID, make(map[int]int), make(map[int]PlacementNode))
}

// isDiamond8WithCache 判断用户是否是8钻（带缓存）
func (h *InviteHandler) isDiamond8WithCache(userID int, cache map[int]int, nodeMap map[int]PlacementNode) bool {
	// 查询所有直推下级（zj_fid = userID）
	var directReferrals []model.User
	if err := h.DB.Where("zj_fid = ?", userID).Find(&directReferrals).Error; err != nil {
		return false
	}

	// 统计直推下级中7钻的数量
	diamond7Count := 0
	for _, referral := range directReferrals {
		if h.isDiamond7WithCache(int(referral.Id), cache, nodeMap) {
			diamond7Count++
		}
	}

	// 8钻条件：有5个7钻直推下级
	return diamond7Count >= 5
}

// isDiamond9 判断用户是否是9钻（有5个8钻直推下级）
func (h *InviteHandler) isDiamond9(userID int) bool {
	return h.isDiamond9WithCache(userID, make(map[int]int), make(map[int]PlacementNode))
}

// isDiamond9WithCache 判断用户是否是9钻（带缓存）
func (h *InviteHandler) isDiamond9WithCache(userID int, cache map[int]int, nodeMap map[int]PlacementNode) bool {
	// 查询所有直推下级（zj_fid = userID）
	var directReferrals []model.User
	if err := h.DB.Where("zj_fid = ?", userID).Find(&directReferrals).Error; err != nil {
		return false
	}

	// 统计直推下级中8钻的数量
	diamond8Count := 0
	for _, referral := range directReferrals {
		if h.isDiamond8WithCache(int(referral.Id), cache, nodeMap) {
			diamond8Count++
		}
	}

	// 9钻条件：有5个8钻直推下级
	return diamond8Count >= 5
}

// getPlacementTeamMemberIDs 基于安置树获取某个用户下面所有团队成员（不含自己）
func (h *InviteHandler) getPlacementTeamMemberIDs(rootUserID int) ([]int, error) {
	// 只取出 user_id 和 parent_id 即可
	var nodes []PlacementNode
	if err := h.DB.Table("chatgpt_placement_nodes").
		Select("user_id, parent_id").
		Find(&nodes).Error; err != nil {
		return nil, err
	}

	// 构建 parentID -> childrenIDs 映射
	childrenMap := make(map[int][]int)
	for _, n := range nodes {
		if n.ParentID > 0 {
			childrenMap[n.ParentID] = append(childrenMap[n.ParentID], n.UserID)
		}
	}

	// 从 rootUserID 开始做 BFS，收集所有子孙节点
	visited := make(map[int]bool)
	queue := []int{rootUserID}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		for _, childID := range childrenMap[cur] {
			if !visited[childID] {
				visited[childID] = true
				queue = append(queue, childID)
			}
		}
	}

	// visited 里都是团队成员（不含 rootUserID）
	result := make([]int, 0, len(visited))
	for id := range visited {
		result = append(result, id)
	}
	return result, nil
}
