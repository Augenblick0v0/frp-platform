package platform

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

type CertificateRenewer struct {
	store      Backend
	automation *Automation
	Email      string
	BeforeDays int
}

type RenewalResult struct {
	Checked int                 `json:"checked"`
	Renewed int                 `json:"renewed"`
	Skipped int                 `json:"skipped"`
	Failed  int                 `json:"failed"`
	Records []CertificateRecord `json:"records"`
}

func NewCertificateRenewer(store Backend, automation *Automation) *CertificateRenewer {
	days, _ := strconv.Atoi(getenv("CERT_RENEW_BEFORE_DAYS", "30"))
	if days <= 0 {
		days = 30
	}
	return &CertificateRenewer{store: store, automation: automation, Email: getenv("LETSENCRYPT_EMAIL", "admin@example.com"), BeforeDays: days}
}

func (r *CertificateRenewer) RenewDue(ctx context.Context, force bool) (RenewalResult, error) {
	var result RenewalResult
	deadline := time.Now().AddDate(0, 0, r.BeforeDays)
	for _, cert := range r.store.Certificates() {
		result.Checked++
		if !force {
			if cert.ExpiresAt == nil || cert.ExpiresAt.After(deadline) {
				result.Skipped++
				continue
			}
		}
		res, err := r.automation.RequestCertificate(ctx, cert.Domain, r.Email)
		status := "issued"
		errorMessage := ""
		if res.DryRun {
			status = "dry_run"
		}
		if err != nil {
			status = "failed"
			errorMessage = err.Error()
			result.Failed++
		} else {
			result.Renewed++
		}
		certPath, keyPath := r.automation.CertificatePaths(cert.Domain)
		issuedAt, expiresAt := r.automation.InspectCertificate(cert.Domain)
		if issuedAt == nil {
			issuedAt = cert.IssuedAt
		}
		if expiresAt == nil {
			expiresAt = cert.ExpiresAt
		}
		record, saveErr := r.store.SaveCertificate(CertificateRecord{Domain: cert.Domain, Status: status, IssuedAt: issuedAt, ExpiresAt: expiresAt, CertPath: certPath, KeyPath: keyPath, LastCommand: res.Command, LastOutput: res.Output, ErrorMessage: errorMessage})
		if saveErr != nil {
			result.Failed++
			return result, saveErr
		}
		result.Records = append(result.Records, record)
	}
	return result, nil
}

func StartCertificateRenewalScheduler(ctx context.Context, store Backend, automation *Automation) {
	intervalText := os.Getenv("CERT_RENEW_INTERVAL")
	if intervalText == "" || intervalText == "0" {
		return
	}
	interval, err := time.ParseDuration(intervalText)
	if err != nil {
		log.Printf("invalid CERT_RENEW_INTERVAL=%q: %v", intervalText, err)
		return
	}
	renewer := NewCertificateRenewer(store, automation)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				res, err := renewer.RenewDue(ctx, false)
				if err != nil {
					log.Printf("certificate renewal failed: %v", err)
					continue
				}
				if res.Renewed > 0 || res.Failed > 0 {
					log.Printf("certificate renewal result: checked=%d renewed=%d failed=%d skipped=%d", res.Checked, res.Renewed, res.Failed, res.Skipped)
				}
			}
		}
	}()
}

func (r RenewalResult) String() string {
	return fmt.Sprintf("checked=%d renewed=%d failed=%d skipped=%d", r.Checked, r.Renewed, r.Failed, r.Skipped)
}
