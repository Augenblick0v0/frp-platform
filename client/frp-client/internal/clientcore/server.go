package clientcore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	mux.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		logs, err := s.manager.Logs(32768)
		if err != nil {
			writeError(w, 500, err)
			return
		}
		writeJSON(w, map[string]any{"logs": logs})
	})
	mux.HandleFunc("/api/config/render", s.renderConfig)
	mux.HandleFunc("/api/config/sync", s.syncConfig)
	mux.HandleFunc("/api/speed-tests/run", s.runSpeedTest)
	mux.HandleFunc("/api/frpc/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		if err := s.manager.Start(); err != nil {
			writeError(w, 400, err)
			return
		}
		writeJSON(w, s.manager.Status())
	})
	mux.HandleFunc("/api/traffic/report", s.reportTraffic)
	mux.HandleFunc("/api/frpc/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		if err := s.manager.Stop(); err != nil {
			writeError(w, 500, err)
			return
		}
		writeJSON(w, s.manager.Status())
	})
	mux.Handle("/", http.FileServer(http.Dir(s.webDir)))
	return mux
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
		APIBase string `json:"api_base"`
		Token   string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, 400, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	text, err := s.manager.SyncFromServer(ctx, in.APIBase, in.Token)
	if err != nil {
		writeError(w, 400, err)
		return
	}
	writeJSON(w, map[string]any{"config": text, "status": s.manager.Status()})
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
