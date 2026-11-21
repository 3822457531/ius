package mj

import (
	"chatplus/core/types"
	"encoding/json"
	"fmt"
	"github.com/imroc/req/v3"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// Digital
type DigiClient struct {
	Config types.DigiConfig
	apiURL string
	client *req.Client
}

func NewDigiClient(config types.DigiConfig) *DigiClient {
	return &DigiClient{
		Config: config,
		apiURL: config.ApiURL,
		client: req.C().SetTimeout(time.Minute).SetUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"),
	}
}

// 获取所有的数字人参数
//func (c *DigiClient) GetParamter() string {
//	url := "https://ifaceai.com/api/ai/managerVideoTaskController/config/list"
//	method := "GET"
//
//	client := &http.Client{}
//	req, err := http.NewRequest(method, url, nil)
//
//	if err != nil {
//		fmt.Println(err)
//		return ""
//	}
//	req.Header.Add("Content-Type", "application/json")
//
//	res, err := client.Do(req)
//	if err != nil {
//		fmt.Println(err)
//		return ""
//	}
//	defer res.Body.Close()
//
//	body, err := ioutil.ReadAll(res.Body)
//	if err != nil {
//		fmt.Println(err)
//		return ""
//	}
//	fmt.Println(string(body))
//	return string(body)
//}

// // back
func (c *DigiClient) DigiVideo(task types.DigitalTask) (VideoRequest, error) {
	url := "https://ai.scloudapi.com/api/ai/managerVideoTaskController/task/start"
	//url := "https://ifaceai.com/api/ai/managerVideoTaskController/task/start"
	method := "POST"
	payload := fmt.Sprintf(`{
		"backgroundImageUrl": "%s",
		"content": "%s",
		"language": "%s",
		"roleName": "%s",
		"roleStyle": "%s",
		"backgroundType": "%s"
	}`, task.BgUrl, task.Content, task.Language, task.RoleName, task.RoleStyle, "IMG")

	logger.Info("task: ", task)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, strings.NewReader(payload))
	if err != nil {
		logger.Error("Failed to create request: ", err)
		return VideoRequest{}, err
	}

	req.Header.Add("accept", "*/*")
	req.Header.Add("Content-Type", "application/json")

	req.Header.Add("apiKey", "267f273a38f811efb88c0242c0a85003")

	res, err := client.Do(req)
	if err != nil {
		logger.Error("Failed to do request: ", err)
		return VideoRequest{}, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		logger.Error("Failed to read response body: ", err)
		return VideoRequest{}, err
	}

	logger.Info("Response Status: ", res.Status)
	logger.Info("Response Body: ", string(body))

	// 解析 JSON 响应体
	var jsonResponse map[string]interface{}
	if err := json.Unmarshal(body, &jsonResponse); err != nil {
		logger.Error("Failed to unmarshal response body: ", err)
		return VideoRequest{}, err
	}

	// 提取并转换数据
	data, _ := jsonResponse["data"].(string)
	code, _ := jsonResponse["code"].(float64)
	msg, _ := jsonResponse["msg"].(string)

	return VideoRequest{
		Data: data,
		Code: int(code),
		Msg:  msg,
	}, nil

	//	return VideoRequest{Data: string(body)}, nil
}

//// 实现 QueryVideo 方法
//func (c *DigiClient) QueryVideo(taskId string) (VideoResponse, error) {
//	url := fmt.Sprintf("https://ifaceai.com/api/ai/managerVideoTaskController/task/status?id=%s", taskId)
//	method := "POST"
//
//	client := &http.Client{}
//	req, err := http.NewRequest(method, url, nil)
//	if err != nil {
//		return VideoResponse{}, err
//	}
//
//	req.Header.Add("accept", "*/*")
//	req.Header.Add("apiKey", "267f273a38f811efb88c0242c0a85003")
//
//	res, err := client.Do(req)
//	if err != nil {
//		return VideoResponse{}, err
//	}
//	defer res.Body.Close()
//
//	body, err := ioutil.ReadAll(res.Body)
//	if err != nil {
//		return VideoResponse{}, err
//	}
//
//	logger.Info("原结果为", string(body))
//
//	// 解析 JSON 响应体
//	var jsonResponse map[string]interface{}
//	if err := json.Unmarshal(body, &jsonResponse); err != nil {
//		logger.Error("Failed to unmarshal response body: ", err)
//		return VideoResponse{}, err
//	}
//
//	// 提取并转换数据
//	data, _ := jsonResponse["data"].(string)
//	code, _ := jsonResponse["code"].(float64)
//	msg, _ := jsonResponse["msg"].(string)
//
//	return VideoResponse{
//		Data: data,
//		Code: int(code),
//		Msg:  msg,
//	}, nil
//
//
//}

// 实现 QueryVideo 方法
func (c *DigiClient) QueryVideo(taskId string) (string, error) {
	url := fmt.Sprintf("https://ai.scloudapi.com/api/ai/managerVideoTaskController/task/status?id=%s", taskId)
	//url := fmt.Sprintf("https://ifaceai.com/api/ai/managerVideoTaskController/task/status?id=%s", taskId)
	method := "POST"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("accept", "*/*")
	req.Header.Add("apiKey", "267f273a38f811efb88c0242c0a85003")

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	logger.Info("原结果为", string(body))

	return string(body), nil

	//// 解析 JSON 响应体
	//var jsonResponse map[string]interface{}
	//if err := json.Unmarshal(body, &jsonResponse); err != nil {
	//	logger.Error("Failed to unmarshal response body: ", err)
	//	return VideoResponse{}, err
	//}
	//
	//// 提取并转换数据
	//var data string
	//if v, ok := jsonResponse["data"].(string); ok {
	//	data = v
	//}
	//
	//code, _ := jsonResponse["code"].(float64)
	//msg, _ := jsonResponse["msg"].(string)
	//
	//return VideoResponse{
	//	Data: data,
	//	Code: int(code),
	//	Msg:  msg,
	//}, nil
}

//
//func (c *DigiClient) QueryVideo(taskId string) (VideoResponse, error) {
//	url := fmt.Sprintf("https://ifaceai.com/api/ai/managerVideoTaskController/task/status?id=%s", taskId)
//	method := "POST"
//
//	client := &http.Client{}
//	req, err := http.NewRequest(method, url, nil)
//	if err != nil {
//		return VideoResponse{}, err
//	}
//
//	req.Header.Add("accept", "*/*")
//	req.Header.Add("apiKey", "267f273a38f811efb88c0242c0a85003")
//
//	res, err := client.Do(req)
//	if err != nil {
//		return VideoResponse{}, err
//	}
//	defer res.Body.Close()
//
//	body, err := ioutil.ReadAll(res.Body)
//	if err != nil {
//		return VideoResponse{}, err
//	}
//
//	logger.Info("原结果为", string(body))
//
//	// 解析 JSON 响应体
//	var jsonResponse map[string]interface{}
//	if err := json.Unmarshal(body, &jsonResponse); err != nil {
//		logger.Error("Failed to unmarshal response body: ", err)
//		return VideoResponse{}, err
//	}
//
//	// 提取并转换数据
//	code, _ := jsonResponse["code"].(float64)
//	if code == 0 {
//		dataMap, _ := jsonResponse["data"].(map[string]interface{})
//		videoData := VideoData{
//			// 假设 data 包含一些 key-value 数据
//			// 你可以根据实际情况进行调整
//			VideoID:  dataMap["videoId"].(string),
//			Status:   dataMap["status"].(string),
//			FilePath: dataMap["filePath"].(string),
//			// 添加更多字段...
//		}
//		return VideoResponse{
//			Data: videoData,
//			Code: int(code),
//			Msg:  jsonResponse["msg"].(string),
//		}, nil
//	}
//
//	// code 不为 0 的情况
//	return VideoResponse{
//		Data: VideoData{},
//		Code: int(code),
//		Msg:  jsonResponse["msg"].(string),
//	}, nil
//}

//DigiVideo(task types.DigitalTask) (VideoRequest, error)
//QueryVideo(taskId string) (VideoResponse, error)

var _ Video = &DigiClient{}
