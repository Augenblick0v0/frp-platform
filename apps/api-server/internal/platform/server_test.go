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

func TestTrafficReportingAndPlanLimit(t *testing.T) {
	store := NewStore()
	s := NewServer(store)
	post(t, s, "/api/auth/send-email-code", map[string]any{"email": "limit@example.com", "purpose": "register"}, "")
	post(t, s, "/api/auth/register", map[string]any{"email": "limit@example.com", "code": "123456", "password": "pass"}, "")
	login := post(t, s, "/api/auth/login", map[string]any{"email": "limit@example.com", "password": "pass"}, "")
	token := login["access_token"].(string)
	plan, err := store.CreatePlan(Plan{Name: "小流量", DurationDays: 1, TrafficLimitBytes: 100, BandwidthKbps: 1000, MaxTunnels: 1, AllowTCP: true, Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	codes, err := store.CreateRedeemCodes(plan.ID, 1, "LIM")
	if err != nil {
		t.Fatal(err)
	}
	post(t, s, "/api/user/redeem", map[string]any{"code": codes[0].Code}, token)
	post(t, s, "/api/tunnels", map[string]any{"name": "one", "type": "tcp", "local_host": "127.0.0.1", "local_port": 22}, token)
	rr := request(t, s, "POST", "/api/tunnels", map[string]any{"name": "two", "type": "tcp", "local_host": "127.0.0.1", "local_port": 23}, token)
	if rr.Code != 403 {
		t.Fatalf("expected tunnel limit 403, got %d body=%s", rr.Code, rr.Body.String())
	}
	summary := post(t, s, "/api/client/traffic", map[string]any{"reports": []map[string]any{{"bytes_in": 60, "bytes_out": 50}}}, token)
	if summary["traffic_used_bytes"].(float64) != 110 {
		t.Fatalf("unexpected traffic summary %#v", summary)
	}
	rr = request(t, s, "POST", "/api/tunnels", map[string]any{"name": "three", "type": "tcp", "local_host": "127.0.0.1", "local_port": 24}, token)
	if rr.Code != 403 {
		t.Fatalf("expected over traffic 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func request(t *testing.T, s *Server, method, path string, body map[string]any, token string) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	return rr
}

func TestAdminLoginProtectsAdminAPI(t *testing.T) {
	s := NewServer(NewStore())
	rr := request(t, s, "GET", "/api/admin/dashboard", nil, "")
	if rr.Code != 401 {
		t.Fatalf("expected admin dashboard without token to be 401, got %d body=%s", rr.Code, rr.Body.String())
	}
	login := post(t, s, "/api/admin/login", map[string]any{"email": "admin@example.com", "password": "admin123456"}, "")
	token := login["access_token"].(string)
	rr = request(t, s, "GET", "/api/admin/dashboard", nil, token)
	if rr.Code != 200 {
		t.Fatalf("expected admin dashboard with token to be 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}
