package astro

import "testing"

// TestHouseForUniformCusps walks evenly-spaced cusps so each house
// is exactly 30° wide.  The boundary cases pin the canonical
// start-inclusive / end-exclusive semantics.
func TestHouseForUniformCusps(t *testing.T) {
	var cusps [12]float64
	for i := 0; i < 12; i++ {
		cusps[i] = float64(i * 30)
	}
	cases := []struct {
		long float64
		want int
	}{
		{0.0, 1},          // exact cusp 1
		{15.0, 1},         // mid house 1
		{29.999, 1},       // last bit of house 1
		{30.0, 2},         // exact cusp 2 -> house 2
		{45.0, 2},
		{60.0, 3},
		{180.0, 7},        // exact cusp 7
		{329.999, 11},
		{330.0, 12},       // exact cusp 12
		{359.999, 12},     // last bit before wrap
	}
	for _, tc := range cases {
		if got := HouseFor(tc.long, cusps); got != tc.want {
			t.Errorf("HouseFor(%v) = %d, want %d", tc.long, got, tc.want)
		}
	}
}

// TestHouseForWrapsThroughZero stresses the 12-to-1 wrap case
// explicitly.  Cusp 1 starts at 350°; the chart spans the 0/360°
// boundary so cusp 2 lives at 20°.  House 1 owns 350°..360° AND
// 0°..20°.
func TestHouseForWrapsThroughZero(t *testing.T) {
	var cusps [12]float64
	cusps[0] = 350.0 // cusp 1
	for i := 1; i < 12; i++ {
		cusps[i] = float64((350 + i*30) % 360)
	}
	// House 1 owns 350° .. 20° (across the 0°/360° wrap).
	cases := []struct {
		long float64
		want int
	}{
		{349.999, 12},     // just before cusp 1 -> house 12 wraps
		{350.0, 1},        // cusp 1 inclusive
		{355.0, 1},        // mid house 1, before wrap
		{0.0, 1},          // wrapped through 360°
		{19.999, 1},       // last bit of house 1 after wrap
		{20.0, 2},         // cusp 2
		{50.0, 3},
		{349.999, 12},     // re-check the closing edge
	}
	for _, tc := range cases {
		if got := HouseFor(tc.long, cusps); got != tc.want {
			t.Errorf("HouseFor(%v) = %d, want %d", tc.long, got, tc.want)
		}
	}
}

// TestHouseForReturnsZeroOnDegenerateCusps confirms the bug-detect
// path: if every cusp shares the same longitude, no longitude can
// fall inside any half-open interval and the result is 0 to flag
// the problem to the caller.
func TestHouseForReturnsZeroOnDegenerateCusps(t *testing.T) {
	var cusps [12]float64
	for i := range cusps {
		cusps[i] = 100.0
	}
	if got := HouseFor(50.0, cusps); got != 0 {
		t.Errorf("HouseFor with degenerate cusps = %d, want 0", got)
	}
}
