package process_input

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"mademanifest-engine/pkg/canon"
)

// import not required for this test
// fields are missing from the input JSON.
func TestProcessInputDefaults(t *testing.T) {
	// Input JSON missing human_design_mapping and node_policy_by_system
	raw := `{
        "birth": {
            "date": "2000-01-01",
            "time_hh_mm": "00:00",
            "seconds_policy": "assume_00",
            "place_name": "Test",
            "latitude": 0,
            "longitude": 0,
            "timezone_iana": "UTC"
        },
        "engine_contract": {
            "ephemeris": "swiss_ephemeris",
            "zodiac": "tropical",
            "houses": "placidus"
        },
        "expected": {}
    }`
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	canonDir := filepath.Clean(filepath.Join(cwd, "..", "..", "..", "canon"))
	canonPaths := canon.Paths{
		MandalaConstants: filepath.Join(canonDir, "mandala_constants.json"),
		NodePolicy:       filepath.Join(canonDir, "node_policy.json"),
		GateSequence:     filepath.Join(canonDir, "gate_sequence_v1.json"),
	}
	caseData, err := ProcessInput(decoder, canonPaths)
	if err != nil {
		t.Fatalf("ProcessInput returned error: %v", err)
	}
	// Verify that human_design_mapping defaults are present
	md := caseData.EngineContract.HumanDesignMapping
	if md.MandalaStartDeg == 0 || md.GateWidthDeg == 0 || md.LineWidthDeg == 0 {
		t.Errorf("default human_design_mapping not applied: %+v", md)
	}
	if md.IntervalRule != "start_inclusive_end_exclusive" {
		t.Errorf("expected default interval_rule, got %s", md.IntervalRule)
	}
	// Verify node policy defaults
	np := caseData.EngineContract.NodePolicyBySystem
	if np.Astrology != "mean" || np.HumanDesign != true || np.GeneKeys != true {
		t.Errorf("default node_policy not applied: %+v", np)
	}
}
