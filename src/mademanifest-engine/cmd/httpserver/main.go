// Package main is the mademanifest-engine HTTP service entry point.
//
// Phase 12 retired the file-based PoC CLI; this is now the only
// runtime surface for the engine.  The handler reads its canonical
// constants directly from pkg/canon (compiled in) and the
// ephemeris data path from SE_EPHE_PATH; no per-request canon JSON
// files are loaded.
//
// Boot-time gates (Phase 9):
//   * canon.SelfCheck()        – validates every compiled-in canon
//                                constant (GateOrder permutation,
//                                ChannelTable well-formedness, etc.).
//   * ephemeris.ValidateEphePath() – verifies the resolved Swiss
//                                Ephemeris directory exists.
// Either failure is fatal; the engine refuses to start.
//
// Environment:
//   PORT              HTTP listen port (default 8080)
//   SE_EPHE_PATH      Swiss Ephemeris data directory (canon-required)
//   TRINITY_DEV_CORS  Set to "1" to enable the same CORS posture as
//                     the --dev-cors flag (k8s-friendly knob; never
//                     set this in production).
//
// Flags:
//   --version, -v     Print pinned versions as JSON and exit.
//   --dev-cors        Enable wildcard CORS + OPTIONS preflight on
//                     every wired route.  Required for the browser
//                     test client at src/scripts/client.html; OFF
//                     by default and never to be enabled in
//                     production deployments (see the docstring on
//                     pkg/httpservice.withCORS for the threat
//                     model).
//
// CANON_DIRECTORY is no longer consulted: Phase 9 made the compiled
// canon authoritative, and Phase 12 removed the legacy JSON
// loaders.  The variable can still be set for backward compat with
// older deployment manifests; it has no effect.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"

	"mademanifest-engine/pkg/canon"
	"mademanifest-engine/pkg/ephemeris"
	"mademanifest-engine/pkg/httpservice"
)

func main() {
	versionFlag := flag.Bool("version", false, "print pinned versions as JSON and exit")
	flag.BoolVar(versionFlag, "v", false, "print pinned versions as JSON and exit")
	devCORSFlag := flag.Bool("dev-cors", false,
		"enable wildcard CORS + OPTIONS preflight (development only; do not enable in production)")
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

	// CORS opt-in: --dev-cors flag wins; TRINITY_DEV_CORS=1 is the
	// k8s-friendly env-var equivalent so deployment manifests can
	// flip the bit without rewriting container args.  Production
	// deployments must leave both unset.
	devCORS := *devCORSFlag || os.Getenv("TRINITY_DEV_CORS") == "1"
	if devCORS {
		log.Printf("WARNING: --dev-cors enabled; never run with this in production")
	}

	handler := httpservice.New()
	handler.DevCORS = devCORS

	mux := http.NewServeMux()
	handler.Register(mux)

	addr := ":" + port
	log.Printf("HTTP service listening on %s (engine_version=%s canon_version=%s ephe_path=%s dev_cors=%v)",
		addr, canon.EngineVersion, canon.CanonVersion, ephemeris.ResolvedEphePath(), devCORS)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("listen and serve: %v", err)
	}
}
