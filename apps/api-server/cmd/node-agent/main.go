package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"frp-platform/apps/api-server/internal/platform"
)

type nodeAgent struct {
	token      string
	automation *platform.Automation
	frps       *platform.FRPSManager
}

func main() {
	a := &nodeAgent{token: os.Getenv("NODE_AGENT_TOKEN"), automation: platform.AutomationFromEnv(), frps: platform.FRPSManagerFromEnv()}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true, "service": "node-agent"})
	})
	mux.HandleFunc("/api/nginx/https-config", a.auth(a.writeHTTPSConfig))
	mux.HandleFunc("/api/certificates/request", a.auth(a.requestCertificate))
	mux.HandleFunc("/api/certificates/inspect", a.auth(a.inspectCertificate))
	mux.HandleFunc("/api/nginx/test", a.auth(a.testNginx))
	mux.HandleFunc("/api/nginx/reload", a.auth(a.reloadNginx))
	mux.HandleFunc("/api/frps/status", a.auth(a.frpsStatus))
	mux.HandleFunc("/api/frps/config", a.auth(a.frpsConfig))
	mux.HandleFunc("/api/frps/logs", a.auth(a.frpsLogs))
	mux.HandleFunc("/api/frps/restart", a.auth(a.frpsRestart))
	mux.HandleFunc("/api/frps/reload", a.auth(a.frpsReload))
	addr := getenv("NODE_AGENT_ADDR", ":8090")
	log.Printf("node-agent listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func (a *nodeAgent) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if a.token != "" {
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if got != a.token {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid node agent token")
				return
			}
		}
		next(w, r)
	}
}

func (a *nodeAgent) writeHTTPSConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var in struct {
		Domain string `json:"domain"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	res, err := a.automation.WriteHTTPSConfig(in.Domain)
	if err != nil {
		writeError(w, 500, "NGINX_CONFIG_FAILED", err.Error())
		return
	}
	writeJSON(w, res)
}

func (a *nodeAgent) requestCertificate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var in struct{ Domain, Email string }
	if !readJSON(w, r, &in) {
		return
	}
	res, err := a.automation.RequestCertificate(r.Context(), in.Domain, in.Email)
	if err != nil {
		writeError(w, 500, "CERTIFICATE_REQUEST_FAILED", err.Error()+"\n"+res.Output)
		return
	}
	writeJSON(w, res)
}

func (a *nodeAgent) inspectCertificate(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	certPath, keyPath := a.automation.CertificatePaths(domain)
	issued, expires := a.automation.InspectCertificate(domain)
	writeJSON(w, platform.CertificateInspectResult{Domain: domain, CertPath: certPath, KeyPath: keyPath, IssuedAt: issued, ExpiresAt: expires})
}

func (a *nodeAgent) testNginx(w http.ResponseWriter, r *http.Request) {
	out, err := a.automation.TestNginx(r.Context())
	if err != nil {
		writeError(w, 500, "NGINX_TEST_FAILED", err.Error()+"\n"+out)
		return
	}
	writeJSON(w, map[string]string{"output": out})
}

func (a *nodeAgent) reloadNginx(w http.ResponseWriter, r *http.Request) {
	out, err := a.automation.ReloadNginx(r.Context())
	if err != nil {
		writeError(w, 500, "NGINX_RELOAD_FAILED", err.Error()+"\n"+out)
		return
	}
	writeJSON(w, map[string]string{"output": out})
}

func (a *nodeAgent) frpsStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, a.frps.Status(r.Context()))
}
func (a *nodeAgent) frpsConfig(w http.ResponseWriter, r *http.Request) {
	text, err := a.frps.Config()
	if err != nil {
		writeError(w, 500, "FRPS_CONFIG_READ_FAILED", err.Error())
		return
	}
	writeJSON(w, map[string]any{"config": text, "path": a.frps.ConfigPath})
}
func (a *nodeAgent) frpsLogs(w http.ResponseWriter, r *http.Request) {
	text, err := a.frps.Logs(65536)
	if err != nil {
		writeError(w, 500, "FRPS_LOG_READ_FAILED", err.Error())
		return
	}
	writeJSON(w, map[string]any{"logs": text, "path": a.frps.LogPath})
}
func (a *nodeAgent) frpsRestart(w http.ResponseWriter, r *http.Request) {
	res, err := a.frps.Restart(r.Context())
	if err != nil {
		writeError(w, 500, "FRPS_RESTART_FAILED", err.Error()+"\n"+res.Output)
		return
	}
	writeJSON(w, res)
}
func (a *nodeAgent) frpsReload(w http.ResponseWriter, r *http.Request) {
	res, err := a.frps.Reload(r.Context())
	if err != nil {
		writeError(w, 500, "FRPS_RELOAD_FAILED", err.Error()+"\n"+res.Output)
		return
	}
	writeJSON(w, res)
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, 400, "BAD_JSON", err.Error())
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, code int, errCode, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": errCode, "message": msg})
}
func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
