//go:build integration_docker

// Build-tagged smoke test for the Docker harness.  Requires a working
// local Docker daemon.  Enable with:
//   go test -tags integration_docker ./integration/...

package integration

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestDockerHarnessBootsAndServesHealthz(t *testing.T) {
	srv := StartDockerContainer(t, DockerOptions{})
	t.Cleanup(srv.Shutdown)

	status, raw, err := GetJSON(srv.BaseURL, "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("GET /healthz status = %d; body = %s", status, raw)
	}
}

func TestDockerHarnessPostsGoldenFixture(t *testing.T) {
	srv := StartDockerContainer(t, DockerOptions{})
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
