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

// TestDockerHarnessSchiedamActivationsMatchOracle is the Phase 6
// regression sentinel running against the production Docker image.
func TestDockerHarnessSchiedamActivationsMatchOracle(t *testing.T) {
	srv := StartDockerContainer(t, DockerOptions{})
	t.Cleanup(srv.Shutdown)

	AssertSchiedamActivationsMatchOracle(t, srv.BaseURL)
}

// TestDockerHarnessSchiedamStructureMatchesOracle is the Phase 7
// regression sentinel running against the production Docker image.
func TestDockerHarnessSchiedamStructureMatchesOracle(t *testing.T) {
	srv := StartDockerContainer(t, DockerOptions{})
	t.Cleanup(srv.Shutdown)

	AssertSchiedamStructureMatchesOracle(t, srv.BaseURL)
}

// TestDockerHarnessSchiedamGeneKeysMatchOracle is the Phase 8
// regression sentinel running against the production Docker image.
func TestDockerHarnessSchiedamGeneKeysMatchOracle(t *testing.T) {
	srv := StartDockerContainer(t, DockerOptions{})
	t.Cleanup(srv.Shutdown)

	AssertSchiedamGeneKeysMatchOracle(t, srv.BaseURL)
}

// TestDockerHarnessEnvImmuneToSENodePolicy is the Phase 9
// determinism sentinel running against the production Docker image.
// The container is launched with -e SE_NODE_POLICY=true (the retired
// Phase 6 env shim) and must still produce bit-identical canonical
// output for every Phase 4-8 oracle.
func TestDockerHarnessEnvImmuneToSENodePolicy(t *testing.T) {
	srv := StartDockerContainer(t, DockerOptions{
		ExtraEnv: []string{"SE_NODE_POLICY=true"},
	})
	t.Cleanup(srv.Shutdown)

	AssertEnvImmuneCanonicalSchiedam(t, srv.BaseURL)
}
