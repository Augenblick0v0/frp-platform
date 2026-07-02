package platform

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type SQLStore struct{ db *sql.DB }

func NewSQLStore(dsn string) (*SQLStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	s := &SQLStore{db: db}
	if err := s.Migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.SeedDefaults(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLStore) Close() error   { return s.db.Close() }
func (s *SQLStore) Migrate() error { _, err := s.db.Exec(postgresSchema); return err }
func (s *SQLStore) SeedDefaults() error {
	adminEmail := strings.ToLower(getenv("ADMIN_EMAIL", "admin@example.com"))
	adminPassword := getenv("ADMIN_PASSWORD", "admin123456")
	if _, err := s.db.Exec(`INSERT INTO admin_users (id,email,password_hash,status) VALUES (1,$1,$2,'active') ON CONFLICT (id) DO NOTHING`, adminEmail, adminPassword); err != nil {
		return err
	}
	_, err := s.db.Exec(`
SELECT setval(pg_get_serial_sequence('admin_users','id'), GREATEST((SELECT MAX(id) FROM admin_users), 1));
INSERT INTO plans (id,name,description,duration_days,traffic_limit_bytes,bandwidth_limit_kbps,max_tunnels,max_tcp_tunnels,max_udp_tunnels,max_http_tunnels,max_https_tunnels,allow_tcp,allow_udp,allow_http,allow_https,allow_custom_domain,max_domains,allow_auto_cert,status)
VALUES (1,'高级套餐','支持 TCP/UDP/HTTP/HTTPS、自定义域名和自动证书',30,107374182400,10240,20,10,10,10,10,true,true,true,true,true,10,true,'active')
ON CONFLICT (id) DO NOTHING;
SELECT setval(pg_get_serial_sequence('plans','id'), GREATEST((SELECT MAX(id) FROM plans), 1));
INSERT INTO redeem_codes (code, plan_id, status) VALUES ('DEMO-PLAN-2026', 1, 'unused') ON CONFLICT (code) DO NOTHING;
INSERT INTO system_settings (key,value) VALUES
('platform_domain','example.com'),('frp_entry_domain','frp.example.com'),('server_addr','frp.example.com'),('frp_server_port','7000'),('tcp_port_start','20000'),('tcp_port_end','29999'),('udp_port_start','30000'),('udp_port_end','39999'),('purchase_url','https://example.com/buy')
ON CONFLICT (key) DO NOTHING;`)
	return err
}

func (s *SQLStore) AdminLogin(email, password string) (string, AdminUser, error) {
	var admin AdminUser
	err := s.db.QueryRow(`SELECT id,email,password_hash,status,created_at FROM admin_users WHERE email=$1`, strings.ToLower(strings.TrimSpace(email))).Scan(&admin.ID, &admin.Email, &admin.Password, &admin.Status, &admin.CreatedAt)
	if err != nil {
		return "", AdminUser{}, ErrUnauthorized
	}
	if admin.Password != password || admin.Status != "active" {
		return "", AdminUser{}, ErrUnauthorized
	}
	token := fmt.Sprintf("admin-token-%d-%d", admin.ID, time.Now().UnixNano())
	_, err = s.db.Exec(`INSERT INTO admin_sessions (token,admin_user_id,expires_at) VALUES ($1,$2,$3)`, token, admin.ID, time.Now().Add(24*time.Hour))
	if err != nil {
		return "", AdminUser{}, err
	}
	return token, admin, nil
}

func (s *SQLStore) AdminByToken(token string) (AdminUser, error) {
	var admin AdminUser
	err := s.db.QueryRow(`SELECT a.id,a.email,a.password_hash,a.status,a.created_at FROM admin_sessions s JOIN admin_users a ON a.id=s.admin_user_id WHERE s.token=$1 AND s.expires_at>now()`, token).Scan(&admin.ID, &admin.Email, &admin.Password, &admin.Status, &admin.CreatedAt)
	if err != nil || admin.Status != "active" {
		return AdminUser{}, ErrUnauthorized
	}
	return admin, nil
}

func (s *SQLStore) SendEmailCode(email, purpose string) string {
	code := "123456"
	_, _ = s.db.Exec(`INSERT INTO email_verification_codes (email, code, purpose, expires_at) VALUES ($1,$2,$3,$4)`, strings.ToLower(email), code, purpose, time.Now().Add(10*time.Minute))
	return code
}

func (s *SQLStore) Register(email, code, password string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var found string
	err := s.db.QueryRow(`SELECT code FROM email_verification_codes WHERE email=$1 AND purpose='register' AND used_at IS NULL AND expires_at>now() ORDER BY created_at DESC LIMIT 1`, email).Scan(&found)
	if err != nil || found != code {
		return User{}, fmt.Errorf("invalid verification code")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback()
	var u User
	err = tx.QueryRow(`INSERT INTO users (email,password_hash,status,email_verified_at) VALUES ($1,$2,'active',now()) RETURNING id,email,password_hash,status,created_at`, email, password).Scan(&u.ID, &u.Email, &u.Password, &u.Status, &u.CreatedAt)
	if err != nil {
		if isUnique(err) {
			return User{}, ErrConflict
		}
		return User{}, err
	}
	_, _ = tx.Exec(`UPDATE email_verification_codes SET used_at=now() WHERE email=$1 AND purpose='register' AND code=$2`, email, code)
	if err := tx.Commit(); err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *SQLStore) Login(email, password string) (string, User, error) {
	var u User
	err := s.db.QueryRow(`SELECT id,email,password_hash,status,created_at FROM users WHERE email=$1`, strings.ToLower(strings.TrimSpace(email))).Scan(&u.ID, &u.Email, &u.Password, &u.Status, &u.CreatedAt)
	if err != nil {
		return "", User{}, ErrUnauthorized
	}
	if u.Password != password || u.Status != "active" {
		return "", User{}, ErrUnauthorized
	}
	token := fmt.Sprintf("token-%d-%d", u.ID, time.Now().UnixNano())
	_, err = s.db.Exec(`INSERT INTO sessions (token,user_id,expires_at) VALUES ($1,$2,$3)`, token, u.ID, time.Now().Add(24*time.Hour))
	if err != nil {
		return "", User{}, err
	}
	return token, u, nil
}

func (s *SQLStore) UserByToken(token string) (User, error) {
	var u User
	err := s.db.QueryRow(`SELECT u.id,u.email,u.password_hash,u.status,u.created_at FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token=$1 AND s.expires_at>now()`, token).Scan(&u.ID, &u.Email, &u.Password, &u.Status, &u.CreatedAt)
	if err != nil {
		return User{}, ErrUnauthorized
	}
	return u, nil
}

func (s *SQLStore) Plans() []Plan {
	rows, err := s.db.Query(`SELECT id,name,coalesce(description,''),duration_days,traffic_limit_bytes,bandwidth_limit_kbps,max_tunnels,max_tcp_tunnels,max_udp_tunnels,max_http_tunnels,max_https_tunnels,allow_tcp,allow_udp,allow_http,allow_https,allow_custom_domain,max_domains,allow_auto_cert,status FROM plans ORDER BY sort_order,id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Plan
	for rows.Next() {
		var p Plan
		_ = rows.Scan(&p.ID, &p.Name, &p.Description, &p.DurationDays, &p.TrafficLimitBytes, &p.BandwidthKbps, &p.MaxTunnels, &p.MaxTCPTunnels, &p.MaxUDPTunnels, &p.MaxHTTPTunnels, &p.MaxHTTPSTunnels, &p.AllowTCP, &p.AllowUDP, &p.AllowHTTP, &p.AllowHTTPS, &p.AllowCustomDomain, &p.MaxDomains, &p.AllowAutoCert, &p.Status)
		out = append(out, p)
	}
	return out
}

func (s *SQLStore) Redeem(userID int64, code string) (Subscription, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Subscription{}, err
	}
	defer tx.Rollback()
	var planID int64
	var status string
	err = tx.QueryRow(`SELECT plan_id,status FROM redeem_codes WHERE code=$1 FOR UPDATE`, strings.TrimSpace(code)).Scan(&planID, &status)
	if err != nil {
		return Subscription{}, ErrNotFound
	}
	if status != "unused" {
		return Subscription{}, ErrForbidden
	}
	p, err := scanPlan(tx.QueryRow(`SELECT id,name,coalesce(description,''),duration_days,traffic_limit_bytes,bandwidth_limit_kbps,max_tunnels,max_tcp_tunnels,max_udp_tunnels,max_http_tunnels,max_https_tunnels,allow_tcp,allow_udp,allow_http,allow_https,allow_custom_domain,max_domains,allow_auto_cert,status FROM plans WHERE id=$1`, planID))
	if err != nil || p.Status != "active" {
		return Subscription{}, ErrNotFound
	}
	now := time.Now()
	expires := now.AddDate(0, 0, p.DurationDays)
	_, err = tx.Exec(`UPDATE redeem_codes SET status='redeemed', redeemed_by_user_id=$1, redeemed_at=$2 WHERE code=$3`, userID, now, code)
	if err != nil {
		return Subscription{}, err
	}
	_, _ = tx.Exec(`UPDATE subscriptions SET status='replaced' WHERE user_id=$1 AND status='active'`, userID)
	_, err = tx.Exec(`INSERT INTO subscriptions (user_id,plan_id,starts_at,expires_at,traffic_limit_bytes,bandwidth_limit_kbps,status) VALUES ($1,$2,$3,$4,$5,$6,'active')`, userID, p.ID, now, expires, p.TrafficLimitBytes, p.BandwidthKbps)
	if err != nil {
		return Subscription{}, err
	}
	if err := tx.Commit(); err != nil {
		return Subscription{}, err
	}
	return planToSub(userID, p, expires), nil
}

func (s *SQLStore) Subscription(userID int64) (Subscription, error) {
	row := s.db.QueryRow(`SELECT p.id,p.name,coalesce(p.description,''),p.duration_days,sub.traffic_limit_bytes,sub.bandwidth_limit_kbps,p.max_tunnels,p.max_tcp_tunnels,p.max_udp_tunnels,p.max_http_tunnels,p.max_https_tunnels,p.allow_tcp,p.allow_udp,p.allow_http,p.allow_https,p.allow_custom_domain,p.max_domains,p.allow_auto_cert,sub.status,sub.expires_at,sub.traffic_used_bytes FROM subscriptions sub JOIN plans p ON p.id=sub.plan_id WHERE sub.user_id=$1 AND sub.status='active' ORDER BY sub.expires_at DESC LIMIT 1`, userID)
	var p Plan
	var expires time.Time
	var used int64
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.DurationDays, &p.TrafficLimitBytes, &p.BandwidthKbps, &p.MaxTunnels, &p.MaxTCPTunnels, &p.MaxUDPTunnels, &p.MaxHTTPTunnels, &p.MaxHTTPSTunnels, &p.AllowTCP, &p.AllowUDP, &p.AllowHTTP, &p.AllowHTTPS, &p.AllowCustomDomain, &p.MaxDomains, &p.AllowAutoCert, &p.Status, &expires, &used)
	if err != nil {
		return Subscription{}, ErrNotFound
	}
	sub := planToSub(userID, p, expires)
	sub.TrafficUsedBytes = used
	if time.Now().After(expires) {
		sub.Status = "expired"
	}
	return sub, nil
}

func (s *SQLStore) CreateTunnel(userID int64, req Tunnel) (Tunnel, error) {
	sub, err := s.Subscription(userID)
	if err != nil || sub.Status != "active" {
		return Tunnel{}, ErrForbidden
	}
	if sub.TrafficLimitBytes > 0 && sub.TrafficUsedBytes >= sub.TrafficLimitBytes {
		return Tunnel{}, ErrForbidden
	}
	if err := s.checkSQLTunnelLimits(userID, sub, strings.ToLower(req.Type)); err != nil {
		return Tunnel{}, err
	}
	st := s.Settings()
	typ := strings.ToLower(req.Type)
	if req.Name == "" || req.LocalHost == "" || req.LocalPort <= 0 {
		return Tunnel{}, fmt.Errorf("invalid tunnel request")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Tunnel{}, err
	}
	defer tx.Rollback()
	t := Tunnel{UserID: userID, Name: req.Name, Type: typ, LocalHost: req.LocalHost, LocalPort: req.LocalPort, UseHTTPS: req.UseHTTPS, CreatedAt: time.Now()}
	switch typ {
	case "tcp":
		if !sub.AllowTCP {
			return Tunnel{}, ErrForbidden
		}
		port, err := s.allocateSQLPort(tx, "tcp", st.TCPPortStart, st.TCPPortEnd)
		if err != nil {
			return Tunnel{}, err
		}
		t.RemotePort = port
		t.Status = "active"
		t.PublicURL = fmt.Sprintf("%s:%d", st.ServerAddr, port)
	case "udp":
		if !sub.AllowUDP {
			return Tunnel{}, ErrForbidden
		}
		port, err := s.allocateSQLPort(tx, "udp", st.UDPPortStart, st.UDPPortEnd)
		if err != nil {
			return Tunnel{}, err
		}
		t.RemotePort = port
		t.Status = "active"
		t.PublicURL = fmt.Sprintf("%s:%d", st.ServerAddr, port)
	case "http", "https":
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
	err = tx.QueryRow(`INSERT INTO tunnels (user_id,name,type,local_host,local_port,remote_port,domain,use_https,status,public_url) VALUES ($1,$2,$3,$4,$5,NULLIF($6,0),NULLIF($7,''),$8,$9,$10) RETURNING id,created_at`, userID, t.Name, t.Type, t.LocalHost, t.LocalPort, t.RemotePort, t.Domain, t.UseHTTPS, t.Status, t.PublicURL).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		if isUnique(err) {
			return Tunnel{}, ErrConflict
		}
		return Tunnel{}, err
	}
	if t.RemotePort != 0 {
		_, err = tx.Exec(`UPDATE port_allocations SET tunnel_id=$1,user_id=$2 WHERE protocol=$3 AND port=$4`, t.ID, userID, typ, t.RemotePort)
		if err != nil {
			return Tunnel{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Tunnel{}, err
	}
	return t, nil
}

func (s *SQLStore) allocateSQLPort(tx *sql.Tx, protocol string, start, end int) (int, error) {
	for p := start; p <= end; p++ {
		_, err := tx.Exec(`INSERT INTO port_allocations (protocol,port,status) VALUES ($1,$2,'allocated')`, protocol, p)
		if err == nil {
			return p, nil
		}
		if !isUnique(err) {
			return 0, err
		}
	}
	return 0, fmt.Errorf("no free port")
}
func (s *SQLStore) Tunnels(userID int64) []Tunnel {
	return s.queryTunnels(`WHERE user_id=$1 ORDER BY id DESC`, userID)
}
func (s *SQLStore) AllTunnels() []Tunnel { return s.queryTunnels(`ORDER BY id DESC`) }
func (s *SQLStore) queryTunnels(suffix string, args ...any) []Tunnel {
	rows, err := s.db.Query(`SELECT id,user_id,name,type,local_host,local_port,coalesce(remote_port,0),coalesce(domain,''),use_https,status,coalesce(public_url,''),coalesce(error_message,''),created_at FROM tunnels `+suffix, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Tunnel
	for rows.Next() {
		var t Tunnel
		_ = rows.Scan(&t.ID, &t.UserID, &t.Name, &t.Type, &t.LocalHost, &t.LocalPort, &t.RemotePort, &t.Domain, &t.UseHTTPS, &t.Status, &t.PublicURL, &t.ErrorMessage, &t.CreatedAt)
		out = append(out, t)
	}
	return out
}
func (s *SQLStore) Settings() Settings {
	get := func(k, d string) string {
		var v string
		if err := s.db.QueryRow(`SELECT value FROM system_settings WHERE key=$1`, k).Scan(&v); err != nil {
			return d
		}
		return v
	}
	atoi := func(v string, d int) int {
		i, err := strconv.Atoi(v)
		if err != nil {
			return d
		}
		return i
	}
	return Settings{PlatformDomain: get("platform_domain", "example.com"), FRPEntryDomain: get("frp_entry_domain", "frp.example.com"), ServerAddr: get("server_addr", "frp.example.com"), FRPServerPort: atoi(get("frp_server_port", "7000"), 7000), TCPPortStart: atoi(get("tcp_port_start", "20000"), 20000), TCPPortEnd: atoi(get("tcp_port_end", "29999"), 29999), UDPPortStart: atoi(get("udp_port_start", "30000"), 30000), UDPPortEnd: atoi(get("udp_port_end", "39999"), 39999), PurchaseURL: get("purchase_url", "https://example.com/buy")}
}
func (s *SQLStore) UpdateSettings(in Settings) Settings {
	vals := map[string]string{"platform_domain": in.PlatformDomain, "frp_entry_domain": in.FRPEntryDomain, "server_addr": in.ServerAddr, "frp_server_port": strconv.Itoa(in.FRPServerPort), "tcp_port_start": strconv.Itoa(in.TCPPortStart), "tcp_port_end": strconv.Itoa(in.TCPPortEnd), "udp_port_start": strconv.Itoa(in.UDPPortStart), "udp_port_end": strconv.Itoa(in.UDPPortEnd), "purchase_url": in.PurchaseURL}
	for k, v := range vals {
		_, _ = s.db.Exec(`INSERT INTO system_settings (key,value) VALUES ($1,$2) ON CONFLICT (key) DO UPDATE SET value=excluded.value, updated_at=now()`, k, v)
	}
	return s.Settings()
}

func scanPlan(row *sql.Row) (Plan, error) {
	var p Plan
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.DurationDays, &p.TrafficLimitBytes, &p.BandwidthKbps, &p.MaxTunnels, &p.MaxTCPTunnels, &p.MaxUDPTunnels, &p.MaxHTTPTunnels, &p.MaxHTTPSTunnels, &p.AllowTCP, &p.AllowUDP, &p.AllowHTTP, &p.AllowHTTPS, &p.AllowCustomDomain, &p.MaxDomains, &p.AllowAutoCert, &p.Status)
	return p, err
}
func planToSub(userID int64, p Plan, expires time.Time) Subscription {
	return Subscription{UserID: userID, PlanID: p.ID, PlanName: p.Name, ExpiresAt: expires, TrafficLimitBytes: p.TrafficLimitBytes, BandwidthKbps: p.BandwidthKbps, AllowTCP: p.AllowTCP, AllowUDP: p.AllowUDP, AllowHTTP: p.AllowHTTP, AllowHTTPS: p.AllowHTTPS, AllowCustomDomain: p.AllowCustomDomain, AllowAutoCert: p.AllowAutoCert, MaxTunnels: p.MaxTunnels, MaxTCPTunnels: p.MaxTCPTunnels, MaxUDPTunnels: p.MaxUDPTunnels, MaxHTTPTunnels: p.MaxHTTPTunnels, MaxHTTPSTunnels: p.MaxHTTPSTunnels, MaxDomains: p.MaxDomains, Status: "active"}
}
func isUnique(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint"))
}

var _ = errors.Is

func (s *SQLStore) CreatePlan(plan Plan) (Plan, error) {
	if plan.Status == "" {
		plan.Status = "active"
	}
	err := s.db.QueryRow(`INSERT INTO plans (name,description,duration_days,traffic_limit_bytes,bandwidth_limit_kbps,max_tunnels,max_tcp_tunnels,max_udp_tunnels,max_http_tunnels,max_https_tunnels,allow_tcp,allow_udp,allow_http,allow_https,allow_custom_domain,max_domains,allow_auto_cert,status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
RETURNING id`, plan.Name, plan.Description, plan.DurationDays, plan.TrafficLimitBytes, plan.BandwidthKbps, plan.MaxTunnels, plan.MaxTCPTunnels, plan.MaxUDPTunnels, plan.MaxHTTPTunnels, plan.MaxHTTPSTunnels, plan.AllowTCP, plan.AllowUDP, plan.AllowHTTP, plan.AllowHTTPS, plan.AllowCustomDomain, plan.MaxDomains, plan.AllowAutoCert, plan.Status).Scan(&plan.ID)
	return plan, err
}

func (s *SQLStore) Users() []User {
	rows, err := s.db.Query(`SELECT id,email,password_hash,status,created_at FROM users ORDER BY id DESC LIMIT 500`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		_ = rows.Scan(&u.ID, &u.Email, &u.Password, &u.Status, &u.CreatedAt)
		out = append(out, u)
	}
	return out
}

func (s *SQLStore) RedeemCodes() []RedeemCode {
	rows, err := s.db.Query(`SELECT code,plan_id,status,expires_at,coalesce(redeemed_by_user_id,0),redeemed_at FROM redeem_codes ORDER BY created_at DESC LIMIT 500`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []RedeemCode
	for rows.Next() {
		var rc RedeemCode
		var expires sql.NullTime
		var redeemed sql.NullTime
		_ = rows.Scan(&rc.Code, &rc.PlanID, &rc.Status, &expires, &rc.RedeemedBy, &redeemed)
		if expires.Valid {
			rc.ExpiresAt = &expires.Time
		}
		if redeemed.Valid {
			rc.RedeemedAt = &redeemed.Time
		}
		out = append(out, rc)
	}
	return out
}

func (s *SQLStore) CreateRedeemCodes(planID int64, count int, prefix string) ([]RedeemCode, error) {
	if count <= 0 || count > 500 {
		return nil, fmt.Errorf("count must be 1-500")
	}
	if prefix == "" {
		prefix = "CODE"
	}
	var exists bool
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM plans WHERE id=$1)`, planID).Scan(&exists); err != nil || !exists {
		return nil, ErrNotFound
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	out := make([]RedeemCode, 0, count)
	for i := 0; i < count; i++ {
		code := fmt.Sprintf("%s-%d-%d", strings.ToUpper(prefix), time.Now().UnixNano(), i+1)
		_, err := tx.Exec(`INSERT INTO redeem_codes (code,plan_id,status) VALUES ($1,$2,'unused')`, code, planID)
		if err != nil {
			return nil, err
		}
		out = append(out, RedeemCode{Code: code, PlanID: planID, Status: "unused"})
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SQLStore) checkSQLTunnelLimits(userID int64, sub Subscription, typ string) error {
	rows, err := s.db.Query(`SELECT type, count(*) FROM tunnels WHERE user_id=$1 AND status NOT IN ('deleted','disabled') GROUP BY type`, userID)
	if err != nil {
		return err
	}
	defer rows.Close()
	total, domains := 0, 0
	counts := map[string]int{}
	for rows.Next() {
		var t string
		var c int
		_ = rows.Scan(&t, &c)
		counts[t] = c
		total += c
		if t == "http" || t == "https" {
			domains += c
		}
	}
	if sub.MaxTunnels > 0 && total >= sub.MaxTunnels {
		return ErrForbidden
	}
	switch typ {
	case "tcp":
		if sub.MaxTCPTunnels > 0 && counts["tcp"] >= sub.MaxTCPTunnels {
			return ErrForbidden
		}
	case "udp":
		if sub.MaxUDPTunnels > 0 && counts["udp"] >= sub.MaxUDPTunnels {
			return ErrForbidden
		}
	case "http":
		if sub.MaxHTTPTunnels > 0 && counts["http"] >= sub.MaxHTTPTunnels {
			return ErrForbidden
		}
		if sub.MaxDomains > 0 && domains >= sub.MaxDomains {
			return ErrForbidden
		}
	case "https":
		if sub.MaxHTTPSTunnels > 0 && counts["https"] >= sub.MaxHTTPSTunnels {
			return ErrForbidden
		}
		if sub.MaxDomains > 0 && domains >= sub.MaxDomains {
			return ErrForbidden
		}
	}
	return nil
}

func (s *SQLStore) ReportTraffic(userID int64, reports []TrafficReport) (TrafficSummary, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return TrafficSummary{}, err
	}
	defer tx.Rollback()
	var added int64
	for _, r := range reports {
		if r.BytesIn < 0 || r.BytesOut < 0 {
			continue
		}
		if r.TunnelID != 0 {
			var owner int64
			if err := tx.QueryRow(`SELECT user_id FROM tunnels WHERE id=$1`, r.TunnelID).Scan(&owner); err != nil || owner != userID {
				continue
			}
		}
		_, err := tx.Exec(`INSERT INTO traffic_logs (user_id,tunnel_id,bytes_in,bytes_out) VALUES ($1,NULLIF($2,0),$3,$4)`, userID, r.TunnelID, r.BytesIn, r.BytesOut)
		if err != nil {
			return TrafficSummary{}, err
		}
		added += r.BytesIn + r.BytesOut
	}
	_, err = tx.Exec(`UPDATE subscriptions SET traffic_used_bytes=traffic_used_bytes+$1, updated_at=now() WHERE user_id=$2 AND status='active'`, added, userID)
	if err != nil {
		return TrafficSummary{}, err
	}
	if err := tx.Commit(); err != nil {
		return TrafficSummary{}, err
	}
	return s.TrafficSummary(userID)
}

func (s *SQLStore) TrafficSummary(userID int64) (TrafficSummary, error) {
	sub, err := s.Subscription(userID)
	if err != nil {
		return TrafficSummary{}, err
	}
	left := sub.TrafficLimitBytes - sub.TrafficUsedBytes
	if left < 0 {
		left = 0
	}
	var today int64
	_ = s.db.QueryRow(`SELECT coalesce(sum(bytes_in+bytes_out),0) FROM traffic_logs WHERE user_id=$1 AND recorded_at >= date_trunc('day', now())`, userID).Scan(&today)
	return TrafficSummary{UserID: userID, TrafficLimitBytes: sub.TrafficLimitBytes, TrafficUsedBytes: sub.TrafficUsedBytes, TrafficLeftBytes: left, TodayBytes: today}, nil
}

func (s *SQLStore) TotalTrafficToday() int64 {
	var today int64
	_ = s.db.QueryRow(`SELECT coalesce(sum(bytes_in+bytes_out),0) FROM traffic_logs WHERE recorded_at >= date_trunc('day', now())`).Scan(&today)
	return today
}
