package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"frp-platform/client/frp-client/internal/clientcore"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:18080", "local webui listen address")
	webDir := flag.String("web", "../../apps/client-webui", "client webui directory")
	workDir := flag.String("workdir", "", "client runtime directory")
	frpcPath := flag.String("frpc", "frpc", "frpc executable path")
	flag.Parse()

	if _, err := os.Stat(*webDir); err != nil {
		log.Printf("webui directory not found: %s", *webDir)
	}
	manager, err := clientcore.NewManager(*workDir, *frpcPath)
	if err != nil {
		log.Fatalf("create manager: %v", err)
	}
	server := clientcore.NewLocalServer(manager, *webDir)
	log.Printf("frp client local webui listening on http://%s", *addr)
	log.Printf("runtime config: %s", manager.Status().ConfigPath)
	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}
	log.Fatal(httpServer.ListenAndServe())
}
