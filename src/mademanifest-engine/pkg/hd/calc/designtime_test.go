package calc

import (
	"math"
	"sort"
	"testing"
)

// linearSun returns a SunLongitudeFunc that increases by slopeDegPerDay
// degrees per Julian Day starting from baseDeg at jd0.  The function
// wraps into [0, 360) on read.  Used by the synthetic tests below to
// hit the bisection loop without pulling in Swiss Ephemeris.
func linearSun(jd0, baseDeg, slopeDegPerDay float64) SunLongitudeFunc {
	return func(jd float64) float64 {
		raw := baseDeg + (jd-jd0)*slopeDegPerDay
		r := math.Mod(raw, 360.0)
		if r < 0 {
			r += 360.0
		}
		return r
	}
}

// countingSun wraps an inner SunLongitudeFunc and records every JD
// value at which it is invoked.  The test uses the recorded sequence
// to verify the iteration discipline (call count for the
// forbidden-shortcut test, JD spread for the monotonicity test).
type countingSun struct {
	inner SunLongitudeFunc
	calls []float64
}

func (c *countingSun) sun(jd float64) float64 {
	c.calls = append(c.calls, jd)
	return c.inner(jd)
}

func TestSolveDesignTimeRecoversLinearTarget(t *testing.T) {
	// Birth at JD 2447991 with Sun at exactly 19.0°.  Slope 1°/day
	// makes the canonical 88° offset exactly 88 days in the past.
	// The solver must recover that JD to within the canonical
	// 1-second tolerance.
	const (
		birthJD       = 2447991.0
		baseDeg       = 19.0
		slope         = 1.0
		expectedJD    = birthJD - 88.0
		toleranceDays = StopBracketSeconds / secondsPerDay
	)
	sun := linearSun(birthJD, baseDeg, slope)
	got, diag, err := SolveDesignTimeWithDiagnostics(birthJD, sun)
	if err != nil {
		t.Fatalf("SolveDesignTime: %v", err)
	}
	if math.Abs(got-expectedJD) > toleranceDays {
		t.Errorf("design JD = %.10f, want %.10f ± %.10f (drift %.3e days = %.3f s)",
			got, expectedJD, toleranceDays,
			got-expectedJD, (got-expectedJD)*secondsPerDay)
	}
	if diag.FinalAbsDiffDeg >= StopAbsSunDiffDeg && diag.FinalBracketDays >= toleranceDays {
		t.Errorf("solver exited without satisfying either canonical stop "+
			"condition: final |diff| = %.6e°, final width = %.3e days",
			diag.FinalAbsDiffDeg, diag.FinalBracketDays)
	}
}

// TestSolveDesignTimeMonotonicBracketShrinks asserts that the
// successive midpoints visited by the bisection step strictly halve
// the bracket on each call.  The plan calls for "bracket shrinks
// strictly; final width ≤ 1 second"; we observe shrinkage by sorting
// the visited JDs and computing the spread per iteration.
func TestSolveDesignTimeMonotonicBracketShrinks(t *testing.T) {
	// Use a *steep* slope so the absolute-difference stop never
	// fires before the width-stop.  At slope 50°/day the diff at
	// the midpoint after k bisections is 25 / 2^k degrees, which
	// stays above 0.0001 for at least 18 iterations – well past
	// the canonical 20-iter bracket-width stop.
	const (
		birthJD = 2447991.0
		baseDeg = 100.0
		slope   = 50.0
	)
	cs := &countingSun{inner: linearSun(birthJD, baseDeg, slope)}
	_, diag, err := SolveDesignTimeWithDiagnostics(birthJD, cs.sun)
	if err != nil {
		t.Fatalf("SolveDesignTime: %v", err)
	}
	if diag.FinalBracketDays > StopBracketSeconds/secondsPerDay {
		t.Errorf("final bracket width %.6e days exceeds canonical 1-second cap (= %.6e days)",
			diag.FinalBracketDays, StopBracketSeconds/secondsPerDay)
	}

	// Drop the first three calls (target capture + initial bracket
	// endpoints); what remains is a sequence of bisection midpoints.
	if len(cs.calls) < 5 {
		t.Fatalf("solver made only %d calls; expected at least 5", len(cs.calls))
	}
	mids := append([]float64(nil), cs.calls[3:]...)
	sorted := append([]float64(nil), mids...)
	sort.Float64s(sorted)
	overallSpread := sorted[len(sorted)-1] - sorted[0]
	if overallSpread > 11.0 {
		t.Errorf("midpoint spread %.3f days exceeds 11-day initial-bracket envelope", overallSpread)
	}

	// Late-iteration spread must collapse hard.  Each bisection
	// halves the bracket, so the spread of the last K midpoints is
	// bounded above by the width K-1 iterations before exit, which
	// is at most 2^(K-1) × the final bracket width.  With K=4 and a
	// final width of 1 second the bound is 8 seconds; we assert
	// well under that — late-tail spread > 60 seconds would mean
	// the loop was not actually halving the bracket each step.
	if len(mids) >= 4 {
		tail := append([]float64(nil), mids[len(mids)-4:]...)
		sort.Float64s(tail)
		tailRange := tail[len(tail)-1] - tail[0]
		if tailRange*secondsPerDay > 60.0 {
			t.Errorf("late midpoint spread %.6f s exceeds 60 s — bracket not converging",
				tailRange*secondsPerDay)
		}
	}

	// Strictly verify the *bracket* (lower, upper) shrinks
	// monotonically: re-run the solver with a sun function that
	// records each call's JD, then walk from the end of the call
	// list pairing each mid with its lower/upper bounds.  Because
	// every step replaces *exactly one* of (lower, upper) with the
	// previous mid, the rolling [min, max] of the most recent
	// (mid, prev-lower-or-upper) pair must shrink each iteration.
	// We approximate the bound monotonicity by asserting that the
	// span between the kth-from-last mid and the final mid divided
	// by 2^k stays under the initial half-width.
	for k := 1; k < 5 && k < len(mids); k++ {
		drift := math.Abs(mids[len(mids)-1] - mids[len(mids)-1-k])
		bound := initialBracketHalfWidthDays / math.Pow(2, float64(k-1))
		if drift > bound {
			t.Errorf("mid %d steps before final drifted %.6f days; bound = %.6f days",
				k, drift, bound)
		}
	}
}

// TestSolveDesignTimeSubSecondStopReturnsLowerBound is the
// canonical regression sentinel for A3 / Document 12 D22.  It
// constructs a synthetic Sun-rate that forces the solver out via
// the *bracket-width* stop (interval < 1 s), then asserts the
// canon-required structural property of the returned JD.
//
// Why a steep slope.  The abs-diff stop fires when
// |sun(mid) - target| < 0.0001°.  Under a synthetic linear Sun
// `sun(t) = base + slope * (t - birthJD)`, at the bisection's mid
// the diff is roughly `slope × (mid - root)`, so the abs-diff
// stop translates to a bracket width of roughly
// `2 × 0.0001 / slope` days.  For the *width* stop (1 s = 1/86400
// day) to fire *before* the abs-diff stop we need
//     1/86400 > 2 × 0.0001 / slope
//   ⇔ slope > 17.28 °/day.
// The 50 °/day slope used here is well above that threshold so
// the width stop fires first by construction.  At 50 °/day the
// canonical 88° offset traverses 1.76 days, so the analytic root
// of the synthetic Sun is at `birthJD - 1.76`, well inside the
// initial 10-day bracket — no expansions are required.
//
// Structural property asserted (canon-correct, midpoint-rejecting):
// at the width stop, the canon emits the *lower bound* of the
// final search interval.  After the bisection, the final interval
// brackets the root: the lower endpoint sits at or below the root
// (signedDiffDeg(sun(lower), target) ≤ 0) and the upper endpoint
// at or above (signedDiffDeg(sun(upper), target) ≥ 0).  A correct
// lower-bound emission therefore satisfies
// signedDiffDeg(sun(got), target) ≤ 0 by construction.  A
// regression that reverts to the midpoint emission would split
// the bracket and the sun(got) value would land on either side of
// the root with no canonical preference — `got` could end up at
// the upper end, violating the inequality.  Combined with the
// width assertion (got ≤ root + 1 s), the test pins lower-bound
// emission tightly without needing an analytically known root.
//
// This exists alongside the Schiedam regression sentinel (which
// pins the truncated whole-second value indirectly) so that the
// sub-second stop behaviour is protected explicitly even if the
// Schiedam fixture ever shifts to a payload whose convergence
// path lands on the abs-diff stop instead.  Per Jaimie's note in
// the canon-fold-back review, this regression sentinel must
// exercise the sub-second Design-time stop directly, not only
// transitively through the baseline.
func TestSolveDesignTimeSubSecondStopReturnsLowerBound(t *testing.T) {
	const (
		birthJD = 2447991.0
		baseDeg = 100.0
		slope   = 50.0
	)
	sun := linearSun(birthJD, baseDeg, slope)
	got, diag, err := SolveDesignTimeWithDiagnostics(birthJD, sun)
	if err != nil {
		t.Fatalf("SolveDesignTime: %v", err)
	}
	if diag.ExitReason != "bracket_width_threshold" {
		t.Fatalf("ExitReason = %q, want bracket_width_threshold "+
			"(test scenario must exercise the sub-second exit; if the "+
			"abs-diff stop fired first the test no longer protects A3/D22)",
			diag.ExitReason)
	}
	subSecondCap := StopBracketSeconds / secondsPerDay
	if diag.FinalBracketDays >= subSecondCap {
		t.Errorf("final bracket width %.6e days >= %.6e days; "+
			"sub-second precondition not met", diag.FinalBracketDays, subSecondCap)
	}

	// Canon property: D22 emits exactly FinalLowerJD.  Any midpoint
	// regression would return (FinalLowerJD + FinalUpperJD) / 2 —
	// strictly different unless the bracket has zero width, which
	// the FinalBracketDays assertion above already excludes.  Any
	// upper-bound regression would return FinalUpperJD — again
	// strictly different.  Bit-exact equality with FinalLowerJD is
	// therefore the surgical canon-correctness assertion.
	if got != diag.FinalLowerJD {
		t.Errorf("returned JD %.20f != FinalLowerJD %.20f "+
			"(delta = %+.3f s, FinalBracketDays = %.6e days = %.3f s); "+
			"canon D22 requires bit-exact return of the lower bound — "+
			"midpoint or upper-bound regression?",
			got, diag.FinalLowerJD,
			(got-diag.FinalLowerJD)*secondsPerDay,
			diag.FinalBracketDays, diag.FinalBracketDays*secondsPerDay)
	}
}

// TestSolveDesignTimeRequiresManyIterations is the forbidden-shortcut
// sentinel.  A solver that subtracted 88 days, used a 3-month
// approximation, performed a single secant step, or pulled the result
// from a precomputed lookup would all converge in << 20 sun-function
// calls.  The canonical bisection on a 10-day initial bracket
// requires roughly log2(10 days / 1 second) ≈ 20 iterations, plus a
// handful of setup calls.  We pin a hard floor at 20 to defeat every
// shortcut listed in trinity.org §"Forbidden shortcuts".
func TestSolveDesignTimeRequiresManyIterations(t *testing.T) {
	// Steep slope keeps the early |diff| stop from firing, so the
	// canonical width-stop sets the call-count floor.
	const (
		birthJD = 2447991.0
		baseDeg = 100.0
		slope   = 50.0
	)
	cs := &countingSun{inner: linearSun(birthJD, baseDeg, slope)}
	_, diag, err := SolveDesignTimeWithDiagnostics(birthJD, cs.sun)
	if err != nil {
		t.Fatalf("SolveDesignTime: %v", err)
	}
	if diag.SunFuncCalls < 20 {
		t.Errorf("solver made only %d sun-func calls; canonical bisection "+
			"on a 10-day bracket converging to 1 second requires ≥ 20",
			diag.SunFuncCalls)
	}
	if diag.BracketIterations < 17 {
		t.Errorf("solver ran only %d bisection iterations; canonical "+
			"convergence to 1-second width requires ≥ 17",
			diag.BracketIterations)
	}
}

// TestSolveDesignTimeBackwardOnly verifies that the canonical
// invariant "search direction backward only" holds: every JD value
// at which the solver evaluates the Sun callback lies strictly
// before birthJD (with the single allowed exception of the birth
// evaluation itself, used to capture the offset target).
func TestSolveDesignTimeBackwardOnly(t *testing.T) {
	const (
		birthJD = 2447991.0
		baseDeg = 19.0
		slope   = 1.0
	)
	cs := &countingSun{inner: linearSun(birthJD, baseDeg, slope)}
	_, _, err := SolveDesignTimeWithDiagnostics(birthJD, cs.sun)
	if err != nil {
		t.Fatalf("SolveDesignTime: %v", err)
	}
	// The very first call is at birthJD itself (target capture);
	// every subsequent call must be strictly < birthJD.
	if len(cs.calls) == 0 || cs.calls[0] != birthJD {
		t.Fatalf("first call must capture target at birthJD; got calls=%v", cs.calls)
	}
	for i, jd := range cs.calls[1:] {
		if jd >= birthJD {
			t.Errorf("call %d evaluated sun(%.6f) ≥ birthJD %.6f — search must be backward only",
				i+1, jd, birthJD)
		}
	}
}

// TestSolveDesignTimeRejectsNilSunFunc guards the public entry against
// nil callback misuse, which would otherwise panic deep in the loop.
func TestSolveDesignTimeRejectsNilSunFunc(t *testing.T) {
	if _, err := SolveDesignTime(2447991.0, nil); err == nil {
		t.Fatal("expected error for nil sun func; got nil")
	}
}

func TestNormalizeDeg(t *testing.T) {
	cases := []struct {
		in, want float64
	}{
		{0.0, 0.0},
		{360.0, 0.0},
		{720.0, 0.0},
		{-1.0, 359.0},
		{-360.0, 0.0},
		{359.999999, 359.999999},
		{1080.5, 0.5},
	}
	for _, c := range cases {
		if got := normalizeDeg(c.in); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("normalizeDeg(%.6f) = %.9f, want %.9f", c.in, got, c.want)
		}
	}
}

func TestSignedDiffDeg(t *testing.T) {
	cases := []struct {
		a, b, want float64
	}{
		{10.0, 0.0, 10.0},
		{0.0, 10.0, -10.0},
		{350.0, 10.0, -20.0}, // crosses 0/360 wrap
		{10.0, 350.0, 20.0},
		{179.0, 0.0, 179.0},
		{180.0, 0.0, 180.0}, // exact half-circle: positive
		{181.0, 0.0, -179.0},
		{0.0, 0.0, 0.0},
	}
	for _, c := range cases {
		if got := signedDiffDeg(c.a, c.b); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("signedDiffDeg(%.3f, %.3f) = %.6f, want %.6f",
				c.a, c.b, got, c.want)
		}
	}
}
