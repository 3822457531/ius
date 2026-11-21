package types

// TaskType 任务类别
type TaskType string

func (t TaskType) String() string {
	return string(t)
}

const (
	TaskImage     = TaskType("image")
	TaskBlend     = TaskType("blend")
	TaskSwapFace  = TaskType("swapFace")
	TaskUpscale   = TaskType("upscale")
	TaskVariation = TaskType("variation")
)

// MjTask MidJourney 任务
type MjTask struct {
	Id          uint     `json:"id"`
	TaskId      string   `json:"task_id"`
	ImgArr      []string `json:"img_arr"`
	ChannelId   string   `json:"channel_id"`
	SessionId   string   `json:"session_id"`
	Type        TaskType `json:"type"`
	UserId      int      `json:"user_id"`
	Prompt      string   `json:"prompt,omitempty"`
	NegPrompt   string   `json:"neg_prompt,omitempty"`
	Params      string   `json:"full_prompt"`
	Index       int      `json:"index,omitempty"`
	MessageId   string   `json:"message_id,omitempty"`
	MessageHash string   `json:"message_hash,omitempty"`
	RetryCount  int      `json:"retry_count"`
}

// 数字人视频
type DigitalTask struct {
	Id         uint     `json:"id"`
	TaskId     string   `json:"task_id"`
	SessionId  string   `json:"session_id"`
	Type       TaskType `json:"type"`
	UserId     int      `json:"user_id"`
	BgUrl      string   `json:"bg_url"`
	Language   string   `json:"language"`  // 语言
	RoleName   string   `json:"roleName"`  // 角色
	RoleStyle  string   `json:"roleStyle"` // 角色形象
	Content    string   `json:"content,omitempty"`
	Params     string   `json:"full_prompt"`
	Index      int      `json:"index,omitempty"`
	RetryCount int      `json:"retry_count"`
}

type SdTask struct {
	Id         int          `json:"id"` // job 数据库ID
	SessionId  string       `json:"session_id"`
	Type       TaskType     `json:"type"`
	UserId     int          `json:"user_id"`
	Params     SdTaskParams `json:"params"`
	RetryCount int          `json:"retry_count"`
}

type SdTaskParams struct {
	TaskId       string  `json:"task_id"`
	Prompt       string  `json:"prompt"`     // 提示词
	NegPrompt    string  `json:"neg_prompt"` // 反向提示词
	Steps        int     `json:"steps"`      // 迭代步数，默认20
	Sampler      string  `json:"sampler"`    // 采样器
	FaceFix      bool    `json:"face_fix"`   // 面部修复
	CfgScale     float32 `json:"cfg_scale"`  //引导系数，默认 7
	Seed         int64   `json:"seed"`       // 随机数种子
	Height       int     `json:"height"`
	Width        int     `json:"width"`
	HdFix        bool    `json:"hd_fix"`         // 启用高清修复
	HdRedrawRate float32 `json:"hd_redraw_rate"` // 高清修复重绘幅度
	HdScale      int     `json:"hd_scale"`       // 放大倍数
	HdScaleAlg   string  `json:"hd_scale_alg"`   // 放大算法
	HdSteps      int     `json:"hd_steps"`       // 高清修复迭代步数
}

// Sora2Task Sora2 视频生成任务
type Sora2Task struct {
	ClientId string      `json:"client_id"`
	Id       uint        `json:"id"`
	Channel  string      `json:"channel"`
	UserId   int         `json:"user_id"`
	TaskId   string      `json:"task_id"`
	Prompt   string      `json:"prompt"` // 提示词
	Images   []string    `json:"images"` // 图生视频的图片数组
	Params   Sora2Params `json:"params"`
	//UpStatus         int         `json:"up_status"`          //上传状态
	TranslateModelId int `json:"translate_model_id"` // 提示词翻译模型ID
}

// Sora2Params Sora2参数
type Sora2Params struct {
	Duration       string `json:"duration"`        // 视频时长: 5, 10, 15, 30
	AspectRatio    string `json:"aspect_ratio"`    // 视频比例: 16:9, 9:16, 1:1
	Quality        string `json:"quality"`         // 视频质量: standard, hd, uhd
	Style          string `json:"style"`           // 视频风格: realistic, animated, artistic, cinematic
	NegativePrompt string `json:"negative_prompt"` // 负面提示词
	Seed           int64  `json:"seed"`            // 随机种子
	Steps          int    `json:"steps"`           // 生成步数
	Model          string `json:"model"`           // 模型: sora-2, sora-2-pro
}
