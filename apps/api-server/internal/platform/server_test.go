package platform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

func TestCertificateRequestPersistsDryRunRecord(t *testing.T) {
	store := NewStore()
	s := NewServerWithServices(store, LogMailer{}, &Automation{NginxConfDir: t.TempDir(), WebrootDir: "/tmp/acme", LetsEncryptDir: "/tmp/letsencrypt", FRPVhostURL: "http://frps:8080", CertbotBin: "certbot", DryRun: true}, nil)
	login := post(t, s, "/api/admin/login", map[string]any{"email": "admin@example.com", "password": "admin123456"}, "")
	token := login["access_token"].(string)
	post(t, s, "/api/admin/certificates/request", map[string]any{"domain": "cert.example.com", "email": "admin@example.com"}, token)
	rr := request(t, s, "GET", "/api/admin/certificates", nil, token)
	if rr.Code != 200 {
		t.Fatalf("cert list status=%d body=%s", rr.Code, rr.Body.String())
	}
	certs := store.Certificates()
	if len(certs) != 1 || certs[0].Domain != "cert.example.com" || certs[0].Status != "dry_run" {
		t.Fatalf("unexpected certs %#v", certs)
	}
}

func TestAdminOperationLogRecordsRedeemCodeCreation(t *testing.T) {
	s := NewServer(NewStore())
	login := post(t, s, "/api/admin/login", map[string]any{"email": "admin@example.com", "password": "admin123456"}, "")
	token := login["access_token"].(string)
	rrCreate := request(t, s, "POST", "/api/admin/redeem-codes", map[string]any{"plan_id": 1, "count": 1, "prefix": "OPS"}, token)
	if rrCreate.Code != 200 {
		t.Fatalf("create code status=%d body=%s", rrCreate.Code, rrCreate.Body.String())
	}
	rr := request(t, s, "GET", "/api/admin/operation-logs", nil, token)
	if rr.Code != 200 {
		t.Fatalf("logs status=%d body=%s", rr.Code, rr.Body.String())
	}
	logs := s.store.AdminOperationLogs(10)
	if len(logs) == 0 || logs[0].Action != "redeem_codes.create" {
		t.Fatalf("unexpected logs %#v", logs)
	}
}

func TestAdminRenewDueCertificates(t *testing.T) {
	store := NewStore()
	expires := time.Now().AddDate(0, 0, 1)
	_, err := store.SaveCertificate(CertificateRecord{Domain: "due.example.com", Status: "issued", ExpiresAt: &expires})
	if err != nil {
		t.Fatal(err)
	}
	s := NewServerWithServices(store, LogMailer{}, &Automation{WebrootDir: "/tmp/acme", LetsEncryptDir: "/tmp/letsencrypt", CertbotBin: "certbot", DryRun: true}, nil)
	login := post(t, s, "/api/admin/login", map[string]any{"email": "admin@example.com", "password": "admin123456"}, "")
	token := login["access_token"].(string)
	rr := request(t, s, "POST", "/api/admin/certificates/renew-due", map[string]any{"force": false}, token)
	if rr.Code != 200 {
		t.Fatalf("renew status=%d body=%s", rr.Code, rr.Body.String())
	}
	certs := store.Certificates()
	if len(certs) != 1 || certs[0].Status != "dry_run" {
		t.Fatalf("unexpected certs %#v", certs)
	}
	logs := store.AdminOperationLogs(10)
	if len(logs) == 0 || logs[0].Action != "certificate.renew_due" {
		t.Fatalf("unexpected logs %#v", logs)
	}
}

func TestAdminNodeCreateBindAndRemoteStatus(t *testing.T) {
	fakeAgent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Fatalf("missing node agent auth header")
		}
		if r.URL.Path != "/api/frps/status" {
			t.Fatalf("unexpected fake agent path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(FRPSStatus{Healthy: true, Output: "fake frps ok"})
	}))
	defer fakeAgent.Close()

	store := NewStore()
	s := NewServer(store)
	login := post(t, s, "/api/admin/login", map[string]any{"email": "admin@example.com", "password": "admin123456"}, "")
	token := login["access_token"].(string)

	created := post(t, s, "/api/admin/nodes", map[string]any{"name": "edge-1", "agent_url": fakeAgent.URL, "frp_entry_domain": "frp.example.com", "server_addr": "frp.example.com", "frp_server_port": 7000}, token)
	bindToken := created["bind_token"].(string)
	if bindToken == "" {
		t.Fatalf("expected bind token in created node %#v", created)
	}

	bound := post(t, s, "/api/nodes/bind", map[string]any{"bind_token": bindToken, "name": "edge-1", "agent_url": fakeAgent.URL, "frp_entry_domain": "frp.example.com", "server_addr": "frp.example.com"}, "")
	if bound["agent_token"].(string) == "" {
		t.Fatalf("expected agent token in bind response %#v", bound)
	}

	nodeID := int64(created["id"].(float64))
	rr := request(t, s, "GET", fmt.Sprintf("/api/admin/nodes/%d/status", nodeID), nil, token)
	if rr.Code != 200 {
		t.Fatalf("node status code=%d body=%s", rr.Code, rr.Body.String())
	}
	if store.Nodes()[0].Status != "online" {
		t.Fatalf("expected node online, got %#v", store.Nodes()[0])
	}
}
