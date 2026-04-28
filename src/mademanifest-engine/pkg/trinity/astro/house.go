package astro

// HouseFor returns the canonical house number (1..12) for a
// normalised ecliptic longitude given the 12 house cusps in
// canonical order: cusps[0] is the cusp of house 1, cusps[1] is
// the cusp of house 2, ..., cusps[11] is the cusp of house 12.
//
// A2 (RESOLVED, Document 12 D21): every cusp is start-inclusive
// and the next cusp is end-exclusive; the cusp-12-to-cusp-1
// boundary is the chart wrap-around case and is handled explicitly
// so wraparound charts (where the chart spans the 0/360° boundary)
// place longitudes in the correct house.  No epsilon, midpoint, or
// dual-inclusive variant is permitted.
//
// Returns 0 if the longitude does not fall in any house, which
// indicates a degenerate cusp table (callers should treat this as
// a programming bug, not as a chart property).
//
// Implementation:
//   - Iterate house i from 1 to 12.
//   - lo = cusps[i-1], hi = cusps[i % 12] (so house 12's hi wraps
//     back to house 1's cusp).
//   - If lo <= hi the segment does not cross 360°: longitude is
//     in this house when lo <= longitude < hi.
//   - If lo >  hi the segment crosses 360°: longitude is in this
//     house when longitude >= lo OR longitude < hi.
//
// The cusp-array index is 0-based here for Go convention; the
// returned house number is 1-based per canon.
func HouseFor(longitude float64, cusps [12]float64) int {
	for i := 0; i < 12; i++ {
		lo := cusps[i]
		hi := cusps[(i+1)%12]
		var in bool
		if lo <= hi {
			in = lo <= longitude && longitude < hi
		} else {
			in = longitude >= lo || longitude < hi
		}
		if in {
			return i + 1
		}
	}
	return 0
}
