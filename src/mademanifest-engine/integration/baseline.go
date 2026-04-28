package integration

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"mademanifest-engine/pkg/trinity/output"
)

// AssertSchiedamAstrologyMatchesOracle runs the canonical Schiedam
// 1990-04-09 18:04 input through POST /manifest on baseURL and
// compares the astrology subsection field-by-field with the frozen
// oracle fixture under src/golden/trinity/baseline/.
//
// The oracle was captured once from a Phase 4 engine run and cross-
// checked against the PoC golden case's published Schiedam chart
// (ascendant Virgo 25°06' ≈ 175.114°, MC Gemini 23°35' ≈ 83.6°).
// Going forward the test acts as a regression sentinel: any drift
// in the astrology pipeline (Swiss Ephemeris pin, time conversion,
// house algorithm, sign mapping, Earth derivation) is caught here.
//
// A7 (RESOLVED, Document 12 D26): the oracle is not self-validating.
// Generation by the implementation is necessary but not sufficient;
// before approval, fixtures must be reviewed against the governing
// canon (timezone-sensitive cases, sign / house / gate / line
// boundaries, Design-time sub-second stop behaviour, error
// classification).  External tools (e.g. the published PoC Schiedam
// chart used here) are sanity checks, not the golden oracle.  The
// canon owner must explicitly approve the frozen fixture.
func AssertSchiedamAstrologyMatchesOracle(t *testing.T, baseURL string) {
	t.Helper()

	baselineDir := filepath.Join(RepoRoot(t), "src", "golden", "trinity", "baseline")
	inputBytes, err := os.ReadFile(
		filepath.Join(baselineDir, "schiedam_1990_04_09_input.json"))
	if err != nil {
		t.Fatalf("read baseline input: %v", err)
	}
	oracleBytes, err := os.ReadFile(
		filepath.Join(baselineDir, "schiedam_1990_04_09_astrology.json"))
	if err != nil {
		t.Fatalf("read baseline oracle: %v", err)
	}

	var oracle output.Astrology
	if err := json.Unmarshal(oracleBytes, &oracle); err != nil {
		t.Fatalf("decode baseline oracle: %v", err)
	}

	status, raw, err := PostManifest(baseURL, inputBytes, nil)
	if err != nil {
		t.Fatalf("POST /manifest: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", status, raw)
	}

	var env output.SuccessEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("decode success envelope: %v\nbody: %s", err, raw)
	}
	if !reflect.DeepEqual(env.Astrology.System, oracle.System) {
		t.Errorf("astrology.system drift:\n got:  %+v\n want: %+v",
			env.Astrology.System, oracle.System)
	}
	if !reflect.DeepEqual(env.Astrology.Angles, oracle.Angles) {
		t.Errorf("astrology.angles drift:\n got:  %+v\n want: %+v",
			env.Astrology.Angles, oracle.Angles)
	}
	if !reflect.DeepEqual(env.Astrology.HouseCusps, oracle.HouseCusps) {
		t.Errorf("astrology.house_cusps drift:\n got:  %+v\n want: %+v",
			env.Astrology.HouseCusps, oracle.HouseCusps)
	}
	if !reflect.DeepEqual(env.Astrology.Objects, oracle.Objects) {
		t.Errorf("astrology.objects drift:\n got:  %+v\n want: %+v",
			env.Astrology.Objects, oracle.Objects)
	}
}
