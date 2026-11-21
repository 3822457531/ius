package mj

import "chatplus/core/types"

type Video interface {
	DigiVideo(task types.DigitalTask) (VideoRequest, error)
	QueryVideo(taskId string) (string, error)
}

type VideoRequest struct {
	Data  string   `json:"data"`
	Code  int      `json:"code"`
	Msg   string   `json:"msg"`
	Extra struct{} `json:"extra"`
}

type VideoResponse struct {
	Data  string   `json:"data"`
	Code  int      `json:"code"`
	Msg   string   `json:"msg"`
	Extra struct{} `json:"extra"`
}

type fuilResponse struct {
	Data  struct{} `json:"data"`
	Code  int      `json:"code"`
	Msg   string   `json:"msg"`
	Extra struct{} `json:"extra"`
}

//"status": "Succeeded",
//"remark": null,
//"videoFileUrl": "https://www.ifaceai.com/ai/assets/video/2c9a808290748325019078f17d470002_water.mp4",
//"durationSecond": 38,
//"currentMoneyFree": 20,
//"createAt": "2024-07-03 22:13:30",
//"completeAt": "2024-07-03 22:15:54"

type VideoData struct {
	Status           string `json:"status"`
	Remark           string `json:"remark"`
	VideoFileUrl     string `json:"videoFileUrl"`
	CreateAt         string `json:"createAt"`
	CompleteAt       string `json:"completeAt"`
	CurrentMoneyFree string `json:"currentMoneyFree"`
}

type ErrRes struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
