package platform

import "testing"

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
