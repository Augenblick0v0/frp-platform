package main

import (
	"frp-platform/apps/api-server/internal/platform"
	"log"
	"net/http"
	"os"
)

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	srv := platform.NewServer(platform.NewStore())
	log.Printf("frp-platform api-server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, srv.Handler()))
}
