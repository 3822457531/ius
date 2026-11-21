package handler

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils/resp"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CoinLogHandler 用户 SOHA 钱包总账明细（user.coin，数据表 chatgpt_q2_logs，对标 power_log）
type CoinLogHandler struct {
	BaseHandler
}

func NewCoinLogHandler(app *core.AppServer, db *gorm.DB) *CoinLogHandler {
	return &CoinLogHandler{BaseHandler: BaseHandler{App: app, DB: db}}
}

func (h *CoinLogHandler) List(c *gin.Context) {
	var data struct {
		Model    string   `json:"model"`
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
	userId := h.GetLoginUserId(c)
	if userId == 0 {
		resp.NotAuth(c)
		return
	}
	session = session.Where("user_id", userId)
	if data.Model != "" {
		session = session.Where("model LIKE ? OR username LIKE ?", "%"+data.Model+"%", "%"+data.Model+"%")
	}
	if len(data.Date) == 2 {
		start := data.Date[0] + " 00:00:00"
		end := data.Date[1] + " 23:59:59"
		session = session.Where("created_at >= ? AND created_at <= ?", start, end)
	}

	var total int64
	session.Model(&model.Q2Log{}).Count(&total)
	var items []model.Q2Log
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
	resp.SUCCESS(c, vo.NewPage(total, data.Page, data.PageSize, list))
}
