package teaching

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	volcSpeechHost    = "open.volcengineapi.com"
	volcSpeechService = "speech_saas_prod"
	volcSpeechRegion  = "cn-north-1"
	volcSpeechVersion = "2023-11-07"
	bigTTSResourceID  = "volc.service_type.10029"
)

// TTSQuotaPack 资源包余量摘要（OpenSpeech / 豆包语音大模型）
type TTSQuotaPack struct {
	Name       string  `json:"name"`
	ResourceID string  `json:"resource_id"`
	Type       string  `json:"type"`
	State      string  `json:"state"`
	Purchased  float64 `json:"purchased"`
	Used       float64 `json:"used"`
	Remaining  float64 `json:"remaining"`
	Unit       string  `json:"unit"`
	Expires    string  `json:"expires"`
}

// TTSQuotaResult OpenSpeech TTS 资源包查询结果
type TTSQuotaResult struct {
	ProjectName string         `json:"project_name"`
	Packs       []TTSQuotaPack `json:"packs"`
	Summary     string         `json:"summary"`
	RawError    string         `json:"raw_error,omitempty"`
}

type resourcePacksStatusResp struct {
	ResponseMetadata *struct {
		Error *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
	} `json:"ResponseMetadata"`
	Result *struct {
		Packs         []packItem         `json:"Packs"`
		TotalHarvests []totalHarvestItem `json:"TotalHarvests"`
	} `json:"Result"`
}

type packItem struct {
	ResourceDisplay *string `json:"ResourceDisplay"`
	ResourceID      *string `json:"ResourceID"`
	Type            *string `json:"Type"`
	State           *string `json:"State"`
	PurchasedAmount *string `json:"PurchasedAmount"`
	CurrentUsage    *string `json:"CurrentUsage"`
	Expires         *string `json:"Expires"`
}

type totalHarvestItem struct {
	ResourceID      *string  `json:"ResourceID"`
	CropType        *string  `json:"CropType"`
	PurchasedAmount *float64 `json:"PurchasedAmount"`
	CurrentUsage    *float64 `json:"CurrentUsage"`
	Unit            *string  `json:"Unit"`
}

// FetchTTSResourceQuota 查询字节 OpenSpeech / 豆包语音大模型资源包余量
func FetchTTSResourceQuota(accessKey, secretKey, projectName string) (*TTSQuotaResult, error) {
	accessKey = strings.TrimSpace(accessKey)
	secretKey = strings.TrimSpace(secretKey)
	projectName = strings.TrimSpace(projectName)
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("请先配置火山 IAM AccessKey 与 SecretKey（tts_volc_access_key / tts_volc_secret_key）")
	}
	if projectName == "" {
		projectName = "default"
	}

	body := map[string]interface{}{
		"ProjectName": projectName,
		"PageSize":    50,
		"PageNumber":  1,
		"Types":       []string{bigTTSResourceID},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("Action", "ResourcePacksStatus")
	query.Set("Version", volcSpeechVersion)

	req, err := http.NewRequest(http.MethodPost, "https://"+volcSpeechHost+"/?"+query.Encode(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	signVolcRequest(req, accessKey, secretKey, bodyBytes)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("火山 ResourcePacksStatus HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var parsed resourcePacksStatusResp
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("解析 ResourcePacksStatus 响应失败: %w", err)
	}
	if parsed.ResponseMetadata != nil && parsed.ResponseMetadata.Error != nil {
		e := parsed.ResponseMetadata.Error
		return nil, fmt.Errorf("火山 ResourcePacksStatus 错误 %s: %s", e.Code, e.Message)
	}

	result := &TTSQuotaResult{
		ProjectName: projectName,
		Packs:       make([]TTSQuotaPack, 0),
	}

	if parsed.Result != nil {
		for _, p := range parsed.Result.Packs {
			pack := packFromItem(p)
			if pack.Name == "" && pack.ResourceID == "" {
				continue
			}
			result.Packs = append(result.Packs, pack)
		}
		if len(result.Packs) == 0 && len(parsed.Result.TotalHarvests) > 0 {
			for _, h := range parsed.Result.TotalHarvests {
				result.Packs = append(result.Packs, packFromHarvest(h))
			}
		}
	}

	if len(result.Packs) == 0 {
		// 不带 Types 过滤再查一次（部分账号 ProjectName 与 AppID 不一致）
		if fallback, err2 := fetchTTSResourceQuotaAllTypes(accessKey, secretKey, projectName); err2 == nil && len(fallback.Packs) > 0 {
			result.Packs = fallback.Packs
		} else {
			result.Summary = "未查询到语音合成大模型资源包，请确认 ProjectName/AppID 与 IAM 密钥是否正确"
			return result, nil
		}
	}

	var totalRemain float64
	for i := range result.Packs {
		if strings.Contains(strings.ToLower(result.Packs[i].Type), "10029") ||
			strings.Contains(strings.ToLower(result.Packs[i].Name), "语音合成") ||
			strings.Contains(strings.ToLower(result.Packs[i].Name), "bigtts") ||
			result.Packs[i].ResourceID == bigTTSResourceID {
			totalRemain += result.Packs[i].Remaining
		}
	}
	if totalRemain > 0 {
		result.Summary = fmt.Sprintf("语音合成大模型资源包合计剩余约 %.0f 字", totalRemain)
	} else if len(result.Packs) > 0 {
		result.Summary = fmt.Sprintf("共 %d 个资源包，详见下表", len(result.Packs))
	}
	return result, nil
}

func fetchTTSResourceQuotaAllTypes(accessKey, secretKey, projectName string) (*TTSQuotaResult, error) {
	body := map[string]interface{}{
		"ProjectName": projectName,
		"PageSize":    50,
		"PageNumber":  1,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("Action", "ResourcePacksStatus")
	query.Set("Version", volcSpeechVersion)
	req, err := http.NewRequest(http.MethodPost, "https://"+volcSpeechHost+"/?"+query.Encode(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	signVolcRequest(req, accessKey, secretKey, bodyBytes)
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var parsed resourcePacksStatusResp
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	out := &TTSQuotaResult{ProjectName: projectName, Packs: make([]TTSQuotaPack, 0)}
	if parsed.Result == nil {
		return out, nil
	}
	for _, p := range parsed.Result.Packs {
		out.Packs = append(out.Packs, packFromItem(p))
	}
	return out, nil
}

func packFromItem(p packItem) TTSQuotaPack {
	name := strPtr(p.ResourceDisplay)
	if name == "" {
		name = strPtr(p.ResourceID)
	}
	purchased := parseAmount(strPtr(p.PurchasedAmount))
	used := parseAmount(strPtr(p.CurrentUsage))
	return TTSQuotaPack{
		Name:       name,
		ResourceID: strPtr(p.ResourceID),
		Type:       strPtr(p.Type),
		State:      strPtr(p.State),
		Purchased:  purchased,
		Used:       used,
		Remaining:  maxFloat(0, purchased-used),
		Unit:       "字",
		Expires:    strPtr(p.Expires),
	}
}

func packFromHarvest(h totalHarvestItem) TTSQuotaPack {
	purchased := float64(0)
	used := float64(0)
	if h.PurchasedAmount != nil {
		purchased = *h.PurchasedAmount
	}
	if h.CurrentUsage != nil {
		used = *h.CurrentUsage
	}
	name := strPtr(h.CropType)
	if name == "" {
		name = strPtr(h.ResourceID)
	}
	unit := strPtr(h.Unit)
	if unit == "" {
		unit = "字"
	}
	return TTSQuotaPack{
		Name:       name,
		ResourceID: strPtr(h.ResourceID),
		Type:       strPtr(h.ResourceID),
		State:      "active",
		Purchased:  purchased,
		Used:       used,
		Remaining:  maxFloat(0, purchased-used),
		Unit:       unit,
	}
}

func strPtr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func parseAmount(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func signVolcRequest(req *http.Request, ak, sk string, body []byte) {
	now := time.Now().UTC()
	xDate := now.Format("20060102T150405Z")
	shortDate := xDate[:8]
	payloadHash := sha256Hex(body)
	contentType := "application/json; charset=utf-8"

	req.Header.Set("Host", volcSpeechHost)
	req.Header.Set("X-Date", xDate)
	req.Header.Set("X-Content-Sha256", payloadHash)
	req.Header.Set("Content-Type", contentType)

	canonicalQuery := normVolcQuery(req.URL.Query())
	canonicalHeaders := strings.Join([]string{
		"content-type:" + contentType,
		"host:" + volcSpeechHost,
		"x-content-sha256:" + payloadHash,
		"x-date:" + xDate,
	}, "\n") + "\n"
	signedHeaders := "content-type;host;x-content-sha256;x-date"
	canonicalRequest := strings.Join([]string{
		req.Method,
		"/",
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	credentialScope := strings.Join([]string{shortDate, volcSpeechRegion, volcSpeechService, "request"}, "/")
	stringToSign := strings.Join([]string{
		"HMAC-SHA256",
		xDate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	kDate := hmacSHA256([]byte(sk), shortDate)
	kRegion := hmacSHA256(kDate, volcSpeechRegion)
	kService := hmacSHA256(kRegion, volcSpeechService)
	kSigning := hmacSHA256(kService, "request")
	signature := hex.EncodeToString(hmacSHA256(kSigning, stringToSign))

	req.Header.Set("Authorization", fmt.Sprintf(
		"HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		ak, credentialScope, signedHeaders, signature,
	))
}

func normVolcQuery(v url.Values) string {
	if len(v) == 0 {
		return ""
	}
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		vals := v[k]
		sort.Strings(vals)
		for _, val := range vals {
			parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(val))
		}
	}
	return strings.Join(parts, "&")
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, content string) []byte {
	m := hmac.New(sha256.New, key)
	_, _ = m.Write([]byte(content))
	return m.Sum(nil)
}
