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

// activationsOracle mirrors the JSON shape of
// schiedam_1990_04_09_activations.json: just the two activation
// arrays without the surrounding success-envelope keys.
type activationsOracle struct {
	PersonalityActivations []output.HDActivation `json:"personality_activations"`
	DesignActivations      []output.HDActivation `json:"design_activations"`
}

// AssertSchiedamActivationsMatchOracle is the Phase 6 regression
// sentinel for the Trinity Human Design activation pipeline.  The
// frozen oracle captures both personality_activations and
// design_activations for the canonical Schiedam input, computed
// with:
//
//   * canon mandala anchor 277.5° (gate 38 first in canon.GateOrder)
//   * SE_TRUE_NODE for north_node (Document 03 §"Node policy by domain")
//   * design Julian Day = the Phase 5 bisection result for the same
//     payload
//
// Per A7 the oracle was captured once from the Phase 6 engine and
// cross-checked against the Schiedam PoC golden case via algebraic
// re-anchoring: the PoC used anchor 313.25° with a different
// GateOrder, so the two emit different gate numbers, but they must
// agree on the underlying ecliptic longitude.  Spot check at Sun:
// PoC personality_sun = 51.5 (anchor 313.25°) corresponds to
// Trinity personality_sun = 42.1 (anchor 277.5°) for the same Sun
// longitude ≈ 19.540°.
func AssertSchiedamActivationsMatchOracle(t *testing.T, baseURL string) {
	t.Helper()

	baselineDir := filepath.Join(RepoRoot(t), "src", "golden", "trinity", "baseline")
	inputBytes, err := os.ReadFile(
		filepath.Join(baselineDir, "schiedam_1990_04_09_input.json"))
	if err != nil {
		t.Fatalf("read baseline input: %v", err)
	}
	oracleBytes, err := os.ReadFile(
		filepath.Join(baselineDir, "schiedam_1990_04_09_activations.json"))
	if err != nil {
		t.Fatalf("read activations oracle: %v", err)
	}

	var oracle activationsOracle
	if err := json.Unmarshal(oracleBytes, &oracle); err != nil {
		t.Fatalf("decode activations oracle: %v", err)
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

	if !reflect.DeepEqual(env.HumanDesign.PersonalityActivations, oracle.PersonalityActivations) {
		t.Errorf("human_design.personality_activations drift:\n got:  %+v\n want: %+v",
			env.HumanDesign.PersonalityActivations, oracle.PersonalityActivations)
	}
	if !reflect.DeepEqual(env.HumanDesign.DesignActivations, oracle.DesignActivations) {
		t.Errorf("human_design.design_activations drift:\n got:  %+v\n want: %+v",
			env.HumanDesign.DesignActivations, oracle.DesignActivations)
	}
}
