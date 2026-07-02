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
	Users() []User
	RedeemCodes() []RedeemCode
	CreateRedeemCodes(planID int64, count int, prefix string) ([]RedeemCode, error)
	Redeem(userID int64, code string) (Subscription, error)
	Subscription(userID int64) (Subscription, error)
	CreateTunnel(userID int64, req Tunnel) (Tunnel, error)
	Tunnels(userID int64) []Tunnel
	AllTunnels() []Tunnel
	Settings() Settings
	UpdateSettings(in Settings) Settings
	ReportTraffic(userID int64, reports []TrafficReport) (TrafficSummary, error)
	TrafficSummary(userID int64) (TrafficSummary, error)
	TotalTrafficToday() int64
	SaveCertificate(record CertificateRecord) (CertificateRecord, error)
	Certificates() []CertificateRecord
	RecordAdminOperation(log AdminOperationLog) error
	AdminOperationLogs(limit int) []AdminOperationLog
}
