package model

import "time"

// ConsumeRewardLog 应用广场算力成功消耗后的团队分红幂等记录
type ConsumeRewardLog struct {
	Id         uint `gorm:"primarykey;column:id"`
	SourceKey  string
	UserId     uint
	Power      float64
	SourceName string
	CreatedAt  time.Time
}

func (ConsumeRewardLog) TableName() string {
	return "chatgpt_consume_reward_logs"
}
