// Package calc holds the canonical Human Design calculation
// primitives used by the Trinity engine.  Phase 5 introduces the
// design-time bisection solver; later phases add gate/line mapping
// and structural derivations under this same package tree.
//
// The solver here replaces the parameter-driven implementation in
// pkg/human_design/SolveDesignTime.  The legacy entry point still
// exists for the Golden PoC code path but is no longer reachable
// from the HTTP handler; it will be retired with the rest of the
// PoC scaffolding in Phase 12.
//
// All canon constants are baked in.  trinity.org §"Human Design
// Design-Time Derivation" lines 252-271 pins:
//
//   * sun-offset target  = exactly 88.0 degrees earlier than birth Sun
//   * search direction   = backward in time only
//   * initial bracket    = centered approximately 88 days before birth
//   * root-finding       = iterative bisection (no secant, no fixed
//                          subtractions, no precomputed lookups)
//   * stop conditions    = |sun(t) - target| < 0.0001 degrees
//                          OR remaining interval width < 1 second
//
// A3 (RESOLVED, Document 12 D22): when the bisection stops because the
// remaining interval width is < 1 second, the canonical Design time is
// the *lower bound* of the final search interval – not the midpoint,
// not the upper bound, not nearest-second rounding.  The solver returns
// that lower bound as a Julian Day; output.DesignTime.MarshalJSON then
// truncates to whole-second precision when serialising.
package calc

import (
	"errors"
	"fmt"
	"math"
)

// Canon-pinned solver constants.  Each value is sourced directly
// from trinity.org and may not be overridden at runtime.  The
// dimensional units are documented inline so callers cannot mistake
// "1 second" for one Julian Day.
const (
	// SunOffsetDeg is the canonical 88° backward offset between the
	// birth Sun longitude and the design-time Sun longitude.
	SunOffsetDeg = 88.0

	// StopAbsSunDiffDeg is the canonical convergence threshold on
	// the absolute Sun-longitude difference from the target.
	StopAbsSunDiffDeg = 0.0001

	// StopBracketSeconds is the canonical convergence threshold on
	// the remaining bracket width in real-time seconds.
	StopBracketSeconds = 1.0

	// initialBracketHalfWidthDays is the half-width of the initial
	// search bracket centered ~88 days before birth.  trinity.org
	// allows "approximately 88 days back"; we use a 5-day half-width
	// (10-day total bracket) which guarantees the 88° offset falls
	// strictly inside the initial bracket for every plausible
	// terrestrial Sun-rate without resorting to expansion iterations
	// in the canonical case.
	initialBracketHalfWidthDays = 5.0

	// secondsPerDay is the Julian Day to second conversion factor.
	secondsPerDay = 86400.0

	// maxBracketExpansion bounds the number of times the initial
	// bracket may be widened on each side if the canonical 10-day
	// window does not bracket the root.  Each expansion adds 2 days
	// per side; sixteen rounds therefore widens the bracket to 64
	// days per side which exceeds any realistic Sun-motion edge
	// case.  Beyond that we abort with an error rather than loop
	// forever.
	maxBracketExpansion = 16

	// bracketExpansionStepDays is the per-side widening step used by
	// the bracket-expansion fallback.
	bracketExpansionStepDays = 2.0

	// maxBisectionIterations bounds the bisection inner loop as a
	// belt-and-braces against numerical pathologies.  log2(10 days /
	// 1 second) ≈ 19.7 iterations; 60 is well above any plausible
	// canonical convergence depth.
	maxBisectionIterations = 60
)

// SunLongitudeFunc returns the geocentric ecliptic Sun longitude in
// degrees, normalised to the canonical [0, 360) range, at a given
// Julian Day.  Implementations are expected to be deterministic:
// repeated calls with the same input return the same output.
type SunLongitudeFunc func(jd float64) float64

// Diagnostics captures auxiliary information about a solver run.
// Production code (the HTTP handler) discards this; tests use it
// to enforce the canonical iteration discipline (forbidden-shortcut
// detection, monotonic bracket shrinkage, final-width bound).
type Diagnostics struct {
	// SunFuncCalls is the total number of SunLongitudeFunc
	// invocations made during the solve, including the initial
	// target capture, bracket-establishment evaluations, and every
	// bisection step.
	SunFuncCalls int

	// BracketIterations is the number of bisection-loop passes
	// (one mid-evaluation per pass).  Distinct from SunFuncCalls
	// because the initial-bracket setup also contributes calls.
	BracketIterations int

	// BracketExpansions is the number of times the initial bracket
	// had to be widened beyond the canonical 10-day window before
	// the root was bracketed.  In the typical canonical case this
	// is zero; non-zero values flag input regimes where the Sun's
	// instantaneous rate drives the 88° offset outside the nominal
	// 88-day distance.
	BracketExpansions int

	// FinalBracketDays is the bracket width (upper - lower) at the
	// moment the loop exited, in Julian Days.  The canonical exit
	// rule guarantees this is < StopBracketSeconds / 86400 unless
	// the early |diff| stop fired first, in which case it is the
	// width at the iteration that triggered the early exit.
	FinalBracketDays float64

	// FinalAbsDiffDeg is the absolute Sun-longitude difference from
	// the target at the chosen Julian Day.  The canonical exit rule
	// guarantees this is < StopAbsSunDiffDeg unless the width-stop
	// fired first.
	FinalAbsDiffDeg float64

	// FinalLowerJD and FinalUpperJD are the lower and upper bounds
	// of the final search interval at the moment the loop exited.
	// Exposed so canon-A3/D22 regression sentinels can assert that
	// the returned JD equals FinalLowerJD exactly (any midpoint or
	// upper-bound regression would emit a different value).
	FinalLowerJD float64
	FinalUpperJD float64

	// ExitReason is the human-readable name of the stop condition
	// that terminated the loop: "abs_diff_threshold",
	// "bracket_width_threshold", or "max_iterations".
	ExitReason string
}

// SolveDesignTime returns the Julian Day at which the Sun longitude
// is exactly SunOffsetDeg earlier than at birthJD, searching backward
// in time using pure bisection.  When the abs-diff stop fires, the
// returned value is the converged midpoint at that step.  When the
// bracket-width stop fires (interval < 1 second), the returned value
// is the *lower bound* of the final interval per canon A3 / D22.
//
// SolveDesignTime is the production entry point used by the engine
// to populate human_design.system.design_time_utc.  Tests that need
// to inspect the iteration discipline use SolveDesignTimeWithDiagnostics
// instead.
func SolveDesignTime(birthJD float64, sun SunLongitudeFunc) (float64, error) {
	jd, _, err := solveDesignTime(birthJD, sun)
	return jd, err
}

// SolveDesignTimeWithDiagnostics is the test-facing entry point.  It
// returns the same Julian Day SolveDesignTime would, plus a
// Diagnostics record summarising the solver's iteration discipline.
// Production code never depends on the diagnostic values – the
// canon does not require them in the response envelope.
func SolveDesignTimeWithDiagnostics(birthJD float64, sun SunLongitudeFunc) (float64, Diagnostics, error) {
	return solveDesignTime(birthJD, sun)
}

// solveDesignTime is the shared implementation behind both public
// entry points.
func solveDesignTime(birthJD float64, sun SunLongitudeFunc) (float64, Diagnostics, error) {
	if sun == nil {
		return 0, Diagnostics{}, errors.New("designtime: sun longitude function is nil")
	}

	diag := Diagnostics{}

	birthSun := sun(birthJD)
	diag.SunFuncCalls++
	target := normalizeDeg(birthSun - SunOffsetDeg)

	// Initial bracket: ~88 days before birth, ±5 days.  Backward
	// search direction is enforced by clamping the upper bound at
	// or before birth.
	lower := birthJD - (SunOffsetDeg + initialBracketHalfWidthDays)
	upper := birthJD - (SunOffsetDeg - initialBracketHalfWidthDays)
	if upper > birthJD {
		upper = birthJD
	}

	diffLower := signedDiffDeg(sun(lower), target)
	diag.SunFuncCalls++
	diffUpper := signedDiffDeg(sun(upper), target)
	diag.SunFuncCalls++

	if diffLower == 0 {
		diag.FinalBracketDays = upper - lower
		diag.FinalAbsDiffDeg = 0
		diag.FinalLowerJD = lower
		diag.FinalUpperJD = upper
		diag.ExitReason = "abs_diff_threshold"
		return lower, diag, nil
	}
	if diffUpper == 0 {
		diag.FinalBracketDays = upper - lower
		diag.FinalAbsDiffDeg = 0
		diag.FinalLowerJD = lower
		diag.FinalUpperJD = upper
		diag.ExitReason = "abs_diff_threshold"
		return upper, diag, nil
	}

	// Expand the bracket only as a fallback for pathological inputs.
	// The canonical case starts already bracketed because the Sun
	// moves ~0.985°/day and the nominal 88-day shift therefore puts
	// the design moment within ±2 days of birth-88, well inside the
	// initial ±5-day half-width.
	//
	// When expansion is needed, the direction of widening is chosen
	// from the sign of the existing diffs:
	//
	//   * both diffs > 0  ⇒ Sun is past the target at both ends ⇒
	//                       the root is *earlier* than lower ⇒
	//                       widen lower further into the past.
	//   * both diffs < 0  ⇒ Sun has not yet reached the target ⇒
	//                       the root is *later* than upper ⇒ widen
	//                       upper forward, capped at birthJD to
	//                       preserve the canonical backward-only
	//                       search direction.
	for diffLower*diffUpper > 0 && diag.BracketExpansions < maxBracketExpansion {
		if diffLower > 0 {
			lower -= bracketExpansionStepDays
			diffLower = signedDiffDeg(sun(lower), target)
			diag.SunFuncCalls++
		} else {
			upper += bracketExpansionStepDays
			if upper > birthJD {
				upper = birthJD
			}
			diffUpper = signedDiffDeg(sun(upper), target)
			diag.SunFuncCalls++
		}
		diag.BracketExpansions++
	}
	if diffLower*diffUpper > 0 {
		return 0, diag, fmt.Errorf(
			"designtime: failed to bracket sun = %.4f° within %d expansions "+
				"(final bracket [%.6f, %.6f] JD, diffs [%.6f°, %.6f°])",
			target, maxBracketExpansion, lower, upper, diffLower, diffUpper)
	}

	// Pure bisection.  Sun longitude is monotonically increasing in
	// time over the canonical ~88-day window (mean Sun motion is
	// ~0.985°/day with no retrograde phase), so the sign of
	// signedDiffDeg(sun(t), target) increases monotonically with t.
	// Hence: diff > 0 ⇒ t too late ⇒ shrink upper.
	stopBracketDays := StopBracketSeconds / secondsPerDay
	for i := 0; i < maxBisectionIterations; i++ {
		diag.BracketIterations++
		mid := (lower + upper) / 2.0
		diff := signedDiffDeg(sun(mid), target)
		diag.SunFuncCalls++

		if math.Abs(diff) < StopAbsSunDiffDeg {
			diag.FinalBracketDays = upper - lower
			diag.FinalAbsDiffDeg = math.Abs(diff)
			diag.FinalLowerJD = lower
			diag.FinalUpperJD = upper
			diag.ExitReason = "abs_diff_threshold"
			return mid, diag, nil
		}
		if diff > 0 {
			upper = mid
			diffUpper = diff
		} else {
			lower = mid
			diffLower = diff
		}
		if (upper - lower) < stopBracketDays {
			// A3 / Document 12 D22: canonical Design time is the
			// lower bound of the final search interval (not midpoint).
			// We re-evaluate sun(lower) so FinalAbsDiffDeg in the
			// diagnostics reflects the returned point, not the
			// last-visited midpoint.
			finalDiff := signedDiffDeg(sun(lower), target)
			diag.SunFuncCalls++
			diag.FinalBracketDays = upper - lower
			diag.FinalAbsDiffDeg = math.Abs(finalDiff)
			diag.FinalLowerJD = lower
			diag.FinalUpperJD = upper
			diag.ExitReason = "bracket_width_threshold"
			return lower, diag, nil
		}
	}
	// Pathological exhaustion path.  Canon does not address this
	// case (the width-threshold should always fire first within
	// maxBisectionIterations), so we return the lower bound for
	// consistency with the canonical width-stop rule rather than
	// inventing a midpoint convention here.
	finalDiff := signedDiffDeg(sun(lower), target)
	diag.SunFuncCalls++
	diag.FinalBracketDays = upper - lower
	diag.FinalAbsDiffDeg = math.Abs(finalDiff)
	diag.FinalLowerJD = lower
	diag.FinalUpperJD = upper
	diag.ExitReason = "max_iterations"
	return lower, diag, nil
}

// normalizeDeg folds an arbitrary angular value into [0, 360).
// Mirrors trinity.org line 234 ("360.0 normalizes to 0.0").
func normalizeDeg(x float64) float64 {
	r := math.Mod(x, 360)
	if r < 0 {
		r += 360
	}
	return r
}

// signedDiffDeg returns a - b expressed in (-180, 180].  This is the
// canonical signed angular difference: zero when a = b modulo 360,
// positive when a leads b by less than half a circle, negative when
// a trails b by less than half a circle.
func signedDiffDeg(a, b float64) float64 {
	d := normalizeDeg(a - b)
	if d > 180.0 {
		d -= 360.0
	}
	return d
}
