package platform

import (
	"strings"
	"testing"
)

func TestRenderHTTPSConfig(t *testing.T) {
	a := &Automation{LetsEncryptDir: "/etc/letsencrypt", WebrootDir: "/var/www/certbot", FRPVhostURL: "http://frps:8080"}
	cfg, err := a.RenderHTTPSConfig("App.User.COM")
	if err != nil {
		t.Fatal(err)
	}
	checks := []string{"server_name app.user.com;", "ssl_certificate /etc/letsencrypt/live/app.user.com/fullchain.pem;", "proxy_pass http://frps:8080;", "proxy_set_header Host $host;"}
	for _, want := range checks {
		if !strings.Contains(cfg, want) {
			t.Fatalf("missing %q in\n%s", want, cfg)
		}
	}
}

func TestWriteHTTPSConfig(t *testing.T) {
	dir := t.TempDir()
	a := &Automation{NginxConfDir: dir, LetsEncryptDir: "/etc/letsencrypt", WebrootDir: "/var/www/certbot", FRPVhostURL: "http://frps:8080"}
	res, err := a.WriteHTTPSConfig("demo.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(res.Path, "demo.example.com.https.conf") {
		t.Fatalf("unexpected path %s", res.Path)
	}
}
