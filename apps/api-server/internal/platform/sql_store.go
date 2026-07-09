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
	if _, err := s.db.Exec(`INSERT INTO admin_users (id,email,password_hash,status) VALUES (1,$1,$2,'active') ON CONFLICT (id) DO NOTHING`, adminEmail, mustHashPassword(adminPassword)); err != nil {
		return err
	}
	_, err := s.db.Exec(`
SELECT setval(pg_get_serial_sequence('admin_users','id'), GREATEST((SELECT MAX(id) FROM admin_users), 1));
INSERT INTO plans (id,name,description,price_cents,duration_days,traffic_limit_bytes,bandwidth_limit_kbps,max_tunnels,max_tcp_tunnels,max_udp_tunnels,max_http_tunnels,max_https_tunnels,allow_tcp,allow_udp,allow_http,allow_https,allow_custom_domain,max_domains,allow_auto_cert,status)
VALUES (1,'高级套餐','支持 TCP/UDP/HTTP/HTTPS、自定义域名和自动证书',990,30,107374182400,10240,20,10,10,10,10,true,true,true,true,true,10,true,'active')
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
	if !VerifyPassword(admin.Password, password) || admin.Status != "active" {
		return "", AdminUser{}, ErrUnauthorized
	}
	token, err := randomToken("admin")
	if err != nil {
		return "", AdminUser{}, err
	}
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
	code, err := randomDigits(6)
	if err != nil {
		code = fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	_, _ = s.db.Exec(`INSERT INTO email_verification_codes (email, code, purpose, expires_at) VALUES ($1,$2,$3,$4)`, strings.ToLower(strings.TrimSpace(email)), code, strings.TrimSpace(purpose), time.Now().Add(10*time.Minute))
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
	err = tx.QueryRow(`INSERT INTO users (email,password_hash,status,email_verified_at) VALUES ($1,$2,'active',now()) RETURNING id,email,password_hash,status,created_at`, email, mustHashPassword(password)).Scan(&u.ID, &u.Email, &u.Password, &u.Status, &u.CreatedAt)
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
	if !VerifyPassword(u.Password, password) || u.Status != "active" {
		return "", User{}, ErrUnauthorized
	}
	token, err := randomToken("token")
	if err != nil {
		return "", User{}, err
	}
	_, err = s.db.Exec(`INSERT INTO sessions (token,user_id,expires_at) VALUES ($1,$2,$3)`, token, u.ID, time.Now().Add(24*time.Hour))
	if err != nil {
		return "", User{}, err
	}
	return token, u, nil
}

func (s *SQLStore) UserByToken(token string) (User, error) {
	var u User
	err := s.db.QueryRow(`SELECT u.id,u.email,u.password_hash,u.status,u.created_at FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token=$1 AND s.expires_at>now() AND u.status='active'`, token).Scan(&u.ID, &u.Email, &u.Password, &u.Status, &u.CreatedAt)
	if err != nil {
		return User{}, ErrUnauthorized
	}
	return u, nil
}

func (s *SQLStore) Plans() []Plan {
	rows, err := s.db.Query(`SELECT id,name,coalesce(description,''),price_cents,duration_days,traffic_limit_bytes,bandwidth_limit_kbps,max_tunnels,max_tcp_tunnels,max_udp_tunnels,max_http_tunnels,max_https_tunnels,allow_tcp,allow_udp,allow_http,allow_https,allow_custom_domain,max_domains,allow_auto_cert,status FROM plans ORDER BY sort_order,id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Plan
	for rows.Next() {
		var p Plan
		_ = rows.Scan(&p.ID, &p.Name, &p.Description, &p.PriceCents, &p.DurationDays, &p.TrafficLimitBytes, &p.BandwidthKbps, &p.MaxTunnels, &p.MaxTCPTunnels, &p.MaxUDPTunnels, &p.MaxHTTPTunnels, &p.MaxHTTPSTunnels, &p.AllowTCP, &p.AllowUDP, &p.AllowHTTP, &p.AllowHTTPS, &p.AllowCustomDomain, &p.MaxDomains, &p.AllowAutoCert, &p.Status)
		out = append(out, p)
	}
	return out
}

func (s *SQLStore) CreatePaymentOrder(order PaymentOrder) (PaymentOrder, error) {
	if order.Status == "" {
		order.Status = "pending"
	}
	if order.Provider == "" {
		order.Provider = "epay"
	}
	err := s.db.QueryRow(`INSERT INTO payment_orders (user_id,plan_id,provider,out_trade_no,pay_type,name,money,status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
RETURNING id,created_at`, order.UserID, order.PlanID, order.Provider, order.OutTradeNo, order.PayType, order.Name, order.Money, order.Status).Scan(&order.ID, &order.CreatedAt)
	if err != nil {
		if isUnique(err) {
			return PaymentOrder{}, ErrConflict
		}
		return PaymentOrder{}, err
	}
	return order, nil
}

func (s *SQLStore) PaymentOrderByOutTradeNo(outTradeNo string) (PaymentOrder, error) {
	row := s.db.QueryRow(`SELECT id,user_id,plan_id,provider,out_trade_no,coalesce(provider_trade_no,''),coalesce(pay_type,''),name,money,status,created_at,paid_at FROM payment_orders WHERE out_trade_no=$1`, strings.TrimSpace(outTradeNo))
	return scanPaymentOrder(row)
}

func (s *SQLStore) MarkPaymentOrderPaid(outTradeNo, providerTradeNo string) (PaymentOrder, Subscription, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return PaymentOrder{}, Subscription{}, err
	}
	defer tx.Rollback()
	order, err := scanPaymentOrder(tx.QueryRow(`SELECT id,user_id,plan_id,provider,out_trade_no,coalesce(provider_trade_no,''),coalesce(pay_type,''),name,money,status,created_at,paid_at FROM payment_orders WHERE out_trade_no=$1 FOR UPDATE`, strings.TrimSpace(outTradeNo)))
	if err != nil {
		return PaymentOrder{}, Subscription{}, ErrNotFound
	}
	p, err := scanPlan(tx.QueryRow(`SELECT id,name,coalesce(description,''),price_cents,duration_days,traffic_limit_bytes,bandwidth_limit_kbps,max_tunnels,max_tcp_tunnels,max_udp_tunnels,max_http_tunnels,max_https_tunnels,allow_tcp,allow_udp,allow_http,allow_https,allow_custom_domain,max_domains,allow_auto_cert,status FROM plans WHERE id=$1`, order.PlanID))
	if err != nil || p.Status != "active" {
		return PaymentOrder{}, Subscription{}, ErrNotFound
	}
	if order.Status != "paid" {
		now := time.Now()
		_, err = tx.Exec(`UPDATE payment_orders SET status='paid', provider_trade_no=$1, paid_at=$2 WHERE id=$3`, strings.TrimSpace(providerTradeNo), now, order.ID)
		if err != nil {
			return PaymentOrder{}, Subscription{}, err
		}
		_, _ = tx.Exec(`UPDATE subscriptions SET status='replaced' WHERE user_id=$1 AND status='active'`, order.UserID)
		expires := now.AddDate(0, 0, p.DurationDays)
		_, err = tx.Exec(`INSERT INTO subscriptions (user_id,plan_id,plan_name,starts_at,expires_at,traffic_limit_bytes,bandwidth_limit_kbps,allow_tcp,allow_udp,allow_http,allow_https,allow_custom_domain,allow_auto_cert,max_tunnels,max_tcp_tunnels,max_udp_tunnels,max_http_tunnels,max_https_tunnels,max_domains,status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,'active')`, order.UserID, p.ID, p.Name, now, expires, p.TrafficLimitBytes, p.BandwidthKbps, p.AllowTCP, p.AllowUDP, p.AllowHTTP, p.AllowHTTPS, p.AllowCustomDomain, p.AllowAutoCert, p.MaxTunnels, p.MaxTCPTunnels, p.MaxUDPTunnels, p.MaxHTTPTunnels, p.MaxHTTPSTunnels, p.MaxDomains)
		if err != nil {
			return PaymentOrder{}, Subscription{}, err
		}
		order.Status = "paid"
		order.ProviderTradeNo = strings.TrimSpace(providerTradeNo)
		order.PaidAt = &now
	}
	if err := tx.Commit(); err != nil {
		return PaymentOrder{}, Subscription{}, err
	}
	sub, err := s.Subscription(order.UserID)
	if err != nil {
		return PaymentOrder{}, Subscription{}, err
	}
	return order, sub, nil
}

func (s *SQLStore) Redeem(userID int64, code string) (Subscription, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Subscription{}, err
	}
	defer tx.Rollback()
	var planID int64
	var status string
	var expiresAt sql.NullTime
	err = tx.QueryRow(`SELECT plan_id,status,expires_at FROM redeem_codes WHERE code=$1 FOR UPDATE`, strings.TrimSpace(code)).Scan(&planID, &status, &expiresAt)
	if err != nil {
		return Subscription{}, ErrNotFound
	}
	if status != "unused" {
		return Subscription{}, ErrForbidden
	}
	if expiresAt.Valid && time.Now().After(expiresAt.Time) {
		return Subscription{}, ErrForbidden
	}
	p, err := scanPlan(tx.QueryRow(`SELECT id,name,coalesce(description,''),price_cents,duration_days,traffic_limit_bytes,bandwidth_limit_kbps,max_tunnels,max_tcp_tunnels,max_udp_tunnels,max_http_tunnels,max_https_tunnels,allow_tcp,allow_udp,allow_http,allow_https,allow_custom_domain,max_domains,allow_auto_cert,status FROM plans WHERE id=$1`, planID))
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
	_, err = tx.Exec(`INSERT INTO subscriptions (user_id,plan_id,plan_name,starts_at,expires_at,traffic_limit_bytes,bandwidth_limit_kbps,allow_tcp,allow_udp,allow_http,allow_https,allow_custom_domain,allow_auto_cert,max_tunnels,max_tcp_tunnels,max_udp_tunnels,max_http_tunnels,max_https_tunnels,max_domains,status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,'active')`, userID, p.ID, p.Name, now, expires, p.TrafficLimitBytes, p.BandwidthKbps, p.AllowTCP, p.AllowUDP, p.AllowHTTP, p.AllowHTTPS, p.AllowCustomDomain, p.AllowAutoCert, p.MaxTunnels, p.MaxTCPTunnels, p.MaxUDPTunnels, p.MaxHTTPTunnels, p.MaxHTTPSTunnels, p.MaxDomains)
	if err != nil {
		return Subscription{}, err
	}
	if err := tx.Commit(); err != nil {
		return Subscription{}, err
	}
	return planToSub(userID, p, expires), nil
}

func (s *SQLStore) Subscription(userID int64) (Subscription, error) {
	row := s.db.QueryRow(`SELECT plan_id,coalesce(nullif(plan_name,''),'plan'),traffic_limit_bytes,traffic_used_bytes,bandwidth_limit_kbps,allow_tcp,allow_udp,allow_http,allow_https,allow_custom_domain,allow_auto_cert,max_tunnels,max_tcp_tunnels,max_udp_tunnels,max_http_tunnels,max_https_tunnels,max_domains,status,expires_at FROM subscriptions WHERE user_id=$1 AND status='active' ORDER BY expires_at DESC LIMIT 1`, userID)
	var sub Subscription
	err := row.Scan(&sub.PlanID, &sub.PlanName, &sub.TrafficLimitBytes, &sub.TrafficUsedBytes, &sub.BandwidthKbps, &sub.AllowTCP, &sub.AllowUDP, &sub.AllowHTTP, &sub.AllowHTTPS, &sub.AllowCustomDomain, &sub.AllowAutoCert, &sub.MaxTunnels, &sub.MaxTCPTunnels, &sub.MaxUDPTunnels, &sub.MaxHTTPTunnels, &sub.MaxHTTPSTunnels, &sub.MaxDomains, &sub.Status, &sub.ExpiresAt)
	if err != nil {
		return Subscription{}, ErrNotFound
	}
	sub.UserID = userID
	if time.Now().After(sub.ExpiresAt) {
		sub.Status = "expired"
	}
	return sub, nil
}

func (s *SQLStore) CreateTunnel(userID int64, req Tunnel) (Tunnel, error) {
	s.CleanupExpiredSpeedTestTunnels(time.Now())
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
	if req.NodeID > 0 {
		node, err := s.Node(req.NodeID)
		if err != nil {
			return Tunnel{}, err
		}
		st = settingsFromNode(st, node)
	}
	typ := strings.ToLower(req.Type)
	if req.Name == "" || req.LocalHost == "" || req.LocalPort <= 0 {
		return Tunnel{}, fmt.Errorf("invalid tunnel request")
	}
	if err := validateTunnelBandwidth(sub, req.BandwidthKbps); err != nil {
		return Tunnel{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Tunnel{}, err
	}
	defer tx.Rollback()
	t := Tunnel{UserID: userID, NodeID: req.NodeID, Name: req.Name, Type: typ, LocalHost: req.LocalHost, LocalPort: req.LocalPort, UseHTTPS: req.UseHTTPS, BandwidthKbps: req.BandwidthKbps, EffectiveBandwidthKbps: effectiveBandwidth(sub.BandwidthKbps, req.BandwidthKbps), CreatedAt: time.Now()}
	switch typ {
	case "tcp":
		if !sub.AllowTCP {
			return Tunnel{}, ErrForbidden
		}
		port, err := s.allocateSQLPort(tx, req.NodeID, "tcp", st.TCPPortStart, st.TCPPortEnd)
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
		port, err := s.allocateSQLPort(tx, req.NodeID, "udp", st.UDPPortStart, st.UDPPortEnd)
		if err != nil {
			return Tunnel{}, err
		}
		t.RemotePort = port
		t.Status = "active"
		t.PublicURL = fmt.Sprintf("%s:%d", st.ServerAddr, port)
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
	err = tx.QueryRow(`INSERT INTO tunnels (user_id,node_id,name,type,local_host,local_port,remote_port,domain,use_https,bandwidth_limit_kbps,status,public_url) VALUES ($1,NULLIF($2,0),$3,$4,$5,$6,NULLIF($7,0),NULLIF($8,''),$9,$10,$11,$12) RETURNING id,created_at`, userID, t.NodeID, t.Name, t.Type, t.LocalHost, t.LocalPort, t.RemotePort, t.Domain, t.UseHTTPS, t.BandwidthKbps, t.Status, t.PublicURL).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		if isUnique(err) {
			return Tunnel{}, ErrConflict
		}
		return Tunnel{}, err
	}
	if t.RemotePort != 0 {
		_, err = tx.Exec(`UPDATE port_allocations SET tunnel_id=$1,user_id=$2 WHERE node_id=$3 AND protocol=$4 AND port=$5`, t.ID, userID, t.NodeID, typ, t.RemotePort)
		if err != nil {
			return Tunnel{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Tunnel{}, err
	}
	return t, nil
}

func (s *SQLStore) CreateSpeedTestTunnel(userID int64, req SpeedTestTunnelRequest) (SpeedTestTunnel, error) {
	s.CleanupExpiredSpeedTestTunnels(time.Now())
	sub, err := s.Subscription(userID)
	if err != nil || sub.Status != "active" {
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
	st := s.Settings()
	if req.NodeID > 0 {
		node, err := s.Node(req.NodeID)
		if err != nil {
			return SpeedTestTunnel{}, err
		}
		st = settingsFromNode(st, node)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return SpeedTestTunnel{}, err
	}
	defer tx.Rollback()
	expires := time.Now().Add(15 * time.Minute)
	t := Tunnel{UserID: userID, NodeID: req.NodeID, Type: typ, LocalHost: req.LocalHost, LocalPort: req.LocalPort, BandwidthKbps: 0, EffectiveBandwidthKbps: sub.BandwidthKbps, SpeedTest: true, ExpiresAt: &expires, Status: "active"}
	switch typ {
	case "tcp":
		if !sub.AllowTCP {
			return SpeedTestTunnel{}, ErrForbidden
		}
		port, err := s.allocateSQLPort(tx, req.NodeID, "tcp", st.TCPPortStart, st.TCPPortEnd)
		if err != nil {
			return SpeedTestTunnel{}, err
		}
		t.RemotePort = port
		t.PublicURL = fmt.Sprintf("%s:%d", st.ServerAddr, port)
	case "udp":
		if !sub.AllowUDP {
			return SpeedTestTunnel{}, ErrForbidden
		}
		port, err := s.allocateSQLPort(tx, req.NodeID, "udp", st.UDPPortStart, st.UDPPortEnd)
		if err != nil {
			return SpeedTestTunnel{}, err
		}
		t.RemotePort = port
		t.PublicURL = fmt.Sprintf("%s:%d", st.ServerAddr, port)
	case "http", "https":
		if typ == "http" && !sub.AllowHTTP {
			return SpeedTestTunnel{}, ErrForbidden
		}
		if typ == "https" && !sub.AllowHTTPS {
			return SpeedTestTunnel{}, ErrForbidden
		}
		t.Domain = fmt.Sprintf("speed-%d.%s", time.Now().UnixNano(), strings.TrimPrefix(st.FRPEntryDomain, "*."))
		t.UseHTTPS = typ == "https"
		if typ == "https" {
			t.PublicURL = "https://" + t.Domain
		} else {
			t.PublicURL = "http://" + t.Domain
		}
	default:
		return SpeedTestTunnel{}, fmt.Errorf("unsupported tunnel type")
	}
	t.Name = fmt.Sprintf("__speedtest_%s_%d", typ, time.Now().UnixNano())
	err = tx.QueryRow(`INSERT INTO tunnels (user_id,node_id,name,type,local_host,local_port,remote_port,domain,use_https,bandwidth_limit_kbps,speed_test,expires_at,status,public_url) VALUES ($1,NULLIF($2,0),$3,$4,$5,$6,NULLIF($7,0),NULLIF($8,''),$9,$10,true,$11,$12,$13) RETURNING id,created_at`, userID, t.NodeID, t.Name, t.Type, t.LocalHost, t.LocalPort, t.RemotePort, t.Domain, t.UseHTTPS, t.BandwidthKbps, expires, t.Status, t.PublicURL).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		if isUnique(err) {
			return SpeedTestTunnel{}, ErrConflict
		}
		return SpeedTestTunnel{}, err
	}
	if t.RemotePort != 0 {
		if _, err := tx.Exec(`UPDATE port_allocations SET tunnel_id=$1,user_id=$2 WHERE node_id=$3 AND protocol=$4 AND port=$5`, t.ID, userID, t.NodeID, typ, t.RemotePort); err != nil {
			return SpeedTestTunnel{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return SpeedTestTunnel{}, err
	}
	return speedTestTunnelFromTunnel(t), nil
}

func (s *SQLStore) SpeedTestTunnel(userID int64, tunnelID int64) (SpeedTestTunnel, error) {
	rows := s.queryTunnels(`WHERE user_id=$1 AND id=$2 AND speed_test=true AND status <> 'deleted' LIMIT 1`, userID, tunnelID)
	if len(rows) == 0 {
		return SpeedTestTunnel{}, ErrNotFound
	}
	t := rows[0]
	if sub, err := s.Subscription(userID); err == nil && sub.Status == "active" {
		t.EffectiveBandwidthKbps = effectiveBandwidth(sub.BandwidthKbps, t.BandwidthKbps)
	}
	return speedTestTunnelFromTunnel(t), nil
}

func (s *SQLStore) FinishSpeedTestTunnel(userID int64, tunnelID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var t Tunnel
	var expires sql.NullTime
	err = tx.QueryRow(`SELECT id,user_id,coalesce(node_id,0),type,coalesce(remote_port,0),coalesce(domain,''),speed_test,expires_at FROM tunnels WHERE id=$1 FOR UPDATE`, tunnelID).Scan(&t.ID, &t.UserID, &t.NodeID, &t.Type, &t.RemotePort, &t.Domain, &t.SpeedTest, &expires)
	if err != nil || t.UserID != userID || !t.SpeedTest {
		return ErrNotFound
	}
	if _, err := tx.Exec(`UPDATE tunnels SET status='deleted', updated_at=now() WHERE id=$1`, tunnelID); err != nil {
		return err
	}
	if t.RemotePort > 0 {
		if _, err := tx.Exec(`DELETE FROM port_allocations WHERE node_id=$1 AND protocol=$2 AND port=$3`, t.NodeID, t.Type, t.RemotePort); err != nil {
			return err
		}
	}
	if t.Domain != "" {
		if _, err := tx.Exec(`UPDATE tunnels SET domain=NULL WHERE id=$1 AND status='deleted'`, tunnelID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLStore) UpdateTunnelStatus(userID int64, tunnelID int64, status string) (Tunnel, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "active" && status != "disabled" {
		return Tunnel{}, fmt.Errorf("unsupported tunnel status")
	}
	if status == "active" {
		sub, err := s.Subscription(userID)
		if err != nil || sub.Status != "active" {
			return Tunnel{}, ErrForbidden
		}
		if sub.TrafficLimitBytes > 0 && sub.TrafficUsedBytes >= sub.TrafficLimitBytes {
			return Tunnel{}, ErrForbidden
		}
	}
	res, err := s.db.Exec(`UPDATE tunnels SET status=$3,updated_at=now() WHERE id=$1 AND user_id=$2 AND speed_test=false AND status <> 'deleted'`, tunnelID, userID, status)
	if err != nil {
		return Tunnel{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Tunnel{}, ErrNotFound
	}
	rows := s.queryTunnels(`WHERE user_id=$1 AND id=$2 LIMIT 1`, userID, tunnelID)
	if len(rows) == 0 {
		return Tunnel{}, ErrNotFound
	}
	return rows[0], nil
}

func (s *SQLStore) DeleteTunnel(userID int64, tunnelID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var t Tunnel
	err = tx.QueryRow(`SELECT id,user_id,coalesce(node_id,0),type,coalesce(remote_port,0),coalesce(domain,''),status FROM tunnels WHERE id=$1 FOR UPDATE`, tunnelID).Scan(&t.ID, &t.UserID, &t.NodeID, &t.Type, &t.RemotePort, &t.Domain, &t.Status)
	if err != nil || t.UserID != userID || t.Status == "deleted" {
		return ErrNotFound
	}
	if _, err := tx.Exec(`UPDATE tunnels SET status='deleted',domain=NULL,updated_at=now() WHERE id=$1`, tunnelID); err != nil {
		return err
	}
	if t.RemotePort > 0 {
		if _, err := tx.Exec(`DELETE FROM port_allocations WHERE node_id=$1 AND protocol=$2 AND port=$3`, t.NodeID, t.Type, t.RemotePort); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SQLStore) CleanupExpiredSpeedTestTunnels(now time.Time) int {
	tx, err := s.db.Begin()
	if err != nil {
		return 0
	}
	defer tx.Rollback()
	rows, err := tx.Query(`SELECT id,coalesce(node_id,0),type,coalesce(remote_port,0) FROM tunnels WHERE speed_test=true AND status <> 'deleted' AND expires_at IS NOT NULL AND expires_at < $1 FOR UPDATE`, now)
	if err != nil {
		return 0
	}
	type expiredTunnel struct {
		id     int64
		nodeID int64
		typ    string
		port   int
	}
	expired := []expiredTunnel{}
	for rows.Next() {
		var item expiredTunnel
		if err := rows.Scan(&item.id, &item.nodeID, &item.typ, &item.port); err == nil {
			expired = append(expired, item)
		}
	}
	_ = rows.Close()
	for _, item := range expired {
		_, _ = tx.Exec(`UPDATE tunnels SET status='deleted',domain=NULL,updated_at=now() WHERE id=$1`, item.id)
		if item.port > 0 {
			_, _ = tx.Exec(`DELETE FROM port_allocations WHERE node_id=$1 AND protocol=$2 AND port=$3`, item.nodeID, item.typ, item.port)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0
	}
	return len(expired)
}

func (s *SQLStore) allocateSQLPort(tx *sql.Tx, nodeID int64, protocol string, start, end int) (int, error) {
	for p := start; p <= end; p++ {
		_, err := tx.Exec(`INSERT INTO port_allocations (node_id,protocol,port,status) VALUES ($1,$2,$3,'allocated')`, nodeID, protocol, p)
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
	rows, err := s.db.Query(`SELECT id,user_id,coalesce(node_id,0),name,type,local_host,local_port,coalesce(remote_port,0),coalesce(domain,''),use_https,bandwidth_limit_kbps,speed_test,expires_at,status,coalesce(public_url,''),coalesce(error_message,''),created_at FROM tunnels `+suffix, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []Tunnel
	for rows.Next() {
		var t Tunnel
		var expires sql.NullTime
		_ = rows.Scan(&t.ID, &t.UserID, &t.NodeID, &t.Name, &t.Type, &t.LocalHost, &t.LocalPort, &t.RemotePort, &t.Domain, &t.UseHTTPS, &t.BandwidthKbps, &t.SpeedTest, &expires, &t.Status, &t.PublicURL, &t.ErrorMessage, &t.CreatedAt)
		if expires.Valid {
			t.ExpiresAt = &expires.Time
		}
		out = append(out, t)
	}
	return out
}

func (s *SQLStore) Nodes() []Node {
	rows, err := s.db.Query(`SELECT id,name,coalesce(agent_url,''),agent_token,bind_token,coalesce(public_url,''),coalesce(frp_entry_domain,''),coalesce(server_addr,''),frp_server_port,tcp_port_start,tcp_port_end,udp_port_start,udp_port_end,status,last_seen_at,coalesce(last_error,''),created_at,updated_at FROM nodes ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []Node{}
	for rows.Next() {
		n, err := scanNode(rows)
		if err == nil {
			out = append(out, n)
		}
	}
	return out
}

func (s *SQLStore) Node(id int64) (Node, error) {
	return scanNode(s.db.QueryRow(`SELECT id,name,coalesce(agent_url,''),agent_token,bind_token,coalesce(public_url,''),coalesce(frp_entry_domain,''),coalesce(server_addr,''),frp_server_port,tcp_port_start,tcp_port_end,udp_port_start,udp_port_end,status,last_seen_at,coalesce(last_error,''),created_at,updated_at FROM nodes WHERE id=$1`, id))
}

func (s *SQLStore) CreateNode(node Node) (Node, error) {
	if strings.TrimSpace(node.Name) == "" {
		node.Name = "node"
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
	if node.Status == "" {
		node.Status = "pending"
	}
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
	return scanNode(s.db.QueryRow(`INSERT INTO nodes (name,agent_url,agent_token,bind_token,public_url,frp_entry_domain,server_addr,frp_server_port,tcp_port_start,tcp_port_end,udp_port_start,udp_port_end,status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
RETURNING id,name,coalesce(agent_url,''),agent_token,bind_token,coalesce(public_url,''),coalesce(frp_entry_domain,''),coalesce(server_addr,''),frp_server_port,tcp_port_start,tcp_port_end,udp_port_start,udp_port_end,status,last_seen_at,coalesce(last_error,''),created_at,updated_at`, node.Name, node.AgentURL, node.AgentToken, node.BindToken, node.PublicURL, node.FRPEntryDomain, node.ServerAddr, node.FRPServerPort, node.TCPPortStart, node.TCPPortEnd, node.UDPPortStart, node.UDPPortEnd, node.Status))
}

func (s *SQLStore) DeleteNode(id int64) error {
	res, err := s.db.Exec(`DELETE FROM nodes WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SQLStore) BindNode(req NodeBindRequest) (Node, error) {
	now := time.Now()
	res, err := s.db.Exec(`UPDATE nodes SET
name=COALESCE(NULLIF($2,''),name),
agent_url=COALESCE(NULLIF($3,''),agent_url),
public_url=COALESCE(NULLIF($4,''),public_url),
frp_entry_domain=COALESCE(NULLIF($5,''),frp_entry_domain),
server_addr=COALESCE(NULLIF($6,''),server_addr),
frp_server_port=CASE WHEN $7>0 THEN $7 ELSE frp_server_port END,
tcp_port_start=CASE WHEN $8>0 THEN $8 ELSE tcp_port_start END,
tcp_port_end=CASE WHEN $9>0 THEN $9 ELSE tcp_port_end END,
udp_port_start=CASE WHEN $10>0 THEN $10 ELSE udp_port_start END,
udp_port_end=CASE WHEN $11>0 THEN $11 ELSE udp_port_end END,
status='online', last_error='', last_seen_at=$12, updated_at=$12
WHERE bind_token=$1`, strings.TrimSpace(req.BindToken), strings.TrimSpace(req.Name), strings.TrimRight(strings.TrimSpace(req.AgentURL), "/"), strings.TrimRight(strings.TrimSpace(req.PublicURL), "/"), strings.TrimSpace(req.FRPEntryDomain), strings.TrimSpace(req.ServerAddr), req.FRPServerPort, req.TCPPortStart, req.TCPPortEnd, req.UDPPortStart, req.UDPPortEnd, now)
	if err != nil {
		return Node{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Node{}, ErrUnauthorized
	}
	return scanNode(s.db.QueryRow(`SELECT id,name,coalesce(agent_url,''),agent_token,bind_token,coalesce(public_url,''),coalesce(frp_entry_domain,''),coalesce(server_addr,''),frp_server_port,tcp_port_start,tcp_port_end,udp_port_start,udp_port_end,status,last_seen_at,coalesce(last_error,''),created_at,updated_at FROM nodes WHERE bind_token=$1`, strings.TrimSpace(req.BindToken)))
}

func (s *SQLStore) UpdateNodeStatus(id int64, status string, lastError string) (Node, error) {
	return scanNode(s.db.QueryRow(`UPDATE nodes SET status=$2,last_error=$3,last_seen_at=CASE WHEN $2='online' THEN now() ELSE last_seen_at END,updated_at=now() WHERE id=$1 RETURNING id,name,coalesce(agent_url,''),agent_token,bind_token,coalesce(public_url,''),coalesce(frp_entry_domain,''),coalesce(server_addr,''),frp_server_port,tcp_port_start,tcp_port_end,udp_port_start,udp_port_end,status,last_seen_at,coalesce(last_error,''),created_at,updated_at`, id, status, lastError))
}

func scanNode(scanner interface{ Scan(dest ...any) error }) (Node, error) {
	var n Node
	var seen sql.NullTime
	err := scanner.Scan(&n.ID, &n.Name, &n.AgentURL, &n.AgentToken, &n.BindToken, &n.PublicURL, &n.FRPEntryDomain, &n.ServerAddr, &n.FRPServerPort, &n.TCPPortStart, &n.TCPPortEnd, &n.UDPPortStart, &n.UDPPortEnd, &n.Status, &seen, &n.LastError, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Node{}, ErrNotFound
		}
		return Node{}, err
	}
	if seen.Valid {
		n.LastSeenAt = &seen.Time
	}
	return n, nil
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
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.PriceCents, &p.DurationDays, &p.TrafficLimitBytes, &p.BandwidthKbps, &p.MaxTunnels, &p.MaxTCPTunnels, &p.MaxUDPTunnels, &p.MaxHTTPTunnels, &p.MaxHTTPSTunnels, &p.AllowTCP, &p.AllowUDP, &p.AllowHTTP, &p.AllowHTTPS, &p.AllowCustomDomain, &p.MaxDomains, &p.AllowAutoCert, &p.Status)
	return p, err
}

type paymentOrderScanner interface {
	Scan(dest ...any) error
}

func scanPaymentOrder(row paymentOrderScanner) (PaymentOrder, error) {
	var o PaymentOrder
	var paid sql.NullTime
	err := row.Scan(&o.ID, &o.UserID, &o.PlanID, &o.Provider, &o.OutTradeNo, &o.ProviderTradeNo, &o.PayType, &o.Name, &o.Money, &o.Status, &o.CreatedAt, &paid)
	if paid.Valid {
		o.PaidAt = &paid.Time
	}
	return o, err
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
	err := s.db.QueryRow(`INSERT INTO plans (name,description,price_cents,duration_days,traffic_limit_bytes,bandwidth_limit_kbps,max_tunnels,max_tcp_tunnels,max_udp_tunnels,max_http_tunnels,max_https_tunnels,allow_tcp,allow_udp,allow_http,allow_https,allow_custom_domain,max_domains,allow_auto_cert,status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
RETURNING id`, plan.Name, plan.Description, plan.PriceCents, plan.DurationDays, plan.TrafficLimitBytes, plan.BandwidthKbps, plan.MaxTunnels, plan.MaxTCPTunnels, plan.MaxUDPTunnels, plan.MaxHTTPTunnels, plan.MaxHTTPSTunnels, plan.AllowTCP, plan.AllowUDP, plan.AllowHTTP, plan.AllowHTTPS, plan.AllowCustomDomain, plan.MaxDomains, plan.AllowAutoCert, plan.Status).Scan(&plan.ID)
	return plan, err
}

func (s *SQLStore) UpdatePlan(id int64, plan Plan) (Plan, error) {
	if plan.Status == "" {
		plan.Status = "active"
	}
	plan.ID = id
	res, err := s.db.Exec(`UPDATE plans SET name=$1,description=$2,price_cents=$3,duration_days=$4,traffic_limit_bytes=$5,bandwidth_limit_kbps=$6,max_tunnels=$7,max_tcp_tunnels=$8,max_udp_tunnels=$9,max_http_tunnels=$10,max_https_tunnels=$11,allow_tcp=$12,allow_udp=$13,allow_http=$14,allow_https=$15,allow_custom_domain=$16,max_domains=$17,allow_auto_cert=$18,status=$19,updated_at=now() WHERE id=$20`,
		plan.Name, plan.Description, plan.PriceCents, plan.DurationDays, plan.TrafficLimitBytes, plan.BandwidthKbps, plan.MaxTunnels, plan.MaxTCPTunnels, plan.MaxUDPTunnels, plan.MaxHTTPTunnels, plan.MaxHTTPSTunnels, plan.AllowTCP, plan.AllowUDP, plan.AllowHTTP, plan.AllowHTTPS, plan.AllowCustomDomain, plan.MaxDomains, plan.AllowAutoCert, plan.Status, id)
	if err != nil {
		return Plan{}, err
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return Plan{}, ErrNotFound
	}
	return plan, nil
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

func (s *SQLStore) UpdateUser(id int64, status string, planID int64) (User, Subscription, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return User{}, Subscription{}, err
	}
	defer tx.Rollback()
	if strings.TrimSpace(status) != "" {
		if _, err := tx.Exec(`UPDATE users SET status=$1, updated_at=now() WHERE id=$2`, strings.TrimSpace(status), id); err != nil {
			return User{}, Subscription{}, err
		}
	}
	var u User
	if err := tx.QueryRow(`SELECT id,email,password_hash,status,created_at FROM users WHERE id=$1`, id).Scan(&u.ID, &u.Email, &u.Password, &u.Status, &u.CreatedAt); err != nil {
		return User{}, Subscription{}, ErrNotFound
	}
	var sub Subscription
	if planID > 0 {
		row := tx.QueryRow(`SELECT id,name,coalesce(description,''),price_cents,duration_days,traffic_limit_bytes,bandwidth_limit_kbps,max_tunnels,max_tcp_tunnels,max_udp_tunnels,max_http_tunnels,max_https_tunnels,allow_tcp,allow_udp,allow_http,allow_https,allow_custom_domain,max_domains,allow_auto_cert,status FROM plans WHERE id=$1 AND status='active'`, planID)
		var p Plan
		if err := row.Scan(&p.ID, &p.Name, &p.Description, &p.PriceCents, &p.DurationDays, &p.TrafficLimitBytes, &p.BandwidthKbps, &p.MaxTunnels, &p.MaxTCPTunnels, &p.MaxUDPTunnels, &p.MaxHTTPTunnels, &p.MaxHTTPSTunnels, &p.AllowTCP, &p.AllowUDP, &p.AllowHTTP, &p.AllowHTTPS, &p.AllowCustomDomain, &p.MaxDomains, &p.AllowAutoCert, &p.Status); err != nil {
			return User{}, Subscription{}, ErrNotFound
		}
		now := time.Now()
		expires := now.AddDate(0, 0, p.DurationDays)
		_, _ = tx.Exec(`UPDATE subscriptions SET status='replaced', updated_at=now() WHERE user_id=$1 AND status='active'`, id)
		if _, err := tx.Exec(`INSERT INTO subscriptions (user_id,plan_id,plan_name,starts_at,expires_at,traffic_limit_bytes,bandwidth_limit_kbps,allow_tcp,allow_udp,allow_http,allow_https,allow_custom_domain,allow_auto_cert,max_tunnels,max_tcp_tunnels,max_udp_tunnels,max_http_tunnels,max_https_tunnels,max_domains,status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,'active')`, id, p.ID, p.Name, now, expires, p.TrafficLimitBytes, p.BandwidthKbps, p.AllowTCP, p.AllowUDP, p.AllowHTTP, p.AllowHTTPS, p.AllowCustomDomain, p.AllowAutoCert, p.MaxTunnels, p.MaxTCPTunnels, p.MaxUDPTunnels, p.MaxHTTPTunnels, p.MaxHTTPSTunnels, p.MaxDomains); err != nil {
			return User{}, Subscription{}, err
		}
		sub = planToSub(id, p, expires)
	} else {
		_ = tx.QueryRow(`SELECT plan_id,coalesce(nullif(plan_name,''),'plan'),traffic_limit_bytes,traffic_used_bytes,bandwidth_limit_kbps,allow_tcp,allow_udp,allow_http,allow_https,allow_custom_domain,allow_auto_cert,max_tunnels,max_tcp_tunnels,max_udp_tunnels,max_http_tunnels,max_https_tunnels,max_domains,status,expires_at FROM subscriptions WHERE user_id=$1 AND status='active' ORDER BY expires_at DESC LIMIT 1`, id).Scan(&sub.PlanID, &sub.PlanName, &sub.TrafficLimitBytes, &sub.TrafficUsedBytes, &sub.BandwidthKbps, &sub.AllowTCP, &sub.AllowUDP, &sub.AllowHTTP, &sub.AllowHTTPS, &sub.AllowCustomDomain, &sub.AllowAutoCert, &sub.MaxTunnels, &sub.MaxTCPTunnels, &sub.MaxUDPTunnels, &sub.MaxHTTPTunnels, &sub.MaxHTTPSTunnels, &sub.MaxDomains, &sub.Status, &sub.ExpiresAt)
		sub.UserID = id
	}
	if err := tx.Commit(); err != nil {
		return User{}, Subscription{}, err
	}
	return u, sub, nil
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

func (s *SQLStore) SaveCertificate(record CertificateRecord) (CertificateRecord, error) {
	domain := sanitizeDomain(record.Domain)
	if domain == "" {
		return CertificateRecord{}, fmt.Errorf("domain required")
	}
	record.Domain = domain
	err := s.db.QueryRow(`INSERT INTO certificates (user_id,domain,status,issued_at,expires_at,cert_path,key_path,last_command,last_output,error_message)
VALUES (NULLIF($1,0),$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (domain) DO UPDATE SET user_id=COALESCE(excluded.user_id,certificates.user_id), status=excluded.status, issued_at=excluded.issued_at, expires_at=excluded.expires_at, cert_path=excluded.cert_path, key_path=excluded.key_path, last_command=excluded.last_command, last_output=excluded.last_output, error_message=excluded.error_message, updated_at=now()
RETURNING id,coalesce(user_id,0),domain,status,issued_at,expires_at,coalesce(cert_path,''),coalesce(key_path,''),coalesce(last_command,''),coalesce(last_output,''),coalesce(error_message,''),created_at,updated_at`, record.UserID, record.Domain, record.Status, record.IssuedAt, record.ExpiresAt, record.CertPath, record.KeyPath, record.LastCommand, record.LastOutput, record.ErrorMessage).Scan(&record.ID, &record.UserID, &record.Domain, &record.Status, &record.IssuedAt, &record.ExpiresAt, &record.CertPath, &record.KeyPath, &record.LastCommand, &record.LastOutput, &record.ErrorMessage, &record.CreatedAt, &record.UpdatedAt)
	return record, err
}

func (s *SQLStore) Certificates() []CertificateRecord {
	rows, err := s.db.Query(`SELECT id,coalesce(user_id,0),domain,status,issued_at,expires_at,coalesce(cert_path,''),coalesce(key_path,''),coalesce(last_command,''),coalesce(last_output,''),coalesce(error_message,''),created_at,updated_at FROM certificates ORDER BY updated_at DESC LIMIT 500`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []CertificateRecord{}
	for rows.Next() {
		var cert CertificateRecord
		var issued sql.NullTime
		var expires sql.NullTime
		_ = rows.Scan(&cert.ID, &cert.UserID, &cert.Domain, &cert.Status, &issued, &expires, &cert.CertPath, &cert.KeyPath, &cert.LastCommand, &cert.LastOutput, &cert.ErrorMessage, &cert.CreatedAt, &cert.UpdatedAt)
		if issued.Valid {
			cert.IssuedAt = &issued.Time
		}
		if expires.Valid {
			cert.ExpiresAt = &expires.Time
		}
		out = append(out, cert)
	}
	return out
}

func (s *SQLStore) RecordAdminOperation(logEntry AdminOperationLog) error {
	_, err := s.db.Exec(`INSERT INTO admin_operation_logs (admin_id,admin_email,action,target,detail,ip,created_at) VALUES ($1,$2,$3,$4,$5,$6,coalesce(nullif($7,'0001-01-01T00:00:00Z')::timestamptz, now()))`, logEntry.AdminID, logEntry.AdminEmail, logEntry.Action, logEntry.Target, logEntry.Detail, logEntry.IP, logEntry.CreatedAt.Format(time.RFC3339))
	return err
}

func (s *SQLStore) AdminOperationLogs(limit int) []AdminOperationLog {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.Query(`SELECT id,coalesce(admin_id,0),coalesce(admin_email,''),action,coalesce(target,''),coalesce(detail,''),coalesce(ip,''),created_at FROM admin_operation_logs ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := []AdminOperationLog{}
	for rows.Next() {
		var item AdminOperationLog
		_ = rows.Scan(&item.ID, &item.AdminID, &item.AdminEmail, &item.Action, &item.Target, &item.Detail, &item.IP, &item.CreatedAt)
		out = append(out, item)
	}
	return out
}
