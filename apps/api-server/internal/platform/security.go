package platform

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"
)

func randomDigits(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("digit length required")
	}
	out := make([]byte, n)
	for i := range out {
		v, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		out[i] = byte('0' + v.Int64())
	}
	return string(out), nil
}

func randomToken(prefix string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	if prefix == "" {
		return token, nil
	}
	return prefix + "-" + token, nil
}

func ValidateRequiredSecrets() error {
	if getenv("ALLOW_INSECURE_DEFAULTS", "false") == "true" {
		return nil
	}
	if v := getenv("ADMIN_PASSWORD", ""); v == "" || v == "admin123456" {
		return fmt.Errorf("ADMIN_PASSWORD must be set to a non-default value")
	}
	if v := getenv("FRP_TOKEN", ""); v == "" || v == "change-me" {
		return fmt.Errorf("FRP_TOKEN must be set to a non-default value")
	}
	return nil
}

func allowedCORSOrigin(origin string) (string, bool) {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return "", true
	}
	for _, item := range strings.Split(getenv("CORS_ALLOWED_ORIGINS", ""), ",") {
		if strings.TrimSpace(item) == origin {
			return origin, true
		}
	}
	if getenv("ALLOW_INSECURE_DEFAULTS", "false") == "true" {
		if strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") {
			return origin, true
		}
	}
	return "", false
}
