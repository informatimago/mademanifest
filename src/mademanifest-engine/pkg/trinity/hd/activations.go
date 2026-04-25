package hd

import (
	"math"

	"mademanifest-engine/pkg/astronomy"
	"mademanifest-engine/pkg/canon"
	"mademanifest-engine/pkg/ephemeris"
	"mademanifest-engine/pkg/hd/calc"
	"mademanifest-engine/pkg/trinity/input"
	"mademanifest-engine/pkg/trinity/output"
)

// activations.go implements the Phase 6 Trinity Human Design
// activation pipeline.  For each of the 13 canonical HD snapshot
// bodies (canon.HDSnapshotOrder) it:
//
//  1. Looks up the geocentric ecliptic longitude at a given JD,
//     applying the canon's "true node" policy for Human Design.
//  2. Maps the longitude through the canonical mandala
//     (calc.MapToGateLine) to a (gate, line) pair.
//
// The pipeline is invoked twice per request: once at the birth JD
// (personality activations) and once at the design-time JD computed
// in Phase 5 (design activations).
//
// Determinism: the function below depends only on the JD argument,
// the Swiss Ephemeris pinned via pkg/ephemeris (cross-checked at
// boot), and the compiled-in canon constants.  No environment
// variables are consulted; the SE_NODE_POLICY shim that previously
// toggled mean vs. true node was removed in this same phase.

// snapshotLongitudes returns the 13 canonical HD snapshot longitudes
// at the given Julian Day, keyed by canon.HDSnapshotOrder name.
// north_node is always SE_TRUE_NODE for the Human Design domain.
// south_node is mathematically derived as north_node + 180° mod 360.
// earth is mathematically derived as sun + 180° mod 360 (matching
// the astrology pipeline; trinity.org line 240).
func snapshotLongitudes(jd float64) map[string]float64 {
	out := make(map[string]float64, len(canon.HDSnapshotOrder))
	for _, body := range canon.HDSnapshotOrder {
		switch body {
		case "earth":
			// Earth requires Sun's value; HDSnapshotOrder lists
			// sun before earth so the value is already populated
			// by the time we reach this case.
			out["earth"] = mod360(out["sun"] + 180.0)
		case "north_node":
			// Human Design canon (Document 03 §"Node policy by
			// domain") fixes node_type = true, so we resolve the
			// HD north_node via SE_TRUE_NODE.  The astrology
			// pipeline uses SE_MEAN_NODE under "north_node_mean"
			// instead; the two paths are now strictly separate.
			out["north_node"] = ephemeris.GetPlanetLongAtTime(jd, "north_node_true")
		case "south_node":
			// south_node = north_node (true) + 180° mod 360.
			// HDSnapshotOrder lists north_node before south_node
			// so out["north_node"] is already populated.
			out["south_node"] = mod360(out["north_node"] + 180.0)
		default:
			out[body] = ephemeris.GetPlanetLongAtTime(jd, body)
		}
	}
	return out
}

// activationsFor builds the canonically-ordered HDActivation slice
// for one snapshot.  The output order matches canon.HDSnapshotOrder
// element-for-element, so json.Marshal preserves it without a custom
// encoder.
func activationsFor(longs map[string]float64) []output.HDActivation {
	acts := make([]output.HDActivation, 0, len(canon.HDSnapshotOrder))
	for _, body := range canon.HDSnapshotOrder {
		gate, line := calc.MapToGateLine(longs[body])
		acts = append(acts, output.HDActivation{
			ObjectID: body,
			Gate:     gate,
			Line:     line,
		})
	}
	return acts
}

// ComputeActivations builds personality and design HDActivation
// slices for a validated Trinity payload.  It consumes the
// already-computed design Julian Day (so the caller can reuse the
// expensive bisection result rather than running it twice) and
// returns the two slices in canon order.  Errors here are wrapped
// engine-internal time-conversion failures; a non-nil error must be
// surfaced as execution_failure (HTTP 500) by the caller.
func ComputeActivations(p input.Payload, designJD float64) (personality, design []output.HDActivation, err error) {
	utcBirth, err := localToUTC(p)
	if err != nil {
		return nil, nil, err
	}
	birthJD := astronomy.ConvertUTCToJulianDay(utcBirth)

	personality = activationsFor(snapshotLongitudes(birthJD))
	design = activationsFor(snapshotLongitudes(designJD))
	return personality, design, nil
}

// BirthJDFromPayload exposes the local→UTC→JD conversion used by
// ComputeActivations and ComputeDesignTime so callers (notably the
// HTTP handler) can compute the birth JD once and pass the design
// JD into ComputeActivations without re-parsing the payload.
func BirthJDFromPayload(p input.Payload) (float64, error) {
	utcBirth, err := localToUTC(p)
	if err != nil {
		return 0, err
	}
	return astronomy.ConvertUTCToJulianDay(utcBirth), nil
}

// DesignJDFromTime mirrors julianDayToUTC in the opposite direction.
// Exposed so the HTTP handler can convert the design-time time.Time
// produced by ComputeDesignTime back into a JD for ComputeActivations
// without taking on the unit-conversion responsibility itself.
func DesignJDFromTime(t timeLike) float64 {
	const unixEpochJD = 2440587.5
	secondsSinceEpoch := float64(t.Unix()) + float64(t.Nanosecond())/1e9
	return secondsSinceEpoch/86400.0 + unixEpochJD
}

// timeLike captures the subset of time.Time the JD conversion
// needs.  Defined as a local interface so the package can avoid an
// extra time-package import here (the conversion is symmetric with
// julianDayToUTC in designtime.go).
type timeLike interface {
	Unix() int64
	Nanosecond() int
}

func mod360(x float64) float64 {
	r := math.Mod(x, 360)
	if r < 0 {
		r += 360
	}
	return r
}
