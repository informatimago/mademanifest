package canon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSelfCheckPassesOnCompiledConstants pins the post-condition
// that the engine ships with a self-consistent canon.  Boot-time
// SelfCheck() returning nil is the canonical "constants are sane"
// signal; if this test ever fails, the constants in this package
// drifted out of trinity.org alignment.
func TestSelfCheckPassesOnCompiledConstants(t *testing.T) {
	if err := SelfCheck(); err != nil {
		t.Fatalf("SelfCheck() = %v, want nil", err)
	}
}

// TestCheckGateOrderRejectsTamperedSequences exercises the
// permutation invariants directly so the test does not have to
// mutate the package-level GateOrder.  Each case asserts a clean,
// localised failure message so a tampered fixture in a future test
// fails with a useful diagnostic.
func TestCheckGateOrderRejectsTamperedSequences(t *testing.T) {
	canonical := append([]int(nil), GateOrder[:]...)

	cases := []struct {
		name string
		mut  func([]int) []int
		want string
	}{
		{
			name: "wrong length",
			mut:  func(s []int) []int { return s[:63] },
			want: "want 64",
		},
		{
			name: "out of range high",
			mut: func(s []int) []int {
				cp := append([]int(nil), s...)
				cp[5] = 65
				return cp
			},
			want: "out of range",
		},
		{
			name: "out of range low",
			mut: func(s []int) []int {
				cp := append([]int(nil), s...)
				cp[5] = 0
				return cp
			},
			want: "out of range",
		},
		{
			name: "duplicate entry",
			mut: func(s []int) []int {
				cp := append([]int(nil), s...)
				cp[10] = cp[20] // forces duplicate
				return cp
			},
			want: "duplicate",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			seq := c.mut(canonical)
			err := checkGateOrder(seq)
			if err == nil {
				t.Fatalf("checkGateOrder(%v) = nil, want error", seq)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %q does not contain %q", err.Error(), c.want)
			}
		})
	}
}

// TestCheckGateOrderFuzzPermutesPairs is the Phase 9 fuzz test the
// implementation plan calls for.  We do *not* mutate the
// package-level GateOrder (that would spoil downstream tests in the
// same binary).  Instead, we copy GateOrder into a private slice,
// swap each pair (i, j) for i < j, and confirm checkGateOrder still
// accepts the result — because swapping two entries within a valid
// permutation yields another valid permutation.
//
// The complementary fuzz on the *file*-level cross-check
// (AssertGateSequenceFileMatchesGateOrder) is in
// TestAssertGateSequenceFileFuzzRejectsPermutations below and is
// the actual canon-tamper detector: a swapped-pair permutation
// is a valid permutation but no longer matches GateOrder
// element-for-element, so the cross-check must reject it.
func TestCheckGateOrderFuzzPermutesPairs(t *testing.T) {
	canonical := append([]int(nil), GateOrder[:]...)
	for i := 0; i < len(canonical); i++ {
		for j := i + 1; j < len(canonical); j++ {
			swapped := append([]int(nil), canonical...)
			swapped[i], swapped[j] = swapped[j], swapped[i]
			if err := checkGateOrder(swapped); err != nil {
				t.Errorf("checkGateOrder rejected a valid permutation (swap %d <-> %d): %v",
					i, j, err)
			}
		}
	}
}

// TestAssertGateSequenceFileMatchesGateOrderHappyPath writes the
// canonical sequence to a temp file and asserts the cross-check
// accepts it.  This pins the contract that a JSON file shaped per
// trinity.org and matching canon.GateOrder is the canonical input
// the cross-check expects.
func TestAssertGateSequenceFileMatchesGateOrderHappyPath(t *testing.T) {
	path := writeGateSequenceFile(t, GateOrder[:])
	if err := AssertGateSequenceFileMatchesGateOrder(path); err != nil {
		t.Fatalf("cross-check rejected matching file: %v", err)
	}
}

// TestAssertGateSequenceFileFuzzRejectsPermutations is the
// canonical Phase 9 fuzz test: a JSON file whose gate sequence
// equals canon.GateOrder with two entries swapped must be rejected.
// Walking every (i, j) pair with i<j gives 64*63/2 = 2016 cases;
// rejection by all of them rules out a "first divergence wins"
// implementation that misses tail-end swaps.
func TestAssertGateSequenceFileFuzzRejectsPermutations(t *testing.T) {
	canonical := append([]int(nil), GateOrder[:]...)
	for i := 0; i < len(canonical); i++ {
		for j := i + 1; j < len(canonical); j++ {
			swapped := append([]int(nil), canonical...)
			swapped[i], swapped[j] = swapped[j], swapped[i]
			path := writeGateSequenceFile(t, swapped)
			err := AssertGateSequenceFileMatchesGateOrder(path)
			if err == nil {
				t.Errorf("cross-check accepted a swapped permutation (%d <-> %d)",
					i, j)
				continue
			}
			if !strings.Contains(err.Error(), "canon.GateOrder") {
				t.Errorf("error %q does not name canon.GateOrder for swap (%d, %d)",
					err.Error(), i, j)
			}
		}
	}
}

// TestAssertGateSequenceFileRejectsMalformed exercises the file-IO
// and decode failure paths.
func TestAssertGateSequenceFileRejectsMalformed(t *testing.T) {
	tmp := t.TempDir()

	missing := filepath.Join(tmp, "missing.json")
	if err := AssertGateSequenceFileMatchesGateOrder(missing); err == nil {
		t.Errorf("missing file: err = nil, want non-nil")
	}

	bad := filepath.Join(tmp, "bad.json")
	if err := os.WriteFile(bad, []byte("{not-json}"), 0o600); err != nil {
		t.Fatalf("write bad json: %v", err)
	}
	if err := AssertGateSequenceFileMatchesGateOrder(bad); err == nil {
		t.Errorf("malformed JSON: err = nil, want non-nil")
	}

	short := filepath.Join(tmp, "short.json")
	if err := os.WriteFile(short, []byte(`{"gate_sequence":[1,2,3]}`), 0o600); err != nil {
		t.Fatalf("write short json: %v", err)
	}
	if err := AssertGateSequenceFileMatchesGateOrder(short); err == nil {
		t.Errorf("short sequence: err = nil, want non-nil")
	}

	dupe := filepath.Join(tmp, "dupe.json")
	dupeSeq := append([]int(nil), GateOrder[:]...)
	dupeSeq[0] = dupeSeq[1] // create duplicate
	dupeBytes, _ := json.Marshal(map[string]any{"gate_sequence": dupeSeq})
	if err := os.WriteFile(dupe, dupeBytes, 0o600); err != nil {
		t.Fatalf("write dupe json: %v", err)
	}
	if err := AssertGateSequenceFileMatchesGateOrder(dupe); err == nil {
		t.Errorf("duplicate-entry sequence: err = nil, want non-nil")
	}
}

func writeGateSequenceFile(t *testing.T, seq []int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gate_sequence.json")
	body, err := json.Marshal(map[string]any{"gate_sequence": seq})
	if err != nil {
		t.Fatalf("marshal gate sequence: %v", err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write gate sequence: %v", err)
	}
	return path
}

// TestIsLowerSnake exercises the canon-identifier shape predicate.
func TestIsLowerSnake(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"sun", true},
		{"north_node_mean", true},
		{"", false},
		{"Sun", false},
		{"sun-moon", false},
		{"sun ", false},
		{"sun1", false}, // digits not allowed in canon identifiers
	}
	for _, c := range cases {
		if got := isLowerSnake(c.in); got != c.want {
			t.Errorf("isLowerSnake(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
