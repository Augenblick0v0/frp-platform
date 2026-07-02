package clientcore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagerWriteConfigAndLogs(t *testing.T) {
	dir := t.TempDir()
	m, err := NewManager(dir, "frpc")
	if err != nil {
		t.Fatal(err)
	}
	text, err := m.WriteConfig(ServerConfig{ServerAddr: "frp.example.com", ServerPort: 7000, Token: "secret", Tunnels: []Tunnel{{ID: 1, Name: "ssh", Type: "tcp", LocalHost: "127.0.0.1", LocalPort: 22, RemotePort: 20000, Status: "active"}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, `remotePort = 20000`) {
		t.Fatal(text)
	}
	if _, err := os.Stat(filepath.Join(dir, "frpc.toml")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "logs", "frpc.log"), []byte("hello log"), 0644); err != nil {
		t.Fatal(err)
	}
	logs, err := m.Logs(1024)
	if err != nil {
		t.Fatal(err)
	}
	if logs != "hello log" {
		t.Fatalf("logs=%q", logs)
	}
}
