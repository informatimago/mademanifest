package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"mademanifest-engine/pkg/canon"
	"mademanifest-engine/pkg/engine"
	"mademanifest-engine/pkg/httpservice"
)

func main() {
	versionFlag := flag.Bool("version", false, "print pinned versions as JSON and exit")
	flag.BoolVar(versionFlag, "v", false, "print pinned versions as JSON and exit")
	flag.Parse()

	if *versionFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(canon.Versions()); err != nil {
			log.Fatalf("encode versions: %v", err)
		}
		return
	}

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
	log.Printf("HTTP service listening on %s (engine_version=%s canon_version=%s)",
		addr, canon.EngineVersion, canon.CanonVersion)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("listen and serve: %v", err)
	}
}
