package handler

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils"
	"chatplus/utils/resp"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type MintHandler struct {
	BaseHandler
}

func NewMintHandler(app *core.AppServer, db *gorm.DB) *MintHandler {
	return &MintHandler{BaseHandler: BaseHandler{App: app, DB: db}}
}

// List 模型列表
func (h *MintHandler) List(c *gin.Context) {
	var items []model.Mint
	var list = make([]vo.Mint, 0)
	res := h.DB.Where("enabled", true).Order("sort_num ASC").Find(&items)
	if res.Error == nil {
		// 按订单实时统计已认购台数（比 sales 字段更准确，兼容历史数据）
		type mintSalesRow struct {
			MintId uint
			Cnt    int64
		}
		var rows []mintSalesRow
		_ = h.DB.Model(&model.MintOrder{}).
			Select("mint_id as mint_id, count(*) as cnt").
			Group("mint_id").
			Scan(&rows).Error
		salesMap := make(map[uint]int, len(rows))
		for _, r := range rows {
			salesMap[r.MintId] = int(r.Cnt)
		}

		for _, item := range items {
			var mint vo.Mint
			err := utils.CopyObject(item, &mint)
			if err == nil {
				mint.Id = item.Id
				mint.CreatedAt = item.CreatedAt.Unix()
				mint.UpdatedAt = item.UpdatedAt.Unix()
				if n, ok := salesMap[item.Id]; ok {
					mint.Sales = n
				} else {
					mint.Sales = item.Sales
				}
				list = append(list, mint)
			} else {
				logger.Error(err)
			}
		}
	}
	resp.SUCCESS(c, list)
}

// ReleaseConfig 矿机 SOHA 产出分段释放比例（供前端展示预估日产出）
func (h *MintHandler) ReleaseConfig(c *gin.Context) {
	cfg := h.App.SysConfig
	resp.SUCCESS(c, gin.H{
		"kj_zj_plus":        cfg.KjZjPlus,
		"kj_plus_11_20":     cfg.KjPlus1120,
		"kj_plus_21_30":     cfg.KjPlus2130,
		"kj_plus_31_40":     cfg.KjPlus3140,
		"kj_plus_41_50":     cfg.KjPlus4150,
		"kj_plus_51_60":     cfg.KjPlus5160,
		"kj_plus_61_70":     cfg.KjPlus6170,
		"kj_plus_71_80":     cfg.KjPlus7180,
		"kj_plus_81_90":     cfg.KjPlus8190,
		"kj_plus_91_100":    cfg.KjPlus91100,
		"kj_jj_plus":        cfg.KjJjPlus,
		"kj_san_plus":       cfg.KjSanPlus,
		"kj_si_plus":        cfg.KjSiPlus,
		"kj_other_plus":     cfg.KjOtherPlus,
		"periods": []gin.H{
			{"label": "0-10天", "rate": cfg.KjReleaseRateByDays(5)},
			{"label": "11-20天", "rate": cfg.KjReleaseRateByDays(15)},
			{"label": "21-30天", "rate": cfg.KjReleaseRateByDays(25)},
			{"label": "31-40天", "rate": cfg.KjReleaseRateByDays(35)},
			{"label": "41-50天", "rate": cfg.KjReleaseRateByDays(45)},
			{"label": "51-60天", "rate": cfg.KjReleaseRateByDays(55)},
			{"label": "61-70天", "rate": cfg.KjReleaseRateByDays(65)},
			{"label": "71-80天", "rate": cfg.KjReleaseRateByDays(75)},
			{"label": "81-90天", "rate": cfg.KjReleaseRateByDays(85)},
			{"label": "91-100天+", "rate": cfg.KjReleaseRateByDays(95)},
		},
	})
}

// 获取当前矿机的 SOHA 产出明细（chatgpt_coin_logs；历史数据从 q2_logs 自动补齐）
func (h *MintHandler) GetCoinItems(c *gin.Context) {
	userId := h.GetLoginUserId(c)
	if userId == 0 {
		resp.NotAuth(c)
		return
	}
	productIdStr := c.Query("productId")
	if productIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "productId is required"})
		return
	}

	// 校验矿机归属
	var mintOrder model.MintOrder
	if err := h.DB.Where("order_no = ? AND user_id = ?", productIdStr, userId).First(&mintOrder).Error; err != nil {
		resp.SUCCESS(c, make([]vo.CoinLog, 0))
		return
	}

	// 将 q2_logs 中该矿机的历史产出同步到 coin_logs（幂等）
	h.syncMintCoinLogsFromQ2(productIdStr, userId)

	type coinItemRow struct {
		Id        uint
		Username  string
		Amount    float64
		Balance   float64
		Type      types.PowerType
		CreatedAt time.Time
	}

	var items []coinItemRow
	var list = make([]vo.CoinLog, 0)

	res := h.DB.Raw(`
		SELECT c.id AS id, orders.subject AS username, c.amount AS amount,
		       c.balance AS balance, c.type AS type, c.created_at AS created_at
		FROM chatgpt_mint_orders AS orders
		INNER JOIN chatgpt_coin_logs AS c
			ON orders.order_no = c.order_no AND c.user_id = orders.user_id
		WHERE orders.order_no = ? AND orders.user_id = ? AND c.mark = ?
		ORDER BY c.created_at DESC
	`, productIdStr, userId, types.PowerAdd).Scan(&items)

	if res.Error == nil {
		for _, item := range items {
			list = append(list, vo.CoinLog{
				Id:        item.Id,
				Username:  item.Username,
				Amount:    item.Amount,
				Balance:   item.Balance,
				Type:      item.Type,
				CreatedAt: item.CreatedAt.Unix(),
			})
		}
	} else {
		logger.Error("查询矿机产出明细出错: ", res.Error)
	}

	// coin_logs 尚未建表或未回填时，直接读 q2_logs（与总账同源）
	if len(list) == 0 {
		items = nil
		fallback := h.DB.Raw(`
			SELECT q.id AS id, orders.subject AS username, q.amount AS amount,
			       q.balance AS balance, q.type AS type, q.created_at AS created_at
			FROM chatgpt_mint_orders AS orders
			INNER JOIN chatgpt_q2_logs AS q
				ON q.user_id = orders.user_id AND q.order_no = orders.order_no
			WHERE orders.order_no = ? AND orders.user_id = ? AND q.mark = ?
			ORDER BY q.created_at DESC
		`, productIdStr, userId, types.PowerAdd).Scan(&items)
		if fallback.Error == nil {
			for _, item := range items {
				list = append(list, vo.CoinLog{
					Id:        item.Id,
					Username:  item.Username,
					Amount:    item.Amount,
					Balance:   item.Balance,
					Type:      item.Type,
					CreatedAt: item.CreatedAt.Unix(),
				})
			}
		}
	}

	// 历史 q2_logs 未写 order_no 时：仅一台运行中矿机则归入该矿机明细
	if len(list) == 0 {
		var runningCount int64
		h.DB.Model(&model.MintOrder{}).
			Where("user_id = ? AND status = ?", userId, types.OrderZyz).
			Count(&runningCount)
		if runningCount == 1 {
			items = nil
			legacy := h.DB.Raw(`
				SELECT q.id AS id, ? AS username, q.amount AS amount,
				       q.balance AS balance, q.type AS type, q.created_at AS created_at
				FROM chatgpt_q2_logs AS q
				WHERE q.user_id = ? AND q.mark = ? AND q.model = 'SOHA产出'
				  AND (q.order_no = ? OR q.order_no = '' OR q.order_no IS NULL)
				ORDER BY q.created_at DESC
			`, mintOrder.Subject, userId, types.PowerAdd, productIdStr).Scan(&items)
			if legacy.Error == nil {
				for _, item := range items {
					list = append(list, vo.CoinLog{
						Id:        item.Id,
						Username:  item.Username,
						Amount:    item.Amount,
						Balance:   item.Balance,
						Type:      item.Type,
						CreatedAt: item.CreatedAt.Unix(),
					})
				}
			}
		}
	}

	// 明细合计与 js_coin 不一致时，回写校正（与列表页累计产出对齐）
	if len(list) > 0 {
		var total float64
		for _, item := range list {
			total += item.Amount
		}
		if total > mintOrder.JsCoin+0.0001 {
			_ = h.DB.Model(&model.MintOrder{}).
				Where("order_no = ? AND user_id = ?", productIdStr, userId).
				UpdateColumn("js_coin", total).Error
		}
	}

	resp.SUCCESS(c, list)
}

// syncMintCoinLogsFromQ2 把钱包总账里带 order_no 的矿机产出，补写入 coin_logs
func (h *MintHandler) syncMintCoinLogsFromQ2(orderNo string, userId uint) {
	_ = h.DB.Exec(`
		INSERT INTO chatgpt_coin_logs (user_id, username, type, amount, balance, model, remark, mark, created_at, order_no)
		SELECT q.user_id, q.username, q.type, q.amount, q.balance, q.model, q.remark, q.mark, q.created_at, q.order_no
		FROM chatgpt_q2_logs q
		LEFT JOIN chatgpt_coin_logs c
			ON c.user_id = q.user_id AND c.order_no = q.order_no AND c.amount = q.amount AND c.created_at = q.created_at
		WHERE q.user_id = ? AND q.order_no = ? AND q.mark = ? AND c.id IS NULL
	`, userId, orderNo, types.PowerAdd).Error
}
