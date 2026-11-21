package model

import "time"

type InviteCode struct {
	Id        uint `gorm:"primarykey;column:id"`
	UserId    uint
	FId       string //上级用户id
	Code      string
	Hits      int // 点击次数
	RegNum    int // 注册人数
	CreatedAt time.Time
}
