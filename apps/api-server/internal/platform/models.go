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
	PriceCents        int64  `json:"price_cents"`
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
	MaxTCPTunnels     int       `json:"max_tcp_tunnels"`
	MaxUDPTunnels     int       `json:"max_udp_tunnels"`
	MaxHTTPTunnels    int       `json:"max_http_tunnels"`
	MaxHTTPSTunnels   int       `json:"max_https_tunnels"`
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

type PaymentOrder struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	PlanID          int64      `json:"plan_id"`
	Provider        string     `json:"provider"`
	OutTradeNo      string     `json:"out_trade_no"`
	ProviderTradeNo string     `json:"provider_trade_no,omitempty"`
	PayType         string     `json:"pay_type"`
	Name            string     `json:"name"`
	Money           string     `json:"money"`
	Status          string     `json:"status"`
	PayURL          string     `json:"pay_url,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
}

type Tunnel struct {
	ID                     int64      `json:"id"`
	UserID                 int64      `json:"user_id"`
	NodeID                 int64      `json:"node_id,omitempty"`
	Name                   string     `json:"name"`
	Type                   string     `json:"type"`
	LocalHost              string     `json:"local_host"`
	LocalPort              int        `json:"local_port"`
	RemotePort             int        `json:"remote_port,omitempty"`
	Domain                 string     `json:"domain,omitempty"`
	UseHTTPS               bool       `json:"use_https"`
	BandwidthKbps          int        `json:"bandwidth_limit_kbps"`
	EffectiveBandwidthKbps int        `json:"effective_bandwidth_limit_kbps"`
	SpeedTest              bool       `json:"speed_test"`
	ExpiresAt              *time.Time `json:"expires_at,omitempty"`
	Status                 string     `json:"status"`
	PublicURL              string     `json:"public_url"`
	ErrorMessage           string     `json:"error_message,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
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

type Node struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	AgentURL       string     `json:"agent_url"`
	AgentToken     string     `json:"-"`
	BindToken      string     `json:"bind_token,omitempty"`
	PublicURL      string     `json:"public_url"`
	FRPEntryDomain string     `json:"frp_entry_domain"`
	ServerAddr     string     `json:"server_addr"`
	FRPServerPort  int        `json:"frp_server_port"`
	TCPPortStart   int        `json:"tcp_port_start"`
	TCPPortEnd     int        `json:"tcp_port_end"`
	UDPPortStart   int        `json:"udp_port_start"`
	UDPPortEnd     int        `json:"udp_port_end"`
	Status         string     `json:"status"`
	LastSeenAt     *time.Time `json:"last_seen_at,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type NodeBindRequest struct {
	BindToken      string `json:"bind_token"`
	Name           string `json:"name"`
	AgentURL       string `json:"agent_url"`
	PublicURL      string `json:"public_url"`
	FRPEntryDomain string `json:"frp_entry_domain"`
	ServerAddr     string `json:"server_addr"`
	FRPServerPort  int    `json:"frp_server_port"`
	TCPPortStart   int    `json:"tcp_port_start"`
	TCPPortEnd     int    `json:"tcp_port_end"`
	UDPPortStart   int    `json:"udp_port_start"`
	UDPPortEnd     int    `json:"udp_port_end"`
}

type TrafficReport struct {
	TunnelID int64 `json:"tunnel_id"`
	BytesIn  int64 `json:"bytes_in"`
	BytesOut int64 `json:"bytes_out"`
}

type SpeedTestTunnelRequest struct {
	Type          string `json:"type"`
	LocalHost     string `json:"local_host"`
	LocalPort     int    `json:"local_port"`
	NodeID        int64  `json:"node_id,omitempty"`
	BandwidthKbps int    `json:"bandwidth_limit_kbps"`
}

type SpeedTestTunnel struct {
	ID                     int64     `json:"id"`
	NodeID                 int64     `json:"node_id,omitempty"`
	Type                   string    `json:"type"`
	LocalHost              string    `json:"local_host"`
	LocalPort              int       `json:"local_port"`
	RemotePort             int       `json:"remote_port,omitempty"`
	Domain                 string    `json:"domain,omitempty"`
	PublicURL              string    `json:"public_url"`
	EffectiveBandwidthKbps int       `json:"effective_bandwidth_limit_kbps"`
	ExpiresAt              time.Time `json:"expires_at"`
}

type SpeedTestProbeRequest struct {
	DownloadBytes   int64 `json:"download_bytes"`
	UploadBytes     int64 `json:"upload_bytes"`
	DurationSeconds int   `json:"duration_seconds"`
}

type SpeedTestProbeMetrics struct {
	DownloadAverageKbps float64 `json:"download_average_kbps"`
	DownloadPeakKbps    float64 `json:"download_peak_kbps"`
	UploadAverageKbps   float64 `json:"upload_average_kbps"`
	UploadPeakKbps      float64 `json:"upload_peak_kbps"`
	LatencyMs           float64 `json:"latency_ms"`
	BytesIn             int64   `json:"bytes_in"`
	BytesOut            int64   `json:"bytes_out"`
}

type SpeedTestRunResult struct {
	Tunnel                      SpeedTestTunnel       `json:"tunnel"`
	Metrics                     SpeedTestProbeMetrics `json:"metrics"`
	EffectiveBandwidthLimitKbps int                   `json:"effective_bandwidth_limit_kbps"`
	LimitRatio                  float64               `json:"limit_ratio"`
	BottleneckHint              string                `json:"bottleneck_hint"`
	Finished                    bool                  `json:"finished"`
}

type TrafficSummary struct {
	UserID            int64 `json:"user_id"`
	TrafficLimitBytes int64 `json:"traffic_limit_bytes"`
	TrafficUsedBytes  int64 `json:"traffic_used_bytes"`
	TrafficLeftBytes  int64 `json:"traffic_left_bytes"`
	TodayBytes        int64 `json:"today_bytes"`
}

type AdminUser struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type CertificateRecord struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id,omitempty"`
	Domain       string     `json:"domain"`
	Status       string     `json:"status"`
	IssuedAt     *time.Time `json:"issued_at,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CertPath     string     `json:"cert_path,omitempty"`
	KeyPath      string     `json:"key_path,omitempty"`
	LastCommand  string     `json:"last_command,omitempty"`
	LastOutput   string     `json:"last_output,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type AdminOperationLog struct {
	ID         int64     `json:"id"`
	AdminID    int64     `json:"admin_id"`
	AdminEmail string    `json:"admin_email"`
	Action     string    `json:"action"`
	Target     string    `json:"target"`
	Detail     string    `json:"detail"`
	IP         string    `json:"ip"`
	CreatedAt  time.Time `json:"created_at"`
}

type SafeNode struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	PublicURL      string     `json:"public_url"`
	FRPEntryDomain string     `json:"frp_entry_domain"`
	ServerAddr     string     `json:"server_addr"`
	FRPServerPort  int        `json:"frp_server_port"`
	TCPPortStart   int        `json:"tcp_port_start"`
	TCPPortEnd     int        `json:"tcp_port_end"`
	UDPPortStart   int        `json:"udp_port_start"`
	UDPPortEnd     int        `json:"udp_port_end"`
	Status         string     `json:"status"`
	LastSeenAt     *time.Time `json:"last_seen_at,omitempty"`
}

type TopologyLink struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Description string `json:"description"`
}

type DownloadArtifact struct {
	Platform string `json:"platform"`
	Label    string `json:"label"`
	URL      string `json:"url"`
}

type UserTopology struct {
	Role         string             `json:"role"`
	User         User               `json:"user"`
	Subscription Subscription       `json:"subscription"`
	Traffic      TrafficSummary     `json:"traffic"`
	TunnelCount  int                `json:"tunnel_count"`
	TunnelCounts map[string]int     `json:"tunnel_counts"`
	Nodes        []SafeNode         `json:"nodes"`
	Downloads    []DownloadArtifact `json:"downloads"`
	RoleFlow     []TopologyLink     `json:"role_flow"`
	GeneratedAt  time.Time          `json:"generated_at"`
}

type PaymentMethodStatus struct {
	Provider  string `json:"provider"`
	Method    string `json:"method"`
	PayType   string `json:"pay_type"`
	Channel   string `json:"channel"`
	Online    bool   `json:"online"`
	APIBase   string `json:"api_base"`
	SubmitURL string `json:"submit_url"`
}

type AdminTopology struct {
	Role                    string                `json:"role"`
	UserCount               int                   `json:"user_count"`
	ActiveSubscriptionCount int                   `json:"active_subscription_count"`
	TunnelCount             int                   `json:"tunnel_count"`
	TunnelCounts            map[string]int        `json:"tunnel_counts"`
	NodeCount               int                   `json:"node_count"`
	OnlineNodeCount         int                   `json:"online_node_count"`
	TodayTrafficBytes       int64                 `json:"today_traffic_bytes"`
	PaymentMethods          []PaymentMethodStatus `json:"payment_methods"`
	RecentOrders            []PaymentOrder        `json:"recent_orders"`
	RecentOperations        []AdminOperationLog   `json:"recent_operations"`
	Nodes                   []Node                `json:"nodes"`
	RoleFlow                []TopologyLink        `json:"role_flow"`
	GeneratedAt             time.Time             `json:"generated_at"`
}
