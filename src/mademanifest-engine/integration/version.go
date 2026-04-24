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
// application/json and a JSON body deep-equal to canon.Versions().
//
// Shared by the local, docker, and k8s harness smoke tests so every
// runtime exercises the same invariant: the build deployed into that
// runtime must expose exactly the compiled-in pinned versions.
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
