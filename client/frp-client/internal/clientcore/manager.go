package clientcore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type Manager struct {
	mu         sync.Mutex
	workDir    string
	frpcPath   string
	configPath string
	logPath    string
	cmd        *exec.Cmd
	startedAt  *time.Time
}

type Status struct {
	Running    bool       `json:"running"`
	PID        int        `json:"pid,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	ConfigPath string     `json:"config_path"`
	LogPath    string     `json:"log_path"`
}

func NewManager(workDir, frpcPath string) (*Manager, error) {
	if workDir == "" {
		workDir = defaultWorkDir()
	}
	if frpcPath == "" {
		frpcPath = "frpc"
	}
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(workDir, "logs"), 0755); err != nil {
		return nil, err
	}
	return &Manager{workDir: workDir, frpcPath: frpcPath, configPath: filepath.Join(workDir, "frpc.toml"), logPath: filepath.Join(workDir, "logs", "frpc.log")}, nil
}

func defaultWorkDir() string {
	if v := os.Getenv("FRP_CLIENT_HOME"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".frp-client"
	}
	return filepath.Join(home, ".frp-client")
}

func (m *Manager) WriteConfig(cfg ServerConfig) (string, error) {
	text, err := RenderFRPCConfig(cfg)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := os.WriteFile(m.configPath, []byte(text), 0600); err != nil {
		return "", err
	}
	return text, nil
}

func (m *Manager) SyncFromServer(ctx context.Context, apiBase, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, trimSlash(apiBase)+"/api/client/tunnels", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}
	var envelope struct {
		Success bool         `json:"success"`
		Data    ServerConfig `json:"data"`
		Message string       `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return "", err
	}
	if !envelope.Success {
		return "", fmt.Errorf("server response failed: %s", envelope.Message)
	}
	return m.WriteConfig(envelope.Data)
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd != nil && m.cmd.Process != nil && m.cmd.ProcessState == nil {
		return nil
	}
	if _, err := os.Stat(m.configPath); err != nil {
		return fmt.Errorf("frpc config not found: %s", m.configPath)
	}
	logFile, err := os.OpenFile(m.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	cmd := exec.Command(m.frpcPath, "-c", m.configPath)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return err
	}
	now := time.Now()
	m.startedAt = &now
	m.cmd = cmd
	go func() {
		_ = cmd.Wait()
		_ = logFile.Close()
		m.mu.Lock()
		if m.cmd == cmd {
			m.cmd = nil
			m.startedAt = nil
		}
		m.mu.Unlock()
	}()
	return nil
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd == nil || m.cmd.Process == nil {
		return nil
	}
	err := m.cmd.Process.Kill()
	m.cmd = nil
	m.startedAt = nil
	return err
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	st := Status{ConfigPath: m.configPath, LogPath: m.logPath}
	if m.cmd != nil && m.cmd.Process != nil {
		st.Running = true
		st.PID = m.cmd.Process.Pid
		st.StartedAt = m.startedAt
	}
	return st
}

func (m *Manager) Logs(limit int64) (string, error) {
	if limit <= 0 {
		limit = 8192
	}
	b, err := os.ReadFile(m.logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if int64(len(b)) > limit {
		b = b[int64(len(b))-limit:]
	}
	return string(bytes.TrimLeft(b, "\x00")), nil
}
