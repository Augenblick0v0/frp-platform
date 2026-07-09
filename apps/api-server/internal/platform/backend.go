package platform

type Backend interface {
	AdminLogin(email, password string) (string, AdminUser, error)
	AdminByToken(token string) (AdminUser, error)
	SendEmailCode(email, purpose string) string
	Register(email, code, password string) (User, error)
	Login(email, password string) (string, User, error)
	UserByToken(token string) (User, error)
	Plans() []Plan
	CreatePlan(plan Plan) (Plan, error)
	UpdatePlan(id int64, plan Plan) (Plan, error)
	CreatePaymentOrder(order PaymentOrder) (PaymentOrder, error)
	PaymentOrderByOutTradeNo(outTradeNo string) (PaymentOrder, error)
	MarkPaymentOrderPaid(outTradeNo, providerTradeNo string) (PaymentOrder, Subscription, error)
	Users() []User
	UpdateUser(id int64, status string, planID int64) (User, Subscription, error)
	RedeemCodes() []RedeemCode
	CreateRedeemCodes(planID int64, count int, prefix string) ([]RedeemCode, error)
	Redeem(userID int64, code string) (Subscription, error)
	Subscription(userID int64) (Subscription, error)
	CreateTunnel(userID int64, req Tunnel) (Tunnel, error)
	CreateSpeedTestTunnel(userID int64, req SpeedTestTunnelRequest) (SpeedTestTunnel, error)
	SpeedTestTunnel(userID int64, tunnelID int64) (SpeedTestTunnel, error)
	FinishSpeedTestTunnel(userID int64, tunnelID int64) error
	Tunnels(userID int64) []Tunnel
	AllTunnels() []Tunnel
	Settings() Settings
	UpdateSettings(in Settings) Settings
	Nodes() []Node
	Node(id int64) (Node, error)
	CreateNode(node Node) (Node, error)
	DeleteNode(id int64) error
	BindNode(req NodeBindRequest) (Node, error)
	UpdateNodeStatus(id int64, status string, lastError string) (Node, error)
	ReportTraffic(userID int64, reports []TrafficReport) (TrafficSummary, error)
	TrafficSummary(userID int64) (TrafficSummary, error)
	TotalTrafficToday() int64
	SaveCertificate(record CertificateRecord) (CertificateRecord, error)
	Certificates() []CertificateRecord
	RecordAdminOperation(log AdminOperationLog) error
	AdminOperationLogs(limit int) []AdminOperationLog
}
