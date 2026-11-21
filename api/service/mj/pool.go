package mj

import (
	"chatplus/core/types"
	logger2 "chatplus/logger"
	"chatplus/service"
	"chatplus/service/oss"
	"chatplus/store"
	"chatplus/store/model"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"

	"gorm.io/gorm"
)

// ServicePool Mj service pool
type ServicePool struct {
	services        []*Service
	taskQueue       *store.RedisQueue
	notifyQueue     *store.RedisQueue
	db              *gorm.DB
	uploaderManager *oss.UploaderManager
	Clients         *types.LMap[uint, *types.WsClient] // UserId => Client
}

var logger = logger2.GetLogger()

func NewServicePool(db *gorm.DB, redisCli *redis.Client, manager *oss.UploaderManager, appConfig *types.AppConfig) *ServicePool {
	services := make([]*Service, 0)
	taskQueue := store.NewRedisQueue("MidJourney_Task_Queue", redisCli)
	notifyQueue := store.NewRedisQueue("MidJourney_Notify_Queue", redisCli)

	for k, config := range appConfig.MjPlusConfigs {
		if config.Enabled == false {
			continue
		}
		cli := NewPlusClient(config)
		name := fmt.Sprintf("mj-plus-service-%d", k)
		service := NewService(name, taskQueue, notifyQueue, db, cli, nil)
		go func() {
			service.Run()
		}()
		services = append(services, service)
	}

	for k, config := range appConfig.MjProxyConfigs {
		if config.Enabled == false {
			continue
		}
		cli := NewProxyClient(config)
		name := fmt.Sprintf("mj-proxy-service-%d", k)
		service := NewService(name, taskQueue, notifyQueue, db, cli, nil)
		go func() {
			service.Run()
		}()
		services = append(services, service)
	}

	// 添加 nano 客户端服务（总是启用，因为它是默认模型）
	nanoCli := NewNanoClient(db)
	nanoService := NewService("nano-service", taskQueue, notifyQueue, db, nanoCli, manager)
	go func() {
		nanoService.Run()
	}()
	services = append(services, nanoService)

	image2Cli := NewGptImage2Client(db)
	image2Service := NewService("image2-service", taskQueue, notifyQueue, db, image2Cli, manager)
	go func() {
		image2Service.Run()
	}()
	services = append(services, image2Service)

	return &ServicePool{
		taskQueue:       taskQueue,
		notifyQueue:     notifyQueue,
		services:        services,
		uploaderManager: manager,
		db:              db,
		Clients:         types.NewLMap[uint, *types.WsClient](),
	}
}

func (p *ServicePool) CheckTaskNotify() {
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

func (p *ServicePool) DownloadImages() {
	go func() {
		var items []model.MidJourneyJob
		for {
			// img_url 为空，或误存为外链时需从 org_url 拉取并落盘
			res := p.db.Where(
				"progress = ? AND org_url != '' AND (img_url = ? OR img_url LIKE 'http%')",
				100, "",
			).Find(&items)
			if res.Error != nil {
				continue
			}

			// download images
			for _, v := range items {
				sourceURL := strings.TrimSpace(v.OrgURL)
				if sourceURL == "" {
					continue
				}
				if strings.HasPrefix(strings.TrimSpace(v.ImgURL), "/static/upload/") {
					continue
				}

				logger.Infof("try to download image: %s", sourceURL)
				var imgURL string
				var err error
				if servicePlus := p.getService(v.ChannelId); servicePlus != nil {
					task, _ := servicePlus.Client.QueryTask(v.TaskId)
					if len(task.Buttons) > 0 {
						v.Hash = GetImageHash(task.Buttons[0].CustomId)
					}
					imgURL, err = p.uploaderManager.GetUploadHandler().PutImg(sourceURL, true)
				} else {
					imgURL, err = p.uploaderManager.GetUploadHandler().PutImg(sourceURL, true)
				}
				if err != nil {
					logger.Errorf("error with download image %s, %v", sourceURL, err)
					continue
				} else {
					logger.Infof("download image %s successfully.", sourceURL)
				}

				v.ImgURL = normalizeLocalImgURL(imgURL)
				if isRemoteImageURL(sourceURL) {
					v.OrgURL = sourceURL
				}
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
func (p *ServicePool) PushTask(task types.MjTask) {
	logger.Debugf("add a new MidJourney task to the task list: %+v", task)
	p.taskQueue.RPush(task)
}

// HasAvailableService check if it has available mj service in pool
func (p *ServicePool) HasAvailableService() bool {
	return len(p.services) > 0
}

// SyncTaskProgress 异步拉取任务
func (p *ServicePool) SyncTaskProgress() {
	go func() {
		var items []model.MidJourneyJob

		//获取进度条，然后更新进度条
		for {
			res := p.db.Where("progress < ? AND power > ?", 100, 0).Find(&items)

			if res.Error != nil {
				continue
			}

			for _, job := range items {
				// 失败或者 30 分钟还没完成的任务退回算力（幂等，参考音乐任务）
				if time.Now().Sub(job.CreatedAt) > time.Minute*30 || job.Progress == -1 {
					errMsg := strings.TrimSpace(job.ErrMsg)
					if errMsg == "" {
						if job.Progress == -1 {
							errMsg = "任务失败"
						} else {
							errMsg = "任务超时"
						}
					}
					modelName := ResolveImagePowerLogModel(job.Prompt, "gemini-mj")
					service.RefundMidJourneyJob(p.db, &job, modelName, errMsg)
					continue
				}

				if servicePlus := p.getService(job.ChannelId); servicePlus != nil {
					_ = servicePlus.Notify(job)
				}
			}

			time.Sleep(time.Second)
		}
	}()
}

func (p *ServicePool) getService(name string) *Service {
	for _, s := range p.services {
		if s.Name == name {
			return s
		}
	}
	return nil
}
