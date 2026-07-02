package platform

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	store Backend
	mux   *http.ServeMux
}

func NewServer(store Backend) *Server {
	s := &Server{store: store, mux: http.NewServeMux()}
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
	s.mux.HandleFunc("/api/user/redeem", s.auth(s.redeem))
	s.mux.HandleFunc("/api/user/purchase-info", s.auth(s.purchaseInfo))
	s.mux.HandleFunc("/api/tunnels", s.auth(s.tunnels))
	s.mux.HandleFunc("/api/client/heartbeat", s.auth(s.clientHeartbeat))
	s.mux.HandleFunc("/api/client/tunnels", s.auth(s.clientTunnels))
	s.mux.HandleFunc("/api/admin/dashboard", s.adminDashboard)
	s.mux.HandleFunc("/api/admin/plans", s.adminPlans)
	s.mux.HandleFunc("/api/admin/tunnels", s.adminTunnels)
	s.mux.HandleFunc("/api/admin/settings", s.adminSettings)
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
	ok(w, map[string]any{"expires_in": 600, "dev_code": code})
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
	ok(w, map[string]any{"title": "购买套餐", "description": "请通过购买链接获取兑换码", "button_text": "立即购买", "purchase_url": st.PurchaseURL})
}
func (s *Server) tunnels(w http.ResponseWriter, r *http.Request, u User) {
	switch r.Method {
	case http.MethodGet:
		ok(w, s.store.Tunnels(u.ID))
	case http.MethodPost:
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
func (s *Server) clientHeartbeat(w http.ResponseWriter, r *http.Request, u User) {
	ok(w, map[string]any{"status": "online", "server_time": time.Now().Format(time.RFC3339)})
}
func (s *Server) clientTunnels(w http.ResponseWriter, r *http.Request, u User) {
	st := s.store.Settings()
	ok(w, map[string]any{"server_addr": st.ServerAddr, "server_port": st.FRPServerPort, "token": "runtime-token-placeholder", "tunnels": s.store.Tunnels(u.ID)})
}
func (s *Server) adminDashboard(w http.ResponseWriter, r *http.Request) {
	all := s.store.AllTunnels()
	counts := map[string]int{}
	for _, t := range all {
		counts[t.Type]++
	}
	ok(w, map[string]any{"total_tunnels": len(all), "tunnel_counts": counts, "online_clients": 0, "today_traffic_bytes": 0})
}
func (s *Server) adminPlans(w http.ResponseWriter, r *http.Request)   { ok(w, s.store.Plans()) }
func (s *Server) adminTunnels(w http.ResponseWriter, r *http.Request) { ok(w, s.store.AllTunnels()) }
func (s *Server) adminSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPut {
		var in Settings
		if !decode(w, r, &in) {
			return
		}
		ok(w, s.store.UpdateSettings(in))
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
