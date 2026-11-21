package model

import "time"

type MusicJob struct {
	Id        uint      `gorm:"primarykey;column:id"`
	UserId    int       `gorm:"column:user_id"`
	TaskId    string    `gorm:"column:task_id"`
	Type      string    `gorm:"column:type"`
	Content   string    `gorm:"column:content"`
	AudioUrl  string    `gorm:"column:audio_url"`
	VideoUrl  string    `gorm:"column:video_url"`
	ImageUrl  string    `gorm:"column:image_url"`
	Lyrics    string    `gorm:"column:lyrics;type:longtext"`
	Duration  float64   `gorm:"column:duration"`
	Progress  int       `gorm:"column:progress"`
	ErrMsg    string    `gorm:"column:err_msg"`
	Power     float64   `gorm:"column:power"`
	RoleName  string    `gorm:"column:role_name"`
	RoleStyle string    `gorm:"column:role_style"`
	Language  string    `gorm:"column:language"`
	Publish   *bool     `gorm:"column:publish"`
	Status    string    `gorm:"column:status"`
	StatusId  string    `gorm:"column:status_id"`
	ChannelId string    `gorm:"column:channel_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (MusicJob) TableName() string {
	return "chatgpt_music_jobs"
}
