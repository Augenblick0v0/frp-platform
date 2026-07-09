package main

import (
	"testing"

	"frp-platform/apps/api-server/internal/platform"
)

func TestRequireDatabaseURLInProduction(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEFAULTS", "false")
	t.Setenv("DATABASE_URL", "")
	if err := platform.RequireDatabaseURL(); err == nil {
		t.Fatal("expected DATABASE_URL to be required in production")
	}
}

func TestRequireDatabaseURLAllowsExplicitDevMode(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEFAULTS", "true")
	t.Setenv("DATABASE_URL", "")
	if err := platform.RequireDatabaseURL(); err != nil {
		t.Fatalf("expected dev mode to allow memory store, got %v", err)
	}
}
