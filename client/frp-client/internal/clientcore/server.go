package clientcore

import (
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
