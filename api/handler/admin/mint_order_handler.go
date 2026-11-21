package admin

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/handler"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils"
	"chatplus/utils/resp"
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MintOrderHandler struct {
	handler.BaseHandler
}

func NewMintOrderHandler(app *core.AppServer, db *gorm.DB) *MintOrderHandler {
	return &MintOrderHandler{BaseHandler: handler.BaseHandler{App: app, DB: db}}
}

func (h *MintOrderHandler) List(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		resp.NotPermission(c)
		return
	}

	var data struct {
		Username string `json:"username"`
		OrderNo  string `json:"order_no"`
		Status   int    `json:"status"`
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	if data.Page <= 0 {
		data.Page = 1
	}
	if data.PageSize <= 0 {
		data.PageSize = 15
	}

	session := h.DB.Session(&gorm.Session{}).Model(&model.MintOrder{})
	if data.Username != "" {
		session = session.Where("username LIKE ?", "%"+data.Username+"%")
	}
	if data.OrderNo != "" {
		session = session.Where("order_no = ?", data.OrderNo)
	}
	if data.Status >= 0 {
		session = session.Where("status = ?", data.Status)
	}

	var total int64
	session.Count(&total)

	var items []model.MintOrder
	list := make([]vo.MintOrder, 0)
	offset := (data.Page - 1) * data.PageSize
	res := session.Order("id DESC").Offset(offset).Limit(data.PageSize).Find(&items)
	if res.Error == nil {
		for _, item := range items {
			var order vo.MintOrder
			if err := utils.CopyObject(item, &order); err == nil {
				order.Id = item.Id
				order.CreatedAt = item.CreatedAt.Unix()
				order.UpdatedAt = item.UpdatedAt.Unix()
				list = append(list, order)
			} else {
				logger.Error(err)
			}
		}
	}
	resp.SUCCESS(c, vo.NewPage(total, data.Page, data.PageSize, list))
}

func (h *MintOrderHandler) Close(c *gin.Context) {
	if err := utils.CheckPermission(c, h.DB); err != nil {
		resp.NotPermission(c)
		return
	}

	id := utils.IntValue(c.Query("id"), 0)
	if id <= 0 {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	var order model.MintOrder
	if err := h.DB.First(&order, id).Error; err != nil {
		resp.ERROR(c, "算力机订单不存在")
		return
	}
	if order.Status != types.OrderZyz {
		resp.ERROR(c, "仅运行中的算力机可关闭")
		return
	}

	res := h.DB.Model(&model.MintOrder{}).
		Where("id = ? AND status = ?", id, types.OrderZyz).
		Update("status", types.OrderYjs)
	if res.Error != nil {
		resp.ERROR(c, "关闭失败："+res.Error.Error())
		return
	}
	if res.RowsAffected == 0 {
		resp.ERROR(c, "关闭失败：订单状态已变更")
		return
	}

	resp.SUCCESS(c, fmt.Sprintf("已关闭用户 %s 的算力机订单 %s，将不再产出 SOHA", order.Username, order.OrderNo))
}
