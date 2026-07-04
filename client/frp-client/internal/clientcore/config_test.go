package clientcore

import (
	"strings"
	"testing"
)

func TestRenderFRPCConfigForAllTunnelTypes(t *testing.T) {
	got, err := RenderFRPCConfig(ServerConfig{ServerAddr: "frp.example.com", ServerPort: 7000, Token: "secret", Tunnels: []Tunnel{
		{ID: 2, Name: "web", Type: "http", LocalHost: "127.0.0.1", LocalPort: 8080, Domain: "app.user.com", Status: "pending_domain_check"},
		{ID: 1, Name: "ssh", Type: "tcp", LocalHost: "127.0.0.1", LocalPort: 22, RemotePort: 20000, Status: "active"},
		{ID: 3, Name: "secure web", Type: "https", LocalHost: "127.0.0.1", LocalPort: 8443, Domain: "secure.user.com", Status: "pending_certificate"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	checks := []string{
		`serverAddr = "frp.example.com"`,
		`serverPort = 7000`,
		`auth.token = "secret"`,
		`name = "ssh-1"`,
		`type = "tcp"`,
		`remotePort = 20000`,
		`name = "web-2"`,
		`customDomains = ["app.user.com"]`,
		`name = "secure-web-3"`,
		`customDomains = ["secure.user.com"]`,
	}
	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in config:\n%s", want, got)
		}
	}
}

func TestRenderFRPCConfigRejectsInvalidTunnel(t *testing.T) {
	_, err := RenderFRPCConfig(ServerConfig{ServerAddr: "frp.example.com", ServerPort: 7000, Token: "secret", Tunnels: []Tunnel{{ID: 1, Name: "bad", Type: "tcp", LocalHost: "127.0.0.1", LocalPort: 22}}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRenderFRPCConfigIncludesEffectiveBandwidthLimit(t *testing.T) {
	got, err := RenderFRPCConfig(ServerConfig{ServerAddr: "frp.example.com", ServerPort: 7000, Token: "secret", BandwidthLimitKbps: 512, Tunnels: []Tunnel{
		{ID: 1, Name: "tcp", Type: "tcp", LocalHost: "127.0.0.1", LocalPort: 22, RemotePort: 20000, Status: "active"},
		{ID: 2, Name: "http", Type: "http", LocalHost: "127.0.0.1", LocalPort: 8080, Domain: "app.user.com", EffectiveBandwidthLimitKbps: 256, Status: "active"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `transport.bandwidthLimit = "64KB"`) {
		t.Fatalf("missing inherited bandwidth limit:\n%s", got)
	}
	if !strings.Contains(got, `transport.bandwidthLimit = "32KB"`) {
		t.Fatalf("missing tunnel bandwidth override:\n%s", got)
	}
}
