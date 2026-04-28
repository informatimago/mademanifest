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

// AssertSchiedamGeneKeysMatchOracle is the Phase 8 regression
// sentinel for the Trinity Gene Keys block.  The frozen oracle
// pins:
//
//   * system.derivation_basis = "human_design"
//   * activations.life_work  = personality_sun  = 42.1 (Schiedam)
//   * activations.evolution  = personality_earth = 32.1
//   * activations.radiance   = design_sun       = 61.3
//   * activations.purpose    = design_earth     = 62.3
//
// A7 (RESOLVED, Document 12 D26): the oracle is a freeze of live
// Phase 8 engine output, cross-checked against the Phase 6 activation
// oracle (Gene Keys activations are pure copies of the four HD pillar
// activations, so any drift here is necessarily upstream).  Per D26,
// engine generation is necessary but not sufficient; the canon owner
// must explicitly approve before the freeze is authoritative.
//
// The test asserts the entire gene_keys subtree via reflect.DeepEqual
// so a drift in any of the seven leaf values surfaces with a clean
// got/want diff.
func AssertSchiedamGeneKeysMatchOracle(t *testing.T, baseURL string) {
	t.Helper()

	baselineDir := filepath.Join(RepoRoot(t), "src", "golden", "trinity", "baseline")
	inputBytes, err := os.ReadFile(
		filepath.Join(baselineDir, "schiedam_1990_04_09_input.json"))
	if err != nil {
		t.Fatalf("read baseline input: %v", err)
	}
	oracleBytes, err := os.ReadFile(
		filepath.Join(baselineDir, "schiedam_1990_04_09_gene_keys.json"))
	if err != nil {
		t.Fatalf("read gene keys oracle: %v", err)
	}

	var oracle output.GeneKeysOut
	if err := json.Unmarshal(oracleBytes, &oracle); err != nil {
		t.Fatalf("decode gene keys oracle: %v", err)
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

	if !reflect.DeepEqual(env.GeneKeys, oracle) {
		t.Errorf("gene_keys drift:\n got:  %+v\n want: %+v", env.GeneKeys, oracle)
	}
}
