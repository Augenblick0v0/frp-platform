package platform

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
)

const passwordHashPrefix = "sha256$"
const passwordHashIterations = 120000

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password required")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	digest := stretchPassword([]byte(password), salt, passwordHashIterations)
	return fmt.Sprintf("%s%d$%s$%s", passwordHashPrefix, passwordHashIterations, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(digest)), nil
}

func VerifyPassword(stored, password string) bool {
	if strings.HasPrefix(stored, passwordHashPrefix) {
		parts := strings.Split(stored, "$")
		if len(parts) != 4 {
			return false
		}
		var iterations int
		if _, err := fmt.Sscanf(parts[1], "%d", &iterations); err != nil || iterations <= 0 {
			return false
		}
		salt, err := base64.RawStdEncoding.DecodeString(parts[2])
		if err != nil {
			return false
		}
		want, err := base64.RawStdEncoding.DecodeString(parts[3])
		if err != nil {
			return false
		}
		got := stretchPassword([]byte(password), salt, iterations)
		return subtle.ConstantTimeCompare(got, want) == 1
	}
	if getenv("ALLOW_LEGACY_PLAINTEXT_PASSWORDS", "false") == "true" {
		return subtle.ConstantTimeCompare([]byte(stored), []byte(password)) == 1
	}
	return false
}

func stretchPassword(password, salt []byte, iterations int) []byte {
	h := sha256.Sum256(append(append([]byte{}, salt...), password...))
	out := h[:]
	for i := 1; i < iterations; i++ {
		next := sha256.Sum256(append(out, password...))
		out = next[:]
	}
	return append([]byte{}, out...)
}

func mustHashPassword(password string) string {
	h, err := HashPassword(password)
	if err != nil {
		panic(err)
	}
	return h
}
