package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type NodeAgentClient struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

type CertificateInspectResult struct {
	Domain    string     `json:"domain"`
	CertPath  string     `json:"cert_path"`
	KeyPath   string     `json:"key_path"`
	IssuedAt  *time.Time `json:"issued_at,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func NodeAgentClientFromEnv() *NodeAgentClient {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("NODE_AGENT_URL")), "/")
	if base == "" {
		return nil
	}
	return &NodeAgentClient{BaseURL: base, Token: os.Getenv("NODE_AGENT_TOKEN"), Client: &http.Client{Timeout: 2 * time.Minute}}
}

func (c *NodeAgentClient) enabled() bool { return c != nil && strings.TrimSpace(c.BaseURL) != "" }

func (c *NodeAgentClient) do(ctx context.Context, method, path string, in any, out any) error {
	if !c.enabled() {
		return fmt.Errorf("node agent not configured")
	}
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	hc := c.Client
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("node agent %s %s failed: %s: %s", method, path, resp.Status, string(b))
	}
	if out != nil {
		if err := json.Unmarshal(b, out); err != nil {
			return err
		}
	}
	return nil
}

func (c *NodeAgentClient) WriteHTTPSConfig(ctx context.Context, domain string) (NginxConfigResult, error) {
	var out NginxConfigResult
	err := c.do(ctx, http.MethodPost, "/api/nginx/https-config", map[string]string{"domain": domain}, &out)
	return out, err
}

func (c *NodeAgentClient) RequestCertificate(ctx context.Context, domain, email string) (CertificateResult, error) {
	var out CertificateResult
	err := c.do(ctx, http.MethodPost, "/api/certificates/request", map[string]string{"domain": domain, "email": email}, &out)
	return out, err
}

func (c *NodeAgentClient) InspectCertificate(ctx context.Context, domain string) (CertificateInspectResult, error) {
	var out CertificateInspectResult
	path := "/api/certificates/inspect?domain=" + url.QueryEscape(domain)
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

func (c *NodeAgentClient) TestNginx(ctx context.Context) (string, error) {
	var out map[string]string
	err := c.do(ctx, http.MethodPost, "/api/nginx/test", nil, &out)
	return out["output"], err
}

func (c *NodeAgentClient) ReloadNginx(ctx context.Context) (string, error) {
	var out map[string]string
	err := c.do(ctx, http.MethodPost, "/api/nginx/reload", nil, &out)
	return out["output"], err
}

func (c *NodeAgentClient) FRPSStatus(ctx context.Context) (FRPSStatus, error) {
	var out FRPSStatus
	err := c.do(ctx, http.MethodGet, "/api/frps/status", nil, &out)
	return out, err
}

func (c *NodeAgentClient) FRPSConfig(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	err := c.do(ctx, http.MethodGet, "/api/frps/config", nil, &out)
	return out, err
}

func (c *NodeAgentClient) FRPSLogs(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	err := c.do(ctx, http.MethodGet, "/api/frps/logs", nil, &out)
	return out, err
}

func (c *NodeAgentClient) FRPSRestart(ctx context.Context) (FRPSCommandResult, error) {
	var out FRPSCommandResult
	err := c.do(ctx, http.MethodPost, "/api/frps/restart", nil, &out)
	return out, err
}

func (c *NodeAgentClient) FRPSReload(ctx context.Context) (FRPSCommandResult, error) {
	var out FRPSCommandResult
	err := c.do(ctx, http.MethodPost, "/api/frps/reload", nil, &out)
	return out, err
}
