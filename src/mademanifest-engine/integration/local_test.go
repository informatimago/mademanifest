//go:build integration_local

// Build-tagged smoke test for the local-process harness.  This file is
// only compiled when -tags=integration_local is passed to go test, so
// the default "go test ./..." does not try to build the cgo binary or
// load ephemeris data.
//
// Prerequisites:
//   - libswe.so (or libswe.dylib) installed and discoverable via the
//     loader path (see src/Makefile).
//   - Ephemeris data present at <repo>/src/ephemeris/data/REQUIRED_EPHEMERIS_FILES.

package integration

import (
	"net/http"
	"testing"
)

func TestLocalHarnessBootsAndServesHealthz(t *testing.T) {
	srv := StartLocalServer(t, LocalServerOptions{})
	t.Cleanup(srv.Shutdown)

	status, raw, err := GetJSON(srv.BaseURL, "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("GET /healthz status = %d; body = %s", status, raw)
	}
}

func TestLocalHarnessServesVersion(t *testing.T) {
	srv := StartLocalServer(t, LocalServerOptions{})
	t.Cleanup(srv.Shutdown)

	AssertVersionEndpointMatchesCanon(t, srv.BaseURL)
}

// TestLocalHarnessTrinityRejectionMatrix exercises the Phase 2
// validator end-to-end through the real binary: every rejection
// category from trinity.org § Validation Rules plus the placeholder
// success path.  The POC golden fixture is no longer driven through
// /manifest – it now violates the Trinity input contract (unknown
// fields case_id, birth, engine_contract, expected) and is rejected
// as invalid_input.
func TestLocalHarnessTrinityRejectionMatrix(t *testing.T) {
	srv := StartLocalServer(t, LocalServerOptions{})
	t.Cleanup(srv.Shutdown)

	AssertTrinityRejectionMatrix(t, srv.BaseURL)
}

// TestLocalHarnessSchiedamAstrologyMatchesOracle is the Phase 4
// regression sentinel: the canonical Schiedam payload run through
// the live HTTP service must produce the frozen astrology oracle
// captured under src/golden/trinity/baseline/.
func TestLocalHarnessSchiedamAstrologyMatchesOracle(t *testing.T) {
	srv := StartLocalServer(t, LocalServerOptions{})
	t.Cleanup(srv.Shutdown)

	AssertSchiedamAstrologyMatchesOracle(t, srv.BaseURL)
}

// TestLocalHarnessSchiedamDesignTimeMatchesOracle is the Phase 5
// regression sentinel: the canonical Schiedam payload must produce
// the frozen human_design.system oracle (node_type + design_time_utc).
func TestLocalHarnessSchiedamDesignTimeMatchesOracle(t *testing.T) {
	srv := StartLocalServer(t, LocalServerOptions{})
	t.Cleanup(srv.Shutdown)

	AssertSchiedamDesignTimeMatchesOracle(t, srv.BaseURL)
}
