package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"frp-platform/apps/api-server/internal/platform"
)

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	if err := platform.ValidateRequiredSecrets(); err != nil {
		log.Fatalf("security configuration error: %v", err)
	}
	if err := platform.RequireDatabaseURL(); err != nil {
		log.Fatalf("storage configuration error: %v", err)
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
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}
	log.Fatal(httpServer.ListenAndServe())
}
