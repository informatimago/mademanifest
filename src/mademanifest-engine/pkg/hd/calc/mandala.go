package calc

import (
	"math"

	"mademanifest-engine/pkg/canon"
)

// mandala.go implements the canonical Trinity mandala lookup that
// converts an absolute ecliptic longitude into a Human Design
// (gate, line) pair.  Every constant the function consumes comes
// from pkg/canon — the function MUST NOT accept input-driven
// parameters, per Phase 6 of the implementation plan
// (trinity.org §"Human Design Gate And Line Mapping").
//
// Canon rules (trinity.org lines 273-294):
//
//   * mandala anchor = canon.MandalaAnchorDeg (277.5°, gate 38)
//   * gate width     = canon.GateWidthDeg (5.625°)
//   * line width     = canon.LineWidthDeg (0.9375°)
//   * direction      = ascending ecliptic longitude
//   * intervals      = start-inclusive / end-exclusive
//   * gate order     = canon.GateOrder (64-entry fixed lookup)
//   * boundary rule  = "an interval end belongs to the next segment"
//
// The boundary rule means a longitude that lands exactly on a gate
// boundary belongs to the *new* gate, never to the gate that just
// ended.  Likewise for line boundaries inside a gate.

// MapToGateLine converts an absolute ecliptic longitude in degrees
// into (gate ∈ 1..64, line ∈ 1..6).  The input is normalised to
// [0, 360) before lookup so the function behaves identically for
// any equivalent longitude (e.g. -10 ≡ 350).
//
// Determinism guarantees:
//
//   * Result depends only on the input longitude and the compiled-in
//     canon constants.  No environment variables, no I/O, no
//     time-of-day side effects.
//   * Two calls with bit-identical inputs always produce identical
//     outputs.
func MapToGateLine(longitudeDeg float64) (gate, line int) {
	r := normalizeDeg(longitudeDeg - canon.MandalaAnchorDeg)

	// floor(r / GateWidthDeg) gives the zero-based gate index in
	// canon.GateOrder.  With r ∈ [0, 360) and GateWidthDeg = 5.625
	// the index is mathematically in [0, 63].  We clamp defensively
	// against accumulated floating-point error at r ≈ 360 - ε,
	// which can produce r / 5.625 slightly above 63.
	gateIndex := int(math.Floor(r / canon.GateWidthDeg))
	if gateIndex < 0 {
		gateIndex = 0
	} else if gateIndex > 63 {
		gateIndex = 63
	}

	// Line width is exactly GateWidthDeg / 6, so r mod GateWidthDeg
	// gives the offset inside the current gate, and floor of that
	// over LineWidthDeg gives the line index in [0, 5].
	offsetInGate := r - float64(gateIndex)*canon.GateWidthDeg
	lineIndex := int(math.Floor(offsetInGate / canon.LineWidthDeg))
	if lineIndex < 0 {
		lineIndex = 0
	} else if lineIndex > 5 {
		lineIndex = 5
	}

	return canon.GateOrder[gateIndex], lineIndex + 1
}
