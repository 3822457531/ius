package core

import (
	"bytes"
	"chatplus/core/types"
	logger2 "chatplus/logger"
	"chatplus/utils"
	"errors"
	"os"

	"github.com/BurntSushi/toml"
)

var logger = logger2.GetLogger()

func NewDefaultConfig() *types.AppConfig {
	userSecret, _ := utils.RandomJWTSecret()
	adminSecret, _ := utils.RandomJWTSecret()
	return &types.AppConfig{
		Listen:    "0.0.0.0:5678",
		ProxyURL:  "",
		StaticDir: "./static",
		StaticUrl: "http://localhost/5678/static",
		Redis:     types.RedisConfig{Host: "localhost", Port: 6379, Password: ""},
		Session: types.Session{
			SecretKey: userSecret,
			MaxAge:    86400,
		},
		AdminSession: types.Session{
			SecretKey: adminSecret,
			MaxAge:    86400,
		},
		ApiConfig: types.ChatPlusApiConfig{},
		OSS: types.OSSConfig{
			Active: "local",
			Local: types.LocalStorageConfig{
				BaseURL:  "http://localhost/5678/static/upload",
				BasePath: "./static/upload",
			},
		},
		WeChatBot:    false,
		AlipayConfig: types.AlipayConfig{Enabled: false, SandBox: false},
	}
}

func LoadConfig(configFile string) (*types.AppConfig, error) {
	var config *types.AppConfig
	_, err := os.Stat(configFile)
	if err != nil {
		logger.Info("creating new config file: ", configFile)
		config = NewDefaultConfig()
		config.Path = configFile
		// save config
		err := SaveConfig(config)
		if err != nil {
			return nil, err
		}

		return config, nil
	}
	_, err = toml.DecodeFile(configFile, &config)
	if err != nil {
		return nil, err
	}
	if err := sanitizeSessionSecrets(config); err != nil {
		return nil, err
	}

	return config, err
}

// sanitizeSessionSecrets 启动时消除空/过短 JWT 密钥，防止 HS256 空密钥验签绕过
func sanitizeSessionSecrets(config *types.AppConfig) error {
	if config == nil {
		return errors.New("nil config")
	}
	var err error
	config.Session.SecretKey, err = utils.EnsureJWTSecret(config.Session.SecretKey, "Session")
	if err != nil {
		return err
	}
	if config.Session.MaxAge <= 0 {
		config.Session.MaxAge = 86400
	}
	config.AdminSession.SecretKey, err = utils.EnsureJWTSecret(config.AdminSession.SecretKey, "AdminSession")
	if err != nil {
		return err
	}
	if config.AdminSession.MaxAge <= 0 {
		config.AdminSession.MaxAge = 86400
	}
	// 后台与前台密钥必须隔离；若相同则强制换后台密钥
	if config.AdminSession.SecretKey == config.Session.SecretKey {
		config.AdminSession.SecretKey, err = utils.RandomJWTSecret()
		if err != nil {
			return err
		}
		logger.Info("[SECURITY] AdminSession SecretKey equaled Session SecretKey; regenerated AdminSession secret")
	}
	return nil
}

func SaveConfig(config *types.AppConfig) error {
	buf := new(bytes.Buffer)
	encoder := toml.NewEncoder(buf)
	if err := encoder.Encode(&config); err != nil {
		return err
	}

	return os.WriteFile(config.Path, buf.Bytes(), 0644)
}
