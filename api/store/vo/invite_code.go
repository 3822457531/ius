package vo

type InviteCode struct {
	Id        uint   `json:"id"`
	UserId    uint   `json:"user_id"`
	Code      string `json:"code"`
	Hits      int    `json:"hits"`
	RegNum    int    `json:"reg_num"`
	CreatedAt int64  `json:"created_at"`
	JhNum     int64  `json:"jh_num"`
	IsJh      int    `json:"is_jh"` // 0 未激活：无邀请码/二维码/外链；1 已激活
}
