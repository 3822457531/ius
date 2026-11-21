package admin

import (
	"chatplus/core"
	"chatplus/core/types"
	"chatplus/handler"
	"chatplus/store/model"
	"chatplus/store/vo"
	"chatplus/utils"
	"chatplus/utils/resp"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"time"
)

type MintHandler struct {
	handler.BaseHandler
}

func NewMintHandler(app *core.AppServer, db *gorm.DB) *MintHandler {
	return &MintHandler{BaseHandler: handler.BaseHandler{App: app, DB: db}}
}

func (h *MintHandler) Save(c *gin.Context) {
	var data struct {
		Id   uint    `json:"id"`
		Name string  `json:"name"`
		Fee  float64 `json:"fee"`
		//Discount  float64 `json:"discount"`
		Enabled   bool    `json:"enabled"`
		Days      float64 `json:"days"`
		Power     float64 `json:"power"`
		CreatedAt int64   `json:"created_at"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	item := model.Mint{
		Name: data.Name,
		Fee:  data.Fee,
		//Discount: data.Discount,
		Days:    data.Days,
		Power:   data.Power,
		Enabled: data.Enabled}
	item.Id = data.Id
	if item.Id > 0 {
		item.CreatedAt = time.Unix(data.CreatedAt, 0)
	}
	res := h.DB.Save(&item)
	logger.Info(res)
	if res.Error != nil {
		resp.ERROR(c, "更新数据库失败！")
		return
	}

	var itemVo vo.Mint
	err := utils.CopyObject(item, &itemVo)
	if err != nil {
		resp.ERROR(c, "数据拷贝失败！")
		return
	}
	itemVo.Id = item.Id
	itemVo.UpdatedAt = item.UpdatedAt.Unix()
	resp.SUCCESS(c, itemVo)
}

// List 数据列表
func (h *MintHandler) List(c *gin.Context) {
	var items []model.Mint
	var list = make([]vo.Mint, 0)
	res := h.DB.Order("sort_num ASC").Find(&items)
	if res.Error == nil {
		for _, item := range items {
			var mint vo.Mint
			err := utils.CopyObject(item, &mint)
			if err == nil {
				mint.Id = item.Id
				mint.CreatedAt = item.CreatedAt.Unix()
				mint.UpdatedAt = item.UpdatedAt.Unix()
				list = append(list, mint)
			} else {
				logger.Error(err)
			}
		}
	}
	resp.SUCCESS(c, list)
}

func (h *MintHandler) Enable(c *gin.Context) {
	var data struct {
		Id      uint `json:"id"`
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	res := h.DB.Model(&model.Mint{}).Where("id", data.Id).UpdateColumn("enabled", data.Enabled)
	if res.Error != nil {
		resp.ERROR(c, "更新数据库失败！")
		return
	}
	resp.SUCCESS(c)
}

func (h *MintHandler) Sort(c *gin.Context) {
	var data struct {
		Ids   []uint `json:"ids"`
		Sorts []int  `json:"sorts"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		resp.ERROR(c, types.InvalidArgs)
		return
	}

	for index, id := range data.Ids {
		res := h.DB.Model(&model.Mint{}).Where("id", id).Update("sort_num", data.Sorts[index])
		logger.Info(res)
		if res.Error != nil {
			resp.ERROR(c, "更新数据库失败！")
			return
		}
	}

	resp.SUCCESS(c)
}

func (h *MintHandler) Remove(c *gin.Context) {
	id := h.GetInt(c, "id", 0)

	if id > 0 {
		res := h.DB.Where("id", id).Delete(&model.Mint{})
		if res.Error != nil {
			resp.ERROR(c, "更新数据库失败！")
			return
		}
	}
	resp.SUCCESS(c)
}
