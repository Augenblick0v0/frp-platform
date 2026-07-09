package platform

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestEpayCreateOrderAndNotifyActivatesPlan(t *testing.T) {
	t.Setenv("EPAY_API_BASE", "https://pay.flwi.top")
	t.Setenv("EPAY_PID", "1000")
	t.Setenv("EPAY_KEY", "test-secret")
	t.Setenv("PUBLIC_BASE_URL", "https://panel.example.com")
	store := NewStore()
	s := NewServer(store)
	token := registerTestUser(t, s, store, "pay@example.com", "pass")
	plan, err := store.CreatePlan(Plan{Name: "Paid", DurationDays: 7, TrafficLimitBytes: 1024, BandwidthKbps: 1000, MaxTunnels: 1, AllowTCP: true, Status: "active", PriceCents: 990})
	if err != nil {
		t.Fatal(err)
	}
	created := post(t, s, "/api/payments/epay/orders", map[string]any{"plan_id": plan.ID, "pay_type": "alipay"}, token)
	if created["pay_url"] == "" || created["out_trade_no"] == "" {
		t.Fatalf("unexpected payment order %#v", created)
	}
	if created["money"] != "9.90" {
		t.Fatalf("expected amount 9.90, got %#v", created["money"])
	}
	outTradeNo := created["out_trade_no"].(string)
	notify := epaySignedValues(map[string]string{
		"pid":          "1000",
		"trade_no":     "EPAY123",
		"out_trade_no": outTradeNo,
		"type":         "alipay",
		"name":         "Paid",
		"money":        "9.90",
		"trade_status": "TRADE_SUCCESS",
	}, "test-secret")
	rr := formRequest(t, s, "POST", "/api/payments/epay/notify", notify)
	if rr.Code != 200 || strings.TrimSpace(rr.Body.String()) != "success" {
		t.Fatalf("notify status=%d body=%s", rr.Code, rr.Body.String())
	}
	sub, err := store.Subscription(1)
	if err != nil || sub.PlanID != plan.ID || sub.PlanName != "Paid" {
		t.Fatalf("expected paid subscription, sub=%#v err=%v", sub, err)
	}
}

func TestEpayNotifyRejectsBadSignature(t *testing.T) {
	t.Setenv("EPAY_PID", "1000")
	t.Setenv("EPAY_KEY", "test-secret")
	store := NewStore()
	store.SendEmailCode("badpay@example.com", "register")
	code := store.DebugEmailCode("badpay@example.com", "register")
	if _, err := store.Register("badpay@example.com", code, "pass"); err != nil {
		t.Fatal(err)
	}
	order, err := store.CreatePaymentOrder(PaymentOrder{UserID: 1, PlanID: 1, Provider: "epay", OutTradeNo: "FPTESTBAD", Name: "高级套餐", Money: "1.00", Status: "pending"})
	if err != nil {
		t.Fatal(err)
	}
	s := NewServer(store)
	rr := formRequest(t, s, "POST", "/api/payments/epay/notify", map[string]string{
		"pid":          "1000",
		"trade_no":     "EPAY-BAD",
		"out_trade_no": order.OutTradeNo,
		"type":         "alipay",
		"name":         order.Name,
		"money":        order.Money,
		"trade_status": "TRADE_SUCCESS",
		"sign":         "bad",
		"sign_type":    "MD5",
	})
	if rr.Code != 400 || strings.TrimSpace(rr.Body.String()) == "success" {
		t.Fatalf("expected bad notify to fail, status=%d body=%s", rr.Code, rr.Body.String())
	}
	if _, err := store.Subscription(1); err == nil {
		t.Fatalf("bad signature should not activate subscription")
	}
}

func TestUserRedeemAndCreateTCPTunnel(t *testing.T) {
	store := NewStore()
	s := NewServer(store)
	token := registerTestUser(t, s, store, "u@example.com", "pass")
	post(t, s, "/api/user/redeem", map[string]any{"code": "DEMO-PLAN-2026"}, token)
	created := post(t, s, "/api/tunnels", map[string]any{"name": "ssh", "type": "tcp", "local_host": "127.0.0.1", "local_port": 22}, token)
	if created["remote_port"].(float64) != 20000 {
		t.Fatalf("expected first tcp port 20000, got %#v", created["remote_port"])
	}
}

func registerTestUser(t *testing.T, s *Server, store *Store, email, password string) string {
	t.Helper()
	post(t, s, "/api/auth/send-email-code", map[string]any{"email": email, "purpose": "register"}, "")
	code := store.DebugEmailCode(email, "register")
	post(t, s, "/api/auth/register", map[string]any{"email": email, "code": code, "password": password}, "")
	login := post(t, s, "/api/auth/login", map[string]any{"email": email, "password": password}, "")
	return login["access_token"].(string)
}

func TestSendEmailCodeDoesNotReturnDevCode(t *testing.T) {
	s := NewServer(NewStore())
	rr := request(t, s, "POST", "/api/auth/send-email-code", map[string]any{
		"email": "secure@example.com", "purpose": "register",
	}, "")
	if rr.Code != 200 {
		t.Fatalf("send code status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "dev_code") || strings.Contains(rr.Body.String(), "123456") {
		t.Fatalf("verification code leaked in response: %s", rr.Body.String())
	}
}

func TestAuthTokensAreNotTimestampOnly(t *testing.T) {
	store := NewStore()
	s := NewServer(store)
	token := registerTestUser(t, s, store, "token-random@example.com", "pass")
	if strings.HasPrefix(token, "token-1-") || len(token) < 32 {
		t.Fatalf("weak token generated: %q", token)
	}
}

func TestDisabledUserTokenIsRejected(t *testing.T) {
	store := NewStore()
	s := NewServer(store)
	token := registerTestUser(t, s, store, "disabled@example.com", "pass")
	if _, _, err := store.UpdateUser(1, "disabled", 0); err != nil {
		t.Fatal(err)
	}
	rr := request(t, s, "GET", "/api/auth/me", nil, token)
	if rr.Code != 401 {
		t.Fatalf("disabled token should be rejected, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPRequiresCustomDomainEntitlement(t *testing.T) {
	store := NewStore()
	s := NewServer(store)
	token := registerTestUser(t, s, store, "domain-denied@example.com", "pass")
	plan, err := store.CreatePlan(Plan{Name: "No Domain", DurationDays: 1, TrafficLimitBytes: 1024 * 1024, BandwidthKbps: 1000, MaxTunnels: 3, MaxHTTPTunnels: 3, AllowHTTP: true, AllowCustomDomain: false, Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	codes, err := store.CreateRedeemCodes(plan.ID, 1, "NODOM")
	if err != nil {
		t.Fatal(err)
	}
	post(t, s, "/api/user/redeem", map[string]any{"code": codes[0].Code}, token)
	rr := request(t, s, "POST", "/api/tunnels", map[string]any{"name": "web", "type": "http", "local_host": "127.0.0.1", "local_port": 8080, "domain": "app.example.com"}, token)
	if rr.Code != 403 {
		t.Fatalf("expected custom domain entitlement 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRedeemCodeExpirationIsHonored(t *testing.T) {
	store := NewStore()
	s := NewServer(store)
	token := registerTestUser(t, s, store, "expired-code@example.com", "pass")
	expired := time.Now().Add(-time.Hour)
	store.redeemCodes["OLD-CODE"] = RedeemCode{Code: "OLD-CODE", PlanID: 1, Status: "unused", ExpiresAt: &expired}
	rr := request(t, s, "POST", "/api/user/redeem", map[string]any{"code": "OLD-CODE"}, token)
	if rr.Code != 403 {
		t.Fatalf("expected expired code 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestSamePortCanBeAllocatedOnDifferentNodes(t *testing.T) {
	store := NewStore()
	n1, _ := store.CreateNode(Node{Name: "n1", ServerAddr: "n1.example.com", TCPPortStart: 20000, TCPPortEnd: 20000})
	n2, _ := store.CreateNode(Node{Name: "n2", ServerAddr: "n2.example.com", TCPPortStart: 20000, TCPPortEnd: 20000})
	s := NewServer(store)
	token1 := registerTestUser(t, s, store, "node1@example.com", "pass")
	token2 := registerTestUser(t, s, store, "node2@example.com", "pass")
	post(t, s, "/api/user/redeem", map[string]any{"code": "DEMO-PLAN-2026"}, token1)
	codes, _ := store.CreateRedeemCodes(1, 1, "NODE2")
	post(t, s, "/api/user/redeem", map[string]any{"code": codes[0].Code}, token2)
	t1 := post(t, s, "/api/tunnels", map[string]any{"name": "ssh1", "type": "tcp", "local_host": "127.0.0.1", "local_port": 22, "node_id": n1.ID}, token1)
	t2 := post(t, s, "/api/tunnels", map[string]any{"name": "ssh2", "type": "tcp", "local_host": "127.0.0.1", "local_port": 22, "node_id": n2.ID}, token2)
	if t1["remote_port"].(float64) != 20000 || t2["remote_port"].(float64) != 20000 {
		t.Fatalf("expected same port on different nodes, got %#v %#v", t1["remote_port"], t2["remote_port"])
	}
}

func TestDeleteTunnelReleasesPort(t *testing.T) {
	store := NewStore()
	s := NewServer(store)
	token := registerTestUser(t, s, store, "delete-port@example.com", "pass")
	post(t, s, "/api/user/redeem", map[string]any{"code": "DEMO-PLAN-2026"}, token)
	created := post(t, s, "/api/tunnels", map[string]any{"name": "one", "type": "tcp", "local_host": "127.0.0.1", "local_port": 22}, token)
	id := int64(created["id"].(float64))
	rr := request(t, s, "DELETE", fmt.Sprintf("/api/tunnels/%d", id), nil, token)
	if rr.Code != 200 {
		t.Fatalf("delete status=%d body=%s", rr.Code, rr.Body.String())
	}
	again := post(t, s, "/api/tunnels", map[string]any{"name": "two", "type": "tcp", "local_host": "127.0.0.1", "local_port": 23}, token)
	if again["remote_port"].(float64) != created["remote_port"].(float64) {
		t.Fatalf("expected released port reuse, first=%#v second=%#v", created["remote_port"], again["remote_port"])
	}
}

func TestSpeedProbeRejectsHugePayload(t *testing.T) {
	if _, err := normalizeSpeedProbeBytes(1<<40, defaultSpeedProbeBytes); err == nil {
		t.Fatal("expected huge speed payload to be rejected")
	}
}

func TestClientTunnelsRejectsDefaultFRPToken(t *testing.T) {
	t.Setenv("FRP_TOKEN", "")
	store := NewStore()
	s := NewServer(store)
	token := registerTestUser(t, s, store, "no-frp-token@example.com", "pass")
	rr := request(t, s, "GET", "/api/client/tunnels", nil, token)
	if rr.Code != 500 {
		t.Fatalf("expected missing frp token 500, got %d body=%s", rr.Code, rr.Body.String())
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
	token := registerTestUser(t, s, store, "limit@example.com", "pass")
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

func TestTunnelBandwidthOverrideCannotExceedPlanLimit(t *testing.T) {
	store := NewStore()
	s := NewServer(store)
	token := registerTestUser(t, s, store, "speed@example.com", "pass")
	plan, err := store.CreatePlan(Plan{Name: "Speed", DurationDays: 1, TrafficLimitBytes: 1024 * 1024 * 1024, BandwidthKbps: 512, MaxTunnels: 3, MaxTCPTunnels: 3, AllowTCP: true, Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	codes, err := store.CreateRedeemCodes(plan.ID, 1, "SPD")
	if err != nil {
		t.Fatal(err)
	}
	post(t, s, "/api/user/redeem", map[string]any{"code": codes[0].Code}, token)
	rr := request(t, s, "POST", "/api/tunnels", map[string]any{"name": "too-fast", "type": "tcp", "local_host": "127.0.0.1", "local_port": 22, "bandwidth_limit_kbps": 1024}, token)
	if rr.Code != 403 {
		t.Fatalf("expected too-fast tunnel 403, got %d body=%s", rr.Code, rr.Body.String())
	}
	created := post(t, s, "/api/tunnels", map[string]any{"name": "limited", "type": "tcp", "local_host": "127.0.0.1", "local_port": 22, "bandwidth_limit_kbps": 256}, token)
	if created["bandwidth_limit_kbps"].(float64) != 256 {
		t.Fatalf("unexpected tunnel bandwidth %#v", created)
	}
}

func TestClientTunnelsReturnsEffectiveBandwidthLimit(t *testing.T) {
	t.Setenv("FRP_TOKEN", "test-runtime-token")
	store := NewStore()
	s := NewServer(store)
	token := registerTestUser(t, s, store, "client-speed@example.com", "pass")
	plan, err := store.CreatePlan(Plan{Name: "Client Speed", DurationDays: 1, TrafficLimitBytes: 1024 * 1024 * 1024, BandwidthKbps: 512, MaxTunnels: 3, MaxTCPTunnels: 3, AllowTCP: true, Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	codes, err := store.CreateRedeemCodes(plan.ID, 1, "CSPD")
	if err != nil {
		t.Fatal(err)
	}
	post(t, s, "/api/user/redeem", map[string]any{"code": codes[0].Code}, token)
	post(t, s, "/api/tunnels", map[string]any{"name": "limited", "type": "tcp", "local_host": "127.0.0.1", "local_port": 22, "bandwidth_limit_kbps": 256}, token)
	rr := request(t, s, "GET", "/api/client/tunnels", nil, token)
	if rr.Code != 200 {
		t.Fatalf("client tunnels status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out struct {
		Success bool `json:"success"`
		Data    struct {
			BandwidthLimitKbps int      `json:"bandwidth_limit_kbps"`
			Tunnels            []Tunnel `json:"tunnels"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Data.BandwidthLimitKbps != 512 {
		t.Fatalf("expected package limit 512, got %d", out.Data.BandwidthLimitKbps)
	}
	if len(out.Data.Tunnels) != 1 || out.Data.Tunnels[0].EffectiveBandwidthKbps != 256 {
		t.Fatalf("unexpected tunnels %#v", out.Data.Tunnels)
	}
}

func TestSpeedTestTunnelLifecycleAndTrafficAccounting(t *testing.T) {
	store := NewStore()
	s := NewServer(store)
	token := registerTestUser(t, s, store, "speedtest@example.com", "pass")
	post(t, s, "/api/user/redeem", map[string]any{"code": "DEMO-PLAN-2026"}, token)
	rrLimit := request(t, s, "POST", "/api/speed-tests/tunnels", map[string]any{"type": "tcp", "local_host": "127.0.0.1", "local_port": 18080, "bandwidth_limit_kbps": 512}, token)
	if rrLimit.Code != 400 {
		t.Fatalf("expected custom speed test bandwidth to fail, got %d body=%s", rrLimit.Code, rrLimit.Body.String())
	}
	created := post(t, s, "/api/speed-tests/tunnels", map[string]any{"type": "tcp", "local_host": "127.0.0.1", "local_port": 18080}, token)
	id := int64(created["id"].(float64))
	if created["effective_bandwidth_limit_kbps"].(float64) != 10240 {
		t.Fatalf("unexpected speed test tunnel %#v", created)
	}
	summary := post(t, s, "/api/client/traffic", map[string]any{"reports": []map[string]any{{"tunnel_id": id, "bytes_in": 1000, "bytes_out": 2000}}}, token)
	if summary["traffic_used_bytes"].(float64) != 3000 {
		t.Fatalf("unexpected traffic summary %#v", summary)
	}
	rr := request(t, s, "POST", fmt.Sprintf("/api/speed-tests/%d/finish", id), nil, token)
	if rr.Code != 200 {
		t.Fatalf("finish status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAPIServerRunsSpeedProbeAndCleansTunnel(t *testing.T) {
	ln := startTestTCPBenchmark(t)
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	store := NewStore()
	store.UpdateSettings(Settings{PlatformDomain: "example.com", FRPEntryDomain: "frp.example.com", ServerAddr: "127.0.0.1", FRPServerPort: 7000, TCPPortStart: port, TCPPortEnd: port, UDPPortStart: 30000, UDPPortEnd: 39999})
	t.Setenv("FRP_TOKEN", "test-runtime-token")
	s := NewServer(store)
	token := registerTestUser(t, s, store, "api-speed@example.com", "pass")
	post(t, s, "/api/user/redeem", map[string]any{"code": "DEMO-PLAN-2026"}, token)
	created := post(t, s, "/api/speed-tests/tunnels", map[string]any{"type": "tcp", "local_host": "127.0.0.1", "local_port": 18080}, token)
	id := int64(created["id"].(float64))

	rrCfg := request(t, s, "GET", fmt.Sprintf("/api/client/tunnels?speed_test_id=%d", id), nil, token)
	if rrCfg.Code != 200 {
		t.Fatalf("client speed config status=%d body=%s", rrCfg.Code, rrCfg.Body.String())
	}
	var cfg struct {
		Success bool `json:"success"`
		Data    struct {
			ServerAddr string   `json:"server_addr"`
			Tunnels    []Tunnel `json:"tunnels"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rrCfg.Body.Bytes(), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Data.ServerAddr != "127.0.0.1" || len(cfg.Data.Tunnels) != 1 || cfg.Data.Tunnels[0].ID != id {
		t.Fatalf("unexpected speed config %#v", cfg.Data)
	}

	run := post(t, s, fmt.Sprintf("/api/speed-tests/%d/run", id), map[string]any{"download_bytes": 32768, "upload_bytes": 32768, "duration_seconds": 10}, token)
	metrics := run["metrics"].(map[string]any)
	if metrics["download_average_kbps"].(float64) <= 0 || metrics["upload_average_kbps"].(float64) <= 0 {
		t.Fatalf("unexpected run result %#v", run)
	}
	if run["finished"] != true {
		t.Fatalf("expected finished result %#v", run)
	}
	if _, err := store.SpeedTestTunnel(1, id); err == nil {
		t.Fatalf("speed tunnel should be cleaned after run")
	}
}

func startTestTCPBenchmark(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				var op [1]byte
				if _, err := io.ReadFull(conn, op[:]); err != nil {
					return
				}
				switch op[0] {
				case 'p':
					_, _ = conn.Write([]byte("p"))
				case 'd':
					n := readTestInt64(conn)
					_, _ = io.CopyN(conn, testPatternReader{}, n)
				case 'u':
					n := readTestInt64(conn)
					_, _ = io.CopyN(io.Discard, conn, n)
					_, _ = conn.Write([]byte("ok"))
				}
			}(conn)
		}
	}()
	return ln
}

func readTestInt64(r io.Reader) int64 {
	var b [8]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0
	}
	return int64(binary.BigEndian.Uint64(b[:]))
}

type testPatternReader struct{}

func (testPatternReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(i % 251)
	}
	return len(p), nil
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

func formRequest(t *testing.T, s *Server, method, path string, vals map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	form := make(url.Values)
	for k, v := range vals {
		form.Set(k, v)
	}
	req := httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

func TestAdminNodeDelete(t *testing.T) {
	store := NewStore()
	s := NewServer(store)
	login := post(t, s, "/api/admin/login", map[string]any{"email": "admin@example.com", "password": "admin123456"}, "")
	token := login["access_token"].(string)

	created := post(t, s, "/api/admin/nodes", map[string]any{"name": "delete-me", "agent_url": "http://127.0.0.1:8090", "frp_entry_domain": "frp.example.com", "server_addr": "frp.example.com"}, token)
	nodeID := int64(created["id"].(float64))
	rr := request(t, s, "DELETE", fmt.Sprintf("/api/admin/nodes/%d/delete", nodeID), nil, token)
	if rr.Code != 200 {
		t.Fatalf("delete code=%d body=%s", rr.Code, rr.Body.String())
	}
	if len(store.Nodes()) != 0 {
		t.Fatalf("expected node deleted, got %#v", store.Nodes())
	}
	rr = request(t, s, "DELETE", fmt.Sprintf("/api/admin/nodes/%d/delete", nodeID), nil, token)
	if rr.Code != 404 {
		t.Fatalf("expected deleting missing node to return 404, got code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestClientTunnelsReturnsRuntimeFRPToken(t *testing.T) {
	t.Setenv("FRP_TOKEN", "test-runtime-token")
	store := NewStore()
	s := NewServer(store)
	token := registerTestUser(t, s, store, "token@example.com", "pass")
	rr := request(t, s, "GET", "/api/client/tunnels", nil, token)
	if rr.Code != 200 {
		t.Fatalf("client tunnels status=%d body=%s", rr.Code, rr.Body.String())
	}
	var out struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Data["token"] != "test-runtime-token" {
		t.Fatalf("expected runtime token, got %#v", out.Data["token"])
	}
}

func TestUserTopologyExposesOnlySafeNodeFields(t *testing.T) {
	store := NewStore()
	s := NewServer(store)
	post(t, s, "/api/auth/send-email-code", map[string]any{"email": "topology@example.com", "purpose": "register"}, "")
	post(t, s, "/api/auth/register", map[string]any{"email": "topology@example.com", "code": "123456", "password": "pass"}, "")
	login := post(t, s, "/api/auth/login", map[string]any{"email": "topology@example.com", "password": "pass"}, "")
	token := login["access_token"].(string)
	post(t, s, "/api/user/redeem", map[string]any{"code": "DEMO-PLAN-2026"}, token)
	_, err := store.CreateNode(Node{Name: "edge-safe", AgentURL: "http://node-agent:8090", AgentToken: "secret-agent", BindToken: "secret-bind", FRPEntryDomain: "frp.example.com", ServerAddr: "frp.example.com", FRPServerPort: 7000, Status: "online"})
	if err != nil {
		t.Fatal(err)
	}
	rr := request(t, s, "GET", "/api/user/topology", nil, token)
	if rr.Code != 200 {
		t.Fatalf("topology status=%d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if strings.Contains(body, "secret-agent") || strings.Contains(body, "secret-bind") || strings.Contains(body, "agent_token") || strings.Contains(body, "bind_token") {
		t.Fatalf("user topology leaked node secret: %s", body)
	}
	var out struct {
		Success bool `json:"success"`
		Data    struct {
			Role        string     `json:"role"`
			Nodes       []SafeNode `json:"nodes"`
			TunnelCount int        `json:"tunnel_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Data.Role != "User Console" || len(out.Data.Nodes) < 2 {
		t.Fatalf("unexpected user topology %#v", out.Data)
	}
}

func TestAdminTopologyAndOrders(t *testing.T) {
	t.Setenv("EPAY_PID", "1000")
	t.Setenv("EPAY_KEY", "test-secret")
	store := NewStore()
	s := NewServer(store)
	post(t, s, "/api/auth/send-email-code", map[string]any{"email": "order@example.com", "purpose": "register"}, "")
	post(t, s, "/api/auth/register", map[string]any{"email": "order@example.com", "code": "123456", "password": "pass"}, "")
	userLogin := post(t, s, "/api/auth/login", map[string]any{"email": "order@example.com", "password": "pass"}, "")
	userToken := userLogin["access_token"].(string)
	created := post(t, s, "/api/payments/epay/orders", map[string]any{"plan_id": 1, "pay_type": "wechatpay"}, userToken)
	if created["pay_type"] != "wxpay" {
		t.Fatalf("expected wxpay alias normalization, got %#v", created["pay_type"])
	}
	adminLogin := post(t, s, "/api/admin/login", map[string]any{"email": "admin@example.com", "password": "admin123456"}, "")
	adminToken := adminLogin["access_token"].(string)
	rr := request(t, s, "GET", "/api/admin/topology", nil, adminToken)
	if rr.Code != 200 {
		t.Fatalf("admin topology status=%d body=%s", rr.Code, rr.Body.String())
	}
	var topo struct {
		Success bool `json:"success"`
		Data    struct {
			Role           string                `json:"role"`
			PaymentMethods []PaymentMethodStatus `json:"payment_methods"`
			RecentOrders   []PaymentOrder        `json:"recent_orders"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &topo); err != nil {
		t.Fatal(err)
	}
	if topo.Data.Role != "Admin Console" || len(topo.Data.PaymentMethods) == 0 || len(topo.Data.RecentOrders) == 0 {
		t.Fatalf("unexpected admin topology %#v", topo.Data)
	}
	rr = request(t, s, "GET", "/api/admin/orders", nil, adminToken)
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), created["out_trade_no"].(string)) {
		t.Fatalf("orders status=%d body=%s", rr.Code, rr.Body.String())
	}
}
