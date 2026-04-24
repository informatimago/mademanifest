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
	"path/filepath"
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

func TestK8sHarnessPostsGoldenFixture(t *testing.T) {
	goldenPath := filepath.Join(RepoRoot(t), "src", "golden", "GOLDEN_TEST_CASE_V1.json")
	body, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden fixture: %v", err)
	}

	status, raw, err := PostManifest(sharedK8s.BaseURL, body, nil)
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
