package mj

import "strings"

// resolveApiKeyURL 任务 ChannelId 在队列中多为服务名（image2-service、nano-service），
// 仅当其为完整 API 根地址时才用于按 api_url 筛选 Key。
func resolveApiKeyURL(channel string) string {
	channel = strings.TrimSpace(channel)
	if strings.HasPrefix(channel, "http://") || strings.HasPrefix(channel, "https://") {
		return channel
	}
	return ""
}
