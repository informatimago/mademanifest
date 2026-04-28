package integration

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"mademanifest-engine/pkg/trinity/output"
)

// AssertSchiedamDesignTimeMatchesOracle is the Phase 5 regression
// sentinel for the canonical Schiedam 1990-04-09 18:04 input.  The
// frozen oracle captures human_design.system, which after Phase 5
// carries:
//
//   * node_type           "true"      (canon-pinned)
//   * design_time_utc     RFC 3339 UTC, whole-second truncation of
//                         the solver's lower-bound timestamp
//                         (A3 RESOLVED, Document 12 D22)
//
// A7 (RESOLVED, Document 12 D26): design_time_utc was captured from
// the live engine and cross-checked for plausibility (approximately
// 88 days earlier than birth, ~1.004°/day Sun rate around perihelion).
// Birth UTC is 1990-04-09 16:04Z; the oracle puts design at
// 1990-01-12 00:38Z, ~87.65 days before birth — consistent with the
// canonical 88° backward Sun offset.  The canon owner must explicitly
// approve this fixture per D26 (engine generation is necessary but
// not sufficient); the cross-check above is a sanity check, not the
// oracle itself.
//
// The helper checks the entire HDSystem block via reflect.DeepEqual.
// Drift in either node_type (engine-level mistake) or
// design_time_utc (solver drift) fails the test.  The check is
// deliberately separate from AssertSchiedamAstrologyMatchesOracle
// so a drift report localises the regression to the Phase that
// owns the affected sub-section.
func AssertSchiedamDesignTimeMatchesOracle(t *testing.T, baseURL string) {
	t.Helper()

	baselineDir := filepath.Join(RepoRoot(t), "src", "golden", "trinity", "baseline")
	inputBytes, err := os.ReadFile(
		filepath.Join(baselineDir, "schiedam_1990_04_09_input.json"))
	if err != nil {
		t.Fatalf("read baseline input: %v", err)
	}
	oracleBytes, err := os.ReadFile(
		filepath.Join(baselineDir, "schiedam_1990_04_09_design_time.json"))
	if err != nil {
		t.Fatalf("read baseline design-time oracle: %v", err)
	}

	var oracle output.HDSystem
	if err := json.Unmarshal(oracleBytes, &oracle); err != nil {
		t.Fatalf("decode design-time oracle: %v", err)
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

	if env.HumanDesign.System.NodeType != oracle.NodeType {
		t.Errorf("human_design.system.node_type drift:\n got:  %q\n want: %q",
			env.HumanDesign.System.NodeType, oracle.NodeType)
	}

	gotTime := time.Time(env.HumanDesign.System.DesignTimeUTC).UTC()
	wantTime := time.Time(oracle.DesignTimeUTC).UTC()
	if !reflect.DeepEqual(gotTime, wantTime) {
		t.Errorf("human_design.system.design_time_utc drift:\n got:  %s\n want: %s",
			gotTime.Format(time.RFC3339), wantTime.Format(time.RFC3339))
	}
}
