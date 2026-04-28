package integration

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"mademanifest-engine/pkg/golden"
	"mademanifest-engine/pkg/trinity/output"
)

// AssertTrinityGoldenPack walks the golden pack at
// <repo>/src/golden/trinity, POSTs every input.json to baseURL, and
// compares the response against the matching expected.json using
// pkg/golden.CompareSuccess / CompareError.  The pack's category
// minimums (3 / 5 / 5 / 5 / 2 / 3) are enforced by
// golden.LoadFixtures up front, so a missing or below-minimum
// category fails the test before any HTTP request fires.
//
// Each fixture runs as a t.Run sub-test so a single drift surfaces
// with a precise category/name path.  A4 (RESOLVED, D23): error
// fixtures compare error_type only; per the Phase 11 plan, success
// fixtures compare the entire envelope minus the metadata block
// (which is asserted separately to equal output.CurrentMetadata(),
// since metadata depends on the build's EngineVersion).
func AssertTrinityGoldenPack(t *testing.T, baseURL string) {
	t.Helper()

	packRoot := filepath.Join(RepoRoot(t), "src", "golden", "trinity")
	fixtures, err := golden.LoadFixtures(packRoot)
	if err != nil {
		t.Fatalf("load golden pack at %s: %v", packRoot, err)
	}

	for _, f := range fixtures {
		f := f
		t.Run(f.RelativePath, func(t *testing.T) {
			input, err := f.LoadInput()
			if err != nil {
				t.Fatalf("read %s: %v", f.InputPath, err)
			}
			status, raw, err := PostManifest(baseURL, input, nil)
			if err != nil {
				t.Fatalf("POST /manifest: %v", err)
			}

			if golden.IsErrorCategory(f.Category) {
				assertGoldenErrorCase(t, f, status, raw)
				return
			}
			assertGoldenSuccessCase(t, f, status, raw)
		})
	}
}

// assertGoldenSuccessCase covers valid_baseline, valid_edge, and
// regression_sentinel.  Status must be 200; the body must decode as
// a SuccessEnvelope; metadata must equal CurrentMetadata; and the
// rest of the envelope must equal the frozen ExpectedSuccess.
func assertGoldenSuccessCase(t *testing.T, f golden.Fixture, status int, raw []byte) {
	t.Helper()
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", status, raw)
	}
	var got output.SuccessEnvelope
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode SuccessEnvelope: %v\nbody: %s", err, raw)
	}
	if got.Metadata != output.CurrentMetadata() {
		t.Errorf("metadata drift:\n got:  %+v\n want: %+v",
			got.Metadata, output.CurrentMetadata())
	}
	want, err := f.LoadExpectedSuccess()
	if err != nil {
		t.Fatalf("load expected: %v", err)
	}
	if err := golden.CompareSuccess(got, want); err != nil {
		t.Errorf("golden drift in %s: %v", f.RelativePath, err)
	}
}

// assertGoldenErrorCase covers invalid_input, incomplete_input,
// unsupported_input.  Status must match the canon mapping for the
// expected error_type; the body must decode as an ErrorEnvelope
// whose error_type matches the fixture and whose metadata is
// canonical.
func assertGoldenErrorCase(t *testing.T, f golden.Fixture, status int, raw []byte) {
	t.Helper()
	want, err := f.LoadExpectedError()
	if err != nil {
		t.Fatalf("load expected: %v", err)
	}
	wantStatus := output.StatusCodeForErrorType(want.Error.ErrorType)
	if status != wantStatus {
		t.Errorf("status = %d, want %d; body = %s",
			status, wantStatus, raw)
	}
	var got output.ErrorEnvelope
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode ErrorEnvelope: %v\nbody: %s", err, raw)
	}
	if err := golden.CompareError(got, want, output.CurrentMetadata()); err != nil {
		t.Errorf("golden drift in %s: %v", f.RelativePath, err)
	}
}
