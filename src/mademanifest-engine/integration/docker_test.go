//go:build integration_docker

// Build-tagged smoke test for the Docker harness.  Requires a working
// local Docker daemon.  Enable with:
//   go test -tags integration_docker ./integration/...

package integration

import (
	"net/http"
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

func TestDockerHarnessServesVersion(t *testing.T) {
	srv := StartDockerContainer(t, DockerOptions{})
	t.Cleanup(srv.Shutdown)

	AssertVersionEndpointMatchesCanon(t, srv.BaseURL)
}

func TestDockerHarnessTrinityRejectionMatrix(t *testing.T) {
	srv := StartDockerContainer(t, DockerOptions{})
	t.Cleanup(srv.Shutdown)

	AssertTrinityRejectionMatrix(t, srv.BaseURL)
}

// TestDockerHarnessSchiedamAstrologyMatchesOracle is the Phase 4
// regression sentinel running against the production Docker image.
func TestDockerHarnessSchiedamAstrologyMatchesOracle(t *testing.T) {
	srv := StartDockerContainer(t, DockerOptions{})
	t.Cleanup(srv.Shutdown)

	AssertSchiedamAstrologyMatchesOracle(t, srv.BaseURL)
}

// TestDockerHarnessSchiedamDesignTimeMatchesOracle is the Phase 5
// regression sentinel running against the production Docker image.
func TestDockerHarnessSchiedamDesignTimeMatchesOracle(t *testing.T) {
	srv := StartDockerContainer(t, DockerOptions{})
	t.Cleanup(srv.Shutdown)

	AssertSchiedamDesignTimeMatchesOracle(t, srv.BaseURL)
}
