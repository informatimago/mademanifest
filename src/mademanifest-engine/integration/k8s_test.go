//go:build integration_k8s

// Build-tagged smoke test for the Kubernetes harness.  Requires kind,
// kubectl, and a working local Docker daemon.  Enable with:
//   go test -tags integration_k8s ./integration/...
//
// All Test* functions in this file share a single kind cluster and
// port-forward managed by TestMain.  That makes the startup cost
// (~45s on a cold run) a one-time charge rather than per-test.

package integration

import (
	"fmt"
	"net/http"
	"os"
	"testing"
)

// sharedK8s is the handle created once by TestMain and shared across
// every Test* in this file.  Individual tests read it and must not
// call Shutdown.
var sharedK8s ServerHandle

func TestMain(m *testing.M) {
	handle, err := StartKindCluster(K8sOptions{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "kubernetes harness setup failed: %v\n", err)
		os.Exit(2)
	}
	sharedK8s = handle
	code := m.Run()
	sharedK8s.Shutdown()
	os.Exit(code)
}

func TestK8sHarnessServesHealthz(t *testing.T) {
	status, raw, err := GetJSON(sharedK8s.BaseURL, "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("GET /healthz status = %d; body = %s", status, raw)
	}
}

func TestK8sHarnessServesVersion(t *testing.T) {
	AssertVersionEndpointMatchesCanon(t, sharedK8s.BaseURL)
}

func TestK8sHarnessTrinityRejectionMatrix(t *testing.T) {
	AssertTrinityRejectionMatrix(t, sharedK8s.BaseURL)
}

// TestK8sHarnessSchiedamAstrologyMatchesOracle is the Phase 4
// regression sentinel running against the kind-deployed service.
func TestK8sHarnessSchiedamAstrologyMatchesOracle(t *testing.T) {
	AssertSchiedamAstrologyMatchesOracle(t, sharedK8s.BaseURL)
}

// TestK8sHarnessSchiedamDesignTimeMatchesOracle is the Phase 5
// regression sentinel running against the kind-deployed service.
func TestK8sHarnessSchiedamDesignTimeMatchesOracle(t *testing.T) {
	AssertSchiedamDesignTimeMatchesOracle(t, sharedK8s.BaseURL)
}

// TestK8sHarnessSchiedamActivationsMatchOracle is the Phase 6
// regression sentinel running against the kind-deployed service.
func TestK8sHarnessSchiedamActivationsMatchOracle(t *testing.T) {
	AssertSchiedamActivationsMatchOracle(t, sharedK8s.BaseURL)
}

// TestK8sHarnessSchiedamStructureMatchesOracle is the Phase 7
// regression sentinel running against the kind-deployed service.
func TestK8sHarnessSchiedamStructureMatchesOracle(t *testing.T) {
	AssertSchiedamStructureMatchesOracle(t, sharedK8s.BaseURL)
}

// TestK8sHarnessSchiedamGeneKeysMatchOracle is the Phase 8
// regression sentinel running against the kind-deployed service.
func TestK8sHarnessSchiedamGeneKeysMatchOracle(t *testing.T) {
	AssertSchiedamGeneKeysMatchOracle(t, sharedK8s.BaseURL)
}

// TestK8sHarnessEnvImmuneToSENodePolicy is the Phase 9 determinism
// sentinel running against the kind-deployed service.  The shared
// k8s harness injects SE_NODE_POLICY=true into the mademanifest
// deployment via the kustomize overlay; this test re-asserts every
// Phase 4-8 oracle through that deployment to make the env-immunity
// guarantee an explicit, named test rather than an implicit side
// effect of the per-section sentinels.
func TestK8sHarnessEnvImmuneToSENodePolicy(t *testing.T) {
	AssertEnvImmuneCanonicalSchiedam(t, sharedK8s.BaseURL)
}

// TestK8sHarnessHTTPContract is the Phase 10 contract sentinel
// running against the kind-deployed service.
func TestK8sHarnessHTTPContract(t *testing.T) {
	AssertHTTPContract(t, sharedK8s.BaseURL)
}

// TestK8sHarnessTrinityGoldenPack is the Phase 11 sentinel running
// against the kind-deployed service.
func TestK8sHarnessTrinityGoldenPack(t *testing.T) {
	AssertTrinityGoldenPack(t, sharedK8s.BaseURL)
}

// TestK8sHarnessHardenedManifestsApplied is the Phase 13 sentinel:
// the kind-deployed service must satisfy every hardening invariant
// (NetworkPolicy + HPA exist with canon targets; pod runs as
// non-root; container has readOnlyRootFilesystem and dropped
// capabilities; resource requests/limits set).
func TestK8sHarnessHardenedManifestsApplied(t *testing.T) {
	AssertHardenedDeploymentShape(t, "default")
}

// TestK8sHarnessNetworkPolicyEnforced is the Phase 13 enforcement
// sentinel.  Skipped unless TRINITY_NETWORK_POLICY_ENFORCED=1
// because the default kind cluster does not enforce NetworkPolicy.
func TestK8sHarnessNetworkPolicyEnforced(t *testing.T) {
	AssertNetworkPolicyEnforced(t, "default")
}

// TestK8sHarnessHPAScalesUnderLoad is the Phase 13 horizontal
// scaling sentinel.  Skipped unless TRINITY_LOAD_TEST=1 because
// metrics-server is not installed in the default kind cluster.
func TestK8sHarnessHPAScalesUnderLoad(t *testing.T) {
	AssertHPAScalesUnderLoad(t, "default", sharedK8s.BaseURL)
}
