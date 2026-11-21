package teaching

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"chatplus/core/types"
	"chatplus/utils"
)

// VideoAsset 教学记录中的单张配图
type VideoAsset struct {
	SegmentID   string `json:"segmentId"`
	SegmentIDSn string `json:"segment_id"`
	ImageURL    string `json:"imageUrl"`
	ImageURLSn  string `json:"image_url"`
}

func (a VideoAsset) segmentKey() string {
	s := strings.TrimSpace(a.SegmentID)
	if s != "" {
		return s
	}
	return strings.TrimSpace(a.SegmentIDSn)
}

func (a VideoAsset) imageURL() string {
	s := strings.TrimSpace(a.ImageURL)
	if s != "" {
		return s
	}
	return strings.TrimSpace(a.ImageURLSn)
}

// VideoTTSItem 教学记录中的 TTS 条目
type VideoTTSItem struct {
	SegmentID   string `json:"segmentId"`
	SegmentIDSn string `json:"segment_id"`
	URL         string `json:"url"`
	DurationSec int    `json:"durationSec"`
}

func (t VideoTTSItem) segmentKey() string {
	s := strings.TrimSpace(t.SegmentID)
	if s != "" {
		return s
	}
	return strings.TrimSpace(t.SegmentIDSn)
}

// ParentSegmentID 子分镜 seg-1__2 → seg-1
func ParentSegmentID(segmentID string) string {
	s := strings.TrimSpace(segmentID)
	if i := strings.Index(s, "__"); i > 0 {
		return s[:i]
	}
	return s
}

func resolveFFmpegBin(cfg types.FFmpegConfig) (ffmpegBin, ffprobeBin string, err error) {
	ffmpegBin, err = resolveBinary(strings.TrimSpace(cfg.FFmpegPath), "ffmpeg")
	if err != nil {
		return "", "", err
	}
	ffprobeBin, err = resolveBinary(strings.TrimSpace(cfg.FFprobePath), "ffprobe")
	if err != nil {
		return "", "", err
	}
	return ffmpegBin, ffprobeBin, nil
}

func resolveBinary(configured, defaultName string) (string, error) {
	if configured != "" {
		if _, statErr := os.Stat(configured); statErr != nil {
			return "", fmt.Errorf("未找到 %s（config.toml [FFmpeg]）: %w", configured, statErr)
		}
		return configured, nil
	}
	path, lookErr := exec.LookPath(defaultName)
	if lookErr != nil {
		return "", fmt.Errorf("未找到 %s，请在 config.toml [FFmpeg] 中配置路径或加入 PATH", defaultName)
	}
	return path, nil
}

func probeDurationSeconds(ctx context.Context, ffprobeBin, mediaPath string) (float64, error) {
	cmd := exec.CommandContext(ctx, ffprobeBin,
		"-v", "error", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", mediaPath)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %w", err)
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return 0, fmt.Errorf("ffprobe 无输出")
	}
	sec, err := strconv.ParseFloat(s, 64)
	if err != nil || sec <= 0 {
		return 0, fmt.Errorf("无效时长: %q", s)
	}
	return sec, nil
}

func resolveLocalFromURL(staticDir, u string) (string, error) {
	u = strings.TrimSpace(u)
	if u == "" {
		return "", fmt.Errorf("空地址")
	}
	if strings.HasPrefix(u, "/static/") {
		rel := strings.TrimPrefix(u, "/static/")
		return filepath.Join(staticDir, filepath.FromSlash(rel)), nil
	}
	if strings.HasPrefix(u, "file://") {
		p, err := url.Parse(u)
		if err != nil {
			return "", err
		}
		path := p.Path
		if len(path) >= 3 && path[0] == '/' && path[2] == ':' {
			path = path[1:]
		}
		return filepath.FromSlash(path), nil
	}
	return "", fmt.Errorf("仅支持 /static/ 下的本地文件，当前: %s", u)
}

// resolveImageInput 配图：/static/ 走本地；http(s) 需下载
func resolveImageInput(staticDir, u string) (localPath string, tempCopy bool, err error) {
	u = strings.TrimSpace(u)
	if u == "" {
		return "", false, fmt.Errorf("空配图地址")
	}
	if strings.HasPrefix(u, "/static/") {
		p, e := resolveLocalFromURL(staticDir, u)
		return p, false, e
	}
	if strings.HasPrefix(strings.ToLower(u), "http://") || strings.HasPrefix(strings.ToLower(u), "https://") {
		return u, true, nil
	}
	return "", false, fmt.Errorf("不支持的配图地址: %s", u)
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func downloadFile(ctx context.Context, client *http.Client, rawURL, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载 %s: HTTP %d", rawURL, resp.StatusCode)
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func runFFmpeg(ctx context.Context, ffmpegBin, workDir string, args ...string) error {
	cmd := exec.CommandContext(ctx, ffmpegBin, args...)
	cmd.Dir = workDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runFFmpegAbs(ctx context.Context, ffmpegBin string, args ...string) error {
	cmd := exec.CommandContext(ctx, ffmpegBin, args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RenderTeachingMP4 将配图序列与对应父段 TTS 合成 MP4（一父多图时均分该父段音频时长）。需 ffmpeg、ffprobe（config.toml [FFmpeg] 或 PATH）；TTS 须为已落盘到 /static/ 的本地路径。
func RenderTeachingMP4(staticDir, basePath, baseURL string, ffmpegCfg types.FFmpegConfig, assets []VideoAsset, tts []VideoTTSItem) (publicURL string, err error) {
	if len(assets) == 0 {
		return "", fmt.Errorf("没有配图，无法合成视频")
	}
	ffmpegBin, ffprobeBin, err := resolveFFmpegBin(ffmpegCfg)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(basePath) == "" || strings.TrimSpace(baseURL) == "" {
		return "", fmt.Errorf("未配置本地 OSS BasePath/BaseURL，无法写入成片")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()

	workDir, err := os.MkdirTemp("", "teaching-video-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(workDir)

	ttsBySeg := make(map[string]VideoTTSItem)
	for _, t := range tts {
		if k := t.segmentKey(); k != "" {
			ttsBySeg[k] = t
		}
	}

	var parentOrder []string
	seen := make(map[string]bool)
	for _, a := range assets {
		p := ParentSegmentID(a.segmentKey())
		if p == "" {
			continue
		}
		if !seen[p] {
			seen[p] = true
			parentOrder = append(parentOrder, p)
		}
	}
	if len(parentOrder) == 0 {
		return "", fmt.Errorf("无法解析分镜父段")
	}

	countByParent := make(map[string]int)
	for _, a := range assets {
		countByParent[ParentSegmentID(a.segmentKey())]++
	}

	durByParent := make(map[string]float64)
	resolvedTTS := make(map[string]VideoTTSItem)
	client := &http.Client{Timeout: 5 * time.Minute}
	for _, p := range parentOrder {
		it, ok := ttsBySeg[p]
		if !ok || strings.TrimSpace(it.URL) == "" {
			for _, t := range tts {
				if ParentSegmentID(t.segmentKey()) == p && strings.TrimSpace(t.URL) != "" {
					it, ok = t, true
					break
				}
			}
		}
		if !ok || strings.TrimSpace(it.URL) == "" {
			return "", fmt.Errorf("缺少父段 %s 的 TTS 音频", p)
		}
		resolvedTTS[p] = it
		localPath, e := resolveLocalFromURL(staticDir, it.URL)
		if e != nil {
			return "", fmt.Errorf("父段 %s: %w", p, e)
		}
		if _, e := os.Stat(localPath); e != nil {
			return "", fmt.Errorf("父段 %s 音频文件不存在: %s", p, localPath)
		}
		sec, e := probeDurationSeconds(ctx, ffprobeBin, localPath)
		if e != nil || sec <= 0 {
			sec = 8.0
		}
		durByParent[p] = sec
	}

	type shot struct {
		imgPath string
		sec     float64
	}
	var shots []shot
	for i, a := range assets {
		p := ParentSegmentID(a.segmentKey())
		n := countByParent[p]
		if n <= 0 {
			continue
		}
		total := durByParent[p]
		per := total / float64(n)
		idxInParent := 0
		for j := 0; j < i; j++ {
			if ParentSegmentID(assets[j].segmentKey()) == p {
				idxInParent++
			}
		}
		d := per
		if idxInParent == n-1 {
			used := per * float64(n-1)
			d = total - used
			if d < 0.1 {
				d = 0.1
			}
		}

		imgU := a.imageURL()
		ext := strings.ToLower(filepath.Ext(imgU))
		if ext == "" || len(ext) > 6 {
			ext = ".jpg"
		}
		imgPath := filepath.Join(workDir, fmt.Sprintf("src_%04d%s", i, ext))
		src, needDL, e := resolveImageInput(staticDir, imgU)
		if e != nil {
			return "", fmt.Errorf("配图 %s: %w", a.segmentKey(), e)
		}
		if needDL {
			if err = downloadFile(ctx, client, src, imgPath); err != nil {
				return "", fmt.Errorf("下载配图 %s: %w", a.segmentKey(), err)
			}
		} else {
			if err = copyFile(src, imgPath); err != nil {
				return "", fmt.Errorf("复制配图 %s: %w", a.segmentKey(), err)
			}
		}
		shots = append(shots, shot{imgPath: imgPath, sec: d})
	}

	for i, sh := range shots {
		outClip := filepath.Join(workDir, fmt.Sprintf("clip_%04d.mp4", i))
		d := sh.sec
		if d < 0.2 {
			d = 0.2
		}
		err = runFFmpeg(ctx, ffmpegBin, workDir,
			"-y", "-loop", "1", "-i", filepath.Base(sh.imgPath),
			"-c:v", "libx264", "-t", fmt.Sprintf("%.6f", d),
			"-pix_fmt", "yuv420p",
			"-vf", "scale=1280:720:force_original_aspect_ratio=decrease,pad=1280:720:(ow-iw)/2:(oh-ih)/2,fps=25",
			"-movflags", "+faststart",
			filepath.Base(outClip),
		)
		if err != nil {
			return "", fmt.Errorf("生成视频片段 %d: %w", i, err)
		}
	}

	var listLines []string
	for i := range shots {
		p := filepath.Join(workDir, fmt.Sprintf("clip_%04d.mp4", i))
		line := "file " + escapeConcatPath(filepath.ToSlash(p))
		listLines = append(listLines, line)
	}
	listPath := filepath.Join(workDir, "concat_v.txt")
	if err = os.WriteFile(listPath, []byte(strings.Join(listLines, "\n")+"\n"), 0644); err != nil {
		return "", err
	}
	videoOnly := filepath.Join(workDir, "video_only.mp4")
	if err = runFFmpeg(ctx, ffmpegBin, workDir, "-y", "-f", "concat", "-safe", "0", "-i", filepath.Base(listPath), "-c", "copy", filepath.Base(videoOnly)); err != nil {
		return "", fmt.Errorf("拼接视频轨: %w", err)
	}

	audioOut := filepath.Join(workDir, "audio.m4a")
	ffArgs := []string{"-y"}
	for _, p := range parentOrder {
		lp, _ := resolveLocalFromURL(staticDir, resolvedTTS[p].URL)
		ffArgs = append(ffArgs, "-i", lp)
	}
	if len(parentOrder) == 1 {
		ffArgs = append(ffArgs, "-c:a", "aac", "-b:a", "192k", audioOut)
	} else {
		var fc strings.Builder
		for i := range parentOrder {
			fc.WriteString(fmt.Sprintf("[%d:a]", i))
		}
		fc.WriteString(fmt.Sprintf("concat=n=%d:v=0:a=1[outa]", len(parentOrder)))
		ffArgs = append(ffArgs, "-filter_complex", fc.String(), "-map", "[outa]", "-c:a", "aac", "-b:a", "192k", audioOut)
	}
	if err = runFFmpegAbs(ctx, ffmpegBin, ffArgs...); err != nil {
		return "", fmt.Errorf("处理音轨: %w", err)
	}

	finalPath, err := utils.GenUploadPath(basePath, "teaching_out.mp4", false)
	if err != nil {
		return "", err
	}
	if err = os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return "", err
	}
	muxArgs := []string{
		"-y", "-i", videoOnly, "-i", audioOut,
		"-c:v", "copy", "-c:a", "aac",
		"-map", "0:v:0", "-map", "1:a:0",
		"-shortest", finalPath,
	}
	if err = runFFmpegAbs(ctx, ffmpegBin, muxArgs...); err != nil {
		return "", fmt.Errorf("合成音画: %w", err)
	}

	return utils.GenUploadUrl(basePath, baseURL, finalPath), nil
}

func escapeConcatPath(p string) string {
	return "'" + strings.ReplaceAll(p, "'", "'\\''") + "'"
}

// ParseVideoAssetsJSON 从记录 JSON 文本解析配图数组
func ParseVideoAssetsJSON(raw string) ([]VideoAsset, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("assets 为空")
	}
	var v interface{}
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil, err
	}
	if s, ok := v.(string); ok {
		t := strings.TrimSpace(s)
		if t != "" {
			_ = json.Unmarshal([]byte(t), &v)
		}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out []VideoAsset
	if err = json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ParseVideoTTSJSON 从记录 JSON 文本解析 TTS 数组
func ParseVideoTTSJSON(raw string) ([]VideoTTSItem, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("tts 为空")
	}
	var v interface{}
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil, err
	}
	if s, ok := v.(string); ok {
		t := strings.TrimSpace(s)
		if t != "" {
			_ = json.Unmarshal([]byte(t), &v)
		}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out []VideoTTSItem
	if err = json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}
