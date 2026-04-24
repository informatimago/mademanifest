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
	"os"
	"path/filepath"
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

func TestLocalHarnessPostsGoldenFixture(t *testing.T) {
	srv := StartLocalServer(t, LocalServerOptions{})
	t.Cleanup(srv.Shutdown)

	goldenPath := filepath.Join(RepoRoot(t), "src", "golden", "GOLDEN_TEST_CASE_V1.json")
	body, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden fixture: %v", err)
	}

	status, raw, err := PostManifest(srv.BaseURL, body, nil)
	if err != nil {
		t.Fatalf("POST /manifest: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("POST /manifest status = %d; body = %s", status, raw)
	}
	if len(raw) == 0 {
		t.Fatal("POST /manifest returned empty body")
	}
}
