package integration

// env_immunity.go hosts the Phase 9 env-immunity helpers.  The
// canon (trinity.org §"Determinism And Versioning" lines 585-597)
// forbids any "implicit environment defaults" from affecting
// engine output.  Phase 6 retired the SE_NODE_POLICY shim that
// previously toggled north-node policy at runtime; Phase 9
// integration tests pin the rule that any value of that variable
// (or any other historically-influential env) leaves the engine
// output bit-identical to the no-env baseline.
//
// The helpers in this file are runtime-agnostic: they take a
// pre-built ServerHandle and probe it through the network
// surface.  Per-runtime test files wire up the handle with the
// SE_NODE_POLICY=true environment in place and call the helpers
// to assert canonical output.

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"mademanifest-engine/pkg/trinity/output"
)

// AssertEnvImmuneCanonicalSchiedam runs the canonical Schiedam
// payload through baseURL and asserts the astrology, activations,
// structure, and gene_keys subsections match every Phase 4-8
// frozen oracle.  Test wrapper test files configure the
// ServerHandle with non-canonical environment (e.g. the retired
// SE_NODE_POLICY=true variable) before calling this helper.  A
// post-Phase-6 engine must be immune to that environment and
// produce bit-identical output.
//
// We deliberately do not re-use the per-section AssertSchiedam*
// helpers individually because each starts and stops its own
// server.  In env-immunity tests the ServerHandle is already
// configured with the test environment by the caller, so we want
// one POST and many comparisons.
func AssertEnvImmuneCanonicalSchiedam(t *testing.T, baseURL string) {
	t.Helper()

	baselineDir := filepath.Join(RepoRoot(t), "src", "golden", "trinity", "baseline")
	inputBytes, err := os.ReadFile(
		filepath.Join(baselineDir, "schiedam_1990_04_09_input.json"))
	if err != nil {
		t.Fatalf("read baseline input: %v", err)
	}

	var astroOracle output.Astrology
	mustReadJSON(t, filepath.Join(baselineDir, "schiedam_1990_04_09_astrology.json"), &astroOracle)

	var dtOracle output.HDSystem
	mustReadJSON(t, filepath.Join(baselineDir, "schiedam_1990_04_09_design_time.json"), &dtOracle)

	var actOracle activationsOracle
	mustReadJSON(t, filepath.Join(baselineDir, "schiedam_1990_04_09_activations.json"), &actOracle)

	var strOracle structureOracle
	mustReadJSON(t, filepath.Join(baselineDir, "schiedam_1990_04_09_structure.json"), &strOracle)

	var gkOracle output.GeneKeysOut
	mustReadJSON(t, filepath.Join(baselineDir, "schiedam_1990_04_09_gene_keys.json"), &gkOracle)

	status, raw, err := PostManifest(baseURL, inputBytes, nil)
	if err != nil {
		t.Fatalf("POST /manifest: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", status, raw)
	}

	var env output.SuccessEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("decode envelope: %v\nbody: %s", err, raw)
	}

	// Astrology (Phase 4 oracle).
	if !reflect.DeepEqual(env.Astrology, astroOracle) {
		t.Errorf("astrology drift under non-canonical env:\n got:  %+v\n want: %+v",
			env.Astrology, astroOracle)
	}
	// Human Design system (Phase 5 design_time_utc).
	if env.HumanDesign.System.NodeType != dtOracle.NodeType {
		t.Errorf("human_design.system.node_type drift: got %q, want %q",
			env.HumanDesign.System.NodeType, dtOracle.NodeType)
	}
	if !reflect.DeepEqual(env.HumanDesign.System.DesignTimeUTC, dtOracle.DesignTimeUTC) {
		t.Errorf("design_time_utc drift under non-canonical env:\n got:  %+v\n want: %+v",
			env.HumanDesign.System.DesignTimeUTC, dtOracle.DesignTimeUTC)
	}
	// Activations (Phase 6).
	if !reflect.DeepEqual(env.HumanDesign.PersonalityActivations, actOracle.PersonalityActivations) {
		t.Errorf("personality_activations drift:\n got:  %+v\n want: %+v",
			env.HumanDesign.PersonalityActivations, actOracle.PersonalityActivations)
	}
	if !reflect.DeepEqual(env.HumanDesign.DesignActivations, actOracle.DesignActivations) {
		t.Errorf("design_activations drift:\n got:  %+v\n want: %+v",
			env.HumanDesign.DesignActivations, actOracle.DesignActivations)
	}
	// Structure (Phase 7).
	if !reflect.DeepEqual(env.HumanDesign.Channels, strOracle.Channels) {
		t.Errorf("channels drift")
	}
	if !reflect.DeepEqual(env.HumanDesign.Centers, strOracle.Centers) {
		t.Errorf("centers drift")
	}
	if env.HumanDesign.Definition != strOracle.Definition {
		t.Errorf("definition drift: got %q, want %q",
			env.HumanDesign.Definition, strOracle.Definition)
	}
	if env.HumanDesign.Type != strOracle.Type {
		t.Errorf("type drift: got %q, want %q",
			env.HumanDesign.Type, strOracle.Type)
	}
	if env.HumanDesign.Authority != strOracle.Authority {
		t.Errorf("authority drift: got %q, want %q",
			env.HumanDesign.Authority, strOracle.Authority)
	}
	// Gene Keys (Phase 8).
	if !reflect.DeepEqual(env.GeneKeys, gkOracle) {
		t.Errorf("gene_keys drift under non-canonical env:\n got:  %+v\n want: %+v",
			env.GeneKeys, gkOracle)
	}
}

func mustReadJSON(t *testing.T, path string, into any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, into); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}
