package clientcore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type LocalServer struct {
	manager *Manager
	webDir  string
}

func NewLocalServer(manager *Manager, webDir string) *LocalServer {
	return &LocalServer{manager: manager, webDir: webDir}
}
func (s *LocalServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, map[string]any{"status": "ok"}) })
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, s.manager.Status()) })
	mux.HandleFunc("/api/local-token", s.localToken)
	mux.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		logs, err := s.manager.Logs(32768)
		if err != nil {
			writeError(w, 500, err)
			return
		}
		writeJSON(w, map[string]any{"logs": logs})
	})
	mux.HandleFunc("/api/config/render", s.requireLocalToken(s.renderConfig))
	mux.HandleFunc("/api/config/sync", s.requireLocalToken(s.syncConfig))
	mux.HandleFunc("/api/speed-tests/prepare", s.requireLocalToken(s.prepareSpeedTest))
	mux.HandleFunc("/api/speed-tests/cleanup", s.requireLocalToken(s.cleanupSpeedTest))
	mux.HandleFunc("/api/speed-tests/run", s.requireLocalToken(s.runSpeedTest))
	mux.HandleFunc("/api/frpc/restart", s.requireLocalToken(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		if err := s.manager.Restart(); err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, s.manager.Status())
	}))
	mux.HandleFunc("/api/frpc/start", s.requireLocalToken(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		if err := s.manager.Start(); err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, s.manager.Status())
	}))
	mux.HandleFunc("/api/traffic/report", s.requireLocalToken(s.reportTraffic))
	mux.HandleFunc("/api/frpc/stop", s.requireLocalToken(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		if err := s.manager.Stop(); err != nil {
			writeError(w, 500, err)
			return
		}
		writeJSON(w, s.manager.Status())
	}))
	mux.Handle("/", http.FileServer(http.Dir(s.webDir)))
	return localCORS(mux)
}

func (s *LocalServer) localToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(405)
		return
	}
	if !sameOriginLocalRequest(r) {
		writeError(w, http.StatusForbidden, fmt.Errorf("local token is only available to the local web ui"))
		return
	}
	token, err := s.manager.LocalAPIToken()
	if err != nil {
		writeError(w, 500, err)
		return
	}
	writeJSON(w, map[string]string{"token": token})
}

func (s *LocalServer) requireLocalToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next(w, r)
			return
		}
		want, err := s.manager.LocalAPIToken()
		if err != nil {
			writeError(w, 500, err)
			return
		}
		if r.Header.Get("X-Local-Token") != want {
			writeError(w, http.StatusUnauthorized, fmt.Errorf("invalid local api token"))
			return
		}
		next(w, r)
	}
}

func (s *LocalServer) prepareSpeedTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var in struct {
		Type string `json:"type"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	bench, err := s.manager.PrepareSpeedBenchmark(in.Type)
	if err != nil {
		writeError(w, 400, err)
		return
	}
	writeJSON(w, bench)
}

func (s *LocalServer) cleanupSpeedTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	s.manager.CloseSpeedBenchmark()
	writeJSON(w, map[string]any{"cleaned": true})
}

func syncAPIBase(apiBase string, speedTestID int64) string {
	if speedTestID <= 0 {
		return apiBase
	}
	sep := "?"
	if strings.Contains(apiBase, "?") {
		sep = "&"
	}
	return apiBase + sep + "speed_test_id=" + fmt.Sprint(speedTestID)
}

func (s *LocalServer) reportTraffic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var in struct {
		APIBase string          `json:"api_base"`
		Token   string          `json:"token"`
		Reports []TrafficReport `json:"reports"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, 400, err)
		return
	}
	body, _ := json.Marshal(map[string]any{"reports": in.Reports})
	req, err := http.NewRequest(http.MethodPost, trimSlash(in.APIBase)+"/api/client/traffic", bytes.NewReader(body))
	if err != nil {
		writeError(w, 400, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+in.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeError(w, 400, err)
		return
	}
	defer resp.Body.Close()
	var out any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		writeError(w, 500, err)
		return
	}
	if resp.StatusCode > 299 {
		w.WriteHeader(resp.StatusCode)
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (s *LocalServer) runSpeedTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var in SpeedTestRunRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, 400, err)
		return
	}
	timeout := in.DurationSeconds + 30
	if timeout < 45 {
		timeout = 45
	}
	if timeout > 120 {
		timeout = 120
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeout)*time.Second)
	defer cancel()
	result, err := s.manager.RunSpeedTest(ctx, in)
	if err != nil {
		writeError(w, 400, err)
		return
	}
	writeJSON(w, result)
}

func (s *LocalServer) renderConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var cfg ServerConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, 400, err)
		return
	}
	text, err := s.manager.WriteConfig(cfg)
	if err != nil {
		writeError(w, 400, err)
		return
	}
	writeJSON(w, map[string]any{"config": text, "status": s.manager.Status()})
}
func (s *LocalServer) syncConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var in struct {
		APIBase     string `json:"api_base"`
		Token       string `json:"token"`
		SpeedTestID int64  `json:"speed_test_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, 400, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	text, err := s.manager.SyncFromServer(ctx, syncAPIBase(in.APIBase, in.SpeedTestID), in.Token)
	if err != nil {
		writeError(w, 400, err)
		return
	}
	writeJSON(w, map[string]any{"config": text, "status": s.manager.Status()})
}

func localCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" || isAllowedLocalOrigin(origin) || origin == os.Getenv("USER_PORTAL_ORIGIN") {
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Local-Token")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func sameOriginLocalRequest(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return (u.Host == r.Host) && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" || u.Hostname() == "::1")
}

func isAllowedLocalOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
}
func writeError(w http.ResponseWriter, status int, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "message": fmt.Sprint(err)})
}
