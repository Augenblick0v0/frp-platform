package platform

import (
	"context"
	"testing"
	"time"
)

func TestCertificateRenewerRenewsExpiringCert(t *testing.T) {
	store := NewStore()
	expires := time.Now().AddDate(0, 0, 1)
	_, err := store.SaveCertificate(CertificateRecord{Domain: "renew.example.com", Status: "issued", ExpiresAt: &expires})
	if err != nil {
		t.Fatal(err)
	}
	renewer := &CertificateRenewer{store: store, automation: &Automation{WebrootDir: "/tmp/acme", LetsEncryptDir: "/tmp/letsencrypt", CertbotBin: "certbot", DryRun: true}, Email: "admin@example.com", BeforeDays: 30}
	res, err := renewer.RenewDue(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Renewed != 1 || len(res.Records) != 1 || res.Records[0].Status != "dry_run" {
		t.Fatalf("unexpected result %#v", res)
	}
}

func TestCertificateRenewerSkipsFreshCert(t *testing.T) {
	store := NewStore()
	expires := time.Now().AddDate(0, 0, 90)
	_, err := store.SaveCertificate(CertificateRecord{Domain: "fresh.example.com", Status: "issued", ExpiresAt: &expires})
	if err != nil {
		t.Fatal(err)
	}
	renewer := &CertificateRenewer{store: store, automation: &Automation{WebrootDir: "/tmp/acme", LetsEncryptDir: "/tmp/letsencrypt", CertbotBin: "certbot", DryRun: true}, Email: "admin@example.com", BeforeDays: 30}
	res, err := renewer.RenewDue(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Renewed != 0 || res.Skipped != 1 {
		t.Fatalf("unexpected result %#v", res)
	}
}
