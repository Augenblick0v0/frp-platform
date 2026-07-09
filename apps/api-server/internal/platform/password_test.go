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
