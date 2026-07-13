package platform

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
)

func TestSQLStoreConcurrentTunnelLimit(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not configured")
	}
	t.Setenv("ALLOW_INSECURE_DEFAULTS", "true")
	store, err := NewSQLStore(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	email := fmt.Sprintf("concurrent-%s@example.com", randomTestSuffix(t))
	var userID int64
	if err := store.db.QueryRow(`INSERT INTO users (email,password_hash,status,email_verified_at) VALUES ($1,$2,'active',now()) RETURNING id`, email, mustHashPassword("integration-pass")).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	plan, err := store.CreatePlan(Plan{Name: "Concurrent Limit", DurationDays: 1, TrafficLimitBytes: 1 << 30, BandwidthKbps: 1000, MaxTunnels: 1, MaxTCPTunnels: 1, AllowTCP: true, Status: "active"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.UpdateUser(userID, "active", plan.ID); err != nil {
		t.Fatal(err)
	}

	var successes int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := store.CreateTunnel(userID, Tunnel{Name: fmt.Sprintf("tcp-%d", i), Type: "tcp", LocalHost: "127.0.0.1", LocalPort: 10000 + i})
			if err == nil {
				atomic.AddInt32(&successes, 1)
			}
		}(i)
	}
	wg.Wait()
	if successes != 1 {
		t.Fatalf("expected exactly one successful tunnel, got %d", successes)
	}
}

func randomTestSuffix(t *testing.T) string {
	t.Helper()
	token, err := randomToken("")
	if err != nil {
		t.Fatal(err)
	}
	return token[:12]
}
