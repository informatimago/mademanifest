// Package hd is the Trinity Human Design pipeline.  Phase 5
// introduces the design-time computation: a thin engine-side wrapper
// around pkg/hd/calc.SolveDesignTime that plugs in the canonical
// Swiss Ephemeris Sun longitude function and converts between Julian
// Day, the solver's native time domain, and time.Time, the canonical
// Trinity output domain.
//
// Later phases (6-8) extend this package with gate/line mapping,
// activations, and the structural derivations.
package hd

import (
	"fmt"
	"math"
	"time"

	"mademanifest-engine/pkg/astronomy"
	"mademanifest-engine/pkg/ephemeris"
	"mademanifest-engine/pkg/hd/calc"
	"mademanifest-engine/pkg/trinity/input"
)

// ComputeDesignTime returns the Human Design design-time as a UTC
// time.Time for a validated Trinity input payload.  It runs the
// canonical bisection solver against the Swiss Ephemeris Sun
// longitude function over Julian Day.
//
// The payload's birth_date / birth_time / timezone fields have
// already been validated by pkg/trinity/input; we therefore treat
// parse failures here as engine bugs rather than user input errors,
// and surface them as wrapped Go errors that the HTTP handler will
// emit as execution_failure (HTTP 500).
//
// Sun longitude is identical for personality and design — node
// policy (mean for astrology, true for human_design) does not enter
// the design-time computation.  We therefore call ephemeris's raw
// "sun" body lookup directly, avoiding any node policy crosstalk.
func ComputeDesignTime(p input.Payload) (time.Time, error) {
	utcBirth, err := localToUTC(p)
	if err != nil {
		return time.Time{}, err
	}
	birthJD := astronomy.ConvertUTCToJulianDay(utcBirth)
	sun := func(jd float64) float64 {
		return ephemeris.GetPlanetLongAtTime(jd, "sun")
	}
	designJD, err := calc.SolveDesignTime(birthJD, sun)
	if err != nil {
		return time.Time{}, fmt.Errorf("solve design time: %w", err)
	}
	return julianDayToUTC(designJD), nil
}

// localToUTC mirrors astro.localToUTC: the validator has already
// proved these strings parse, so any error here is an engine bug.
func localToUTC(p input.Payload) (time.Time, error) {
	loc, err := time.LoadLocation(p.Timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("load timezone %q: %w", p.Timezone, err)
	}
	var year, month, day int
	if _, err := fmt.Sscanf(p.BirthDate, "%d-%d-%d", &year, &month, &day); err != nil {
		return time.Time{}, fmt.Errorf("parse birth_date %q: %w", p.BirthDate, err)
	}
	var hour, minute int
	if _, err := fmt.Sscanf(p.BirthTime, "%d:%d", &hour, &minute); err != nil {
		return time.Time{}, fmt.Errorf("parse birth_time %q: %w", p.BirthTime, err)
	}
	local := time.Date(year, time.Month(month), day, hour, minute, 0, 0, loc)
	return local.UTC(), nil
}

// julianDayToUTC inverts astronomy.ConvertUTCToJulianDay.  The
// astronomy package only ships the forward direction, so we
// implement the reverse here at nanosecond precision.
// DesignTime.MarshalJSON applies the canonical truncation to whole
// seconds (A3 RESOLVED, D22) on the way out, so any sub-second
// remainder here is dropped at serialisation time.
func julianDayToUTC(jd float64) time.Time {
	const unixEpochJD = 2440587.5
	secondsSinceEpoch := (jd - unixEpochJD) * 86400.0
	whole := math.Floor(secondsSinceEpoch)
	frac := secondsSinceEpoch - whole
	nanos := int64(frac * 1e9)
	return time.Unix(int64(whole), nanos).UTC()
}
