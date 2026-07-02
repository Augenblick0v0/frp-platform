package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"frp-platform/apps/api-server/internal/platform"
)

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	var backend platform.Backend
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		sqlStore, err := platform.NewSQLStore(dsn)
		if err != nil {
			log.Fatalf("connect postgres: %v", err)
		}
		defer sqlStore.Close()
		backend = sqlStore
		log.Printf("storage backend: postgres")
	} else {
		backend = platform.NewStore()
		log.Printf("storage backend: in-memory")
	}

	automation := platform.AutomationFromEnv()
	platform.StartCertificateRenewalScheduler(context.Background(), backend, automation)
	srv := platform.NewServerWithServices(backend, platform.MailerFromEnv(), automation, platform.FRPSManagerFromEnv())
	log.Printf("frp-platform api-server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, srv.Handler()))
}
