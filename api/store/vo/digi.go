package vo

import "time"

type DigiJob struct {
	Id        uint      `json:"id"`
	Type      string    `json:"type"`
	UserId    int       `json:"user_id"`
	ChannelId string    `json:"channel_id"`
	TaskId    string    `json:"task_id"`
	Content   string    `json:"content"` //文案
	BgUrl     string    `json:"bg_url"`
	PreUrl    string    `json:"pre_url"`   //用户背景url
	VideoUrl  string    `json:"video_url"` // 上游视频url
	WaterUrl  string    `json:"water_url"` // 平台视频url
	Progress  int       `json:"progress"`
	ErrMsg    string    `json:"err_msg"`    // 报错信息
	Power     float64   `json:"power"`      // 消耗算力
	Language  string    `json:"language"`   // 语言
	RoleName  string    `json:"role_name"`  // 角色
	RoleStyle string    `json:"role_style"` // 角色形象
	Status    string    `json:"status"`     // Status
	StatusId  string    `json:"status_id"`  // 状态id
	CreatedAt time.Time `json:"created_at"` // 状态id

}
