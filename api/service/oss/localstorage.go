package oss

import (
	"bytes"
	"chatplus/core/types"
	"chatplus/utils"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type LocalStorage struct {
	config   *types.LocalStorageConfig
	proxyURL string
}

func NewLocalStorage(config *types.AppConfig) LocalStorage {
	return LocalStorage{
		config:   &config.OSS.Local,
		proxyURL: config.ProxyURL,
	}
}

func (s LocalStorage) PutFile(ctx *gin.Context, name string) (File, error) {
	file, err := ctx.FormFile(name)
	if err != nil {
		return File{}, fmt.Errorf("error with get form: %v", err)
	}
	if file.Size > 5<<20 {
		return File{}, fmt.Errorf("文件过大，最大 5MB")
	}

	src, err := file.Open()
	if err != nil {
		return File{}, fmt.Errorf("无法读取上传文件")
	}
	defer src.Close()

	head := make([]byte, 16)
	n, _ := io.ReadFull(src, head)
	head = head[:n]
	ext, ok := detectAllowedImageExt(head)
	if !ok {
		return File{}, fmt.Errorf("仅支持 jpg/png/gif/webp")
	}
	// 服务端决定扩展名，忽略客户端伪造的 filename / Content-Type
	safeName := strings.TrimSuffix(filepath.Base(file.Filename), filepath.Ext(file.Filename)) + ext
	path, err := utils.GenUploadPath(s.config.BasePath, safeName, false)
	if err != nil {
		return File{}, fmt.Errorf("error with generate filename: %s", err.Error())
	}
	out, err := os.Create(path)
	if err != nil {
		return File{}, fmt.Errorf("error with create file: %s", err.Error())
	}
	defer out.Close()
	if _, err := io.Copy(out, io.MultiReader(bytes.NewReader(head), src)); err != nil {
		_ = os.Remove(path)
		return File{}, fmt.Errorf("error with save upload file: %s", err.Error())
	}

	return File{
		Name:   filepath.Base(path),
		ObjKey: path,
		URL:    utils.GenUploadUrl(s.config.BasePath, s.config.BaseURL, path),
		Ext:    ext,
		Size:   file.Size,
	}, nil
}

func detectAllowedImageExt(head []byte) (string, bool) {
	if len(head) >= 3 && head[0] == 0xFF && head[1] == 0xD8 && head[2] == 0xFF {
		return ".jpg", true
	}
	if len(head) >= 8 && bytes.Equal(head[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return ".png", true
	}
	if len(head) >= 6 && (bytes.Equal(head[:6], []byte("GIF87a")) || bytes.Equal(head[:6], []byte("GIF89a"))) {
		return ".gif", true
	}
	if len(head) >= 12 && bytes.Equal(head[:4], []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP")) {
		return ".webp", true
	}
	return "", false
}

func (s LocalStorage) PutImg(imageURL string, useProxy bool) (string, error) {
	parse, err := url.Parse(imageURL)
	if err != nil {
		return "", fmt.Errorf("error with parse image URL: %v", err)
	}
	filename := filepath.Base(parse.Path)
	filePath, err := utils.GenUploadPath(s.config.BasePath, filename, true)
	if err != nil {
		return "", fmt.Errorf("error with generate image dir: %v", err)
	}

	if useProxy {
		err = utils.DownloadFile(imageURL, filePath, s.proxyURL)
	} else {
		err = utils.DownloadFile(imageURL, filePath, "")
	}
	if err != nil {
		return "", fmt.Errorf("error with download image: %v", err)
	}

	return utils.GenUploadUrl(s.config.BasePath, s.config.BaseURL, filePath), nil
}

func (s LocalStorage) PutBase64(base64Img string) (string, error) {
	imageData, err := base64.StdEncoding.DecodeString(base64Img)
	if err != nil {
		return "", fmt.Errorf("error decoding base64:%v", err)
	}
	filePath, err := utils.GenUploadPath(s.config.BasePath, "", true)
	err = os.WriteFile(filePath, imageData, 0644)
	if err != nil {
		return "", fmt.Errorf("error writing to file:%v", err)
	}

	return utils.GenUploadUrl(s.config.BasePath, s.config.BaseURL, filePath), nil
}

func (s LocalStorage) Delete(fileURL string) error {
	if _, err := os.Stat(fileURL); err == nil {
		return os.Remove(fileURL)
	}
	filePath := strings.Replace(fileURL, s.config.BaseURL, s.config.BasePath, 1)
	return os.Remove(filePath)
}

var _ Uploader = LocalStorage{}
