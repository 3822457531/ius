package chatimpl

import (
	"chatplus/core/types"
	"chatplus/store/model"
	"chatplus/utils"

	"github.com/gin-gonic/gin"
)

const builtinWebSearchTool = "web_search"

func (h *ChatHandler) attachOpenAITools(req *types.ApiRequest, webSearch bool) {
	if webSearch {
		req.Tools = []interface{}{
			gin.H{
				"type": "function",
				"function": gin.H{
					"name":        builtinWebSearchTool,
					"description": "调用 DeepSeek 网页搜索 API，获取实时新闻、热点与网页摘要。参数 query 为搜索词（1～128 字）。",
					"parameters": gin.H{
						"type": "object",
						"properties": gin.H{
							"query": gin.H{
								"type":        "string",
								"description": "搜索关键词，如：国内科技新闻 2026",
							},
						},
						"required": []string{"query"},
					},
				},
			},
		}
		req.ToolChoice = "auto"
		return
	}

	var items []model.Function
	res := h.DB.Where("enabled", true).Find(&items)
	if res.Error != nil {
		return
	}
	tools := make([]interface{}, 0)
	for _, v := range items {
		var parameters map[string]interface{}
		if err := utils.JsonDecode(v.Parameters, &parameters); err != nil {
			continue
		}
		required := parameters["required"]
		delete(parameters, "required")
		tools = append(tools, gin.H{
			"type": "function",
			"function": gin.H{
				"name":        v.Name,
				"description": v.Description,
				"parameters":  parameters,
				"required":    required,
			},
		})
	}
	if len(tools) > 0 {
		req.Tools = tools
		req.ToolChoice = "auto"
	}
}
