package platform

import (
	"strings"
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("secret")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "secret" {
		t.Fatal("password hash equals plain password")
	}
	if !VerifyPassword(hash, "secret") {
		t.Fatal("expected password to verify")
	}
	if VerifyPassword(hash, "wrong") {
		t.Fatal("wrong password verified")
	}
}

func TestPlaintextPasswordFallbackDisabledByDefault(t *testing.T) {
	t.Setenv("ALLOW_LEGACY_PLAINTEXT_PASSWORDS", "")
	if VerifyPassword("secret", "secret") {
		t.Fatal("plaintext password fallback must be disabled by default")
	}
}

func TestNormalizeRegistrationInputRejectsEmptyPassword(t *testing.T) {
	_, err := NormalizeRegistrationInput("user@example.com", "")
	if err == nil || err.Error() != "email and password required" {
		t.Fatalf("expected email/password required, got %v", err)
	}
}

func TestNormalizeRegistrationInputNormalizesEmail(t *testing.T) {
	email, err := NormalizeRegistrationInput(" USER@Example.COM ", "pass")
	if err != nil {
		t.Fatal(err)
	}
	if email != "user@example.com" {
		t.Fatalf("unexpected normalized email %q", email)
	}
}

func TestValidateRequiredSecretsRejectsPlaceholders(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEFAULTS", "false")
	t.Setenv("ADMIN_PASSWORD", "replace-with-strong-admin-password")
	t.Setenv("FRP_TOKEN", "frp-token-with-at-least-32-randomish-chars")
	if err := ValidateRequiredSecrets(); err == nil {
		t.Fatal("expected weak admin password to be rejected")
	}
}

func TestValidateRequiredSecretsAcceptsStrongValues(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEFAULTS", "false")
	t.Setenv("ADMIN_PASSWORD", "A9f3Jk8Lm2Np6Qr4Tu7Vx")
	t.Setenv("FRP_TOKEN", "Zx7Yp2Lm9Qw4Er8Ty1Ui5Op3As6Df0Gh")
	if err := ValidateRequiredSecrets(); err != nil {
		t.Fatalf("expected strong values to pass, got %v", err)
	}
}

func TestValidateRequiredSecretsRejectsSMTPTLSSkipVerify(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEFAULTS", "false")
	t.Setenv("ADMIN_PASSWORD", "A7xQ9mL2vN8kR4tZ6pW3")
	t.Setenv("FRP_TOKEN", "Zx7Yp2Lm9Qw4Er8Ty1Ui5Op3As6Df0Gh")
	t.Setenv("SMTP_SKIP_VERIFY", "true")
	if err := ValidateRequiredSecrets(); err == nil || !strings.Contains(err.Error(), "SMTP_SKIP_VERIFY") {
		t.Fatalf("expected SMTP_SKIP_VERIFY rejection, got %v", err)
	}
}
