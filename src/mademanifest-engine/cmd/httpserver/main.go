package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"mademanifest-engine/pkg/canon"
	"mademanifest-engine/pkg/engine"
	"mademanifest-engine/pkg/ephemeris"
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

	// Phase 9 boot-time self-checks.  The engine refuses to start
	// when any of these fail; the alternative is silently serving
	// non-canonical results to clients, which violates the canon
	// determinism rules (trinity.org §"Determinism And Versioning").
	if err := canon.SelfCheck(); err != nil {
		log.Fatalf("canon self-check failed: %v", err)
	}
	if err := ephemeris.ValidateEphePath(); err != nil {
		log.Fatalf("ephemeris path validation failed: %v", err)
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
	log.Printf("HTTP service listening on %s (engine_version=%s canon_version=%s ephe_path=%s)",
		addr, canon.EngineVersion, canon.CanonVersion, ephemeris.ResolvedEphePath())
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("listen and serve: %v", err)
	}
}
