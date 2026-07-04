package clientcore

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
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

func TestLocalBenchmarkProbesReturnSpeedMetrics(t *testing.T) {
	for _, typ := range []string{"http", "tcp", "udp"} {
		t.Run(typ, func(t *testing.T) {
			bench, err := startBenchmarkService(typ)
			if err != nil {
				t.Fatal(err)
			}
			defer bench.Close()
			public := "127.0.0.1:" + strconv.Itoa(bench.port)
			if typ == "http" {
				public = "http://" + public
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			got, err := runProtocolProbe(ctx, typ, public, 32*1024, 32*1024)
			if err != nil {
				t.Fatal(err)
			}
			if got.BytesIn <= 0 || got.BytesOut <= 0 {
				t.Fatalf("expected traffic bytes, got %#v", got)
			}
			if got.DownloadAverageKbps <= 0 || got.UploadAverageKbps <= 0 || got.LatencyMs < 0 {
				t.Fatalf("expected speed metrics, got %#v", got)
			}
		})
	}
}
