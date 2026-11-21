package model

import "time"

type DigiJob struct {
	Id        uint `gorm:"primarykey;column:id"`
	Type      string
	UserId    int
	TaskId    string
	Content   string //文案
	BgUrl     string
	VideoUrl  string // 上游视频url
	WaterUrl  string // 平台视频url
	Progress  int
	ErrMsg    string  // 报错信息
	Power     float64 // 消耗算力
	Language  string  // 语言
	RoleName  string  // 角色
	RoleStyle string  // 角色形象
	Status    string  // Status
	StatusId  string  // 状态id
	ChannelId string  //频道service
	PreUrl    string  //用户背景url
	//CompleteAt string  //完成时间

	CreatedAt time.Time
}

func (DigiJob) TableName() string {
	return "chatgpt_digi_jobs"
}
