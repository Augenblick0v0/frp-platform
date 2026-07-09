package platform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	store      Backend
	mailer     Mailer
	automation *Automation
	frps       *FRPSManager
	mux        *http.ServeMux
}

func NewServer(store Backend) *Server { return NewServerWithMailer(store, LogMailer{}) }

func NewServerWithMailer(store Backend, mailer Mailer) *Server {
	return NewServerWithServices(store, mailer, AutomationFromEnv(), FRPSManagerFromEnv())
}

func NewServerWithServices(store Backend, mailer Mailer, automation *Automation, frps *FRPSManager) *Server {
	if mailer == nil {
		mailer = LogMailer{}
	}
	if automation == nil {
		automation = AutomationFromEnv()
	}
	if frps == nil {
		frps = FRPSManagerFromEnv()
	}
	s := &Server{store: store, mailer: mailer, automation: automation, frps: frps, mux: http.NewServeMux()}
	s.routes()
	return s
}
func (s *Server) Handler() http.Handler { return cors(s.mux) }

func (s *Server) routes() {
	s.mux.HandleFunc("/health", s.health)
	s.mux.HandleFunc("/api/auth/send-email-code", s.sendCode)
	s.mux.HandleFunc("/api/auth/register", s.register)
	s.mux.HandleFunc("/api/auth/login", s.login)
	s.mux.HandleFunc("/api/auth/me", s.auth(s.me))
	s.mux.HandleFunc("/api/user/subscription", s.auth(s.subscription))
	s.mux.HandleFunc("/api/user/plans", s.auth(s.paymentPlans))
	s.mux.HandleFunc("/api/user/redeem", s.auth(s.redeem))
	s.mux.HandleFunc("/api/user/purchase-info", s.auth(s.purchaseInfo))
	s.mux.HandleFunc("/api/user/traffic", s.auth(s.userTraffic))
	s.mux.HandleFunc("/api/user/nodes", s.auth(s.userNodes))
	s.mux.HandleFunc("/api/user/topology", s.auth(s.userTopology))
	s.mux.HandleFunc("/api/user/certificates/request", s.auth(s.userRequestCertificate))
	s.mux.HandleFunc("/api/tunnels", s.auth(s.tunnels))
	s.mux.HandleFunc("/api/tunnels/", s.auth(s.tunnelAction))
	s.mux.HandleFunc("/api/speed-tests/tunnels", s.auth(s.createSpeedTestTunnel))
	s.mux.HandleFunc("/api/speed-tests/", s.auth(s.finishSpeedTestTunnel))
	s.mux.HandleFunc("/api/payments/epay/orders", s.auth(s.createEpayOrder))
	s.mux.HandleFunc("/api/payments/epay/notify", s.epayNotify)
	s.mux.HandleFunc("/api/payments/epay/return", s.epayReturn)
	s.mux.HandleFunc("/api/client/heartbeat", s.auth(s.clientHeartbeat))
	s.mux.HandleFunc("/api/client/tunnels", s.auth(s.clientTunnels))
	s.mux.HandleFunc("/api/client/traffic", s.auth(s.clientTraffic))
	s.mux.HandleFunc("/api/nodes/bind", s.nodeBind)
	s.mux.HandleFunc("/api/admin/login", s.adminLogin)
	s.mux.HandleFunc("/api/admin/me", s.adminAuth(s.adminMe))
	s.mux.HandleFunc("/api/admin/dashboard", s.adminAuth(s.adminDashboard))
	s.mux.HandleFunc("/api/admin/topology", s.adminAuth(s.adminTopology))
	s.mux.HandleFunc("/api/admin/plans", s.adminAuth(s.adminPlans))
	s.mux.HandleFunc("/api/admin/plans/", s.adminAuth(s.adminPlanAction))
	s.mux.HandleFunc("/api/admin/users", s.adminAuth(s.adminUsers))
	s.mux.HandleFunc("/api/admin/users/", s.adminAuth(s.adminUserAction))
	s.mux.HandleFunc("/api/admin/redeem-codes", s.adminAuth(s.adminRedeemCodes))
	s.mux.HandleFunc("/api/admin/tunnels", s.adminAuth(s.adminTunnels))
	s.mux.HandleFunc("/api/admin/orders", s.adminAuth(s.adminOrders))
	s.mux.HandleFunc("/api/admin/nodes", s.adminAuth(s.adminNodes))
	s.mux.HandleFunc("/api/admin/nodes/", s.adminAuth(s.adminNodeAction))
	s.mux.HandleFunc("/api/admin/settings", s.adminAuth(s.adminSettings))
	s.mux.HandleFunc("/api/admin/payment-config", s.adminAuth(s.adminPaymentConfig))
	s.mux.HandleFunc("/api/admin/settings/test-mail", s.adminAuth(s.adminTestMail))
	s.mux.HandleFunc("/api/admin/domains/check-cname", s.adminAuth(s.adminCheckCNAME))
	s.mux.HandleFunc("/api/admin/nginx/render-https", s.adminAuth(s.adminRenderHTTPSNginx))
	s.mux.HandleFunc("/api/admin/nginx/test", s.adminAuth(s.adminTestNginx))
	s.mux.HandleFunc("/api/admin/nginx/reload", s.adminAuth(s.adminReloadNginx))
	s.mux.HandleFunc("/api/admin/certificates", s.adminAuth(s.adminCertificates))
	s.mux.HandleFunc("/api/admin/certificates/request", s.adminAuth(s.adminRequestCertificate))
	s.mux.HandleFunc("/api/admin/certificates/renew-due", s.adminAuth(s.adminRenewDueCertificates))
	s.mux.HandleFunc("/api/admin/frps/status", s.adminAuth(s.adminFRPSStatus))
	s.mux.HandleFunc("/api/admin/frps/config", s.adminAuth(s.adminFRPSConfig))
	s.mux.HandleFunc("/api/admin/frps/logs", s.adminAuth(s.adminFRPSLogs))
	s.mux.HandleFunc("/api/admin/frps/restart", s.adminAuth(s.adminFRPSRestart))
	s.mux.HandleFunc("/api/admin/frps/reload", s.adminAuth(s.adminFRPSReload))
	s.mux.HandleFunc("/api/admin/operation-logs", s.adminAuth(s.adminOperationLogs))
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ok(w, map[string]any{"status": "ok", "time": time.Now().Format(time.RFC3339)})
}

type ctxUser struct{ User }

func (s *Server) auth(next func(http.ResponseWriter, *http.Request, User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		u, err := s.store.UserByToken(token)
		if err != nil {
			fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "未登录或登录已过期")
			return
		}
		next(w, r, u)
	}
}

func (s *Server) adminFromRequest(r *http.Request) (AdminUser, bool) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	admin, err := s.store.AdminByToken(token)
	return admin, err == nil
}

func (s *Server) recordAdminOperation(r *http.Request, action, target, detail string) {
	admin, ok := s.adminFromRequest(r)
	if !ok {
		return
	}
	_ = s.store.RecordAdminOperation(AdminOperationLog{AdminID: admin.ID, AdminEmail: admin.Email, Action: action, Target: target, Detail: detail, IP: clientIP(r), CreatedAt: time.Now()})
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return xr
	}
	return r.RemoteAddr
}

func (s *Server) adminOperationLogs(w http.ResponseWriter, r *http.Request) {
	ok(w, s.store.AdminOperationLogs(100))
}

func (s *Server) adminLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var in struct{ Email, Password string }
	if !decode(w, r, &in) {
		return
	}
	token, admin, err := s.store.AdminLogin(in.Email, in.Password)
	if err != nil {
		handleErr(w, err)
		return
	}
	ok(w, map[string]any{"access_token": token, "expires_in": 86400, "admin": admin})
}

func (s *Server) adminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if _, err := s.store.AdminByToken(token); err != nil {
			fail(w, http.StatusUnauthorized, "ADMIN_UNAUTHORIZED", "管理员未登录或登录已过期")
			return
		}
		next(w, r)
	}
}

func (s *Server) adminMe(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	admin, err := s.store.AdminByToken(token)
	if err != nil {
		handleErr(w, err)
		return
	}
	ok(w, admin)
}

func (s *Server) sendCode(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email   string `json:"email"`
		Purpose string `json:"purpose"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Purpose == "" {
		in.Purpose = "register"
	}
	code := s.store.SendEmailCode(in.Email, in.Purpose)
	if err := s.mailer.SendVerificationCode(in.Email, code, in.Purpose); err != nil {
		fail(w, 500, "MAIL_SEND_FAILED", err.Error())
		return
	}
	ok(w, map[string]any{"expires_in": 600})
}
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Code, Password string }
	if !decode(w, r, &in) {
		return
	}
	u, err := s.store.Register(in.Email, in.Code, in.Password)
	if err != nil {
		handleErr(w, err)
		return
	}
	ok(w, u)
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	if !decode(w, r, &in) {
		return
	}
	token, u, err := s.store.Login(in.Email, in.Password)
	if err != nil {
		handleErr(w, err)
		return
	}
	ok(w, map[string]any{"access_token": token, "expires_in": 86400, "user": u})
}
func (s *Server) me(w http.ResponseWriter, r *http.Request, u User) { ok(w, u) }
func (s *Server) subscription(w http.ResponseWriter, r *http.Request, u User) {
	sub, err := s.store.Subscription(u.ID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			ok(w, Subscription{UserID: u.ID, PlanName: "未开通", Status: "inactive"})
			return
		}
		handleErr(w, err)
		return
	}
	ok(w, sub)
}
func (s *Server) redeem(w http.ResponseWriter, r *http.Request, u User) {
	var in struct {
		Code string `json:"code"`
	}
	if !decode(w, r, &in) {
		return
	}
	sub, err := s.store.Redeem(u.ID, in.Code)
	if err != nil {
		handleErr(w, err)
		return
	}
	ok(w, sub)
}
func (s *Server) purchaseInfo(w http.ResponseWriter, r *http.Request, u User) {
	st := s.store.Settings()
	ok(w, map[string]any{"title": "购买套餐", "description": "请通过购买链接获取兑换码或直接支付开通套餐", "button_text": "立即购买", "purchase_url": st.PurchaseURL})
}
func (s *Server) userTraffic(w http.ResponseWriter, r *http.Request, u User) {
	summary, err := s.store.TrafficSummary(u.ID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			ok(w, TrafficSummary{UserID: u.ID})
			return
		}
		handleErr(w, err)
		return
	}
	ok(w, summary)
}
func (s *Server) userNodes(w http.ResponseWriter, r *http.Request, u User) {
	ok(w, s.safeNodes())
}

func (s *Server) safeNodes() []SafeNode {
	nodes := s.store.Nodes()
	out := make([]SafeNode, 0, len(nodes)+1)
	st := s.store.Settings()
	out = append(out, SafeNode{ID: 0, Name: "Default Node", FRPEntryDomain: st.FRPEntryDomain, ServerAddr: st.ServerAddr, FRPServerPort: st.FRPServerPort, TCPPortStart: st.TCPPortStart, TCPPortEnd: st.TCPPortEnd, UDPPortStart: st.UDPPortStart, UDPPortEnd: st.UDPPortEnd, Status: "online"})
	for _, n := range nodes {
		out = append(out, SafeNode{ID: n.ID, Name: n.Name, PublicURL: n.PublicURL, FRPEntryDomain: n.FRPEntryDomain, ServerAddr: n.ServerAddr, FRPServerPort: n.FRPServerPort, TCPPortStart: n.TCPPortStart, TCPPortEnd: n.TCPPortEnd, UDPPortStart: n.UDPPortStart, UDPPortEnd: n.UDPPortEnd, Status: n.Status, LastSeenAt: n.LastSeenAt})
	}
	return out
}

func (s *Server) userTopology(w http.ResponseWriter, r *http.Request, u User) {
	sub, err := s.store.Subscription(u.ID)
	if err != nil {
		sub = Subscription{UserID: u.ID, PlanName: "Inactive", Status: "inactive"}
	}
	traffic, err := s.store.TrafficSummary(u.ID)
	if err != nil {
		left := sub.TrafficLimitBytes - sub.TrafficUsedBytes
		if left < 0 {
			left = 0
		}
		traffic = TrafficSummary{UserID: u.ID, TrafficLimitBytes: sub.TrafficLimitBytes, TrafficUsedBytes: sub.TrafficUsedBytes, TrafficLeftBytes: left}
	}
	tunnels := s.store.Tunnels(u.ID)
	counts := map[string]int{}
	activeCount := 0
	for _, t := range tunnels {
		if t.Status == "deleted" {
			continue
		}
		activeCount++
		counts[t.Type]++
	}
	ok(w, UserTopology{
		Role:         "User Console",
		User:         u,
		Subscription: sub,
		Traffic:      traffic,
		TunnelCount:  activeCount,
		TunnelCounts: counts,
		Nodes:        s.safeNodes(),
		Downloads: []DownloadArtifact{
			{Platform: "windows", Label: "Windows Client", URL: "/downloads/windows/FrpTunnelClient-0.1.3-windows-amd64.zip"},
			{Platform: "linux", Label: "Linux Client", URL: "/downloads/linux/FrpTunnelClient-0.1.3-linux-amd64.tar.gz"},
		},
		RoleFlow: []TopologyLink{
			{From: "User Console", To: "Master", Description: "Create tunnels, purchase plans, redeem plans"},
			{From: "Client(FRPC)", To: "Master", Description: "Fetch current user's frpc config"},
			{From: "Client(FRPC)", To: "Server(FRPS)", Description: "Connect to FRPS node using generated config"},
			{From: "Visitor", To: "Server(FRPS)", Description: "Access public entry and forward to local service"},
		},
		GeneratedAt: time.Now(),
	})
}

func (s *Server) userRequestCertificate(w http.ResponseWriter, r *http.Request, u User) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var in struct {
		Domain string `json:"domain"`
		Email  string `json:"email"`
	}
	if !decode(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Email) == "" {
		in.Email = u.Email
	}
	sub, err := s.store.Subscription(u.ID)
	if err != nil || sub.Status != "active" || !sub.AllowCustomDomain || !sub.AllowAutoCert {
		handleErr(w, ErrForbidden)
		return
	}
	domain := sanitizeDomain(in.Domain)
	if domain == "" || !s.userOwnsDomain(u.ID, domain) {
		handleErr(w, ErrForbidden)
		return
	}
	in.Domain = domain
	res, err := s.automation.RequestCertificate(r.Context(), in.Domain, in.Email)
	status := "issued"
	errorMessage := ""
	if res.DryRun {
		status = "dry_run"
	}
	if err != nil {
		status = "failed"
		errorMessage = err.Error()
	}
	certPath, keyPath := s.automation.CertificatePaths(in.Domain)
	issuedAt, expiresAt := s.automation.InspectCertificate(in.Domain)
	record, saveErr := s.store.SaveCertificate(CertificateRecord{UserID: u.ID, Domain: in.Domain, Status: status, IssuedAt: issuedAt, ExpiresAt: expiresAt, CertPath: certPath, KeyPath: keyPath, LastCommand: res.Command, LastOutput: res.Output, ErrorMessage: errorMessage})
	if saveErr != nil {
		fail(w, 500, "CERTIFICATE_SAVE_FAILED", saveErr.Error())
		return
	}
	if err != nil {
		fail(w, 500, "CERTIFICATE_REQUEST_FAILED", err.Error()+"\n"+res.Output)
		return
	}
	ok(w, map[string]any{"result": res, "record": record})
}

func (s *Server) userOwnsDomain(userID int64, domain string) bool {
	for _, t := range s.store.Tunnels(userID) {
		if t.Status == "deleted" || t.SpeedTest {
			continue
		}
		if sanitizeDomain(t.Domain) == domain && (t.Type == "http" || t.Type == "https") {
			return true
		}
	}
	return false
}

func (s *Server) tunnels(w http.ResponseWriter, r *http.Request, u User) {
	switch r.Method {
	case http.MethodGet:
		s.store.CleanupExpiredSpeedTestTunnels(time.Now())
		tunnels := s.store.Tunnels(u.ID)
		if tunnels == nil {
			tunnels = []Tunnel{}
		}
		ok(w, tunnels)
	case http.MethodPost:
		s.store.CleanupExpiredSpeedTestTunnels(time.Now())
		var in Tunnel
		if !decode(w, r, &in) {
			return
		}
		t, err := s.store.CreateTunnel(u.ID, in)
		if err != nil {
			handleErr(w, err)
			return
		}
		ok(w, t)
	default:
		w.WriteHeader(405)
	}
}

func (s *Server) tunnelAction(w http.ResponseWriter, r *http.Request, u User) {
	id, action, okPath := parseTunnelAction(r.URL.Path)
	if !okPath {
		fail(w, 404, "TUNNEL_ACTION_NOT_FOUND", "tunnel action not found")
		return
	}
	if action == "" {
		if r.Method != http.MethodDelete {
			w.WriteHeader(405)
			return
		}
		if err := s.store.DeleteTunnel(u.ID, id); err != nil {
			handleErr(w, err)
			return
		}
		ok(w, map[string]any{"deleted": true, "id": id})
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	switch action {
	case "start":
		t, err := s.store.UpdateTunnelStatus(u.ID, id, "active")
		if err != nil {
			handleErr(w, err)
			return
		}
		ok(w, t)
	case "stop":
		t, err := s.store.UpdateTunnelStatus(u.ID, id, "disabled")
		if err != nil {
			handleErr(w, err)
			return
		}
		ok(w, t)
	default:
		fail(w, 404, "TUNNEL_ACTION_NOT_FOUND", "unknown tunnel action")
	}
}

func parseTunnelAction(path string) (int64, string, bool) {
	rest := strings.TrimPrefix(path, "/api/tunnels/")
	if rest == path || rest == "" {
		return 0, "", false
	}
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		return 0, "", false
	}
	if len(parts) == 1 {
		return id, "", true
	}
	return id, parts[1], true
}

func (s *Server) clientHeartbeat(w http.ResponseWriter, r *http.Request, u User) {
	ok(w, map[string]any{"status": "online", "server_time": time.Now().Format(time.RFC3339)})
}
func (s *Server) clientTraffic(w http.ResponseWriter, r *http.Request, u User) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var in struct {
		Reports []TrafficReport `json:"reports"`
	}
	if !decode(w, r, &in) {
		return
	}
	summary, err := s.store.ReportTraffic(u.ID, in.Reports)
	if err != nil {
		handleErr(w, err)
		return
	}
	ok(w, summary)
}
func (s *Server) clientTunnels(w http.ResponseWriter, r *http.Request, u User) {
	frpToken := getenv("FRP_TOKEN", "")
	if frpToken == "" || frpToken == "change-me" {
		fail(w, 500, "FRP_TOKEN_NOT_CONFIGURED", "FRP_TOKEN must be configured")
		return
	}
	s.store.CleanupExpiredSpeedTestTunnels(time.Now())
	st := s.store.Settings()
	sub, err := s.store.Subscription(u.ID)
	bandwidth := 0
	if err == nil && sub.Status == "active" {
		bandwidth = sub.BandwidthKbps
	}
	tunnels := s.store.Tunnels(u.ID)
	if idText := strings.TrimSpace(r.URL.Query().Get("speed_test_id")); idText != "" {
		id, _ := strconv.ParseInt(idText, 10, 64)
		filtered := []Tunnel{}
		for _, t := range tunnels {
			if t.ID == id && t.SpeedTest && t.Status != "deleted" {
				filtered = append(filtered, t)
				if t.NodeID > 0 {
					if node, err := s.store.Node(t.NodeID); err == nil {
						st = settingsFromNode(st, node)
					}
				}
			}
		}
		tunnels = filtered
	}
	for i := range tunnels {
		tunnels[i].EffectiveBandwidthKbps = effectiveBandwidth(bandwidth, tunnels[i].BandwidthKbps)
	}
	ok(w, map[string]any{"server_addr": st.ServerAddr, "server_port": st.FRPServerPort, "token": frpToken, "bandwidth_limit_kbps": bandwidth, "tunnels": tunnels})
}

func (s *Server) createSpeedTestTunnel(w http.ResponseWriter, r *http.Request, u User) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var in SpeedTestTunnelRequest
	if !decode(w, r, &in) {
		return
	}
	s.store.CleanupExpiredSpeedTestTunnels(time.Now())
	t, err := s.store.CreateSpeedTestTunnel(u.ID, in)
	if err != nil {
		handleErr(w, err)
		return
	}
	ok(w, t)
}

func (s *Server) finishSpeedTestTunnel(w http.ResponseWriter, r *http.Request, u User) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	id, action, okPath := parseSpeedTestAction(r.URL.Path)
	if !okPath {
		fail(w, 404, "SPEED_TEST_ACTION_NOT_FOUND", "speed test action not found")
		return
	}
	switch action {
	case "finish":
		if err := s.store.FinishSpeedTestTunnel(u.ID, id); err != nil {
			handleErr(w, err)
			return
		}
		ok(w, map[string]any{"finished": true, "id": id})
	case "run":
		s.runSpeedTestProbe(w, r, u, id)
	default:
		fail(w, 404, "SPEED_TEST_ACTION_NOT_FOUND", "speed test action not found")
	}
}

func (s *Server) runSpeedTestProbe(w http.ResponseWriter, r *http.Request, u User, id int64) {
	var in SpeedTestProbeRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&in)
	}
	if in.DurationSeconds <= 0 {
		in.DurationSeconds = 45
	}
	if in.DurationSeconds > 120 {
		in.DurationSeconds = 120
	}
	ctx, cancel := contextWithRequestTimeout(r, time.Duration(in.DurationSeconds)*time.Second)
	defer cancel()
	tunnel, err := s.store.SpeedTestTunnel(u.ID, id)
	if err != nil {
		handleErr(w, err)
		return
	}
	metrics, err := runSpeedProbe(ctx, tunnel.Type, tunnel.PublicURL, in.DownloadBytes, in.UploadBytes)
	if err != nil {
		fail(w, 502, "SPEED_TEST_PROBE_FAILED", err.Error())
		return
	}
	_, _ = s.store.ReportTraffic(u.ID, []TrafficReport{{TunnelID: id, BytesIn: metrics.BytesIn, BytesOut: metrics.BytesOut}})
	finished := false
	if err := s.store.FinishSpeedTestTunnel(u.ID, id); err == nil {
		finished = true
	}
	observed := metrics.DownloadAverageKbps
	if metrics.UploadAverageKbps > observed {
		observed = metrics.UploadAverageKbps
	}
	limitRatio := 0.0
	if tunnel.EffectiveBandwidthKbps > 0 {
		limitRatio = observed / float64(tunnel.EffectiveBandwidthKbps)
	}
	ok(w, SpeedTestRunResult{Tunnel: tunnel, Metrics: metrics, EffectiveBandwidthLimitKbps: tunnel.EffectiveBandwidthKbps, LimitRatio: limitRatio, BottleneckHint: speedBottleneckHint(limitRatio), Finished: finished})
}

func speedBottleneckHint(limitRatio float64) string {
	if limitRatio >= 0.85 {
		return "Speed is close to the package limit; the likely bottleneck is package bandwidth or node egress."
	}
	if limitRatio > 0 {
		return "Speed is below the package limit; the likely bottleneck is local bandwidth, node load, or network path."
	}
	return "No valid throughput data was collected; check the local client, temporary tunnel, and node connectivity."
}

func contextWithRequestTimeout(r *http.Request, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), timeout)
}

func parseSpeedTestAction(path string) (int64, string, bool) {
	rest := strings.TrimPrefix(path, "/api/speed-tests/")
	if rest == path || rest == "" {
		return 0, "", false
	}
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 || len(parts) < 2 {
		return 0, "", false
	}
	return id, parts[1], true
}
func (s *Server) adminDashboard(w http.ResponseWriter, r *http.Request) {
	all := s.store.AllTunnels()
	counts := map[string]int{}
	for _, t := range all {
		counts[t.Type]++
	}
	ok(w, map[string]any{"total_tunnels": len(all), "tunnel_counts": counts, "online_clients": 0, "today_traffic_bytes": s.store.TotalTrafficToday()})
}

func (s *Server) paymentMethodStatuses() []PaymentMethodStatus {
	cfg := epayFromEnv()
	online := cfg.enabled()
	return []PaymentMethodStatus{
		{Provider: "epay", Method: "WeChat Pay", PayType: "wxpay", Channel: "wxpay_zg", Online: online, APIBase: cfg.BaseURL, SubmitURL: cfg.SubmitURL},
		{Provider: "epay", Method: "Alipay", PayType: "alipay", Channel: "alipay_zg", Online: online, APIBase: cfg.BaseURL, SubmitURL: cfg.SubmitURL},
	}
}

func (s *Server) adminTopology(w http.ResponseWriter, r *http.Request) {
	all := s.store.AllTunnels()
	counts := map[string]int{}
	for _, t := range all {
		if t.Status == "deleted" {
			continue
		}
		counts[t.Type]++
	}
	nodes := s.store.Nodes()
	onlineNodes := 0
	for _, n := range nodes {
		if n.Status == "online" {
			onlineNodes++
		}
	}
	ok(w, AdminTopology{
		Role:                    "Admin Console",
		UserCount:               len(s.store.Users()),
		ActiveSubscriptionCount: s.store.ActiveSubscriptionCount(),
		TunnelCount:             len(all),
		TunnelCounts:            counts,
		NodeCount:               len(nodes),
		OnlineNodeCount:         onlineNodes,
		TodayTrafficBytes:       s.store.TotalTrafficToday(),
		PaymentMethods:          s.paymentMethodStatuses(),
		RecentOrders:            s.store.PaymentOrders(20),
		RecentOperations:        s.store.AdminOperationLogs(20),
		Nodes:                   nodes,
		RoleFlow: []TopologyLink{
			{From: "Admin Console", To: "Master", Description: "Manage users, plans, orders, nodes and certificates"},
			{From: "User Console", To: "Master", Description: "Users create tunnels, purchase plans and redeem plans"},
			{From: "Master", To: "Server(FRPS)", Description: "Operate nodes through node-agent"},
			{From: "Client(FRPC)", To: "Server(FRPS)", Description: "Local client connects to frps"},
			{From: "Visitor", To: "Server(FRPS)", Description: "Access public entrypoint"},
		},
		GeneratedAt: time.Now(),
	})
}

func (s *Server) adminPlans(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ok(w, s.store.Plans())
	case http.MethodPost:
		var in Plan
		if !decode(w, r, &in) {
			return
		}
		plan, err := s.store.CreatePlan(in)
		if err != nil {
			handleErr(w, err)
			return
		}
		s.recordAdminOperation(r, "plan.create", fmt.Sprintf("plan:%d", plan.ID), plan.Name)
		ok(w, plan)
	default:
		w.WriteHeader(405)
	}
}

func (s *Server) adminPlanAction(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/plans/"), "/")
	id, err := strconv.ParseInt(rest, 10, 64)
	if rest == "" || err != nil || id <= 0 {
		fail(w, 404, "PLAN_NOT_FOUND", "plan not found")
		return
	}
	switch r.Method {
	case http.MethodPut:
		var in Plan
		if !decode(w, r, &in) {
			return
		}
		plan, err := s.store.UpdatePlan(id, in)
		if err != nil {
			handleErr(w, err)
			return
		}
		s.recordAdminOperation(r, "plan.update", fmt.Sprintf("plan:%d", plan.ID), plan.Name)
		ok(w, plan)
	default:
		w.WriteHeader(405)
	}
}

func (s *Server) adminPaymentConfig(w http.ResponseWriter, r *http.Request) {
	cfg := epayFromEnv()
	ok(w, map[string]any{
		"provider":         "epay",
		"enabled":          cfg.enabled(),
		"pid_set":          cfg.PID != "",
		"key_set":          cfg.Key != "",
		"api_base":         cfg.BaseURL,
		"submit_url":       cfg.SubmitURL,
		"site_name":        cfg.SiteName,
		"public_url":       cfg.PublicURL,
		"default_pay_type": cfg.DefaultPayType,
		"configured_by":    "environment",
		"methods":          s.paymentMethodStatuses(),
	})
}
func (s *Server) adminUsers(w http.ResponseWriter, r *http.Request) { ok(w, s.store.Users()) }

func (s *Server) adminUserAction(w http.ResponseWriter, r *http.Request) {
	idText := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/users/"), "/")
	id, err := strconv.ParseInt(idText, 10, 64)
	if idText == "" || err != nil || id <= 0 {
		fail(w, 404, "USER_NOT_FOUND", "user not found")
		return
	}
	if r.Method != http.MethodPut {
		w.WriteHeader(405)
		return
	}
	var in struct {
		Status string `json:"status"`
		PlanID int64  `json:"plan_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	user, sub, err := s.store.UpdateUser(id, in.Status, in.PlanID)
	if err != nil {
		handleErr(w, err)
		return
	}
	detail := fmt.Sprintf("status=%s plan_id=%d", in.Status, in.PlanID)
	s.recordAdminOperation(r, "user.update", user.Email, detail)
	ok(w, map[string]any{"user": user, "subscription": sub})
}
func (s *Server) adminRedeemCodes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ok(w, s.store.RedeemCodes())
	case http.MethodPost:
		var in struct {
			PlanID int64  `json:"plan_id"`
			Count  int    `json:"count"`
			Prefix string `json:"prefix"`
		}
		if !decode(w, r, &in) {
			return
		}
		codes, err := s.store.CreateRedeemCodes(in.PlanID, in.Count, in.Prefix)
		if err != nil {
			handleErr(w, err)
			return
		}
		s.recordAdminOperation(r, "redeem_codes.create", fmt.Sprintf("plan:%d", in.PlanID), fmt.Sprintf("count=%d prefix=%s", in.Count, in.Prefix))
		ok(w, codes)
	default:
		w.WriteHeader(405)
	}
}
func (s *Server) adminTunnels(w http.ResponseWriter, r *http.Request) { ok(w, s.store.AllTunnels()) }

func (s *Server) adminOrders(w http.ResponseWriter, r *http.Request) {
	ok(w, s.store.PaymentOrders(200))
}

func (s *Server) adminCheckCNAME(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var in struct {
		Domain string `json:"domain"`
		Target string `json:"target"`
	}
	if !decode(w, r, &in) {
		return
	}
	ok(w, s.automation.CheckCNAME(in.Domain, in.Target))
}

func (s *Server) adminRenderHTTPSNginx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var in struct {
		Domain string `json:"domain"`
	}
	if !decode(w, r, &in) {
		return
	}
	res, err := s.automation.WriteHTTPSConfig(in.Domain)
	if err != nil {
		handleErr(w, err)
		return
	}
	ok(w, res)
}

func (s *Server) adminNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var in Node
		if !decode(w, r, &in) {
			return
		}
		node, err := s.store.CreateNode(in)
		if err != nil {
			handleErr(w, err)
			return
		}
		s.recordAdminOperation(r, "node.create", node.Name, node.AgentURL)
		ok(w, node)
		return
	}
	ok(w, s.store.Nodes())
}

func (s *Server) adminNodeAction(w http.ResponseWriter, r *http.Request) {
	id, action, parseOK := parseNodeAction(r.URL.Path)
	if !parseOK {
		fail(w, 404, "NODE_ACTION_NOT_FOUND", "node action not found")
		return
	}
	node, err := s.store.Node(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	if action == "" {
		ok(w, node)
		return
	}
	if action == "delete" {
		if r.Method != http.MethodDelete && r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		if err := s.store.DeleteNode(node.ID); err != nil {
			handleErr(w, err)
			return
		}
		s.recordAdminOperation(r, "node.delete", node.Name, node.AgentURL)
		ok(w, map[string]any{"deleted": true, "id": node.ID})
		return
	}
	client := NewNodeAgentClient(node.AgentURL, node.AgentToken)
	switch action {
	case "status":
		st, err := client.FRPSStatus(r.Context())
		if err != nil {
			_, _ = s.store.UpdateNodeStatus(node.ID, "error", err.Error())
			fail(w, 502, "NODE_STATUS_FAILED", err.Error())
			return
		}
		_, _ = s.store.UpdateNodeStatus(node.ID, "online", "")
		ok(w, st)
	case "frps-config":
		out, err := client.FRPSConfig(r.Context())
		if err != nil {
			fail(w, 502, "NODE_FRPS_CONFIG_FAILED", err.Error())
			return
		}
		ok(w, out)
	case "frps-logs":
		out, err := client.FRPSLogs(r.Context())
		if err != nil {
			fail(w, 502, "NODE_FRPS_LOGS_FAILED", err.Error())
			return
		}
		ok(w, out)
	case "frps-restart":
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		out, err := client.FRPSRestart(r.Context())
		if err != nil {
			fail(w, 502, "NODE_FRPS_RESTART_FAILED", err.Error()+"\n"+out.Output)
			return
		}
		s.recordAdminOperation(r, "node.frps.restart", node.Name, out.Output)
		ok(w, out)
	case "frps-reload":
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		out, err := client.FRPSReload(r.Context())
		if err != nil {
			fail(w, 502, "NODE_FRPS_RELOAD_FAILED", err.Error()+"\n"+out.Output)
			return
		}
		s.recordAdminOperation(r, "node.frps.reload", node.Name, out.Output)
		ok(w, out)
	case "nginx-test":
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		out, err := client.TestNginx(r.Context())
		if err != nil {
			fail(w, 502, "NODE_NGINX_TEST_FAILED", err.Error()+"\n"+out)
			return
		}
		s.recordAdminOperation(r, "node.nginx.test", node.Name, out)
		ok(w, map[string]string{"output": out})
	case "nginx-reload":
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		out, err := client.ReloadNginx(r.Context())
		if err != nil {
			fail(w, 502, "NODE_NGINX_RELOAD_FAILED", err.Error()+"\n"+out)
			return
		}
		s.recordAdminOperation(r, "node.nginx.reload", node.Name, out)
		ok(w, map[string]string{"output": out})
	default:
		fail(w, 404, "NODE_ACTION_NOT_FOUND", "unknown node action")
	}
}

func (s *Server) nodeBind(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var in NodeBindRequest
	if !decode(w, r, &in) {
		return
	}
	node, err := s.store.BindNode(in)
	if err != nil {
		handleErr(w, err)
		return
	}
	ok(w, map[string]any{"node": node, "agent_token": node.AgentToken})
}

func parseNodeAction(path string) (int64, string, bool) {
	rest := strings.TrimPrefix(path, "/api/admin/nodes/")
	if rest == path || rest == "" {
		return 0, "", false
	}
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id <= 0 {
		return 0, "", false
	}
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	return id, action, true
}

func (s *Server) adminFRPSStatus(w http.ResponseWriter, r *http.Request) {
	ok(w, s.frps.Status(r.Context()))
}

func (s *Server) adminFRPSConfig(w http.ResponseWriter, r *http.Request) {
	if s.frps.NodeAgent.enabled() {
		out, err := s.frps.NodeAgent.FRPSConfig(r.Context())
		if err != nil {
			fail(w, 500, "FRPS_CONFIG_READ_FAILED", err.Error())
			return
		}
		ok(w, out)
		return
	}
	text, err := s.frps.Config()
	if err != nil {
		fail(w, 500, "FRPS_CONFIG_READ_FAILED", err.Error())
		return
	}
	ok(w, map[string]any{"config": text, "path": s.frps.ConfigPath})
}

func (s *Server) adminFRPSLogs(w http.ResponseWriter, r *http.Request) {
	if s.frps.NodeAgent.enabled() {
		out, err := s.frps.NodeAgent.FRPSLogs(r.Context())
		if err != nil {
			fail(w, 500, "FRPS_LOG_READ_FAILED", err.Error())
			return
		}
		ok(w, out)
		return
	}
	text, err := s.frps.Logs(65536)
	if err != nil {
		fail(w, 500, "FRPS_LOG_READ_FAILED", err.Error())
		return
	}
	ok(w, map[string]any{"logs": text, "path": s.frps.LogPath})
}

func (s *Server) adminFRPSRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	res, err := s.frps.Restart(r.Context())
	if err != nil {
		fail(w, 500, "FRPS_RESTART_FAILED", err.Error()+"\n"+res.Output)
		return
	}
	s.recordAdminOperation(r, "frps.restart", "frps", res.Output)
	ok(w, res)
}

func (s *Server) adminFRPSReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	res, err := s.frps.Reload(r.Context())
	if err != nil {
		fail(w, 500, "FRPS_RELOAD_FAILED", err.Error()+"\n"+res.Output)
		return
	}
	s.recordAdminOperation(r, "frps.reload", "frps", res.Output)
	ok(w, res)
}

func (s *Server) adminCertificates(w http.ResponseWriter, r *http.Request) {
	ok(w, s.store.Certificates())
}

func (s *Server) adminRenewDueCertificates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var in struct {
		Force bool `json:"force"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	renewer := NewCertificateRenewer(s.store, s.automation)
	res, err := renewer.RenewDue(r.Context(), in.Force)
	if err != nil {
		fail(w, 500, "CERTIFICATE_RENEWAL_FAILED", err.Error())
		return
	}
	s.recordAdminOperation(r, "certificate.renew_due", "certificates", res.String())
	ok(w, res)
}

func (s *Server) adminRequestCertificate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var in struct {
		Domain string `json:"domain"`
		Email  string `json:"email"`
	}
	if !decode(w, r, &in) {
		return
	}
	res, err := s.automation.RequestCertificate(r.Context(), in.Domain, in.Email)
	status := "issued"
	errorMessage := ""
	if res.DryRun {
		status = "dry_run"
	}
	if err != nil {
		status = "failed"
		errorMessage = err.Error()
	}
	certPath, keyPath := s.automation.CertificatePaths(in.Domain)
	issuedAt, expiresAt := s.automation.InspectCertificate(in.Domain)
	record, saveErr := s.store.SaveCertificate(CertificateRecord{Domain: in.Domain, Status: status, IssuedAt: issuedAt, ExpiresAt: expiresAt, CertPath: certPath, KeyPath: keyPath, LastCommand: res.Command, LastOutput: res.Output, ErrorMessage: errorMessage})
	if saveErr != nil {
		fail(w, 500, "CERTIFICATE_SAVE_FAILED", saveErr.Error())
		return
	}
	if err != nil {
		fail(w, 500, "CERTIFICATE_REQUEST_FAILED", err.Error()+"\n"+res.Output)
		return
	}
	s.recordAdminOperation(r, "certificate.request", record.Domain, record.Status)
	ok(w, map[string]any{"result": res, "record": record})
}

func (s *Server) adminTestNginx(w http.ResponseWriter, r *http.Request) {
	out, err := s.automation.TestNginx(r.Context())
	if err != nil {
		fail(w, 500, "NGINX_TEST_FAILED", err.Error()+"\n"+out)
		return
	}
	s.recordAdminOperation(r, "nginx.test", "nginx", out)
	ok(w, map[string]any{"output": out})
}

func (s *Server) adminReloadNginx(w http.ResponseWriter, r *http.Request) {
	out, err := s.automation.ReloadNginx(r.Context())
	if err != nil {
		fail(w, 500, "NGINX_RELOAD_FAILED", err.Error()+"\n"+out)
		return
	}
	s.recordAdminOperation(r, "nginx.reload", "nginx", out)
	ok(w, map[string]any{"output": out})
}

func (s *Server) adminTestMail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var in struct {
		Email string `json:"email"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.mailer.Send(in.Email, "FRP 平台测试邮件", "这是一封来自 FRP 平台的测试邮件。邮件服务器配置可用。\\n"); err != nil {
		fail(w, 500, "MAIL_SEND_FAILED", err.Error())
		return
	}
	s.recordAdminOperation(r, "mail.test", in.Email, "sent")
	ok(w, map[string]any{"sent": true})
}

func (s *Server) adminSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPut {
		var in Settings
		if !decode(w, r, &in) {
			return
		}
		updated := s.store.UpdateSettings(in)
		s.recordAdminOperation(r, "settings.update", "system_settings", "updated")
		ok(w, updated)
		return
	}
	ok(w, s.store.Settings())
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Method != http.MethodGet {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(v); err != nil {
			fail(w, 400, "BAD_REQUEST", fmt.Sprintf("请求体错误: %v", err))
			return false
		}
	}
	return true
}
func ok(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data, "message": "ok"})
}
func fail(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "code": code, "message": msg})
}
func handleErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrUnauthorized):
		fail(w, 401, "UNAUTHORIZED", "认证失败")
	case errors.Is(err, ErrForbidden):
		fail(w, 403, "FORBIDDEN", "套餐无权限、已过期或资源不可用")
	case errors.Is(err, ErrConflict):
		fail(w, 409, "CONFLICT", "资源冲突")
	case errors.Is(err, ErrNotFound):
		fail(w, 404, "NOT_FOUND", "资源不存在")
	default:
		fail(w, 400, "BAD_REQUEST", err.Error())
	}
}
