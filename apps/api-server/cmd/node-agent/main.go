package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"frp-platform/apps/api-server/internal/platform"
)

type nodeAgent struct {
	token      string
	automation *platform.Automation
	frps       *platform.FRPSManager
}

func main() {
	token := strings.TrimSpace(os.Getenv("NODE_AGENT_TOKEN"))
	tokenFile := getenv("NODE_AGENT_TOKEN_FILE", "/app/runtime/node-agent/agent_token")
	if token == "" {
		if data, err := os.ReadFile(tokenFile); err == nil {
			token = strings.TrimSpace(string(data))
		}
	}
	bindToken := strings.TrimSpace(os.Getenv("NODE_BIND_TOKEN"))
	if token == "" && bindToken == "" {
		log.Fatal("NODE_AGENT_TOKEN or NODE_BIND_TOKEN is required")
	}
	a := &nodeAgent{token: token, automation: platform.AutomationFromEnv(), frps: platform.FRPSManagerFromEnv()}
	if a.token == "" {
		control := strings.TrimRight(strings.TrimSpace(os.Getenv("CONTROL_PLANE_URL")), "/")
		if control == "" {
			log.Fatal("CONTROL_PLANE_URL is required when using NODE_BIND_TOKEN")
		}
		if err := a.bindOnce(control, bindToken); err != nil {
			log.Fatalf("initial node bind failed: %v", err)
		}
		if strings.TrimSpace(a.token) == "" {
			log.Fatal("initial node bind did not return NODE_AGENT_TOKEN")
		}
		if err := writePrivateFile(tokenFile, []byte(a.token+"\n")); err != nil {
			log.Fatalf("persist node agent token: %v", err)
		}
	}
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
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}
	log.Fatal(httpServer.ListenAndServe())
}

func writePrivateFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".node-token-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func (a *nodeAgent) bindLoop() {
	control := strings.TrimRight(strings.TrimSpace(os.Getenv("CONTROL_PLANE_URL")), "/")
	bindToken := strings.TrimSpace(os.Getenv("NODE_BIND_TOKEN"))
	if control == "" || bindToken == "" {
		return
	}
	for {
		if err := a.bindOnce(control, bindToken); err != nil {
			log.Printf("node bind failed: %v", err)
		}
		time.Sleep(60 * time.Second)
	}
}

func (a *nodeAgent) bindOnce(control, bindToken string) error {
	payload := platform.NodeBindRequest{
		BindToken:      bindToken,
		Name:           getenv("NODE_NAME", "edge-node"),
		AgentURL:       getenv("NODE_PUBLIC_AGENT_URL", ""),
		PublicURL:      getenv("NODE_PUBLIC_URL", ""),
		FRPEntryDomain: getenv("FRP_ENTRY_DOMAIN", ""),
		ServerAddr:     getenv("SERVER_ADDR", getenv("FRP_ENTRY_DOMAIN", "")),
		FRPServerPort:  atoiEnv("FRP_BIND_PORT", 7000),
		TCPPortStart:   atoiEnv("TCP_PORT_START", 20000),
		TCPPortEnd:     atoiEnv("TCP_PORT_END", 29999),
		UDPPortStart:   atoiEnv("UDP_PORT_START", 30000),
		UDPPortEnd:     atoiEnv("UDP_PORT_END", 39999),
	}
	b, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(control+"/api/nodes/bind", "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var out struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Node       platform.Node `json:"node"`
			AgentToken string        `json:"agent_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("control plane returned %s", resp.Status)
	}
	if !out.Success {
		return fmt.Errorf("control plane bind failed: %s", out.Message)
	}
	if a.token == "" && out.Data.AgentToken != "" {
		a.token = out.Data.AgentToken
		log.Printf("node bound to control plane as node %d", out.Data.Node.ID)
	}
	return nil
}

func atoiEnv(k string, def int) int {
	v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(k)))
	if err != nil || v <= 0 {
		return def
	}
	return v
}

func (a *nodeAgent) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if strings.TrimSpace(a.token) == "" || got != a.token {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid node agent token")
			return
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
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	out, err := a.automation.TestNginx(r.Context())
	if err != nil {
		writeError(w, 500, "NGINX_TEST_FAILED", err.Error()+"\n"+out)
		return
	}
	writeJSON(w, map[string]string{"output": out})
}

func (a *nodeAgent) reloadNginx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
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
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	res, err := a.frps.Restart(r.Context())
	if err != nil {
		writeError(w, 500, "FRPS_RESTART_FAILED", err.Error()+"\n"+res.Output)
		return
	}
	writeJSON(w, res)
}
func (a *nodeAgent) frpsReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	res, err := a.frps.Reload(r.Context())
	if err != nil {
		writeError(w, 500, "FRPS_RELOAD_FAILED", err.Error()+"\n"+res.Output)
		return
	}
	writeJSON(w, res)
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		writeError(w, 400, "BAD_JSON", err.Error())
		return false
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		writeError(w, 400, "BAD_JSON", "request body contains trailing data")
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
