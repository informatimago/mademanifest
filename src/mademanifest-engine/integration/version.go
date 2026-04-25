package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"mademanifest-engine/pkg/canon"
)

// AssertVersionEndpointMatchesCanon calls GET baseURL/version and
// fails the test unless the response is 200 with a Content-Type of
// application/json and a JSON body whose canonical fields equal
// canon.Versions().  Phase 9 added the diagnostic
// "ephe_path_resolved" field; this helper verifies the canon block
// matches and the diagnostic field is present and non-empty.
//
// Shared by the local, docker, and k8s harness smoke tests so every
// runtime exercises the same invariant: the build deployed into that
// runtime must expose exactly the compiled-in pinned versions plus
// a diagnosable resolved ephemeris path.
func AssertVersionEndpointMatchesCanon(t testing.TB, baseURL string) {
	t.Helper()
	status, raw, err := GetJSON(baseURL, "/version")
	if err != nil {
		t.Fatalf("GET /version: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("GET /version status = %d; body = %s", status, raw)
	}
	var got canon.VersionInfo
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode /version body: %v\nbody: %s", err, raw)
	}
	want := canon.Versions()
	if got != want {
		t.Fatalf("/version payload:\n got:  %s\n want: %+v",
			prettyJSON(raw), want)
	}

	// Phase 9: ephe_path_resolved must be present and non-empty.
	// The exact value depends on the runtime (subprocess sees the
	// repo-local checkout, the container sees an in-image path),
	// so we only assert presence and non-emptiness here.
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatalf("decode /version (generic): %v", err)
	}
	pathField, ok := generic["ephe_path_resolved"].(string)
	if !ok {
		t.Fatalf("/version: ephe_path_resolved missing or not a string\nbody: %s",
			prettyJSON(raw))
	}
	if pathField == "" {
		t.Fatalf("/version: ephe_path_resolved is empty\nbody: %s",
			prettyJSON(raw))
	}
}

func prettyJSON(raw []byte) string {
	var buf []byte
	decoded := map[string]any{}
	if err := json.Unmarshal(raw, &decoded); err == nil {
		if formatted, err := json.MarshalIndent(decoded, "", "  "); err == nil {
			buf = formatted
		}
	}
	if buf == nil {
		return fmt.Sprintf("%s", raw)
	}
	return string(buf)
}
