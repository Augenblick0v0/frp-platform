package platform

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

var ErrNotFound = errors.New("not found")
var ErrUnauthorized = errors.New("unauthorized")
var ErrForbidden = errors.New("forbidden")
var ErrConflict = errors.New("conflict")

const sessionTTL = 24 * time.Hour

type sessionRecord struct {
	UserID    int64
	ExpiresAt time.Time
}

type Store struct {
	mu              sync.Mutex
	users           map[int64]User
	usersByEmail    map[string]int64
	admins          map[int64]AdminUser
	adminsByEmail   map[string]int64
	sessions        map[string]sessionRecord
	adminSessions   map[string]sessionRecord
	emailCodes      map[string]string
	plans           map[int64]Plan
	paymentOrders   map[string]PaymentOrder
	redeemCodes     map[string]RedeemCode
	subscriptions   map[int64]Subscription
	tunnels         map[int64]Tunnel
	domains         map[string]int64
	usedPorts       map[string]bool
	nextUserID      int64
	nextAdminID     int64
	nextPlanID      int64
	nextPaymentID   int64
	nextTunnelID    int64
	settings        Settings
	todayTraffic    int64
	certificates    map[string]CertificateRecord
	nodes           map[int64]Node
	nodesByBind     map[string]int64
	nextNodeID      int64
	nextCertID      int64
	operationLogs   []AdminOperationLog
	nextOperationID int64
}

func NewStore() *Store {
	s := &Store{
		users: map[int64]User{}, usersByEmail: map[string]int64{}, admins: map[int64]AdminUser{}, adminsByEmail: map[string]int64{}, sessions: map[string]sessionRecord{}, adminSessions: map[string]sessionRecord{}, emailCodes: map[string]string{},
		plans: map[int64]Plan{}, paymentOrders: map[string]PaymentOrder{}, redeemCodes: map[string]RedeemCode{}, subscriptions: map[int64]Subscription{}, tunnels: map[int64]Tunnel{}, domains: map[string]int64{}, certificates: map[string]CertificateRecord{}, nodes: map[int64]Node{}, nodesByBind: map[string]int64{},
		usedPorts: map[string]bool{}, nextUserID: 1, nextAdminID: 1, nextPlanID: 1, nextPaymentID: 1, nextTunnelID: 1, nextNodeID: 1, nextCertID: 1, nextOperationID: 1,
		settings: Settings{PlatformDomain: "example.com", FRPEntryDomain: "frp.example.com", ServerAddr: "frp.example.com", FRPServerPort: 7000, TCPPortStart: 20000, TCPPortEnd: 29999, UDPPortStart: 30000, UDPPortEnd: 39999, PurchaseURL: "https://example.com/buy"},
	}
	plan := Plan{ID: s.nextPlanID, Name: "高级套餐", Description: "支持 TCP/UDP/HTTP/HTTPS、自定义域名和自动证书", PriceCents: 990, DurationDays: 30, TrafficLimitBytes: 100 * 1024 * 1024 * 1024, BandwidthKbps: 10240, MaxTunnels: 20, MaxTCPTunnels: 10, MaxUDPTunnels: 10, MaxHTTPTunnels: 10, MaxHTTPSTunnels: 10, AllowTCP: true, AllowUDP: true, AllowHTTP: true, AllowHTTPS: true, AllowCustomDomain: true, MaxDomains: 10, AllowAutoCert: true, Status: "active"}
	s.plans[plan.ID] = plan
	s.nextPlanID++
	if InsecureDefaultsAllowed() {
		s.redeemCodes["DEMO-PLAN-2026"] = RedeemCode{Code: "DEMO-PLAN-2026", PlanID: plan.ID, Status: "unused"}
	}
	admin := AdminUser{ID: s.nextAdminID, Email: strings.ToLower(getenv("ADMIN_EMAIL", "admin@example.com")), Password: mustHashPassword(getenv("ADMIN_PASSWORD", "admin123456")), Status: "active", CreatedAt: time.Now()}
	s.nextAdminID++
	s.admins[admin.ID] = admin
	s.adminsByEmail[admin.Email] = admin.ID
	return s
}

func (s *Store) AdminLogin(email, password string) (string, AdminUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.adminsByEmail[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return "", AdminUser{}, ErrUnauthorized
	}
	admin := s.admins[id]
	if !VerifyPassword(admin.Password, password) || admin.Status != "active" {
		return "", AdminUser{}, ErrUnauthorized
	}
	token, err := randomToken("admin")
	if err != nil {
		return "", AdminUser{}, err
	}
	s.adminSessions[token] = sessionRecord{UserID: admin.ID, ExpiresAt: time.Now().Add(sessionTTL)}
	return token, admin, nil
}

func (s *Store) AdminByToken(token string) (AdminUser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.adminSessions[token]
	if !ok {
		return AdminUser{}, ErrUnauthorized
	}
	if time.Now().After(rec.ExpiresAt) {
		delete(s.adminSessions, token)
		return AdminUser{}, ErrUnauthorized
	}
	admin := s.admins[rec.UserID]
	if admin.Status != "active" {
		return AdminUser{}, ErrUnauthorized
	}
	return admin, nil
}

func (s *Store) SendEmailCode(email, purpose string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	code, err := randomDigits(6)
	if err != nil {
		code = fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	s.emailCodes[emailCodeKey(email, purpose)] = code
	return code
}

func emailCodeKey(email, purpose string) string {
	return strings.ToLower(strings.TrimSpace(email)) + ":" + strings.TrimSpace(purpose)
}

func (s *Store) DebugEmailCode(email, purpose string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.emailCodes[emailCodeKey(email, purpose)]
}

func (s *Store) Register(email, code, password string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	email, err = NormalizeRegistrationInput(email, password)
	if err != nil {
		return User{}, err
	}
	if s.emailCodes[emailCodeKey(email, "register")] != code {
		return User{}, fmt.Errorf("invalid verification code")
	}
	if _, ok := s.usersByEmail[email]; ok {
		return User{}, ErrConflict
	}
	hash, err := HashPassword(password)
	if err != nil {
		return User{}, err
	}
	u := User{ID: s.nextUserID, Email: email, Password: hash, Status: "active", CreatedAt: time.Now()}
	s.nextUserID++
	s.users[u.ID] = u
	s.usersByEmail[email] = u.ID
	return u, nil
}

func (s *Store) Login(email, password string) (string, User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.usersByEmail[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return "", User{}, ErrUnauthorized
	}
	u := s.users[id]
	if !VerifyPassword(u.Password, password) || u.Status != "active" {
		return "", User{}, ErrUnauthorized
	}
	token, err := randomToken("token")
	if err != nil {
		return "", User{}, err
	}
	s.sessions[token] = sessionRecord{UserID: u.ID, ExpiresAt: time.Now().Add(sessionTTL)}
	return token, u, nil
}

func (s *Store) UserByToken(token string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.sessions[token]
	if !ok {
		return User{}, ErrUnauthorized
	}
	if time.Now().After(rec.ExpiresAt) {
		delete(s.sessions, token)
		return User{}, ErrUnauthorized
	}
	u := s.users[rec.UserID]
	if u.Status != "active" {
		return User{}, ErrUnauthorized
	}
	return u, nil
}

func (s *Store) Plans() []Plan {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Plan{}
	for _, p := range s.plans {
		out = append(out, p)
	}
	return out
}

func (s *Store) CreatePaymentOrder(order PaymentOrder) (PaymentOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[order.UserID]; !ok {
		return PaymentOrder{}, ErrNotFound
	}
	if _, ok := s.plans[order.PlanID]; !ok {
		return PaymentOrder{}, ErrNotFound
	}
	if strings.TrimSpace(order.OutTradeNo) == "" {
		return PaymentOrder{}, fmt.Errorf("out_trade_no required")
	}
	if _, exists := s.paymentOrders[order.OutTradeNo]; exists {
		return PaymentOrder{}, ErrConflict
	}
	order.ID = s.nextPaymentID
	s.nextPaymentID++
	if order.Status == "" {
		order.Status = "pending"
	}
	if order.Provider == "" {
		order.Provider = "epay"
	}
	if order.CreatedAt.IsZero() {
		order.CreatedAt = time.Now()
	}
	s.paymentOrders[order.OutTradeNo] = order
	return order, nil
}

func (s *Store) PaymentOrderByOutTradeNo(outTradeNo string) (PaymentOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.paymentOrders[strings.TrimSpace(outTradeNo)]
	if !ok {
		return PaymentOrder{}, ErrNotFound
	}
	return order, nil
}

func (s *Store) PaymentOrders(limit int) []PaymentOrder {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 {
		limit = 100
	}
	out := make([]PaymentOrder, 0, len(s.paymentOrders))
	for _, order := range s.paymentOrders {
		out = append(out, order)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (s *Store) MarkPaymentOrderPaid(outTradeNo, providerTradeNo string) (PaymentOrder, Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	order, ok := s.paymentOrders[strings.TrimSpace(outTradeNo)]
	if !ok {
		return PaymentOrder{}, Subscription{}, ErrNotFound
	}
	p, ok := s.plans[order.PlanID]
	if !ok || p.Status != "active" {
		return PaymentOrder{}, Subscription{}, ErrNotFound
	}
	if order.Status == "paid" {
		sub, ok := s.subscriptions[order.UserID]
		if !ok {
			return order, Subscription{}, ErrNotFound
		}
		return order, sub, nil
	}
	now := time.Now()
	order.Status = "paid"
	order.ProviderTradeNo = strings.TrimSpace(providerTradeNo)
	order.PaidAt = &now
	s.paymentOrders[order.OutTradeNo] = order
	sub := planToSub(order.UserID, p, now.AddDate(0, 0, p.DurationDays))
	s.subscriptions[order.UserID] = sub
	return order, sub, nil
}

func (s *Store) Redeem(userID int64, code string) (Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rc, ok := s.redeemCodes[strings.TrimSpace(code)]
	if !ok {
		return Subscription{}, ErrNotFound
	}
	if rc.Status != "unused" {
		return Subscription{}, ErrForbidden
	}
	if rc.ExpiresAt != nil && time.Now().After(*rc.ExpiresAt) {
		return Subscription{}, ErrForbidden
	}
	p, ok := s.plans[rc.PlanID]
	if !ok || p.Status != "active" {
		return Subscription{}, ErrNotFound
	}
	now := time.Now()
	rc.Status = "redeemed"
	rc.RedeemedBy = userID
	rc.RedeemedAt = &now
	s.redeemCodes[rc.Code] = rc
	sub := planToSub(userID, p, now.AddDate(0, 0, p.DurationDays))
	s.subscriptions[userID] = sub
	return sub, nil
}

func (s *Store) Subscription(userID int64) (Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subscriptions[userID]
	if !ok {
		return Subscription{}, ErrNotFound
	}
	if time.Now().After(sub.ExpiresAt) {
		sub.Status = "expired"
	}
	return sub, nil
}

func (s *Store) CreateTunnel(userID int64, req Tunnel) (Tunnel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupExpiredSpeedTestTunnelsLocked(time.Now())
	sub, ok := s.subscriptions[userID]
	if !ok || sub.Status != "active" || time.Now().After(sub.ExpiresAt) {
		return Tunnel{}, ErrForbidden
	}
	if sub.TrafficLimitBytes > 0 && sub.TrafficUsedBytes >= sub.TrafficLimitBytes {
		return Tunnel{}, ErrForbidden
	}
	if err := s.checkTunnelLimitsLocked(userID, sub, strings.ToLower(req.Type)); err != nil {
		return Tunnel{}, err
	}
	typ := strings.ToLower(req.Type)
	if req.Name == "" || req.LocalHost == "" || req.LocalPort <= 0 {
		return Tunnel{}, fmt.Errorf("invalid tunnel request")
	}
	if err := validateTunnelBandwidth(sub, req.BandwidthKbps); err != nil {
		return Tunnel{}, err
	}
	nodeSettings := s.settings
	if req.NodeID > 0 {
		node, ok := s.nodes[req.NodeID]
		if !ok {
			return Tunnel{}, ErrNotFound
		}
		nodeSettings = settingsFromNode(s.settings, node)
	}
	t := Tunnel{ID: s.nextTunnelID, UserID: userID, NodeID: req.NodeID, Name: req.Name, Type: typ, LocalHost: req.LocalHost, LocalPort: req.LocalPort, UseHTTPS: req.UseHTTPS, BandwidthKbps: req.BandwidthKbps, CreatedAt: time.Now()}
	s.nextTunnelID++
	switch typ {
	case "tcp":
		if !sub.AllowTCP {
			return Tunnel{}, ErrForbidden
		}
		port, err := s.allocatePortLocked(req.NodeID, "tcp", nodeSettings.TCPPortStart, nodeSettings.TCPPortEnd)
		if err != nil {
			return Tunnel{}, err
		}
		t.RemotePort = port
		t.Status = "active"
		t.PublicURL = fmt.Sprintf("%s:%d", nodeSettings.ServerAddr, port)
	case "udp":
		if !sub.AllowUDP {
			return Tunnel{}, ErrForbidden
		}
		port, err := s.allocatePortLocked(req.NodeID, "udp", nodeSettings.UDPPortStart, nodeSettings.UDPPortEnd)
		if err != nil {
			return Tunnel{}, err
		}
		t.RemotePort = port
		t.Status = "active"
		t.PublicURL = fmt.Sprintf("%s:%d", nodeSettings.ServerAddr, port)
	case "http", "https":
		if !sub.AllowCustomDomain {
			return Tunnel{}, ErrForbidden
		}
		if typ == "http" && !sub.AllowHTTP {
			return Tunnel{}, ErrForbidden
		}
		if typ == "https" && (!sub.AllowHTTPS || !sub.AllowAutoCert) {
			return Tunnel{}, ErrForbidden
		}
		domain := strings.ToLower(strings.TrimSpace(req.Domain))
		if domain == "" {
			return Tunnel{}, fmt.Errorf("domain required")
		}
		if _, exists := s.domains[domain]; exists {
			return Tunnel{}, ErrConflict
		}
		s.domains[domain] = t.ID
		t.Domain = domain
		if typ == "https" {
			t.UseHTTPS = true
			t.Status = "pending_certificate"
			t.PublicURL = "https://" + domain
		} else {
			t.Status = "pending_domain_check"
			t.PublicURL = "http://" + domain
		}
	default:
		return Tunnel{}, fmt.Errorf("unsupported tunnel type")
	}
	t.EffectiveBandwidthKbps = effectiveBandwidth(sub.BandwidthKbps, t.BandwidthKbps)
	s.tunnels[t.ID] = t
	return t, nil
}

func (s *Store) CreateSpeedTestTunnel(userID int64, req SpeedTestTunnelRequest) (SpeedTestTunnel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupExpiredSpeedTestTunnelsLocked(time.Now())
	sub, ok := s.subscriptions[userID]
	if !ok || sub.Status != "active" || time.Now().After(sub.ExpiresAt) {
		return SpeedTestTunnel{}, ErrForbidden
	}
	if sub.TrafficLimitBytes > 0 && sub.TrafficUsedBytes >= sub.TrafficLimitBytes {
		return SpeedTestTunnel{}, ErrForbidden
	}
	typ := strings.ToLower(strings.TrimSpace(req.Type))
	if req.LocalHost == "" || req.LocalPort <= 0 {
		return SpeedTestTunnel{}, fmt.Errorf("invalid speed test request")
	}
	if req.BandwidthKbps > 0 {
		return SpeedTestTunnel{}, fmt.Errorf("speed test bandwidth limit is inherited from subscription")
	}
	nodeSettings := s.settings
	if req.NodeID > 0 {
		node, ok := s.nodes[req.NodeID]
		if !ok {
			return SpeedTestTunnel{}, ErrNotFound
		}
		nodeSettings = settingsFromNode(s.settings, node)
	}
	expires := time.Now().Add(15 * time.Minute)
	t := Tunnel{ID: s.nextTunnelID, UserID: userID, NodeID: req.NodeID, Name: fmt.Sprintf("__speedtest_%s_%d", typ, s.nextTunnelID), Type: typ, LocalHost: req.LocalHost, LocalPort: req.LocalPort, BandwidthKbps: 0, EffectiveBandwidthKbps: sub.BandwidthKbps, SpeedTest: true, ExpiresAt: &expires, CreatedAt: time.Now()}
	s.nextTunnelID++
	switch typ {
	case "tcp":
		if !sub.AllowTCP {
			return SpeedTestTunnel{}, ErrForbidden
		}
		port, err := s.allocatePortLocked(req.NodeID, "tcp", nodeSettings.TCPPortStart, nodeSettings.TCPPortEnd)
		if err != nil {
			return SpeedTestTunnel{}, err
		}
		t.RemotePort = port
		t.Status = "active"
		t.PublicURL = fmt.Sprintf("%s:%d", nodeSettings.ServerAddr, port)
	case "udp":
		if !sub.AllowUDP {
			return SpeedTestTunnel{}, ErrForbidden
		}
		port, err := s.allocatePortLocked(req.NodeID, "udp", nodeSettings.UDPPortStart, nodeSettings.UDPPortEnd)
		if err != nil {
			return SpeedTestTunnel{}, err
		}
		t.RemotePort = port
		t.Status = "active"
		t.PublicURL = fmt.Sprintf("%s:%d", nodeSettings.ServerAddr, port)
	case "http", "https":
		if typ == "http" && !sub.AllowHTTP {
			return SpeedTestTunnel{}, ErrForbidden
		}
		if typ == "https" && !sub.AllowHTTPS {
			return SpeedTestTunnel{}, ErrForbidden
		}
		domain := fmt.Sprintf("speed-%d.%s", t.ID, strings.TrimPrefix(nodeSettings.FRPEntryDomain, "*."))
		if _, exists := s.domains[domain]; exists {
			return SpeedTestTunnel{}, ErrConflict
		}
		s.domains[domain] = t.ID
		t.Domain = domain
		t.UseHTTPS = typ == "https"
		t.Status = "active"
		if typ == "https" {
			t.PublicURL = "https://" + domain
		} else {
			t.PublicURL = "http://" + domain
		}
	default:
		return SpeedTestTunnel{}, fmt.Errorf("unsupported tunnel type")
	}
	s.tunnels[t.ID] = t
	return speedTestTunnelFromTunnel(t), nil
}

func (s *Store) SpeedTestTunnel(userID int64, tunnelID int64) (SpeedTestTunnel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tunnels[tunnelID]
	if !ok || t.UserID != userID || !t.SpeedTest || t.Status == "deleted" {
		return SpeedTestTunnel{}, ErrNotFound
	}
	return speedTestTunnelFromTunnel(t), nil
}

func settingsFromNode(fallback Settings, node Node) Settings {
	out := fallback
	if strings.TrimSpace(node.FRPEntryDomain) != "" {
		out.FRPEntryDomain = node.FRPEntryDomain
	}
	if strings.TrimSpace(node.ServerAddr) != "" {
		out.ServerAddr = node.ServerAddr
	}
	if node.FRPServerPort > 0 {
		out.FRPServerPort = node.FRPServerPort
	}
	if node.TCPPortStart > 0 {
		out.TCPPortStart = node.TCPPortStart
	}
	if node.TCPPortEnd > 0 {
		out.TCPPortEnd = node.TCPPortEnd
	}
	if node.UDPPortStart > 0 {
		out.UDPPortStart = node.UDPPortStart
	}
	if node.UDPPortEnd > 0 {
		out.UDPPortEnd = node.UDPPortEnd
	}
	return out
}

func (s *Store) FinishSpeedTestTunnel(userID int64, tunnelID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tunnels[tunnelID]
	if !ok || t.UserID != userID || !t.SpeedTest {
		return ErrNotFound
	}
	t.Status = "deleted"
	s.tunnels[tunnelID] = t
	s.releaseTunnelLocked(t)
	return nil
}

func (s *Store) UpdateTunnelStatus(userID int64, tunnelID int64, status string) (Tunnel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tunnels[tunnelID]
	if !ok || t.UserID != userID || t.SpeedTest || t.Status == "deleted" {
		return Tunnel{}, ErrNotFound
	}
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		sub, ok := s.subscriptions[userID]
		if !ok || sub.Status != "active" || time.Now().After(sub.ExpiresAt) {
			return Tunnel{}, ErrForbidden
		}
		if sub.TrafficLimitBytes > 0 && sub.TrafficUsedBytes >= sub.TrafficLimitBytes {
			return Tunnel{}, ErrForbidden
		}
		t.Status = "active"
	case "disabled":
		t.Status = "disabled"
	default:
		return Tunnel{}, fmt.Errorf("unsupported tunnel status")
	}
	s.tunnels[t.ID] = t
	return t, nil
}

func (s *Store) DeleteTunnel(userID int64, tunnelID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tunnels[tunnelID]
	if !ok || t.UserID != userID || t.Status == "deleted" {
		return ErrNotFound
	}
	t.Status = "deleted"
	s.tunnels[t.ID] = t
	s.releaseTunnelLocked(t)
	return nil
}

func (s *Store) CleanupExpiredSpeedTestTunnels(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cleanupExpiredSpeedTestTunnelsLocked(now)
}

func (s *Store) cleanupExpiredSpeedTestTunnelsLocked(now time.Time) int {
	cleaned := 0
	for id, t := range s.tunnels {
		if !t.SpeedTest || t.Status == "deleted" || t.ExpiresAt == nil || now.Before(*t.ExpiresAt) {
			continue
		}
		t.Status = "deleted"
		s.tunnels[id] = t
		s.releaseTunnelLocked(t)
		cleaned++
	}
	return cleaned
}

func (s *Store) releaseTunnelLocked(t Tunnel) {
	if t.RemotePort > 0 {
		delete(s.usedPorts, portKey(t.NodeID, t.Type, t.RemotePort))
	}
	if t.Domain != "" {
		delete(s.domains, t.Domain)
	}
}

func (s *Store) checkTunnelLimitsLocked(userID int64, sub Subscription, typ string) error {
	total, tcp, udp, httpc, httpsc, domains := 0, 0, 0, 0, 0, 0
	for _, t := range s.tunnels {
		if t.UserID != userID || t.Status == "deleted" || t.Status == "disabled" {
			continue
		}
		total++
		switch t.Type {
		case "tcp":
			tcp++
		case "udp":
			udp++
		case "http":
			httpc++
			domains++
		case "https":
			httpsc++
			domains++
		}
	}
	if sub.MaxTunnels > 0 && total >= sub.MaxTunnels {
		return ErrForbidden
	}
	switch typ {
	case "tcp":
		if sub.MaxTCPTunnels > 0 && tcp >= sub.MaxTCPTunnels {
			return ErrForbidden
		}
	case "udp":
		if sub.MaxUDPTunnels > 0 && udp >= sub.MaxUDPTunnels {
			return ErrForbidden
		}
	case "http":
		if sub.MaxHTTPTunnels > 0 && httpc >= sub.MaxHTTPTunnels {
			return ErrForbidden
		}
		if sub.MaxDomains > 0 && domains >= sub.MaxDomains {
			return ErrForbidden
		}
	case "https":
		if sub.MaxHTTPSTunnels > 0 && httpsc >= sub.MaxHTTPSTunnels {
			return ErrForbidden
		}
		if sub.MaxDomains > 0 && domains >= sub.MaxDomains {
			return ErrForbidden
		}
	}
	return nil
}

func portKey(nodeID int64, protocol string, port int) string {
	return fmt.Sprintf("%d:%s:%d", nodeID, strings.ToLower(protocol), port)
}

func (s *Store) allocatePortLocked(nodeID int64, protocol string, start, end int) (int, error) {
	for p := start; p <= end; p++ {
		key := portKey(nodeID, protocol, p)
		if !s.usedPorts[key] {
			s.usedPorts[key] = true
			return p, nil
		}
	}
	return 0, fmt.Errorf("no free port")
}

func validateTunnelBandwidth(sub Subscription, tunnelLimit int) error {
	if tunnelLimit < 0 {
		return fmt.Errorf("bandwidth_limit_kbps must be >= 0")
	}
	if tunnelLimit > 0 && sub.BandwidthKbps > 0 && tunnelLimit > sub.BandwidthKbps {
		return ErrForbidden
	}
	if tunnelLimit > 0 && sub.BandwidthKbps == 0 {
		return ErrForbidden
	}
	return nil
}

func effectiveBandwidth(packageLimit int, tunnelLimit int) int {
	if tunnelLimit > 0 {
		return tunnelLimit
	}
	return packageLimit
}

func speedTestTunnelFromTunnel(t Tunnel) SpeedTestTunnel {
	expires := time.Now()
	if t.ExpiresAt != nil {
		expires = *t.ExpiresAt
	}
	return SpeedTestTunnel{ID: t.ID, NodeID: t.NodeID, Type: t.Type, LocalHost: t.LocalHost, LocalPort: t.LocalPort, RemotePort: t.RemotePort, Domain: t.Domain, PublicURL: t.PublicURL, EffectiveBandwidthKbps: t.EffectiveBandwidthKbps, ExpiresAt: expires}
}

func (s *Store) Tunnels(userID int64) []Tunnel {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Tunnel{}
	for _, t := range s.tunnels {
		if t.UserID == userID {
			out = append(out, t)
		}
	}
	return out
}
func (s *Store) AllTunnels() []Tunnel {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Tunnel{}
	for _, t := range s.tunnels {
		out = append(out, t)
	}
	return out
}

func (s *Store) Nodes() []Node {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Node{}
	for _, n := range s.nodes {
		out = append(out, n)
	}
	return out
}

func (s *Store) Node(id int64) (Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.nodes[id]
	if !ok {
		return Node{}, ErrNotFound
	}
	return n, nil
}

func (s *Store) CreateNode(node Node) (Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if strings.TrimSpace(node.Name) == "" {
		node.Name = fmt.Sprintf("node-%d", s.nextNodeID)
	}
	if node.FRPServerPort == 0 {
		node.FRPServerPort = 7000
	}
	if node.TCPPortStart == 0 {
		node.TCPPortStart = 20000
	}
	if node.TCPPortEnd == 0 {
		node.TCPPortEnd = 29999
	}
	if node.UDPPortStart == 0 {
		node.UDPPortStart = 30000
	}
	if node.UDPPortEnd == 0 {
		node.UDPPortEnd = 39999
	}
	if strings.TrimSpace(node.Status) == "" {
		node.Status = "pending"
	}
	node.ID = s.nextNodeID
	s.nextNodeID++
	if node.BindToken == "" {
		token, err := randomToken("node-bind")
		if err != nil {
			return Node{}, err
		}
		node.BindToken = token
	}
	if node.AgentToken == "" {
		token, err := randomToken("node-agent")
		if err != nil {
			return Node{}, err
		}
		node.AgentToken = token
	}
	node.CreatedAt = now
	node.UpdatedAt = now
	s.nodes[node.ID] = node
	s.nodesByBind[node.BindToken] = node.ID
	return node, nil
}

func (s *Store) DeleteNode(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.nodes[id]
	if !ok {
		return ErrNotFound
	}
	delete(s.nodesByBind, n.BindToken)
	delete(s.nodes, id)
	return nil
}

func (s *Store) BindNode(req NodeBindRequest) (Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.nodesByBind[strings.TrimSpace(req.BindToken)]
	if !ok {
		return Node{}, ErrUnauthorized
	}
	n := s.nodes[id]
	delete(s.nodesByBind, n.BindToken)
	newBindToken, err := randomToken("node-bind")
	if err != nil {
		return Node{}, err
	}
	n.BindToken = newBindToken
	s.nodesByBind[newBindToken] = id
	now := time.Now()
	if strings.TrimSpace(req.Name) != "" {
		n.Name = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.AgentURL) != "" {
		n.AgentURL = strings.TrimRight(strings.TrimSpace(req.AgentURL), "/")
	}
	if strings.TrimSpace(req.PublicURL) != "" {
		n.PublicURL = strings.TrimRight(strings.TrimSpace(req.PublicURL), "/")
	}
	if strings.TrimSpace(req.FRPEntryDomain) != "" {
		n.FRPEntryDomain = strings.TrimSpace(req.FRPEntryDomain)
	}
	if strings.TrimSpace(req.ServerAddr) != "" {
		n.ServerAddr = strings.TrimSpace(req.ServerAddr)
	}
	if req.FRPServerPort > 0 {
		n.FRPServerPort = req.FRPServerPort
	}
	if req.TCPPortStart > 0 {
		n.TCPPortStart = req.TCPPortStart
	}
	if req.TCPPortEnd > 0 {
		n.TCPPortEnd = req.TCPPortEnd
	}
	if req.UDPPortStart > 0 {
		n.UDPPortStart = req.UDPPortStart
	}
	if req.UDPPortEnd > 0 {
		n.UDPPortEnd = req.UDPPortEnd
	}
	n.Status = "online"
	n.LastError = ""
	n.LastSeenAt = &now
	n.UpdatedAt = now
	s.nodes[id] = n
	return n, nil
}

func (s *Store) UpdateNodeStatus(id int64, status string, lastError string) (Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.nodes[id]
	if !ok {
		return Node{}, ErrNotFound
	}
	n.Status = status
	n.LastError = lastError
	n.UpdatedAt = time.Now()
	if status == "online" {
		n.LastSeenAt = &n.UpdatedAt
	}
	s.nodes[id] = n
	return n, nil
}

func (s *Store) Settings() Settings { s.mu.Lock(); defer s.mu.Unlock(); return s.settings }
func (s *Store) UpdateSettings(in Settings) Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = in
	return s.settings
}

func (s *Store) CreatePlan(plan Plan) (Plan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	plan.ID = s.nextPlanID
	s.nextPlanID++
	if plan.Status == "" {
		plan.Status = "active"
	}
	s.plans[plan.ID] = plan
	return plan, nil
}

func (s *Store) UpdatePlan(id int64, plan Plan) (Plan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.plans[id]; !ok {
		return Plan{}, ErrNotFound
	}
	if plan.Status == "" {
		plan.Status = "active"
	}
	plan.ID = id
	s.plans[id] = plan
	return plan, nil
}

func (s *Store) Users() []User {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []User{}
	for _, u := range s.users {
		out = append(out, u)
	}
	return out
}

func (s *Store) ActiveSubscriptionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	count := 0
	for _, sub := range s.subscriptions {
		if sub.Status == "active" && now.Before(sub.ExpiresAt) {
			count++
		}
	}
	return count
}

func (s *Store) UpdateUser(id int64, status string, planID int64) (User, Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		return User{}, Subscription{}, ErrNotFound
	}
	if strings.TrimSpace(status) != "" {
		u.Status = strings.TrimSpace(status)
		s.users[id] = u
	}
	var sub Subscription
	if planID > 0 {
		p, ok := s.plans[planID]
		if !ok || p.Status != "active" {
			return User{}, Subscription{}, ErrNotFound
		}
		now := time.Now()
		sub = planToSub(id, p, now.AddDate(0, 0, p.DurationDays))
		s.subscriptions[id] = sub
	}
	return u, sub, nil
}

func (s *Store) RedeemCodes() []RedeemCode {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []RedeemCode{}
	for _, rc := range s.redeemCodes {
		out = append(out, rc)
	}
	return out
}

func (s *Store) CreateRedeemCodes(planID int64, count int, prefix string) ([]RedeemCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.plans[planID]; !ok {
		return nil, ErrNotFound
	}
	if count <= 0 || count > 500 {
		return nil, fmt.Errorf("count must be 1-500")
	}
	if prefix == "" {
		prefix = "CODE"
	}
	out := make([]RedeemCode, 0, count)
	for i := 0; i < count; i++ {
		code := fmt.Sprintf("%s-%d-%d", strings.ToUpper(prefix), time.Now().UnixNano(), i+1)
		rc := RedeemCode{Code: code, PlanID: planID, Status: "unused"}
		s.redeemCodes[code] = rc
		out = append(out, rc)
	}
	return out, nil
}

func (s *Store) ReportTraffic(userID int64, reports []TrafficReport) (TrafficSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subscriptions[userID]
	if !ok {
		return TrafficSummary{}, ErrNotFound
	}
	var added int64
	for _, r := range reports {
		if r.BytesIn < 0 || r.BytesOut < 0 {
			continue
		}
		if r.TunnelID != 0 {
			if t, ok := s.tunnels[r.TunnelID]; !ok || t.UserID != userID {
				continue
			}
		}
		added += r.BytesIn + r.BytesOut
	}
	sub.TrafficUsedBytes += added
	s.todayTraffic += added
	s.subscriptions[userID] = sub
	return trafficSummaryFromSub(userID, sub, s.todayTraffic), nil
}

func (s *Store) TrafficSummary(userID int64) (TrafficSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subscriptions[userID]
	if !ok {
		return TrafficSummary{}, ErrNotFound
	}
	return trafficSummaryFromSub(userID, sub, s.todayTraffic), nil
}

func (s *Store) TotalTrafficToday() int64 { s.mu.Lock(); defer s.mu.Unlock(); return s.todayTraffic }

func trafficSummaryFromSub(userID int64, sub Subscription, today int64) TrafficSummary {
	left := sub.TrafficLimitBytes - sub.TrafficUsedBytes
	if left < 0 {
		left = 0
	}
	return TrafficSummary{UserID: userID, TrafficLimitBytes: sub.TrafficLimitBytes, TrafficUsedBytes: sub.TrafficUsedBytes, TrafficLeftBytes: left, TodayBytes: today}
}

func (s *Store) SaveCertificate(record CertificateRecord) (CertificateRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	domain := sanitizeDomain(record.Domain)
	if domain == "" {
		return CertificateRecord{}, fmt.Errorf("domain required")
	}
	now := time.Now()
	existing := s.certificates[domain]
	if existing.ID == 0 {
		existing.ID = s.nextCertID
		s.nextCertID++
		existing.CreatedAt = now
	}
	record.ID = existing.ID
	if record.UserID == 0 {
		record.UserID = existing.UserID
	}
	record.Domain = domain
	if record.CreatedAt.IsZero() {
		record.CreatedAt = existing.CreatedAt
	}
	record.UpdatedAt = now
	s.certificates[domain] = record
	return record, nil
}

func (s *Store) Certificates() []CertificateRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []CertificateRecord{}
	for _, cert := range s.certificates {
		out = append(out, cert)
	}
	return out
}

func (s *Store) RecordAdminOperation(logEntry AdminOperationLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	logEntry.ID = s.nextOperationID
	s.nextOperationID++
	if logEntry.CreatedAt.IsZero() {
		logEntry.CreatedAt = time.Now()
	}
	s.operationLogs = append([]AdminOperationLog{logEntry}, s.operationLogs...)
	if len(s.operationLogs) > 1000 {
		s.operationLogs = s.operationLogs[:1000]
	}
	return nil
}

func (s *Store) AdminOperationLogs(limit int) []AdminOperationLog {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if limit > len(s.operationLogs) {
		limit = len(s.operationLogs)
	}
	out := make([]AdminOperationLog, limit)
	copy(out, s.operationLogs[:limit])
	return out
}
