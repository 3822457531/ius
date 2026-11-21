package admin

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/handler"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils/resp"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CoinLogHandler struct {
	handler.BaseHandler
}

func NewCoinLogHandler(app *core.AppServer, db *gorm.DB) *CoinLogHandler {
	return &CoinLogHandler{BaseHandler: handler.BaseHandler{App: app, DB: db}}
}

func (h *CoinLogHandler) List(c *gin.Context) {
	var data struct {
		Username string   `json:"username"`
		Model    string   `json:"model"`
		OrderNo  string   `json:"order_no"`
		Date     []string `json:"date"`
		Page     int      `json:"page"`
		PageSize int      `json:"page_size"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	if data.Page <= 0 {
		data.Page = 1
	}
	if data.PageSize <= 0 {
		data.PageSize = 20
	}

	session := h.DB.Session(&gorm.Session{})
	if data.Username != "" {
		session = session.Where("username LIKE ?", "%"+data.Username+"%")
	}
	if data.Model != "" {
		session = session.Where("model LIKE ?", "%"+data.Model+"%")
	}
	if data.OrderNo != "" {
		session = session.Where("order_no LIKE ?", "%"+data.OrderNo+"%")
	}
	if len(data.Date) == 2 {
		start := data.Date[0] + " 00:00:00"
		end := data.Date[1] + " 23:59:59"
		session = session.Where("created_at >= ? AND created_at <= ?", start, end)
	}

	var total int64
	session.Model(&model.CoinLog{}).Count(&total)
	var items []model.CoinLog
	list := make([]vo.CoinLog, 0)
	offset := (data.Page - 1) * data.PageSize
	res := session.Order("id DESC").Offset(offset).Limit(data.PageSize).Find(&items)
	if res.Error == nil {
		for _, item := range items {
			list = append(list, vo.CoinLog{
				Id:        item.Id,
				UserId:    item.UserId,
				Username:  item.Username,
				Type:      item.Type,
				TypeStr:   item.Type.String(),
				Amount:    item.Amount,
				Mark:      item.Mark,
				Balance:   item.Balance,
				Model:     item.Model,
				Remark:    item.Remark,
				OrderNo:   item.OrderNo,
				CreatedAt: item.CreatedAt.Unix(),
			})
		}
	}

	var totalCoin float64
	statSession := h.DB.Model(&model.CoinLog{}).Where("mark", types.PowerAdd)
	if data.Username != "" {
		statSession = statSession.Where("username LIKE ?", "%"+data.Username+"%")
	}
	if data.Model != "" {
		statSession = statSession.Where("model LIKE ?", "%"+data.Model+"%")
	}
	if data.OrderNo != "" {
		statSession = statSession.Where("order_no LIKE ?", "%"+data.OrderNo+"%")
	}
	if len(data.Date) == 2 {
		start := data.Date[0] + " 00:00:00"
		end := data.Date[1] + " 23:59:59"
		statSession = statSession.Where("created_at >= ? AND created_at <= ?", start, end)
	}
	statSession.Select("COALESCE(SUM(amount), 0)").Scan(&totalCoin)

	resp.SUCCESS(c, gin.H{"data": vo.NewPage(total, data.Page, data.PageSize, list), "stat": totalCoin})
}
