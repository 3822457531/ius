package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const defaultLocale = "en-US"

var (
	geoHTTPClient = &http.Client{Timeout: 2 * time.Second}
	geoCache      sync.Map // ip -> geoCacheItem
	geoCacheTTL   = 24 * time.Hour
)

type geoCacheItem struct {
	country string
	expire  time.Time
}

// Ip2Locale 根据 IP 推荐界面语言：中国→zh-CN，日本→ja，韩国→kr-KR，其余→en-US。
// 优先使用传入的 countryHint（如 Cloudflare CF-IPCountry），否则走在线 Geo 查询。
func Ip2Locale(ip string, countryHint string) string {
	country := strings.ToUpper(strings.TrimSpace(countryHint))
	if country == "" || country == "XX" || country == "T1" {
		country = LookupCountryCode(ip)
	}
	return CountryToLocale(country)
}

// CountryToLocale 国家码/国名 → 界面语言
func CountryToLocale(country string) string {
	c := strings.ToUpper(strings.TrimSpace(country))
	switch c {
	case "CN", "中国", "HK", "MO", "TW": // 港澳台按中文
		return "zh-CN"
	case "JP", "日本":
		return "ja"
	case "KR", "韩国", "南韩", "大韩民国":
		return "kr-KR"
	default:
		return defaultLocale
	}
}

// LookupCountryCode 在线查询 IP 所属国家码（ISO 3166-1 alpha-2），失败返回空串。
func LookupCountryCode(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" || isPrivateOrLocalIP(ip) {
		return ""
	}

	if v, ok := geoCache.Load(ip); ok {
		item := v.(geoCacheItem)
		if time.Now().Before(item.expire) {
			return item.country
		}
		geoCache.Delete(ip)
	}

	country := lookupCountryOnline(ip)
	if country != "" {
		geoCache.Store(ip, geoCacheItem{country: country, expire: time.Now().Add(geoCacheTTL)})
	}
	return country
}

func lookupCountryOnline(ip string) string {
	if code := lookupByIPAPI(ip); code != "" {
		return code
	}
	return lookupByGeoJS(ip)
}

// ip-api.com（免费，无需 key）
func lookupByIPAPI(ip string) string {
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,countryCode", ip)
	resp, err := geoHTTPClient.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return ""
	}
	var data struct {
		Status      string `json:"status"`
		CountryCode string `json:"countryCode"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return ""
	}
	if data.Status != "success" {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(data.CountryCode))
}

// geojs.io 作为备用（HTTPS）
func lookupByGeoJS(ip string) string {
	url := fmt.Sprintf("https://get.geojs.io/v1/ip/country/%s.json", ip)
	resp, err := geoHTTPClient.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return ""
	}
	var data struct {
		Country string `json:"country"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(data.Country))
}

func isPrivateOrLocalIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
		return true
	}
	return false
}
