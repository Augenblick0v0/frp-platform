package clientcore

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

type Tunnel struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	LocalHost     string   `json:"local_host"`
	LocalPort     int      `json:"local_port"`
	RemotePort    int      `json:"remote_port,omitempty"`
	Domain        string   `json:"domain,omitempty"`
	UseHTTPS      bool     `json:"use_https"`
	Status        string   `json:"status"`
	CustomDomains []string `json:"custom_domains,omitempty"`
}

type ServerConfig struct {
	ServerAddr string   `json:"server_addr"`
	ServerPort int      `json:"server_port"`
	Token      string   `json:"token"`
	Tunnels    []Tunnel `json:"tunnels"`
}

func RenderFRPCConfig(cfg ServerConfig) (string, error) {
	if strings.TrimSpace(cfg.ServerAddr) == "" {
		return "", fmt.Errorf("server_addr required")
	}
	if cfg.ServerPort <= 0 {
		return "", fmt.Errorf("server_port required")
	}
	if strings.TrimSpace(cfg.Token) == "" {
		return "", fmt.Errorf("token required")
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "serverAddr = %q\n", cfg.ServerAddr)
	fmt.Fprintf(&b, "serverPort = %d\n", cfg.ServerPort)
	fmt.Fprintf(&b, "auth.method = %q\n", "token")
	fmt.Fprintf(&b, "auth.token = %q\n\n", cfg.Token)

	tunnels := append([]Tunnel(nil), cfg.Tunnels...)
	sort.SliceStable(tunnels, func(i, j int) bool { return tunnels[i].ID < tunnels[j].ID })
	for _, t := range tunnels {
		if t.Status == "disabled" || t.Status == "deleted" {
			continue
		}
		if err := validateTunnel(t); err != nil {
			return "", fmt.Errorf("tunnel %s: %w", t.Name, err)
		}
		name := safeProxyName(t)
		fmt.Fprintf(&b, "[[proxies]]\n")
		fmt.Fprintf(&b, "name = %q\n", name)
		proxyType := strings.ToLower(t.Type)
		if proxyType == "https" {
			proxyType = "http"
		}
		fmt.Fprintf(&b, "type = %q\n", proxyType)
		fmt.Fprintf(&b, "localIP = %q\n", t.LocalHost)
		fmt.Fprintf(&b, "localPort = %d\n", t.LocalPort)
		switch strings.ToLower(t.Type) {
		case "tcp", "udp":
			fmt.Fprintf(&b, "remotePort = %d\n", t.RemotePort)
		case "http", "https":
			domains := t.CustomDomains
			if len(domains) == 0 && t.Domain != "" {
				domains = []string{t.Domain}
			}
			fmt.Fprintf(&b, "customDomains = [%s]\n", quoteList(domains))
		}
		fmt.Fprintf(&b, "\n")
	}
	return b.String(), nil
}

func validateTunnel(t Tunnel) error {
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("name required")
	}
	if strings.TrimSpace(t.LocalHost) == "" {
		return fmt.Errorf("local_host required")
	}
	if t.LocalPort <= 0 {
		return fmt.Errorf("local_port required")
	}
	switch strings.ToLower(t.Type) {
	case "tcp", "udp":
		if t.RemotePort <= 0 {
			return fmt.Errorf("remote_port required")
		}
	case "http", "https":
		if t.Domain == "" && len(t.CustomDomains) == 0 {
			return fmt.Errorf("domain required")
		}
	default:
		return fmt.Errorf("unsupported type %q", t.Type)
	}
	return nil
}

func safeProxyName(t Tunnel) string {
	base := strings.ToLower(strings.TrimSpace(t.Name))
	base = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, base)
	return fmt.Sprintf("%s-%d", base, t.ID)
}

func quoteList(items []string) string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, fmt.Sprintf("%q", item))
		}
	}
	return strings.Join(out, ", ")
}
