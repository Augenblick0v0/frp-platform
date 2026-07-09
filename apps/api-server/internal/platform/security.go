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
	if InsecureDefaultsAllowed() {
		return nil
	}
	if isWeakSecret("ADMIN_PASSWORD", getenv("ADMIN_PASSWORD", "")) {
		return fmt.Errorf("ADMIN_PASSWORD must be set to a strong non-placeholder value")
	}
	if isWeakSecret("FRP_TOKEN", getenv("FRP_TOKEN", "")) {
		return fmt.Errorf("FRP_TOKEN must be set to a strong non-placeholder value")
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
	if InsecureDefaultsAllowed() {
		if strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") {
			return origin, true
		}
	}
	return "", false
}

func InsecureDefaultsAllowed() bool {
	return strings.EqualFold(strings.TrimSpace(getenv("ALLOW_INSECURE_DEFAULTS", "false")), "true")
}

func RequireDatabaseURL() error {
	if InsecureDefaultsAllowed() {
		return nil
	}
	if strings.TrimSpace(getenv("DATABASE_URL", "")) == "" {
		return fmt.Errorf("DATABASE_URL must be set unless ALLOW_INSECURE_DEFAULTS=true")
	}
	return nil
}

func isWeakSecret(name, value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "" || len(v) < 16 {
		return true
	}
	weakFragments := []string{"change-me", "replace-with", "example", "your-", "todo", "password", "secret", "admin123456"}
	for _, fragment := range weakFragments {
		if strings.Contains(v, fragment) {
			return true
		}
	}
	return false
}
