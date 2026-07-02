package platform

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserRedeemAndCreateTCPTunnel(t *testing.T) {
	s := NewServer(NewStore())
	post(t, s, "/api/auth/send-email-code", map[string]any{"email": "u@example.com", "purpose": "register"}, "")
	post(t, s, "/api/auth/register", map[string]any{"email": "u@example.com", "code": "123456", "password": "pass"}, "")
	login := post(t, s, "/api/auth/login", map[string]any{"email": "u@example.com", "password": "pass"}, "")
	token := login["access_token"].(string)
	post(t, s, "/api/user/redeem", map[string]any{"code": "DEMO-PLAN-2026"}, token)
	created := post(t, s, "/api/tunnels", map[string]any{"name": "ssh", "type": "tcp", "local_host": "127.0.0.1", "local_port": 22}, token)
	if created["remote_port"].(float64) != 20000 {
		t.Fatalf("expected first tcp port 20000, got %#v", created["remote_port"])
	}
}

func post(t *testing.T, s *Server, path string, body map[string]any, token string) map[string]any {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code > 299 {
		t.Fatalf("%s status=%d body=%s", path, rr.Code, rr.Body.String())
	}
	var out struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out.Data
}
