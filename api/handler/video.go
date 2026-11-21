package handler

import (
	"bytes"
	"chatplus/core"
	"chatplus/core/types"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type VideoHandler struct {
	BaseHandler
	client *http.Client
	config *types.DigiConfig
}

func NewVideoHandler(app *core.AppServer, config *types.AppConfig) *VideoHandler {
	if len(config.DigiConfig) == 0 {
		return &VideoHandler{
			BaseHandler: BaseHandler{App: app},
			client:      &http.Client{},
			config: &types.DigiConfig{
				ApiURL: "https://api.runwayml.com/v1", // Runway API URL
				ApiKey: "",
			},
		}
	}
	return &VideoHandler{
		BaseHandler: BaseHandler{App: app},
		client:      &http.Client{},
		config:      &config.DigiConfig[0],
	}
}

// Generate 生成视频
func (h *VideoHandler) Generate(c *gin.Context) {
	if deny, msg := DenyIfAppDisabled(h.App.SysConfig, AppFeatureVideo); deny {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  msg,
		})
		return
	}

	if h.config.ApiKey == "" {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "Runway API 未配置",
		})
		return
	}

	var req struct {
		Title       string `json:"title"`
		Prompt      string `json:"prompt"`
		Duration    int    `json:"duration"`
		Resolution  string `json:"resolution"`
		ImagePrompt string `json:"image_prompt,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "参数错误",
		})
		return
	}

	// 构建 Runway API 请求
	runwayReq := map[string]interface{}{
		"taskType": "gen3a", // 使用 Gen-3
		"internal": false,
		"options": map[string]interface{}{
			"name":           req.Title,
			"seconds":        req.Duration,
			"text_prompt":    req.Prompt,
			"seed":           getRandomInt(1375247627, 3975247627),
			"exploreMode":    true,
			"watermark":      false,
			"enhance_prompt": true,
			"resolution":     req.Resolution,
			"init_image":     req.ImagePrompt,
			"assetGroupName": "Generative Video",
		},
	}

	body, err := json.Marshal(runwayReq)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "请求失败",
		})
		return
	}

	request, err := http.NewRequest("POST", h.config.ApiURL+"/tasks", bytes.NewBuffer(body))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "请求失败",
		})
		return
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+h.config.ApiKey)

	resp, err := h.client.Do(request)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "生成失败",
		})
		return
	}
	defer resp.Body.Close()

	var result struct {
		Task struct {
			ID        string `json:"id"`
			Status    string `json:"status"`
			Artifacts []struct {
				URL string `json:"url"`
			} `json:"artifacts"`
			CreatedAt string `json:"createdAt"`
		} `json:"task"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "解析响应失败",
		})
		return
	}

	var videoURL string
	if len(result.Task.Artifacts) > 0 {
		videoURL = result.Task.Artifacts[0].URL
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": map[string]interface{}{
			"id":         result.Task.ID,
			"status":     result.Task.Status,
			"video_url":  videoURL,
			"created_at": result.Task.CreatedAt,
		},
	})
}

// GetStatus 获取视频生成状态
func (h *VideoHandler) GetStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "参数错误",
		})
		return
	}

	request, err := http.NewRequest("GET", fmt.Sprintf("%s/tasks/%s", h.config.ApiURL, id), nil)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "请求失败",
		})
		return
	}

	request.Header.Set("Authorization", "Bearer "+h.config.ApiKey)
	resp, err := h.client.Do(request)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "获取状态失败",
		})
		return
	}
	defer resp.Body.Close()

	var result struct {
		Task struct {
			ID        string `json:"id"`
			Status    string `json:"status"`
			Artifacts []struct {
				URL string `json:"url"`
			} `json:"artifacts"`
			CreatedAt     string `json:"createdAt"`
			Error         string `json:"error,omitempty"`
			ProgressRatio string `json:"progressRatio"`
		} `json:"task"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": -1,
			"msg":  "解析响应失败",
		})
		return
	}

	var videoURL string
	if len(result.Task.Artifacts) > 0 {
		videoURL = result.Task.Artifacts[0].URL
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": map[string]interface{}{
			"id":         result.Task.ID,
			"status":     result.Task.Status,
			"video_url":  videoURL,
			"created_at": result.Task.CreatedAt,
			"error":      result.Task.Error,
			"progress":   result.Task.ProgressRatio,
		},
	})
}

func getRandomInt(min, max int) int {
	return min + (max-min)/2 // 简化的随机数生成
}
