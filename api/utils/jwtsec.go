package utils

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
)

const MinJWTSecretLen = 32

// EnsureJWTSecret 保证密钥非空且足够长；不足时用随机值兜底（并尽量告警）。
// 空密钥会导致 HS256 验签被绕过，必须在启动时消除。
func EnsureJWTSecret(secret, name string) (string, error) {
	s := strings.TrimSpace(secret)
	if len(s) >= MinJWTSecretLen {
		return s, nil
	}
	generated, err := RandomJWTSecret()
	if err != nil {
		return "", fmt.Errorf("%s JWT secret invalid and cannot generate fallback: %w", name, err)
	}
	_, _ = fmt.Fprintf(os.Stderr, "[SECURITY] %s JWT secret missing or too short (< %d); using ephemeral secret — set a stable secret in config/env\n", name, MinJWTSecretLen)
	return generated, nil
}

func RandomJWTSecret() (string, error) {
	b := make([]byte, 48)
	if _, err := rand.Read(b); err != nil {
		return "", errors.New("cannot read entropy: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func AdminSessKey(jti string) string {
	return "admin:sess:" + jti
}

func AdminCurKey(adminID uint) string {
	return fmt.Sprintf("admin:cur:%d", adminID)
}

func UserSessKey(userID uint) string {
	return fmt.Sprintf("users/%d", userID)
}
