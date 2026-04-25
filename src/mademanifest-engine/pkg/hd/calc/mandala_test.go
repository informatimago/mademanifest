package calc

import (
	"math"
	"testing"

	"mademanifest-engine/pkg/canon"
)

// TestMapToGateLineAnchorIsGate38Line1 pins the canon anchor
// (277.5°) to the start of gate 38, which is the first entry in
// canon.GateOrder under the Trinity canon.  trinity.org lines
// 297-300 explicitly fix gate 38 at 277.5° and gate 41 at 300.0°.
func TestMapToGateLineAnchorIsGate38Line1(t *testing.T) {
	gate, line := MapToGateLine(canon.MandalaAnchorDeg)
	if gate != 38 || line != 1 {
		t.Fatalf("MapToGateLine(%.4f) = gate %d line %d, want gate 38 line 1",
			canon.MandalaAnchorDeg, gate, line)
	}
	// trinity.org cross-check: gate 41 is the fifth entry in
	// GateOrder, so its anchor sits at 277.5 + 4*5.625 = 300.0°.
	gate, line = MapToGateLine(300.0)
	if gate != 41 || line != 1 {
		t.Fatalf("MapToGateLine(300.0) = gate %d line %d, want gate 41 line 1",
			gate, line)
	}
}

// TestMapToGateLineAnchorMinusEpsilonWrapsToLastGate exercises the
// canonical boundary rule "an interval end belongs to the next
// segment".  An infinitesimal step before the anchor sits in the
// last gate of the preceding cycle, line 6 (the last line in that
// gate).  The last gate in canon.GateOrder is index 63, value 58.
func TestMapToGateLineAnchorMinusEpsilonWrapsToLastGate(t *testing.T) {
	const eps = 1e-9
	got := canon.MandalaAnchorDeg - eps
	gate, line := MapToGateLine(got)
	wantGate := canon.GateOrder[63]
	if gate != wantGate || line != 6 {
		t.Fatalf("MapToGateLine(anchor - ε = %.10f) = gate %d line %d, want gate %d line 6",
			got, gate, line, wantGate)
	}
}

// TestMapToGateLinePerEntryStart asserts that every canonical gate
// start in canon.GateOrder maps back to the matching gate and to
// line 1.  This pins the bidirectional consistency between the
// mandala formula and the lookup table.
func TestMapToGateLinePerEntryStart(t *testing.T) {
	for idx, want := range canon.GateOrder {
		long := canon.MandalaAnchorDeg + float64(idx)*canon.GateWidthDeg
		gate, line := MapToGateLine(long)
		if gate != want || line != 1 {
			t.Errorf("MapToGateLine(%.6f) [index %d] = gate %d line %d, want gate %d line 1",
				long, idx, gate, line, want)
		}
	}
}

// TestMapToGateLinePerEntryEndIsNextStart verifies the
// start-inclusive / end-exclusive interval rule.  The longitude
// exactly one gate-width past gate i's start belongs to gate i+1
// line 1, never to gate i line 6.
func TestMapToGateLinePerEntryEndIsNextStart(t *testing.T) {
	for idx := 0; idx < 63; idx++ {
		long := canon.MandalaAnchorDeg + float64(idx+1)*canon.GateWidthDeg
		gate, line := MapToGateLine(long)
		want := canon.GateOrder[idx+1]
		if gate != want || line != 1 {
			t.Errorf("MapToGateLine(%.6f) [boundary after index %d] = gate %d line %d, want gate %d line 1",
				long, idx, gate, line, want)
		}
	}
}

// TestMapToGateLineLineBoundaries exercises the per-line interval
// rule inside a single gate.  Within gate 38 (indices 0..5 of the
// six lines), each line k starts at anchor + k * LineWidthDeg.
func TestMapToGateLineLineBoundaries(t *testing.T) {
	for k := 0; k < 6; k++ {
		long := canon.MandalaAnchorDeg + float64(k)*canon.LineWidthDeg
		gate, line := MapToGateLine(long)
		if gate != 38 || line != k+1 {
			t.Errorf("MapToGateLine(%.6f) [line %d start] = gate %d line %d, want gate 38 line %d",
				long, k+1, gate, line, k+1)
		}
	}
	// One ε before the next gate's start: gate 38 line 6.
	const eps = 1e-9
	long := canon.MandalaAnchorDeg + canon.GateWidthDeg - eps
	gate, line := MapToGateLine(long)
	if gate != 38 || line != 6 {
		t.Errorf("MapToGateLine(%.10f) = gate %d line %d, want gate 38 line 6", long, gate, line)
	}
}

// TestMapToGateLineNormalisesInput verifies that out-of-range
// longitudes (negative, > 360°) are folded to the canonical
// [0, 360) before lookup.
func TestMapToGateLineNormalisesInput(t *testing.T) {
	cases := []struct {
		name string
		in   float64
		want float64
	}{
		{"plus 360", canon.MandalaAnchorDeg + 360.0, canon.MandalaAnchorDeg},
		{"minus 360", canon.MandalaAnchorDeg - 360.0, canon.MandalaAnchorDeg},
		{"plus 720", canon.MandalaAnchorDeg + 720.0, canon.MandalaAnchorDeg},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gate, line := MapToGateLine(c.in)
			wantGate, wantLine := MapToGateLine(c.want)
			if gate != wantGate || line != wantLine {
				t.Errorf("MapToGateLine(%.6f) = gate %d line %d, want gate %d line %d",
					c.in, gate, line, wantGate, wantLine)
			}
		})
	}
}

// TestMapToGateLineNoFractionalLeak guards against the trivial
// wrong implementation that uses the input longitude unmodified
// (without subtracting the anchor).  At longitude 0° we expect
// canonical gate sequence index = floor((0 - 277.5) mod 360 / 5.625)
// = floor(82.5 / 5.625) = 14 → GateOrder[14] = 25.
func TestMapToGateLineNoFractionalLeak(t *testing.T) {
	gate, line := MapToGateLine(0.0)
	wantIndex := int(math.Floor(normalizeDeg(0.0-canon.MandalaAnchorDeg) / canon.GateWidthDeg))
	wantGate := canon.GateOrder[wantIndex]
	if gate != wantGate {
		t.Errorf("MapToGateLine(0.0) = gate %d, want gate %d", gate, wantGate)
	}
	if line < 1 || line > 6 {
		t.Errorf("MapToGateLine(0.0) line = %d, must be in 1..6", line)
	}
}
