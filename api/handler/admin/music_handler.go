package admin

import (
	"chatplus/core"
	"chatplus/handler"
	"chatplus/service/suno"
	"chatplus/store/model"
	"chatplus/utils/resp"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MusicHandler struct {
	handler.BaseHandler
}

func NewMusicHandler(app *core.AppServer, db *gorm.DB) *MusicHandler {
	return &MusicHandler{BaseHandler: handler.BaseHandler{App: app, DB: db}}
}

func (h *MusicHandler) FailedStats(c *gin.Context) {
	var total int64
	h.DB.Model(&model.MusicJob{}).
		Where("UPPER(status) = ? AND (audio_url = '' OR audio_url IS NULL)", "FAILED").
		Count(&total)
	resp.SUCCESS(c, gin.H{"failed_count": total})
}

func (h *MusicHandler) RepairHistory(c *gin.Context) {
	result := suno.RepairFailedMusicJobs(h.DB, &h.App.Config.SunoConfig)
	resp.SUCCESS(c, result)
}
