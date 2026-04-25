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

// structureOracle mirrors the JSON shape of
// schiedam_1990_04_09_structure.json: the seven Phase 7 structural
// sub-fields without the surrounding success-envelope keys.
type structureOracle struct {
	Channels         []output.HDChannel        `json:"channels"`
	Centers          []output.HDCenter         `json:"centers"`
	Definition       string                    `json:"definition"`
	Type             string                    `json:"type"`
	Authority        string                    `json:"authority"`
	Profile          string                    `json:"profile"`
	IncarnationCross output.HDIncarnationCross `json:"incarnation_cross"`
}

// AssertSchiedamStructureMatchesOracle is the Phase 7 regression
// sentinel for the Trinity Human Design structural derivations.
// The frozen oracle pins:
//
//   * channels (lexicographic by channel_id) — Schiedam: 24-61,
//     32-54, 7-31
//   * centers in canon.CenterOrder, each defined / undefined
//   * definition = triple_split (three connected components:
//     {ajna,head}, {spleen,root}, {g,throat})
//   * type       = projector (no sacral, no motor-to-throat)
//   * authority  = splenic (no solar_plexus, projector with spleen)
//   * profile    = "1/3" (personality sun line / design sun line)
//   * incarnation_cross 42.1 / 32.1 | 61.3 / 62.3
//
// Per A7 the structural oracle is a freeze of the live Phase 7
// engine output, cross-checked against the canon decision tree
// (trinity.org §"Type derivation" lines 318-325) and the Phase 6
// activation oracle.  The canon owner must approve before final
// acceptance.
func AssertSchiedamStructureMatchesOracle(t *testing.T, baseURL string) {
	t.Helper()

	baselineDir := filepath.Join(RepoRoot(t), "src", "golden", "trinity", "baseline")
	inputBytes, err := os.ReadFile(
		filepath.Join(baselineDir, "schiedam_1990_04_09_input.json"))
	if err != nil {
		t.Fatalf("read baseline input: %v", err)
	}
	oracleBytes, err := os.ReadFile(
		filepath.Join(baselineDir, "schiedam_1990_04_09_structure.json"))
	if err != nil {
		t.Fatalf("read structure oracle: %v", err)
	}

	var oracle structureOracle
	if err := json.Unmarshal(oracleBytes, &oracle); err != nil {
		t.Fatalf("decode structure oracle: %v", err)
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

	if !reflect.DeepEqual(env.HumanDesign.Channels, oracle.Channels) {
		t.Errorf("human_design.channels drift:\n got:  %+v\n want: %+v",
			env.HumanDesign.Channels, oracle.Channels)
	}
	if !reflect.DeepEqual(env.HumanDesign.Centers, oracle.Centers) {
		t.Errorf("human_design.centers drift:\n got:  %+v\n want: %+v",
			env.HumanDesign.Centers, oracle.Centers)
	}
	if env.HumanDesign.Definition != oracle.Definition {
		t.Errorf("human_design.definition = %q, want %q",
			env.HumanDesign.Definition, oracle.Definition)
	}
	if env.HumanDesign.Type != oracle.Type {
		t.Errorf("human_design.type = %q, want %q",
			env.HumanDesign.Type, oracle.Type)
	}
	if env.HumanDesign.Authority != oracle.Authority {
		t.Errorf("human_design.authority = %q, want %q",
			env.HumanDesign.Authority, oracle.Authority)
	}
	if env.HumanDesign.Profile != oracle.Profile {
		t.Errorf("human_design.profile = %q, want %q",
			env.HumanDesign.Profile, oracle.Profile)
	}
	if env.HumanDesign.IncarnationCross != oracle.IncarnationCross {
		t.Errorf("human_design.incarnation_cross drift:\n got:  %+v\n want: %+v",
			env.HumanDesign.IncarnationCross, oracle.IncarnationCross)
	}
}
