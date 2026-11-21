package handler

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils"
	"chatplus/utils/resp"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderHandler struct {
	BaseHandler
}

func NewOrderHandler(app *core.AppServer, db *gorm.DB) *OrderHandler {
	return &OrderHandler{BaseHandler: BaseHandler{App: app, DB: db}}
}

func (h *OrderHandler) List(c *gin.Context) {
	var data struct {
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	userId := h.GetLoginUserId(c)
	session := h.DB.Session(&gorm.Session{}).Where("user_id = ? AND status = ?", userId, types.OrderPaidSuccess)
	var total int64
	session.Model(&model.Order{}).Count(&total)
	var items []model.Order
	var list = make([]vo.Order, 0)
	offset := (data.Page - 1) * data.PageSize
	res := session.Order("id DESC").Offset(offset).Limit(data.PageSize).Find(&items)
	if res.Error == nil {
		for _, item := range items {
			var order vo.Order
			err := utils.CopyObject(item, &order)
			if err == nil {
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

// 获取用户的套餐列表
func (h *OrderHandler) ListByUid(c *gin.Context) {
	var data struct {
		Page     int `json:"page"`
		PageSize int `json:"page_size"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}
	userId := h.GetLoginUserId(c)
	session := h.DB.Session(&gorm.Session{}).Where("user_id = ? AND status = ?", userId, types.OrderPaidSuccess)
	var total int64
	session.Model(&model.Order{}).Count(&total)
	var items []model.Order
	var list = make([]vo.Order, 0)
	offset := (data.Page - 1) * data.PageSize
	res := session.Order("id DESC").Offset(offset).Limit(data.PageSize).Find(&items)
	if res.Error == nil {
		for _, item := range items {
			var order vo.Order
			err := utils.CopyObject(item, &order)
			if err == nil {
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

// GetOrderList 获取订单列表。pending=待产出(仅 status=8)；exited=已产出(status=9 或 status=8 且 js_power>0，即在投中但已有产出的也算已产出)。
// type 可从 query 的 type 或 params 的 JSON（如 params={"type":"exited"}）中读取。
func (h *OrderHandler) GetOrderList(c *gin.Context) {
	userId := h.GetLoginUserId(c)
	orderType := c.Query("type")
	if orderType == "" {
		// 兼容前端传 params={"type":"exited"} 的情况
		if paramsStr := c.Query("params"); paramsStr != "" {
			var params struct {
				Type string `json:"type"`
			}
			if _ = json.Unmarshal([]byte(paramsStr), &params); params.Type != "" {
				orderType = params.Type
			}
		}
	}
	if orderType == "" {
		orderType = "pending"
	}

	var items []model.Order
	list := make([]vo.Order, 0)
	var res *gorm.DB
	switch orderType {
	case "exited":
		// 已产出：status=9(已出局) 或 status=8 且 js_power>0(在投中但已有产出)
		res = h.DB.Where("user_id = ? AND (status = ? OR (status = ? AND js_power >= 0))",
			userId, types.OrderYjs, types.OrderZyz).Order("id DESC").Find(&items)
	default:
		// 待产出：仅 status=8(在运转)
		res = h.DB.Where("user_id = ? AND status = ?", userId, types.OrderZyz).Order("id DESC").Find(&items)
	}
	if res.Error == nil {
		for _, item := range items {
			var order vo.Order
			if err := utils.CopyObject(item, &order); err != nil {
				logger.Error(err)
				continue
			}
			order.Id = item.Id
			order.Subject = item.Subject
			order.Amount = item.Amount
			order.Power = item.Power       // 预产值
			order.NowPower = item.NowPower // 待产值（剩余待产）
			order.JsPower = item.JsPower   // 已解锁/已产出
			order.Sales = item.Sales       // 投入算力
			order.CreatedAt = item.CreatedAt.Unix()
			order.UpdatedAt = item.UpdatedAt.Unix()
			list = append(list, order)
		}
	}
	resp.SUCCESS(c, list)
}

// 获取当前产品的收益详情
func (h *OrderHandler) GetOrderItems(c *gin.Context) {
	//	userId := h.GetLoginUserId(c)
	// 获取查询字符串中的 productId 参数
	//	productIdStr := "202406297212830313435627520" //; c.Query("productId")
	productIdStr := c.Query("productId")
	if productIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "productId is required"})
		return
	}

	logger.Info("productIdStr...", productIdStr)

	//获取当前产品订单号
	var items []model.PowerLog
	var list = make([]vo.PowerLog, 0)

	logger.Info("开始查询数据...")

	// 使用 Scan 映射查询结果到结构体
	res := h.DB.Raw(`
        SELECT 
			LOGS.id AS id,
			orders.subject AS username,
			LOGS.type AS type,
			LOGS.amount AS amount,
			LOGS.mark AS mark,
			LOGS.balance AS balance,
			LOGS.model AS model,
			LOGS.remark AS remark,
			LOGS.created_at AS created_at
        FROM chatgpt_orders AS orders 
        INNER JOIN chatgpt_power_logs AS LOGS ON orders.order_no = LOGS.order_no
		WHERE orders.order_no = ?
		ORDER BY LOGS.id DESC
		`, productIdStr).Scan(&items)

	logger.Info("数据为", items)

	if res.Error == nil {
		for _, item := range items {
			var order vo.PowerLog
			err := utils.CopyObject(item, &order)
			if err == nil {
				order.Id = item.Id
				order.Amount = item.Amount
				order.Username = item.Username
				order.Type = item.Type
				order.Mark = item.Mark
				order.Balance = item.Balance
				order.Model = item.Model
				order.Remark = item.Remark
				order.CreatedAt = item.CreatedAt.Unix()
				list = append(list, order)
			} else {
				logger.Error(err)
			}
		}
	} else {
		logger.Error("查询数据库出错: ", res.Error)
	}

	resp.SUCCESS(c, list)
}

// List 获取进行中的order
func (h *OrderHandler) GetEndOrderList(c *gin.Context) {

	userId := h.GetLoginUserId(c)
	var items []model.Order
	var list = make([]vo.Order, 0)
	//res := h.DB.Where("user_id", userId).Find(&items)
	res := h.DB.Where("user_id = ?  And  status=9 ", userId).Find(&items)
	if res.Error == nil {
		for _, item := range items {
			var order vo.Order
			err := utils.CopyObject(item, &order)
			if err == nil {
				order.Id = item.Id
				order.Subject = item.Subject
				order.Amount = item.Amount
				order.Power = item.Power
				order.JsPower = item.JsPower

				order.CreatedAt = item.CreatedAt.Unix()
				order.UpdatedAt = item.UpdatedAt.Unix()
				list = append(list, order)
			} else {
				logger.Error(err)
			}
		}
	}
	resp.SUCCESS(c, list)
}
