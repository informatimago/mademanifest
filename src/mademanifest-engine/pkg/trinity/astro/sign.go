// Package astro builds the astrology section of a Trinity success
// envelope (output.Astrology) from a validated input payload.
//
// Phase 4 makes the astrology section real – the placeholder values
// emitted by output.NewPlaceholderSuccess are replaced with computed
// longitudes, signs, houses, and angles.  Phases 5-8 do the same for
// human_design and gene_keys.
//
// All input/output conversions happen at the package boundary
// (pkg/trinity/input.Payload in, pkg/trinity/output.Astrology out);
// the package depends on pkg/canon for canonical constants and on
// pkg/ephemeris + Swiss Ephemeris for the underlying numerics.
package astro

import "mademanifest-engine/pkg/canon"

// SignFor returns the canonical lowercase sign identifier for an
// ecliptic longitude in [0, 360).  Each sign spans 30° with start-
// inclusive / end-exclusive intervals: 0° is aries, 29.999...° is
// aries, 30° is taurus, …, 359.999...° is pisces.  This mirrors
// trinity.org §"Astrology Mappings" lines 351-355.
//
// The function does *not* normalise the input; callers must already
// have applied normalizeDeg so the value falls inside [0, 360).
// Out-of-range inputs return the empty string so calling code can
// detect the bug rather than emit a silently-wrong sign.
func SignFor(longitude float64) string {
	if longitude < 0 || longitude >= 360 {
		return ""
	}
	idx := int(longitude / 30.0)
	if idx < 0 || idx > 11 {
		return ""
	}
	return canon.SignOrder[idx]
}
