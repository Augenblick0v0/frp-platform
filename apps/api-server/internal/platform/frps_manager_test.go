package platform

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFRPSManagerReadsConfigAndLogs(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "frps.toml")
	logPath := filepath.Join(dir, "frps.log")
	if err := os.WriteFile(config, []byte("bindPort = 7000\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte("line1\nline2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	m := &FRPSManager{ConfigPath: config, LogPath: logPath}
	cfg, err := m.Config()
	if err != nil || !strings.Contains(cfg, "bindPort") {
		t.Fatalf("cfg=%q err=%v", cfg, err)
	}
	logs, err := m.Logs(6)
	if err != nil || !strings.Contains(logs, "line2") {
		t.Fatalf("logs=%q err=%v", logs, err)
	}
	st := m.Status(context.Background())
	if !st.Healthy {
		t.Fatalf("expected healthy: %#v", st)
	}
}
