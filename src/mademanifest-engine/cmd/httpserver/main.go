package main

import (
	"log"
	"net/http"
	"os"

	"mademanifest-engine/pkg/engine"
	"mademanifest-engine/pkg/httpservice"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	canonDir := os.Getenv("CANON_DIRECTORY")
	if canonDir == "" {
		canonDir = "/app/canon"
	}

	canonPaths, err := engine.ResolveCanonPaths(canonDir, "", "", "")
	if err != nil {
		log.Fatalf("resolve canon paths: %v", err)
	}

	mux := http.NewServeMux()
	httpservice.New(canonPaths).Register(mux)

	addr := ":" + port
	log.Printf("HTTP service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("listen and serve: %v", err)
	}
}
