package canon

import (
	"math"
	"strings"
	"testing"
)

// TestGateOrderIsBijectionOver1To64 verifies that GateOrder contains
// every integer in [1, 64] exactly once.  Trinity.org Document 06
// treats GateOrder as a permutation of the 64-gate set; a duplicate
// or a missing entry would silently corrupt every Human Design
// mapping.
func TestGateOrderIsBijectionOver1To64(t *testing.T) {
	if got, want := len(GateOrder), 64; got != want {
		t.Fatalf("GateOrder length = %d, want %d", got, want)
	}
	seen := [65]bool{}
	for i, gate := range GateOrder {
		if gate < 1 || gate > 64 {
			t.Errorf("GateOrder[%d] = %d, want in [1,64]", i, gate)
			continue
		}
		if seen[gate] {
			t.Errorf("GateOrder[%d] = %d (duplicate)", i, gate)
		}
		seen[gate] = true
	}
	for g := 1; g <= 64; g++ {
		if !seen[g] {
			t.Errorf("gate %d missing from GateOrder", g)
		}
	}
}

// TestGateOrderAnchorFollowsTrinityOrg locks in the two canon
// anchors called out by trinity.org §"Canonical Human Design Gate
// Order" (lines 297-300): Gate 38 at 277.5° and Gate 41 at 300.0°.
// If this test fails the constants have drifted from the canon.
func TestGateOrderAnchorFollowsTrinityOrg(t *testing.T) {
	cases := []struct {
		gate   int
		wantAt float64
	}{
		{38, 277.5},
		{41, 300.0},
	}
	for _, tc := range cases {
		idx := -1
		for i, g := range GateOrder {
			if g == tc.gate {
				idx = i
				break
			}
		}
		if idx < 0 {
			t.Errorf("gate %d not found in GateOrder", tc.gate)
			continue
		}
		got := MandalaAnchorDeg + float64(idx)*GateWidthDeg
		for got >= 360 {
			got -= 360
		}
		if math.Abs(got-tc.wantAt) > 1e-9 {
			t.Errorf("gate %d starts at %.6f°, want %.6f°", tc.gate, got, tc.wantAt)
		}
	}
}

// TestMandalaConstantsSelfConsistent confirms that the three mandala
// numeric constants relate correctly: 64 gates × 5.625° = 360°, and
// 1 gate = 6 × line width.
func TestMandalaConstantsSelfConsistent(t *testing.T) {
	if math.Abs(64*GateWidthDeg-360.0) > 1e-9 {
		t.Errorf("64 * GateWidthDeg = %.6f, want 360.0", 64*GateWidthDeg)
	}
	if math.Abs(6*LineWidthDeg-GateWidthDeg) > 1e-9 {
		t.Errorf("6 * LineWidthDeg = %.6f, want GateWidthDeg=%.6f", 6*LineWidthDeg, GateWidthDeg)
	}
	if MandalaAnchorDeg < 0 || MandalaAnchorDeg >= 360 {
		t.Errorf("MandalaAnchorDeg = %.6f out of [0, 360)", MandalaAnchorDeg)
	}
}

// TestSignOrderIsCanonical verifies the tropical zodiac ordering and
// forbids drift from lowercase snake_case identifiers (per trinity
// "Identifier Normalization", lines 430-437).
func TestSignOrderIsCanonical(t *testing.T) {
	want := [12]string{
		"aries", "taurus", "gemini", "cancer",
		"leo", "virgo", "libra", "scorpio",
		"sagittarius", "capricorn", "aquarius", "pisces",
	}
	if SignOrder != want {
		t.Fatalf("SignOrder = %v\nwant        %v", SignOrder, want)
	}
	for i, s := range SignOrder {
		if s == "" {
			t.Errorf("SignOrder[%d] is empty", i)
		}
		if strings.ToLower(s) != s {
			t.Errorf("SignOrder[%d] = %q is not lowercase", i, s)
		}
	}
}

// TestAstrologyAndHDOrderingsHaveExpectedLengths guards against any
// silent add/remove in the canonical object arrays.
func TestAstrologyAndHDOrderingsHaveExpectedLengths(t *testing.T) {
	if got, want := len(AstrologyObjectOrder), 13; got != want {
		t.Errorf("AstrologyObjectOrder length = %d, want %d", got, want)
	}
	if got, want := len(HDSnapshotOrder), 13; got != want {
		t.Errorf("HDSnapshotOrder length = %d, want %d", got, want)
	}
}

// TestAstrologyObjectOrderStartsWithSun mirrors Document 07's
// explicit emission order: sun first, earth last.
func TestAstrologyObjectOrderStartsWithSun(t *testing.T) {
	if AstrologyObjectOrder[0] != "sun" {
		t.Errorf("AstrologyObjectOrder[0] = %q, want sun", AstrologyObjectOrder[0])
	}
	if AstrologyObjectOrder[len(AstrologyObjectOrder)-1] != "earth" {
		t.Errorf("AstrologyObjectOrder last = %q, want earth",
			AstrologyObjectOrder[len(AstrologyObjectOrder)-1])
	}
}

// TestCenterOrderIsCanonical verifies the canonical center
// sequence head..root.
func TestCenterOrderIsCanonical(t *testing.T) {
	want := [9]string{
		"head", "ajna", "throat", "g", "ego",
		"solar_plexus", "sacral", "spleen", "root",
	}
	if CenterOrder != want {
		t.Errorf("CenterOrder = %v\nwant        %v", CenterOrder, want)
	}
}

// TestMotorCentersSubsetOfCenterOrder enforces that every center
// declared a motor is also present in the canonical center list.
// Without this invariant, Document 05's type-derivation algorithm
// (motor-to-throat connectivity) becomes undefined.
func TestMotorCentersSubsetOfCenterOrder(t *testing.T) {
	centerSet := make(map[string]bool, len(CenterOrder))
	for _, c := range CenterOrder {
		centerSet[c] = true
	}
	for _, m := range MotorCenters {
		if !centerSet[m] {
			t.Errorf("MotorCenters contains %q which is not in CenterOrder", m)
		}
	}
	// Guard against accidental addition of throat (a non-motor per
	// canon) if ever tempted.
	for _, m := range MotorCenters {
		if m == "throat" {
			t.Errorf("MotorCenters must not contain throat")
		}
	}
}

// TestChannelTableIsWellFormed enforces every canon invariant on
// the 36-channel lookup: size, gate-pair bounds, ascending ordering,
// ID format, and center-name membership.
func TestChannelTableIsWellFormed(t *testing.T) {
	if got, want := len(ChannelTable), 36; got != want {
		t.Fatalf("ChannelTable length = %d, want %d", got, want)
	}
	centerSet := make(map[string]bool, len(CenterOrder))
	for _, c := range CenterOrder {
		centerSet[c] = true
	}
	gateInOrder := func(g int) bool {
		for _, x := range GateOrder {
			if x == g {
				return true
			}
		}
		return false
	}
	seenID := make(map[string]bool, len(ChannelTable))
	for i, ch := range ChannelTable {
		if ch.GateA < 1 || ch.GateA > 64 {
			t.Errorf("ChannelTable[%d].GateA=%d out of [1,64]", i, ch.GateA)
		}
		if ch.GateB < 1 || ch.GateB > 64 {
			t.Errorf("ChannelTable[%d].GateB=%d out of [1,64]", i, ch.GateB)
		}
		if ch.GateA >= ch.GateB {
			t.Errorf("ChannelTable[%d] gate pair (%d, %d) not strictly ascending",
				i, ch.GateA, ch.GateB)
		}
		if !gateInOrder(ch.GateA) {
			t.Errorf("ChannelTable[%d].GateA=%d missing from GateOrder", i, ch.GateA)
		}
		if !gateInOrder(ch.GateB) {
			t.Errorf("ChannelTable[%d].GateB=%d missing from GateOrder", i, ch.GateB)
		}
		if !centerSet[ch.CenterA] {
			t.Errorf("ChannelTable[%d].CenterA=%q not in CenterOrder", i, ch.CenterA)
		}
		if !centerSet[ch.CenterB] {
			t.Errorf("ChannelTable[%d].CenterB=%q not in CenterOrder", i, ch.CenterB)
		}
		// Canon ID format: ascending gate pair joined by a hyphen.
		expectedID := ""
		if ch.GateA < ch.GateB {
			expectedID = itoa(ch.GateA) + "-" + itoa(ch.GateB)
		}
		if ch.ID != expectedID {
			t.Errorf("ChannelTable[%d].ID=%q, want %q", i, ch.ID, expectedID)
		}
		if seenID[ch.ID] {
			t.Errorf("ChannelTable[%d].ID=%q is a duplicate", i, ch.ID)
		}
		seenID[ch.ID] = true
	}
}

// itoa avoids importing strconv for a single call site.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
