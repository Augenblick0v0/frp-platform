package platform

import "time"

type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Plan struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	DurationDays      int    `json:"duration_days"`
	TrafficLimitBytes int64  `json:"traffic_limit_bytes"`
	BandwidthKbps     int    `json:"bandwidth_limit_kbps"`
	MaxTunnels        int    `json:"max_tunnels"`
	MaxTCPTunnels     int    `json:"max_tcp_tunnels"`
	MaxUDPTunnels     int    `json:"max_udp_tunnels"`
	MaxHTTPTunnels    int    `json:"max_http_tunnels"`
	MaxHTTPSTunnels   int    `json:"max_https_tunnels"`
	AllowTCP          bool   `json:"allow_tcp"`
	AllowUDP          bool   `json:"allow_udp"`
	AllowHTTP         bool   `json:"allow_http"`
	AllowHTTPS        bool   `json:"allow_https"`
	AllowCustomDomain bool   `json:"allow_custom_domain"`
	MaxDomains        int    `json:"max_domains"`
	AllowAutoCert     bool   `json:"allow_auto_cert"`
	Status            string `json:"status"`
}

type Subscription struct {
	UserID            int64     `json:"user_id"`
	PlanID            int64     `json:"plan_id"`
	PlanName          string    `json:"plan_name"`
	ExpiresAt         time.Time `json:"expires_at"`
	TrafficLimitBytes int64     `json:"traffic_limit_bytes"`
	TrafficUsedBytes  int64     `json:"traffic_used_bytes"`
	BandwidthKbps     int       `json:"bandwidth_limit_kbps"`
	AllowTCP          bool      `json:"allow_tcp"`
	AllowUDP          bool      `json:"allow_udp"`
	AllowHTTP         bool      `json:"allow_http"`
	AllowHTTPS        bool      `json:"allow_https"`
	AllowCustomDomain bool      `json:"allow_custom_domain"`
	AllowAutoCert     bool      `json:"allow_auto_cert"`
	MaxTunnels        int       `json:"max_tunnels"`
	MaxDomains        int       `json:"max_domains"`
	Status            string    `json:"status"`
}

type RedeemCode struct {
	Code       string     `json:"code"`
	PlanID     int64      `json:"plan_id"`
	Status     string     `json:"status"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RedeemedBy int64      `json:"redeemed_by_user_id,omitempty"`
	RedeemedAt *time.Time `json:"redeemed_at,omitempty"`
}

type Tunnel struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	LocalHost    string    `json:"local_host"`
	LocalPort    int       `json:"local_port"`
	RemotePort   int       `json:"remote_port,omitempty"`
	Domain       string    `json:"domain,omitempty"`
	UseHTTPS     bool      `json:"use_https"`
	Status       string    `json:"status"`
	PublicURL    string    `json:"public_url"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type Settings struct {
	PlatformDomain string `json:"platform_domain"`
	FRPEntryDomain string `json:"frp_entry_domain"`
	ServerAddr     string `json:"server_addr"`
	FRPServerPort  int    `json:"frp_server_port"`
	TCPPortStart   int    `json:"tcp_port_start"`
	TCPPortEnd     int    `json:"tcp_port_end"`
	UDPPortStart   int    `json:"udp_port_start"`
	UDPPortEnd     int    `json:"udp_port_end"`
	PurchaseURL    string `json:"purchase_url"`
}
