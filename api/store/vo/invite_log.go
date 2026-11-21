package vo

type InviteLog struct {
	Id         uint    `json:"id"`
	InviterId  uint    `json:"inviter_id"`
	UserId     uint    `json:"user_id"`
	Username   string  `json:"username"`
	InviteCode string  `json:"invite_code"`
	Remark     string  `json:"remark"`
	CreatedAt  int64   `json:"created_at"`
	NextCode   string  `json:"next_code"`
	IsJh       int     `json:"is_jh"`      //是否激活状态
	Yeji       float64 `json:"yeji"`       // 10代以内业绩
	DiamondLevel int   `json:"diamond_level"` // 钻石等级：0表示未激活，1-9表示1-9钻
	MemberTier     string `json:"member_tier"`     // inactive|normal|vip|legacy_vip
	MemberTierLabel string `json:"member_tier_label"`
}
