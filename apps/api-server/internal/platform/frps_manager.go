package platform

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FRPSManager struct {
	NodeAgent  *NodeAgentClient
	ConfigPath string
	LogPath    string
	StatusCmd  string
	RestartCmd string
	ReloadCmd  string
}

type FRPSStatus struct {
	ConfigPath string `json:"config_path"`
	LogPath    string `json:"log_path"`
	StatusCmd  string `json:"status_cmd,omitempty"`
	Output     string `json:"output"`
	Healthy    bool   `json:"healthy"`
}

type FRPSCommandResult struct {
	Command string `json:"command"`
	Output  string `json:"output"`
	OK      bool   `json:"ok"`
}

func FRPSManagerFromEnv() *FRPSManager {
	return &FRPSManager{
		NodeAgent:  NodeAgentClientFromEnv(),
		ConfigPath: getenv("FRPS_CONFIG_PATH", "/app/runtime/frps/frps.toml"),
		LogPath:    getenv("FRPS_LOG_PATH", "/app/runtime/logs/frps/frps.log"),
		StatusCmd:  os.Getenv("FRPS_STATUS_CMD"),
		RestartCmd: os.Getenv("FRPS_RESTART_CMD"),
		ReloadCmd:  os.Getenv("FRPS_RELOAD_CMD"),
	}
}

func (m *FRPSManager) Status(ctx context.Context) FRPSStatus {
	if m.NodeAgent.enabled() {
		st, err := m.NodeAgent.FRPSStatus(ctx)
		if err == nil {
			return st
		}
		return FRPSStatus{Healthy: false, Output: err.Error()}
	}
	st := FRPSStatus{ConfigPath: m.ConfigPath, LogPath: m.LogPath, StatusCmd: m.StatusCmd}
	if strings.TrimSpace(m.StatusCmd) == "" {
		if _, err := os.Stat(m.ConfigPath); err == nil {
			st.Healthy = true
			st.Output = "status command not configured; config file exists"
		} else {
			st.Output = err.Error()
		}
		return st
	}
	out, err := runShell(ctx, m.StatusCmd)
	st.Output = out
	st.Healthy = err == nil
	if err != nil && st.Output == "" {
		st.Output = err.Error()
	}
	return st
}

func (m *FRPSManager) Config() (string, error) {
	if m.NodeAgent.enabled() {
		out, err := m.NodeAgent.FRPSConfig(context.Background())
		if err != nil {
			return "", err
		}
		if s, ok := out["config"].(string); ok {
			return s, nil
		}
		return "", nil
	}
	return readTextFile(m.ConfigPath, 256*1024)
}

func (m *FRPSManager) Logs(limit int64) (string, error) {
	if m.NodeAgent.enabled() {
		out, err := m.NodeAgent.FRPSLogs(context.Background())
		if err != nil {
			return "", err
		}
		if s, ok := out["logs"].(string); ok {
			return s, nil
		}
		return "", nil
	}
	if limit <= 0 {
		limit = 32768
	}
	return readTextFile(m.LogPath, limit)
}

func (m *FRPSManager) Restart(ctx context.Context) (FRPSCommandResult, error) {
	if m.NodeAgent.enabled() {
		return m.NodeAgent.FRPSRestart(ctx)
	}
	return m.run(ctx, m.RestartCmd)
}

func (m *FRPSManager) Reload(ctx context.Context) (FRPSCommandResult, error) {
	if m.NodeAgent.enabled() {
		return m.NodeAgent.FRPSReload(ctx)
	}
	return m.run(ctx, m.ReloadCmd)
}

func (m *FRPSManager) run(ctx context.Context, command string) (FRPSCommandResult, error) {
	res := FRPSCommandResult{Command: command}
	if strings.TrimSpace(command) == "" {
		res.Output = "command not configured"
		res.OK = true
		return res, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, err := runShell(ctx, command)
	res.Output = out
	res.OK = err == nil
	return res, err
}

func readTextFile(path string, limit int64) (string, error) {
	path = filepath.Clean(path)
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if limit > 0 && int64(len(b)) > limit {
		b = b[int64(len(b))-limit:]
	}
	return string(b), nil
}
