package mj

import (
	"chatplus/core/types"
	logger2 "chatplus/logger"
	"chatplus/service/oss"
	"chatplus/store"
	"chatplus/store/model"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
	"time"
)

// ServicePool
type DigiServicePool struct {
	services        []*Digitalervice
	taskQueue       *store.RedisQueue
	notifyQueue     *store.RedisQueue
	db              *gorm.DB
	uploaderManager *oss.UploaderManager
	Clients         *types.LMap[uint, *types.WsClient] // UserId => Client
}

var loggertest = logger2.GetLogger()

func NewDigiServicePool(db *gorm.DB, redisCli *redis.Client, manager *oss.UploaderManager, appConfig *types.AppConfig) *DigiServicePool {
	services := make([]*Digitalervice, 0)
	taskQueue := store.NewRedisQueue("digi_Task_Queue", redisCli)
	notifyQueue := store.NewRedisQueue("digi_Notify_Queue", redisCli)

	for k, config := range appConfig.DigiConfig {
		if config.Enabled == false {
			continue
		}
		//cli := NewDigiClient(config)
		cli := NewDigiClient(config)
		//	//cli := (config)

		name := fmt.Sprintf("digi_service-%d", k)
		service := NewDigitalervice(name, taskQueue, notifyQueue, db, cli)
		go func() {
			service.Run()
		}()
		services = append(services, service)
	}

	for k, config := range appConfig.DigiConfig {
		if config.Enabled == false {
			continue
		}
		//cli := NewProxyClient(config)
		cli := NewDigiClient(config)
		name := fmt.Sprintf("digi-service-%d", k)
		service := NewDigitalervice(name, taskQueue, notifyQueue, db, cli)
		go func() {
			service.Run()
		}()
		services = append(services, service)
	}

	return &DigiServicePool{
		taskQueue:       taskQueue,
		notifyQueue:     notifyQueue,
		services:        services,
		uploaderManager: manager,
		db:              db,
		Clients:         types.NewLMap[uint, *types.WsClient](),
	}
}

func (p *DigiServicePool) CheckTaskNotify() {
	go func() {
		for {
			var userId uint
			err := p.notifyQueue.LPop(&userId)
			if err != nil {
				continue
			}
			cli := p.Clients.Get(userId)
			if cli == nil {
				continue
			}
			err = cli.Send([]byte("Task Updated"))
			if err != nil {
				continue
			}
		}
	}()
}

func (p *DigiServicePool) DownloadVideo() {
	go func() {
		var items []model.DigiJob
		for {
			res := p.db.Where("img_url = ? AND progress = ?", "", 100).Find(&items)
			if res.Error != nil {
				continue
			}

			// download images
			for _, v := range items {
				//videoFileUrl
				if v.VideoUrl == "" {
					continue
				}

				logger.Infof("下载数字人视频: %s", v.VideoUrl)
				//var imgURL string
				var err error
				if servicePlus := p.getService("digi_service-0"); servicePlus != nil {
					task, _ := servicePlus.Video.QueryVideo(v.StatusId)

					status, _ := getStatusFromJSON(task)
					if status == "Succeeded" {
						fmt.Printf("数字人执行结果 succeeded with status: %v\n", status)
						//s.notifyQueue.RPush(job.UserId) //更新前端执行结果
					}

					//	imgURL, err = p.uploaderManager.GetUploadHandler().PutImg(v.OrgURL, false)
					//	file, err:= p.uploaderManager.GetUploadHandler().PutFile(c,"file")
					//file, err := p.uploaderManager.GetUploadHandler().PutFile(c, "file") 下载数字人视频
				}
				if err != nil {
					//logger.Errorf("error with download image %s, %v", v.OrgURL, err)
					continue
				} else {
					logger.Infof("下载视频失败 %s successfully.", v.VideoUrl)
				}

				//v.ImgURL = imgURL
				p.db.Updates(&v)

				cli := p.Clients.Get(uint(v.UserId))
				if cli == nil {
					continue
				}
				err = cli.Send([]byte("Task Updated"))
				if err != nil {
					continue
				}
			}

			time.Sleep(time.Second * 5)
		}
	}()
}

// PushTask push a new mj task in to task queue
func (p *DigiServicePool) PushTask(task types.DigitalTask) {
	logger.Debugf("add a new MidJourney task to the task list: %+v", task)
	p.taskQueue.RPush(task)
}

// PushTask push a new DigiTask
func (p *DigiServicePool) PushDigiTask(task types.DigitalTask) {
	logger.Debugf("add a new Digi task to the task list: %+v", task)
	p.taskQueue.RPush(task)
}

// HasAvailableService check if it has available mj service in pool
func (p *DigiServicePool) HasAvailableService() bool {
	return len(p.services) > 0
}

// SyncTaskProgress 异步拉取任务
func (p *DigiServicePool) SyncTaskProgress() {

	logger.Info("进入数字人异步调用")

	go func() {
		//var items []model.MidJourneyJob
		var items []model.DigiJob

		//获取进度条，然后更新进度条
		//
		logger.Info("获取数字人任务")
		for {
			res := p.db.Where("progress < ?", 100).Find(&items)

			//logger.Info("异步获取绘画任务")
			//logger.Info(items)
			//
			//logger.Info(res)
			//
			//logger.Info(items)
			if res.Error != nil {
				continue
			}

			for _, job := range items {
				// 失败或者 30 分钟还没完成的任务删除并退回算力
				if time.Now().Sub(job.CreatedAt) > time.Minute*30 || job.Progress == -1 {
					// 删除任务
					p.db.Delete(&job)
					// 退回算力
					tx := p.db.Model(&model.User{}).Where("id = ?", job.UserId).UpdateColumn("power", gorm.Expr("power + ?", job.Power))
					if tx.Error == nil && tx.RowsAffected > 0 {
						var user model.User
						p.db.Where("id = ?", job.UserId).First(&user)
						p.db.Create(&model.PowerLog{
							UserId:    user.Id,
							Username:  user.Username,
							Type:      types.PowerConsume,
							Amount:    job.Power,
							Balance:   user.Power + job.Power,
							Mark:      types.PowerAdd,
							Model:     "数字人",
							Remark:    fmt.Sprintf("数字人任务失败，退回算力。任务ID：%s", job.TaskId),
							CreatedAt: time.Now(),
						})
					}
					continue
				}
				//	logger.Info("获取数字人任务")
				//servicePlus := p.getService(job.TaskId)
				//	logger.Info(servicePlus)
				//如果当前有小于100的进度条，则直接进入
				if servicePlus := p.getService("digi_service-0"); servicePlus != nil {
					logger.Info("进入回调状态")
					_ = servicePlus.Notify(job)
				}
			}

			time.Sleep(time.Second)
		}
	}()
}

func (p *DigiServicePool) getService(name string) *Digitalervice {

	for _, s := range p.services {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// isJSON 函数判断给定的字符串是否为 JSON 格式
func isjson(data string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(data), &js) == nil
}

// getStatusFromJSON 从 JSON 字符串中获取 data.status 的值
func getStatus(data string) (string, error) {
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
