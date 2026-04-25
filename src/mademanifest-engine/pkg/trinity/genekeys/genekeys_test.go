package genekeys

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"mademanifest-engine/pkg/trinity/output"
)

// TestComputePureFunctionOfHDActivations pins the canonical
// derivation rule (trinity.org lines 142-149):
//
//   life_work = personality_sun
//   evolution = personality_earth
//   radiance  = design_sun
//   purpose   = design_earth
//
// The (gate, line) pair from the corresponding activation lands on
// (key, line) of the Gene Keys position with no transformation.
func TestComputePureFunctionOfHDActivations(t *testing.T) {
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 42, Line: 1},
		{ObjectID: "earth", Gate: 32, Line: 1},
		// extra bodies must not influence the result
		{ObjectID: "moon", Gate: 57, Line: 2},
		{ObjectID: "mercury", Gate: 24, Line: 3},
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 61, Line: 3},
		{ObjectID: "earth", Gate: 62, Line: 3},
		{ObjectID: "north_node", Gate: 13, Line: 6},
	}
	got, err := Compute(personality, design)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	want := output.GeneKeysOut{
		System: output.GKSystem{DerivationBasis: "human_design"},
		Activations: output.GKActivations{
			LifeWork:  output.GKActivation{Key: 42, Line: 1},
			Evolution: output.GKActivation{Key: 32, Line: 1},
			Radiance:  output.GKActivation{Key: 61, Line: 3},
			Purpose:   output.GKActivation{Key: 62, Line: 3},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Compute(...) =\n %+v\nwant\n %+v", got, want)
	}
}

// TestComputeIgnoresExtraBodies verifies the result depends only on
// sun + earth: shuffling the other 11 activations to absurd values
// must not change the output.
func TestComputeIgnoresExtraBodies(t *testing.T) {
	base := []output.HDActivation{
		{ObjectID: "sun", Gate: 1, Line: 1},
		{ObjectID: "earth", Gate: 2, Line: 2},
	}
	noisy := append([]output.HDActivation(nil), base...)
	for _, body := range []string{"moon", "mercury", "venus", "mars", "jupiter", "saturn"} {
		noisy = append(noisy, output.HDActivation{ObjectID: body, Gate: 64, Line: 6})
	}
	r1, err := Compute(base, base)
	if err != nil {
		t.Fatalf("Compute(base,base): %v", err)
	}
	r2, err := Compute(noisy, noisy)
	if err != nil {
		t.Fatalf("Compute(noisy,noisy): %v", err)
	}
	if r1 != r2 {
		t.Errorf("noise leaked into result:\n base = %+v\n noisy = %+v", r1, r2)
	}
}

// TestComputeRequiresAllFourPillars enumerates the four
// missing-pillar cases.  Each must surface as a non-nil error so
// the HTTP handler can emit execution_failure rather than a
// silently zero-valued envelope.
func TestComputeRequiresAllFourPillars(t *testing.T) {
	full := func() ([]output.HDActivation, []output.HDActivation) {
		p := []output.HDActivation{
			{ObjectID: "sun", Gate: 1, Line: 1},
			{ObjectID: "earth", Gate: 2, Line: 2},
		}
		d := []output.HDActivation{
			{ObjectID: "sun", Gate: 3, Line: 3},
			{ObjectID: "earth", Gate: 4, Line: 4},
		}
		return p, d
	}

	cases := []struct {
		name        string
		mutate      func(*[]output.HDActivation, *[]output.HDActivation)
		wantInError string
	}{
		{
			name: "missing personality sun",
			mutate: func(p, _ *[]output.HDActivation) {
				*p = (*p)[1:] // drop sun
			},
			wantInError: "personality",
		},
		{
			name: "missing personality earth",
			mutate: func(p, _ *[]output.HDActivation) {
				*p = (*p)[:1] // keep only sun
			},
			wantInError: "personality",
		},
		{
			name: "missing design sun",
			mutate: func(_, d *[]output.HDActivation) {
				*d = (*d)[1:]
			},
			wantInError: "design",
		},
		{
			name: "missing design earth",
			mutate: func(_, d *[]output.HDActivation) {
				*d = (*d)[:1]
			},
			wantInError: "design",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, d := full()
			c.mutate(&p, &d)
			_, err := Compute(p, d)
			if err == nil {
				t.Fatalf("Compute returned nil error; want non-nil")
			}
			if !strings.Contains(err.Error(), c.wantInError) {
				t.Errorf("error %q does not mention %q", err.Error(), c.wantInError)
			}
		})
	}
}

// TestComputeJSONShapeIsCanonOnly marshals the result and asserts
// the wire shape is exactly the canon-allowed key set.  No text,
// shadow, gift, essence, or semantic-state fields may appear.
//
// This is the canon-fence test.  If any future code adds fields to
// GKActivations / GKActivation / GKSystem (intentionally or
// otherwise), this test fails until the new keys are vetted against
// trinity.org §"Gene Keys Output".
func TestComputeJSONShapeIsCanonOnly(t *testing.T) {
	personality := []output.HDActivation{
		{ObjectID: "sun", Gate: 42, Line: 1},
		{ObjectID: "earth", Gate: 32, Line: 1},
	}
	design := []output.HDActivation{
		{ObjectID: "sun", Gate: 61, Line: 3},
		{ObjectID: "earth", Gate: 62, Line: 3},
	}
	gk, err := Compute(personality, design)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	raw, err := json.Marshal(gk)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	// Decode into a generic map and walk the tree, collecting all
	// JSON keys and comparing against the canon-allowed set.
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatalf("json.Unmarshal: %v\nbody: %s", err, raw)
	}
	got := keysOf(generic, "")
	sort.Strings(got)
	want := []string{
		"activations",
		"activations.evolution",
		"activations.evolution.key",
		"activations.evolution.line",
		"activations.life_work",
		"activations.life_work.key",
		"activations.life_work.line",
		"activations.purpose",
		"activations.purpose.key",
		"activations.purpose.line",
		"activations.radiance",
		"activations.radiance.key",
		"activations.radiance.line",
		"system",
		"system.derivation_basis",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("JSON keys =\n %v\nwant\n %v\nbody: %s", got, want, raw)
	}

	// Also verify the legacy PoC field "lifes_work" is *not*
	// emitted under any path (case-sensitive substring search).
	if strings.Contains(string(raw), "lifes_work") {
		t.Errorf(`canon rename violated: "lifes_work" present in output: %s`, raw)
	}
}

// keysOf flattens a JSON-decoded tree into dot-separated key
// paths.  It does not descend into JSON arrays because none of the
// Gene Keys output fields are arrays today.
func keysOf(node any, prefix string) []string {
	out := []string{}
	m, ok := node.(map[string]any)
	if !ok {
		return out
	}
	for k, v := range m {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		out = append(out, path)
		out = append(out, keysOf(v, path)...)
	}
	return out
}
