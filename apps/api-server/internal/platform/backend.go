package platform

type Backend interface {
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
}
