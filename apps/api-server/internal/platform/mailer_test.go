package platform

import (
	"strings"
	"testing"
)

func TestVerificationMailBody(t *testing.T) {
	body := VerificationMailBody("123456", "register")
	if !strings.Contains(body, "123456") || !strings.Contains(body, "注册") {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestBase64EncodeHeader(t *testing.T) {
	got := encodeHeader("FRP 平台")
	if !strings.HasPrefix(got, "=?UTF-8?B?") || !strings.HasSuffix(got, "?=") {
		t.Fatalf("bad header: %s", got)
	}
}
