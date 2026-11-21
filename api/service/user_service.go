package service

import (
	"fmt"
	"chatplus/core/types"
	"chatplus/store/model"
	"gorm.io/gorm"
	"sync"
	"time"
)

type UserService struct {
	db   *gorm.DB
	lock sync.Mutex
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db, lock: sync.Mutex{}}
}

// IncreasePower 增加用户算力
func (s *UserService) IncreasePower(userId int, power int, log model.PowerLog) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	tx := s.db.Begin()
	err := tx.Model(&model.User{}).Where("id", userId).UpdateColumn("power", gorm.Expr("power + ?", power)).Error
	if err != nil {
		tx.Rollback()
		return err
	}
	var user model.User
	tx.Where("id", userId).First(&user)
	err = tx.Create(&model.PowerLog{
		UserId:    user.Id,
		Username:  user.Username,
		Type:      log.Type,
		Amount:    float64(power),
		Balance:   user.Power,
		Mark:      types.PowerAdd,
		Model:     log.Model,
		Remark:    log.Remark,
		CreatedAt: time.Now(),
	}).Error
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

// DecreasePower 减少用户算力并写入 chatgpt_power_logs（同一事务，扣费与日志同时成功或同时失败）
func (s *UserService) DecreasePower(userId int, power int, log model.PowerLog) error {
	if power <= 0 || userId <= 0 {
		return nil
	}
	s.lock.Lock()
	defer s.lock.Unlock()

	tx := s.db.Begin()

	var user model.User
	if err := tx.Where("id = ?", userId).First(&user).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("用户不存在：%v", err)
	}
	if user.Power < float64(power) {
		tx.Rollback()
		return fmt.Errorf("当前用户剩余算力不足以完成本次操作")
	}

	res := tx.Model(&model.User{}).Where("id = ? AND power >= ?", userId, power).
		UpdateColumn("power", gorm.Expr("power - ?", power))
	if res.Error != nil {
		tx.Rollback()
		return fmt.Errorf("扣减算力失败：%v", res.Error)
	}
	if res.RowsAffected == 0 {
		tx.Rollback()
		return fmt.Errorf("当前用户剩余算力不足以完成本次操作")
	}

	if err := tx.Where("id = ?", userId).First(&user).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("读取用户算力失败：%v", err)
	}

	if log.Type == 0 {
		log.Type = types.PowerConsume
	}
	err := tx.Create(&model.PowerLog{
		UserId:    user.Id,
		Username:  user.Username,
		Type:      log.Type,
		Amount:    float64(power),
		Balance:   user.Power,
		Mark:      types.PowerSub,
		Model:     log.Model,
		Remark:    log.Remark,
		CreatedAt: time.Now(),
	}).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("记录算力日志失败：%v", err)
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("扣减算力提交失败：%v", err)
	}
	return nil
}

// IncreaseCoin 增加用户 SOHA 余额并记录流水
func (s *UserService) IncreaseCoin(userId uint, amount float64, log model.Q2Log) error {
	if amount <= 0 {
		return fmt.Errorf("invalid coin amount")
	}
	s.lock.Lock()
	defer s.lock.Unlock()

	tx := s.db.Begin()
	if err := tx.Model(&model.User{}).Where("id", userId).UpdateColumn("coin", gorm.Expr("coin + ?", amount)).Error; err != nil {
		tx.Rollback()
		return err
	}
	var user model.User
	if err := tx.Where("id", userId).First(&user).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Create(&model.Q2Log{
		UserId:    user.Id,
		Username:  user.Username,
		Type:      log.Type,
		Amount:    amount,
		Balance:   user.Coin,
		Mark:      types.PowerAdd,
		Model:     log.Model,
		Remark:    log.Remark,
		OrderNo:   log.OrderNo,
		CreatedAt: time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// DecreaseCoin 减少用户 SOHA 余额并记录流水
func (s *UserService) DecreaseCoin(userId uint, amount float64, log model.Q2Log) error {
	if amount <= 0 {
		return fmt.Errorf("invalid coin amount")
	}
	s.lock.Lock()
	defer s.lock.Unlock()

	tx := s.db.Begin()
	if err := tx.Model(&model.User{}).Where("id", userId).UpdateColumn("coin", gorm.Expr("coin - ?", amount)).Error; err != nil {
		tx.Rollback()
		return err
	}
	var user model.User
	if err := tx.Where("id", userId).First(&user).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Create(&model.Q2Log{
		UserId:    user.Id,
		Username:  user.Username,
		Type:      log.Type,
		Amount:    amount,
		Balance:   user.Coin,
		Mark:      types.PowerSub,
		Model:     log.Model,
		Remark:    log.Remark,
		OrderNo:   log.OrderNo,
		CreatedAt: time.Now(),
	}).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// RecordMintCoinOutputInTx 矿机产出：更新 user.coin / mint_order.js_coin，并写入钱包总账与矿机产出明细
func RecordMintCoinOutputInTx(tx *gorm.DB, order model.MintOrder, jsCoin float64) error {
	if jsCoin <= 0 {
		return fmt.Errorf("invalid mint output amount")
	}
	if err := tx.Model(&model.User{}).Where("id = ?", order.UserId).
		UpdateColumn("coin", gorm.Expr("coin + ?", jsCoin)).Error; err != nil {
		return err
	}
	if err := tx.Model(&model.MintOrder{}).Where("order_no = ?", order.OrderNo).
		UpdateColumn("js_coin", gorm.Expr("js_coin + ?", jsCoin)).Error; err != nil {
		return err
	}
	var user model.User
	if err := tx.First(&user, order.UserId).Error; err != nil {
		return err
	}
	now := time.Now()
	walletLog := model.Q2Log{
		UserId:    order.UserId,
		Username:  order.Username,
		Type:      types.CoinCc,
		Amount:    jsCoin,
		Balance:   user.Coin,
		Mark:      types.PowerAdd,
		Model:     "SOHA产出",
		OrderNo:   order.OrderNo,
		CreatedAt: now,
	}
	if err := tx.Create(&walletLog).Error; err != nil {
		return err
	}
	return tx.Create(&model.CoinLog{
		UserId:    order.UserId,
		Username:  order.Username,
		Type:      types.CoinCc,
		Amount:    jsCoin,
		Balance:   user.Coin,
		Mark:      types.PowerAdd,
		Model:     "SOHA产出",
		OrderNo:   order.OrderNo,
		CreatedAt: now,
	}).Error
}
