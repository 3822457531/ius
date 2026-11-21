package mj

import (
	"chatplus/core/types"
	"chatplus/store"
	"chatplus/store/model"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
)

// Service MJ 绘画服务
type Digitalervice struct {
	Name        string // service Name
	Video       Video  // MJ Client
	taskQueue   *store.RedisQueue
	notifyQueue *store.RedisQueue
	db          *gorm.DB
}

func NewDigitalervice(name string, taskQueue *store.RedisQueue, notifyQueue *store.RedisQueue, db *gorm.DB, cli Video) *Digitalervice {
	return &Digitalervice{
		Name:        name,
		db:          db,
		taskQueue:   taskQueue,
		notifyQueue: notifyQueue,
		Video:       cli,
	}
}

func (s *Digitalervice) Run() {
	logger.Infof("数字人任务 for %s", s.Name)
	for {
		var task types.DigitalTask
		err := s.taskQueue.LPop(&task)
		if err != nil {
			logger.Errorf("taking task with error: %v", err)
			continue
		}

		var job model.DigiJob
		tx := s.db.Where("id = ?", task.Id).First(&job)
		if tx.Error != nil {
			logger.Error("任务不存在，任务ID：", task.TaskId)
			continue
		}
		//
		//videoRequest, err := s.Video.DigiVideo(task)
		//if err != nil {
		//	logger.Error("Error in DigiVideo: ", err)
		//} else {
		//	logger.Info("Video Request Data: ", videoRequest.Data)
		//}
		//
		//logger.Infof("%s 执行数字人提交 task: %+v", s.Name, task)
		//var res VideoRequest

		res, err := s.Video.DigiVideo(task)
		logger.Info(res.Data)
		if err != nil || res.Code != 0 || res.Data == "" {
			errMsg := fmt.Sprintf("%v, %s", err, res.Msg)
			logger.Error("数字人任务执行失败：", errMsg)
			job.Progress = -1
			job.ErrMsg = errMsg
			// update the task progress
			s.db.Updates(&job)
			// 任务失败，通知前端
			s.notifyQueue.RPush(task.UserId)
			continue
		} else {
			logger.Info("数字人任务执行成功：", res.Data)
			// 这里可以添加任务成功的逻辑，例如更新任务进度
			job.Progress = 20 //
			job.StatusId = res.Data
			job.ErrMsg = ""
			s.db.Updates(&job)
		}

		logger.Infof("任务提交成功：%+v", res)
		// 更新任务 ID/频道
		job.TaskId = res.Data
		job.StatusId = res.Data
		//job.MessageId = res.Result
		//job.ChannelId = s.Name
		s.db.Updates(&job)
	}
}

func (s *Digitalervice) Notify(job model.DigiJob) error {
	if len(job.StatusId) == 0 {
		logger.Info("StatusId为空")
		return nil
	}

	task, err := s.Video.QueryVideo(job.StatusId)

	logger.Info("获取数字人Notify")

	logger.Info("TaskId", job.StatusId)
	logger.Info("返回的结果", task)
	if err != nil {
		return err
	}

	type Response struct {
		Data struct {
			Status           string `json:"status"`
			Remark           string `json:"remark"`
			VideoFileUrl     string `json:"videoFileUrl"`
			DurationSecond   int    `json:"durationSecond"`
			CurrentMoneyFree int    `json:"currentMoneyFree"`
			CreateAt         string `json:"createAt"`
			CompleteAt       string `json:"completeAt"`
			PreviewFileUrl   string `json:"previewFileUrl"`
		} `json:"data"`
		Code  int         `json:"code"`
		Msg   interface{} `json:"msg"`
		Extra struct{}    `json:"extra"`
	}

	jsonStr1 := task

	// Unmarshal the JSON strings into the struct
	var res1 Response
	json.Unmarshal([]byte(jsonStr1), &res1)
	//解析返回的结构
	// Make judgments based on the "data" field
	if res1.Data.Status != "" {
		//	fmt.Println("数字人生成成功")
		if res1.Data.Status == "Succeeded" {
			fmt.Println("Status:", res1.Data.Status)
			fmt.Println("VideoFileUrl:", res1.Data.VideoFileUrl)
			fmt.Printf("task succeeded with status: %v\n", res1.Data.Status)

			s.db.Model(&model.DigiJob{Id: job.Id}).UpdateColumns(map[string]interface{}{
				"progress":  100,
				"video_url": res1.Data.VideoFileUrl,
				"status":    res1.Data.Status,
				"pre_url":   res1.Data.PreviewFileUrl,
				//"complete_at": res1.Data.CompleteAt,
			})

			s.notifyQueue.RPush(job.UserId)
		}
	}

	//不考虑失败情况
	//else {
	//	fmt.Println("数字人生成失败")
	//	s.db.Model(&model.DigiJob{Id: job.Id}).UpdateColumns(map[string]interface{}{
	//		"progress": -1,
	//		"err_msg":  res1.Msg,
	//		"status":   res1.Data.Status,
	//	})
	//	s.notifyQueue.RPush(job.UserId)
	//	return fmt.Errorf("task failed: %v", res1.Msg)
	//}

	// 判断 task.Data 是否为 JSON 格式，并获取 data.status 的值
	//status, err := getStatusFromJSON(task.Data)

	//oldProgress := job.Progress
	//logger.Info("进度条", oldProgress)
	//
	//job.Progress = utils.IntValue(strings.Replace(task.Progress, "%", "", 1), 0)
	//job.Prompt = task.PromptEn
	//if task.ImageUrl != "" {
	//	job.OrgURL = task.ImageUrl
	//}
	//tx := s.db.Updates(&job)
	//if tx.Error != nil {
	//	return fmt.Errorf("error with update database: %v", tx.Error)
	//}
	//// 通知前端更新任务进度
	//if oldProgress != job.Progress {
	//	s.notifyQueue.RPush(job.UserId)
	//}
	return nil
}

// isJSON 函数判断给定的字符串是否为 JSON 格式
func isJSON(data string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(data), &js) == nil
}

// getStatusFromJSON 从 JSON 字符串中获取 data.status 的值
func getStatusFromJSON(data string) (string, error) {
	if !isJSON(data) {
		return "", fmt.Errorf("not a valid JSON string")
	}

	var parsedData map[string]interface{}
	if err := json.Unmarshal([]byte(data), &parsedData); err != nil {
		return "", err
	}

	dataField, ok := parsedData["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("data field not found or invalid")
	}

	status, ok := dataField["status"].(string)
	if !ok {
		return "", fmt.Errorf("status field not found or invalid")
	}

	return status, nil
}
