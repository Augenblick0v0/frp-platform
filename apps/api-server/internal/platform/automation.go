package platform

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

type Automation struct {
	NodeAgent      *NodeAgentClient
	NginxConfDir   string
	WebrootDir     string
	LetsEncryptDir string
	FRPVhostURL    string
	CertbotBin     string
	NginxTestCmd   string
	NginxReloadCmd string
	DryRun         bool
}

type CNAMECheckResult struct {
	Domain string   `json:"domain"`
	Target string   `json:"target"`
	CNAMEs []string `json:"cnames"`
	Valid  bool     `json:"valid"`
	Error  string   `json:"error,omitempty"`
}

type NginxConfigResult struct {
	Domain string `json:"domain"`
	Path   string `json:"path"`
	Config string `json:"config"`
}

type CertificateResult struct {
	Domain  string `json:"domain"`
	Command string `json:"command"`
	Output  string `json:"output"`
	DryRun  bool   `json:"dry_run"`
}

func AutomationFromEnv() *Automation {
	return &Automation{
		NodeAgent:      NodeAgentClientFromEnv(),
		NginxConfDir:   getenv("NGINX_CONF_DIR", "/app/runtime/nginx-conf.d"),
		WebrootDir:     getenv("ACME_WEBROOT", "/var/www/certbot"),
		LetsEncryptDir: getenv("LETSENCRYPT_DIR", "/etc/letsencrypt"),
		FRPVhostURL:    getenv("FRP_VHOST_UPSTREAM", "http://frps:8080"),
		CertbotBin:     getenv("CERTBOT_BIN", "certbot"),
		NginxTestCmd:   os.Getenv("NGINX_TEST_CMD"),
		NginxReloadCmd: os.Getenv("NGINX_RELOAD_CMD"),
		DryRun:         os.Getenv("CERTBOT_DRY_RUN") == "true",
	}
}

func (a *Automation) CheckCNAME(domain, target string) CNAMECheckResult {
	domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
	target = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(target)), ".")
	res := CNAMECheckResult{Domain: domain, Target: target}
	if domain == "" || target == "" {
		res.Error = "domain and target required"
		return res
	}
	cnames, err := net.LookupCNAME(domain)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	cname := strings.TrimSuffix(strings.ToLower(cnames), ".")
	res.CNAMEs = []string{cname}
	res.Valid = cname == target || strings.HasSuffix(cname, "."+target)
	return res
}

func (a *Automation) RenderHTTPSConfig(domain string) (string, error) {
	domain = sanitizeDomain(domain)
	if domain == "" {
		return "", fmt.Errorf("domain required")
	}
	data := map[string]string{
		"Domain":      domain,
		"Upstream":    a.FRPVhostURL,
		"CertPath":    path.Join(a.LetsEncryptDir, "live", domain, "fullchain.pem"),
		"KeyPath":     path.Join(a.LetsEncryptDir, "live", domain, "privkey.pem"),
		"ACMEWebroot": a.WebrootDir,
	}
	var buf bytes.Buffer
	if err := httpsTemplate.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (a *Automation) WriteHTTPSConfig(domain string) (NginxConfigResult, error) {
	if a.NodeAgent.enabled() {
		return a.NodeAgent.WriteHTTPSConfig(context.Background(), domain)
	}
	cfg, err := a.RenderHTTPSConfig(domain)
	if err != nil {
		return NginxConfigResult{}, err
	}
	if err := os.MkdirAll(a.NginxConfDir, 0755); err != nil {
		return NginxConfigResult{}, err
	}
	path := filepath.Join(a.NginxConfDir, sanitizeDomain(domain)+".https.conf")
	if err := os.WriteFile(path, []byte(cfg), 0644); err != nil {
		return NginxConfigResult{}, err
	}
	return NginxConfigResult{Domain: sanitizeDomain(domain), Path: path, Config: cfg}, nil
}

func (a *Automation) RequestCertificate(ctx context.Context, domain, email string) (CertificateResult, error) {
	if a.NodeAgent.enabled() {
		return a.NodeAgent.RequestCertificate(ctx, domain, email)
	}
	domain = sanitizeDomain(domain)
	if domain == "" || strings.TrimSpace(email) == "" {
		return CertificateResult{}, fmt.Errorf("domain and email required")
	}
	args := []string{"certonly", "--webroot", "-w", a.WebrootDir, "-d", domain, "--email", email, "--agree-tos", "--non-interactive"}
	if a.DryRun {
		args = append(args, "--dry-run")
	}
	cmdText := a.CertbotBin + " " + strings.Join(args, " ")
	if a.DryRun && os.Getenv("EXECUTE_DRY_RUN_CERTBOT") != "true" {
		return CertificateResult{Domain: domain, Command: cmdText, DryRun: true, Output: "dry-run command prepared"}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, a.CertbotBin, args...)
	out, err := cmd.CombinedOutput()
	res := CertificateResult{Domain: domain, Command: cmdText, Output: string(out), DryRun: a.DryRun}
	if err != nil {
		return res, err
	}
	return res, nil
}

func (a *Automation) TestNginx(ctx context.Context) (string, error) {
	if a.NodeAgent.enabled() {
		return a.NodeAgent.TestNginx(ctx)
	}
	return runShell(ctx, a.NginxTestCmd)
}
func (a *Automation) ReloadNginx(ctx context.Context) (string, error) {
	if a.NodeAgent.enabled() {
		return a.NodeAgent.ReloadNginx(ctx)
	}
	return runShell(ctx, a.NginxReloadCmd)
}

func runShell(ctx context.Context, command string) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "command not configured", nil
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "sh", "-c", command).CombinedOutput()
	return string(out), err
}

func sanitizeDomain(domain string) string {
	domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.' {
			return r
		}
		return -1
	}, domain)
}

var httpsTemplate = template.Must(template.New("https").Parse(`server {
    listen 443 ssl http2;
    server_name {{.Domain}};

    ssl_certificate {{.CertPath}};
    ssl_certificate_key {{.KeyPath}};

    location /.well-known/acme-challenge/ {
        root {{.ACMEWebroot}};
    }

    location / {
        proxy_pass {{.Upstream}};
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}
`))

func (a *Automation) CertificatePaths(domain string) (string, string) {
	domain = sanitizeDomain(domain)
	return filepath.Join(a.LetsEncryptDir, "live", domain, "fullchain.pem"), filepath.Join(a.LetsEncryptDir, "live", domain, "privkey.pem")
}

func (a *Automation) InspectCertificate(domain string) (*time.Time, *time.Time) {
	if a.NodeAgent.enabled() {
		res, err := a.NodeAgent.InspectCertificate(context.Background(), domain)
		if err == nil {
			return res.IssuedAt, res.ExpiresAt
		}
	}
	certPath, _ := a.CertificatePaths(domain)
	data, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, nil
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil
	}
	issued := cert.NotBefore
	expires := cert.NotAfter
	return &issued, &expires
}
