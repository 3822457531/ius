package service

import (
	"chatplus/core/types"
	logger2 "chatplus/logger"
	"chatplus/store/model"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var powerRefundLogger = logger2.GetLogger()

func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number == 1062
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique")
}

// RefundPowerIdempotent 幂等退还算力（PPT / 短剧等同步场景；同一 source+refundKey 仅退一次）
func RefundPowerIdempotent(db *gorm.DB, userId uint, amount int, source, refundKey, modelName, remark string) (alreadyRefunded bool, err error) {
	if db == nil || userId == 0 || amount <= 0 {
		return false, fmt.Errorf("invalid refund args")
	}
	source = strings.TrimSpace(source)
	refundKey = strings.TrimSpace(refundKey)
	if source == "" || refundKey == "" {
		return false, fmt.Errorf("refund source/key required")
	}
	if modelName == "" {
		modelName = source
	}
	if remark == "" {
		remark = fmt.Sprintf("%s 算力退还", source)
	}

	tx := db.Begin()
	row := model.AppPowerRefundLog{
		UserId:    userId,
		Source:    source,
		RefundKey: refundKey,
		Amount:    amount,
		CreatedAt: time.Now(),
	}
	if err := tx.Create(&row).Error; err != nil {
		tx.Rollback()
		if isDuplicateKeyErr(err) {
			return true, nil
		}
		return false, err
	}

	if err := tx.Model(&model.User{}).Where("id = ?", userId).
		UpdateColumn("power", gorm.Expr("power + ?", amount)).Error; err != nil {
		tx.Rollback()
		return false, err
	}

	var user model.User
	if err := tx.Where("id = ?", userId).First(&user).Error; err != nil {
		tx.Rollback()
		return false, err
	}

	if err := tx.Create(&model.PowerLog{
		UserId:    user.Id,
		Username:  user.Username,
		Type:      types.PowerRefund,
		Amount:    float64(amount),
		Balance:   user.Power,
		Mark:      types.PowerAdd,
		Model:     modelName,
		Remark:    remark,
		CreatedAt: time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		return false, err
	}

	if err := tx.Commit().Error; err != nil {
		return false, err
	}
	return false, nil
}

// RefundMidJourneyJob 绘画/image2 任务失败退还算力（幂等：power>0 时仅退一次）
func RefundMidJourneyJob(db *gorm.DB, job *model.MidJourneyJob, modelName, errMsg string) {
	if db == nil || job == nil || job.Id == 0 || job.Power <= 0 {
		return
	}
	if errMsg == "" {
		errMsg = "任务失败"
	}
	refundPower := job.Power

	tx := db.Begin()
	res := tx.Model(&model.MidJourneyJob{}).
		Where("id = ? AND power > 0", job.Id).
		Updates(map[string]interface{}{
			"power":    0,
			"progress": 100,
			"err_msg":  errMsg,
		})
	if res.Error != nil || res.RowsAffected == 0 {
		tx.Rollback()
		return
	}

	if err := tx.Model(&model.User{}).Where("id = ?", job.UserId).
		UpdateColumn("power", gorm.Expr("power + ?", refundPower)).Error; err != nil {
		tx.Rollback()
		powerRefundLogger.Errorf("绘画任务退还算力失败 user=%d job=%d: %v", job.UserId, job.Id, err)
		return
	}

	var user model.User
	if err := tx.Where("id = ?", job.UserId).First(&user).Error; err != nil {
		tx.Rollback()
		return
	}

	if modelName == "" {
		modelName = PowerLogModelImage
	}
	taskID := job.TaskId
	if taskID == "" {
		taskID = fmt.Sprintf("%d", job.Id)
	}

	if err := tx.Create(&model.PowerLog{
		UserId:    user.Id,
		Username:  user.Username,
		Type:      types.PowerRefund,
		Amount:    refundPower,
		Balance:   user.Power,
		Mark:      types.PowerAdd,
		Model:     modelName,
		Remark:    fmt.Sprintf("绘画任务失败，退回算力。任务ID：%s", taskID),
		CreatedAt: time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		powerRefundLogger.Errorf("绘画任务退还算力日志失败 user=%d job=%d: %v", job.UserId, job.Id, err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		powerRefundLogger.Errorf("绘画任务退还算力提交失败 job=%d: %v", job.Id, err)
		return
	}

	_ = db.Delete(&model.MidJourneyJob{}, job.Id).Error
}
