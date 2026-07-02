package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:18080", "local webui listen address")
	webDir := flag.String("web", "../../apps/client-webui", "client webui directory")
	flag.Parse()

	if _, err := os.Stat(*webDir); err != nil {
		log.Printf("webui directory not found: %s", *webDir)
	}
	http.Handle("/", http.FileServer(http.Dir(*webDir)))
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { _, _ = fmt.Fprint(w, "ok") })
	log.Printf("frp client local webui listening on http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
