package canon

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// selfcheck.go implements the Phase 9 boot-time canon self-check
// and the optional gate-sequence-file cross-check.  Per the Phase 9
// plan deliverable:
//
//   * pkg/canon is the single source of every calculation constant.
//   * The engine refuses to boot if the compiled-in constants fail
//     their self-checks.
//   * The legacy gate-sequence JSON file becomes a *sanity check*
//     against the compiled-in GateOrder rather than a source of
//     values.
//
// SelfCheck validates the in-memory constants only.  It does not
// read any files and consumes no environment variables, so it is
// safe to call from arbitrary contexts (boot, tests, fuzz).

// SelfCheck verifies that every compiled-in constant in this
// package satisfies the canonical invariants pinned by trinity.org:
//
//   * GateOrder has 64 entries, every value in [1, 64], no duplicates.
//   * SignOrder has 12 entries, all distinct, all lowercase
//     snake_case strings.
//   * AstrologyObjectOrder has 13 entries, all distinct.
//   * HDSnapshotOrder has 13 entries, all distinct.
//   * CenterOrder has 9 entries, all distinct.
//   * MotorCenters is a 4-element subset of CenterOrder.
//   * ChannelTable has 36 entries; for each entry GateA < GateB,
//     both gates in [1,64], both centers in CenterOrder, and ID
//     equals "GateA-GateB".
//   * MandalaAnchorDeg in [0, 360).
//   * GateWidthDeg = 360 / 64 within float tolerance.
//   * LineWidthDeg = GateWidthDeg / 6 within float tolerance.
//
// Returns the first violation as an error.  A successful self-check
// returns nil.
func SelfCheck() error {
	if err := checkGateOrder(GateOrder[:]); err != nil {
		return fmt.Errorf("canon.GateOrder: %w", err)
	}
	if err := checkSignOrder(SignOrder[:]); err != nil {
		return fmt.Errorf("canon.SignOrder: %w", err)
	}
	if err := checkUniqueLen(stringSlice(AstrologyObjectOrder[:]), 13); err != nil {
		return fmt.Errorf("canon.AstrologyObjectOrder: %w", err)
	}
	if err := checkUniqueLen(stringSlice(HDSnapshotOrder[:]), 13); err != nil {
		return fmt.Errorf("canon.HDSnapshotOrder: %w", err)
	}
	if err := checkUniqueLen(stringSlice(CenterOrder[:]), 9); err != nil {
		return fmt.Errorf("canon.CenterOrder: %w", err)
	}
	centerSet := setOf(stringSlice(CenterOrder[:]))
	if len(MotorCenters) != 4 {
		return fmt.Errorf("canon.MotorCenters: want 4 entries, got %d", len(MotorCenters))
	}
	for _, m := range MotorCenters {
		if !centerSet[m] {
			return fmt.Errorf("canon.MotorCenters: %q not in CenterOrder", m)
		}
	}
	if err := checkChannelTable(centerSet); err != nil {
		return fmt.Errorf("canon.ChannelTable: %w", err)
	}
	if MandalaAnchorDeg < 0 || MandalaAnchorDeg >= 360 {
		return fmt.Errorf("canon.MandalaAnchorDeg = %v, want [0, 360)", MandalaAnchorDeg)
	}
	const eps = 1e-9
	if math.Abs(GateWidthDeg-360.0/64.0) > eps {
		return fmt.Errorf("canon.GateWidthDeg = %v, want %v", GateWidthDeg, 360.0/64.0)
	}
	if math.Abs(LineWidthDeg-GateWidthDeg/6.0) > eps {
		return fmt.Errorf("canon.LineWidthDeg = %v, want GateWidthDeg/6 = %v",
			LineWidthDeg, GateWidthDeg/6.0)
	}
	return nil
}

// checkGateOrder verifies the 64-entry permutation of [1, 64].
func checkGateOrder(seq []int) error {
	if len(seq) != 64 {
		return fmt.Errorf("want 64 entries, got %d", len(seq))
	}
	seen := make(map[int]int, 64)
	for i, v := range seq {
		if v < 1 || v > 64 {
			return fmt.Errorf("entry at index %d out of range: %d", i, v)
		}
		if prev, dup := seen[v]; dup {
			return fmt.Errorf("duplicate entry %d at indexes %d and %d", v, prev, i)
		}
		seen[v] = i
	}
	return nil
}

func checkSignOrder(signs []string) error {
	if err := checkUniqueLen(signs, 12); err != nil {
		return err
	}
	for i, s := range signs {
		if !isLowerSnake(s) {
			return fmt.Errorf("sign at index %d not lowercase snake_case: %q", i, s)
		}
	}
	return nil
}

func checkChannelTable(centers map[string]bool) error {
	if len(ChannelTable) != 36 {
		return fmt.Errorf("want 36 entries, got %d", len(ChannelTable))
	}
	seenIDs := make(map[string]int, 36)
	for i, c := range ChannelTable {
		if c.GateA < 1 || c.GateA > 64 || c.GateB < 1 || c.GateB > 64 {
			return fmt.Errorf("entry %d: gate out of range: %+v", i, c)
		}
		if c.GateA >= c.GateB {
			return fmt.Errorf("entry %d: gate_a >= gate_b: %+v", i, c)
		}
		if !centers[c.CenterA] {
			return fmt.Errorf("entry %d: center_a %q not in CenterOrder", i, c.CenterA)
		}
		if !centers[c.CenterB] {
			return fmt.Errorf("entry %d: center_b %q not in CenterOrder", i, c.CenterB)
		}
		if c.CenterA == c.CenterB {
			return fmt.Errorf("entry %d: center_a == center_b (%q)", i, c.CenterA)
		}
		wantID := fmt.Sprintf("%d-%d", c.GateA, c.GateB)
		if c.ID != wantID {
			return fmt.Errorf("entry %d: ID %q != %q", i, c.ID, wantID)
		}
		if prev, dup := seenIDs[c.ID]; dup {
			return fmt.Errorf("duplicate channel ID %q at indexes %d and %d", c.ID, prev, i)
		}
		seenIDs[c.ID] = i
	}
	return nil
}

func checkUniqueLen(items []string, wantLen int) error {
	if len(items) != wantLen {
		return fmt.Errorf("want %d entries, got %d", wantLen, len(items))
	}
	seen := make(map[string]int, wantLen)
	for i, s := range items {
		if prev, dup := seen[s]; dup {
			return fmt.Errorf("duplicate entry %q at indexes %d and %d", s, prev, i)
		}
		seen[s] = i
	}
	return nil
}

// stringSlice copies a fixed-size string array into a slice so the
// helpers above can take a single concrete []string parameter
// without requiring callers to perform the conversion at every site.
func stringSlice(arr []string) []string {
	return append([]string(nil), arr...)
}

func setOf(items []string) map[string]bool {
	out := make(map[string]bool, len(items))
	for _, s := range items {
		out[s] = true
	}
	return out
}

// isLowerSnake reports whether s is a non-empty string consisting
// only of lowercase ASCII letters and underscores.  Used to gate
// canonical identifier shape (canon §"Identifier Normalization").
func isLowerSnake(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r == '_' {
			continue
		}
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}

// AssertGateSequenceFileMatchesGateOrder loads a JSON file with the
// canonical gate-sequence shape ({"gate_sequence": [...]}) and
// verifies it equals the compiled-in canon.GateOrder element-for-
// element.  Phase 9 turns the legacy gate-sequence JSON file into
// a sanity-check artefact: when this function is invoked at boot or
// in CI, a divergence between the file and the compiled canon is a
// configuration bug that must abort the engine.
//
// The function is exported as a standalone tool so callers can opt
// in (e.g. a CLI sanity-check, a CI step, a deployment validator)
// without forcing the trinity HTTP path to depend on the file.  The
// trinity /manifest path itself never reads the file: pkg/canon
// is the authoritative source.
func AssertGateSequenceFileMatchesGateOrder(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var payload struct {
		GateSequence []int `json:"gate_sequence"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	if err := checkGateOrder(payload.GateSequence); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if len(payload.GateSequence) != len(GateOrder) {
		return fmt.Errorf("%s: gate_sequence length %d != canon.GateOrder length %d",
			path, len(payload.GateSequence), len(GateOrder))
	}
	for i, v := range payload.GateSequence {
		if v != GateOrder[i] {
			return fmt.Errorf("%s: gate_sequence[%d] = %d, canon.GateOrder[%d] = %d",
				path, i, v, i, GateOrder[i])
		}
	}
	return nil
}
