package golden

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mademanifest-engine/pkg/trinity/output"
)

// writePack materialises a synthetic fixture pack under root.  Each
// case in the input map is created as <root>/<category>/<name>/{input,expected}.json
// using the byte content provided.  Tests use this to drive
// LoadFixtures against pathological setups (missing categories,
// below-minimum counts, etc.) without touching the real golden tree.
func writePack(t *testing.T, root string, cases map[string]map[string][2][]byte) {
	t.Helper()
	for cat, fixtures := range cases {
		dir := filepath.Join(root, cat)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
		for name, payloads := range fixtures {
			caseDir := filepath.Join(dir, name)
			if err := os.MkdirAll(caseDir, 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", caseDir, err)
			}
			if err := os.WriteFile(filepath.Join(caseDir, "input.json"),
				payloads[0], 0o600); err != nil {
				t.Fatalf("write input: %v", err)
			}
			if err := os.WriteFile(filepath.Join(caseDir, "expected.json"),
				payloads[1], 0o600); err != nil {
				t.Fatalf("write expected: %v", err)
			}
		}
	}
}

// dummyInput / dummyExpectedSuccess / dummyExpectedError build
// minimal payloads that satisfy the LoadFixtures shape constraints.
func dummyInput(name string) []byte {
	return []byte(`{"name":"` + name + `"}`)
}
func dummyExpectedSuccess() []byte {
	v := ExpectedSuccess{
		Status: string(output.StatusSuccess),
		InputEcho: output.InputEcho{
			BirthDate: "1990-04-09",
			BirthTime: "18:04",
			Timezone:  "Europe/Amsterdam",
			Latitude:  output.Longitude(51.9167),
			Longitude: output.Longitude(4.4),
		},
	}
	b, _ := json.Marshal(v)
	return b
}
func dummyExpectedError(errType string) []byte {
	body := map[string]any{
		"status": string(output.StatusError),
		"error":  map[string]string{"error_type": errType},
	}
	b, _ := json.Marshal(body)
	return b
}

// requiredCounts builds a map satisfying every per-category minimum
// with placeholder fixtures.  Tests append their own cases on top.
func requiredCounts(t *testing.T, errOk bool) map[string]map[string][2][]byte {
	t.Helper()
	all := map[string]map[string][2][]byte{}
	for cat, n := range MinimumCounts {
		all[string(cat)] = map[string][2][]byte{}
		for i := 0; i < n; i++ {
			name := "case_" + string(rune('a'+i))
			expected := dummyExpectedSuccess()
			if IsErrorCategory(cat) {
				if !errOk {
					t.Fatalf("category %s requires error fixtures but errOk=false", cat)
				}
				// Use canonical error_type per category.
				switch cat {
				case CategoryInvalidInput:
					expected = dummyExpectedError(output.ErrorInvalidInput)
				case CategoryIncompleteInput:
					expected = dummyExpectedError(output.ErrorIncompleteInput)
				case CategoryUnsupportedInput:
					expected = dummyExpectedError(output.ErrorUnsupportedInput)
				}
			}
			all[string(cat)][name] = [2][]byte{dummyInput(name), expected}
		}
	}
	return all
}

func TestLoadFixturesAcceptsCanonicalMinimums(t *testing.T) {
	root := t.TempDir()
	writePack(t, root, requiredCounts(t, true))

	fxs, err := LoadFixtures(root)
	if err != nil {
		t.Fatalf("LoadFixtures: %v", err)
	}
	wantTotal := 0
	for _, n := range MinimumCounts {
		wantTotal += n
	}
	if len(fxs) != wantTotal {
		t.Fatalf("LoadFixtures returned %d fixtures, want %d", len(fxs), wantTotal)
	}
	// Each category must be represented exactly minimum-count times.
	per := map[Category]int{}
	for _, f := range fxs {
		per[f.Category]++
	}
	for _, cat := range Categories() {
		if per[cat] != MinimumCounts[cat] {
			t.Errorf("category %s: %d fixtures, want %d",
				cat, per[cat], MinimumCounts[cat])
		}
	}
	// Names within a category must come back sorted.
	prev := map[Category]string{}
	for _, f := range fxs {
		if last, ok := prev[f.Category]; ok && last >= f.Name {
			t.Errorf("category %s: fixture %q follows %q (not sorted)",
				f.Category, f.Name, last)
		}
		prev[f.Category] = f.Name
	}
}

func TestLoadFixturesRejectsBelowMinimum(t *testing.T) {
	root := t.TempDir()
	cases := requiredCounts(t, true)
	// Drop one fixture from valid_baseline so the minimum-3 rule
	// fails.
	for k := range cases[string(CategoryValidBaseline)] {
		delete(cases[string(CategoryValidBaseline)], k)
		break
	}
	writePack(t, root, cases)

	_, err := LoadFixtures(root)
	if err == nil {
		t.Fatal("LoadFixtures: err = nil; want non-nil for below-minimum count")
	}
	if !strings.Contains(err.Error(), "valid_baseline") {
		t.Errorf("error %q does not name valid_baseline", err.Error())
	}
}

func TestLoadFixturesRejectsMissingCategoryDir(t *testing.T) {
	root := t.TempDir()
	cases := requiredCounts(t, true)
	delete(cases, string(CategoryRegressionSentinel))
	writePack(t, root, cases)

	_, err := LoadFixtures(root)
	if err == nil {
		t.Fatal("LoadFixtures: err = nil; want non-nil for missing category")
	}
	if !strings.Contains(err.Error(), "regression_sentinel") {
		t.Errorf("error %q does not name regression_sentinel", err.Error())
	}
}

func TestLoadFixturesRejectsMissingFiles(t *testing.T) {
	root := t.TempDir()
	cases := requiredCounts(t, true)
	writePack(t, root, cases)

	// Delete one expected.json from valid_baseline.
	caseDir := filepath.Join(root, string(CategoryValidBaseline), "case_a")
	if err := os.Remove(filepath.Join(caseDir, "expected.json")); err != nil {
		t.Fatalf("remove expected.json: %v", err)
	}

	_, err := LoadFixtures(root)
	if err == nil {
		t.Fatal("LoadFixtures: err = nil; want non-nil for missing expected.json")
	}
	if !strings.Contains(err.Error(), "expected.json") {
		t.Errorf("error %q does not mention expected.json", err.Error())
	}
}

// TestCompareSuccessHappyPath constructs a SuccessEnvelope, projects
// it onto an ExpectedSuccess with metadata stripped, and asserts
// CompareSuccess accepts the round-trip.
func TestCompareSuccessHappyPath(t *testing.T) {
	full := output.SuccessEnvelope{
		Status:   output.StatusSuccess,
		Metadata: output.CurrentMetadata(), // ignored by CompareSuccess
		InputEcho: output.InputEcho{
			BirthDate: "1990-04-09",
			BirthTime: "18:04",
			Timezone:  "Europe/Amsterdam",
			Latitude:  output.Longitude(51.9167),
			Longitude: output.Longitude(4.4),
		},
		Astrology: output.Astrology{
			System: output.AstroSystem{
				Zodiac: "tropical", HouseSystem: "placidus", NodeType: "mean",
			},
			HouseCusps: []output.HouseCusp{},
			Objects:    []output.AstroObject{},
		},
		HumanDesign: output.HumanDesignOut{
			System:                 output.HDSystem{NodeType: "true"},
			PersonalityActivations: []output.HDActivation{},
			DesignActivations:      []output.HDActivation{},
			Channels:               []output.HDChannel{},
			Centers:                []output.HDCenter{},
			Definition:             "none",
			Type:                   "reflector",
			Authority:              "lunar",
			Profile:                "1/1",
		},
		GeneKeys: output.GeneKeysOut{
			System: output.GKSystem{DerivationBasis: "human_design"},
		},
	}
	want := ExpectedSuccess{
		Status:      full.Status,
		InputEcho:   full.InputEcho,
		Astrology:   full.Astrology,
		HumanDesign: full.HumanDesign,
		GeneKeys:    full.GeneKeys,
	}
	if err := CompareSuccess(full, want); err != nil {
		t.Errorf("CompareSuccess: %v", err)
	}
}

// TestCompareSuccessCatchesAstrologyDrift mutates the astrology
// section and confirms CompareSuccess reports the drift.
func TestCompareSuccessCatchesAstrologyDrift(t *testing.T) {
	want := ExpectedSuccess{
		Status: string(output.StatusSuccess),
		Astrology: output.Astrology{
			System: output.AstroSystem{Zodiac: "tropical"},
		},
	}
	got := output.SuccessEnvelope{
		Status: output.StatusSuccess,
		Astrology: output.Astrology{
			System: output.AstroSystem{Zodiac: "sidereal"}, // drift
		},
	}
	err := CompareSuccess(got, want)
	if err == nil {
		t.Fatal("CompareSuccess: err = nil; want astrology drift error")
	}
	if !strings.Contains(err.Error(), "astrology drift") {
		t.Errorf("error %q does not mention astrology drift", err.Error())
	}
}

// TestCompareSuccessCatchesOrderingDrift constructs two astrology
// envelopes whose Objects slices contain identical members in
// different order.  reflect.DeepEqual is order-sensitive, so the
// runner correctly catches ordering drift — which is exactly the
// canon's "Output ordering must match Document 07 order" rule
// (trinity.org §"Astrology Output").
func TestCompareSuccessCatchesOrderingDrift(t *testing.T) {
	a := output.AstroObject{ObjectID: "sun", Longitude: output.Longitude(1)}
	b := output.AstroObject{ObjectID: "moon", Longitude: output.Longitude(2)}
	want := ExpectedSuccess{
		Status: string(output.StatusSuccess),
		Astrology: output.Astrology{
			Objects: []output.AstroObject{a, b},
		},
	}
	got := output.SuccessEnvelope{
		Status: output.StatusSuccess,
		Astrology: output.Astrology{
			Objects: []output.AstroObject{b, a}, // swapped order
		},
	}
	err := CompareSuccess(got, want)
	if err == nil {
		t.Fatal("CompareSuccess accepted swapped objects order")
	}
}

// TestCompareErrorAssertsErrorTypeOnly proves that error_type is
// the only contractual field (A4 RESOLVED, D23): identical error
// types pass even when the message text differs wildly.
func TestCompareErrorAssertsErrorTypeOnly(t *testing.T) {
	current := output.CurrentMetadata()
	got := output.ErrorEnvelope{
		Status:   output.StatusError,
		Metadata: current,
		Error: output.Error{
			Type:    output.ErrorInvalidInput,
			Message: "any prose the engine emits",
		},
	}
	want := ExpectedError{
		Status: string(output.StatusError),
		Error:  struct{ ErrorType string `json:"error_type"` }{ErrorType: output.ErrorInvalidInput},
	}
	if err := CompareError(got, want, current); err != nil {
		t.Errorf("CompareError: %v", err)
	}
}

// TestCompareErrorRejectsTypeMismatch flips the error_type and
// confirms the comparison fails.
func TestCompareErrorRejectsTypeMismatch(t *testing.T) {
	current := output.CurrentMetadata()
	got := output.ErrorEnvelope{
		Status:   output.StatusError,
		Metadata: current,
		Error: output.Error{
			Type:    output.ErrorInvalidInput,
			Message: "not the right type",
		},
	}
	want := ExpectedError{
		Status: string(output.StatusError),
		Error:  struct{ ErrorType string `json:"error_type"` }{ErrorType: output.ErrorIncompleteInput},
	}
	err := CompareError(got, want, current)
	if err == nil || !strings.Contains(err.Error(), "error_type") {
		t.Fatalf("CompareError = %v; want error_type mismatch", err)
	}
}

// TestCompareErrorRejectsEmptyMessage covers the canon rule that
// error.message must be non-empty (A4 RESOLVED, D23: text is
// informational but cannot be elided).
func TestCompareErrorRejectsEmptyMessage(t *testing.T) {
	current := output.CurrentMetadata()
	got := output.ErrorEnvelope{
		Status:   output.StatusError,
		Metadata: current,
		Error: output.Error{
			Type:    output.ErrorInvalidInput,
			Message: "",
		},
	}
	want := ExpectedError{
		Status: string(output.StatusError),
		Error:  struct{ ErrorType string `json:"error_type"` }{ErrorType: output.ErrorInvalidInput},
	}
	err := CompareError(got, want, current)
	if err == nil || !strings.Contains(err.Error(), "message") {
		t.Fatalf("CompareError = %v; want non-empty message error", err)
	}
}

// TestCompareErrorRejectsMetadataDrift catches metadata drift via
// the dedicated check inside CompareError.
func TestCompareErrorRejectsMetadataDrift(t *testing.T) {
	current := output.CurrentMetadata()
	got := output.ErrorEnvelope{
		Status: output.StatusError,
		Metadata: output.Metadata{
			EngineVersion:      "tampered",
			CanonVersion:       current.CanonVersion,
			SourceStackVersion: current.SourceStackVersion,
			InputSchemaVersion: current.InputSchemaVersion,
			MappingVersion:     current.MappingVersion,
		},
		Error: output.Error{
			Type:    output.ErrorInvalidInput,
			Message: "test",
		},
	}
	want := ExpectedError{
		Status: string(output.StatusError),
		Error:  struct{ ErrorType string `json:"error_type"` }{ErrorType: output.ErrorInvalidInput},
	}
	err := CompareError(got, want, current)
	if err == nil || !strings.Contains(err.Error(), "metadata") {
		t.Fatalf("CompareError = %v; want metadata drift error", err)
	}
}

// TestSemanticJSONEqualIgnoresKeyOrder pins the canonical
// equality semantics: two JSON objects with identical leaf values
// but different key ordering are semantically equal.
func TestSemanticJSONEqualIgnoresKeyOrder(t *testing.T) {
	a := []byte(`{"a":1,"b":[1,2,3]}`)
	b := []byte(`{"b":[1,2,3],"a":1}`)
	eq, err := SemanticJSONEqual(a, b)
	if err != nil {
		t.Fatalf("SemanticJSONEqual: %v", err)
	}
	if !eq {
		t.Errorf("expected equal under key-reordering")
	}
}

// TestSemanticJSONEqualCatchesArrayOrderDrift verifies that array
// element order *is* significant — the canon pins ordering for
// every array in the success envelope.
func TestSemanticJSONEqualCatchesArrayOrderDrift(t *testing.T) {
	a := []byte(`[1,2,3]`)
	b := []byte(`[1,3,2]`)
	eq, err := SemanticJSONEqual(a, b)
	if err != nil {
		t.Fatalf("SemanticJSONEqual: %v", err)
	}
	if eq {
		t.Errorf("expected unequal under array-reordering")
	}
}
