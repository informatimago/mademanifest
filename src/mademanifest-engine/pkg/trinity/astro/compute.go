package astro

import (
	"fmt"
	"math"
	"time"

	"github.com/mshafiee/swephgo"
	"mademanifest-engine/pkg/astronomy"
	"mademanifest-engine/pkg/canon"
	"mademanifest-engine/pkg/ephemeris"
	"mademanifest-engine/pkg/sweph"
	"mademanifest-engine/pkg/trinity/input"
	"mademanifest-engine/pkg/trinity/output"
)

// ComputeAstrology builds the complete astrology section of a
// Trinity success envelope from a validated input payload.
//
// Pinned canon (Document 03):
//   * zodiac       = tropical
//   * house_system = placidus
//   * node_type    = mean
//
// Earth derivation: trinity.org line 240 fixes earth = (sun + 180)
// mod 360 (geocentric Earth opposite the geocentric Sun).  Swiss
// Ephemeris's SE_EARTH returns the Earth's heliocentric coordinate,
// which is *not* the canonical astrology Earth – we ignore that
// value and recompute from the Sun longitude.
//
// House placement: the A2 working assumption (start-inclusive /
// end-exclusive, wrap between cusp 12 and cusp 1) is implemented by
// HouseFor.  All longitudes are normalised to [0, 360) before sign
// and house lookup so SignFor and HouseFor never see boundary
// inputs they would reject.
func ComputeAstrology(p input.Payload) (output.Astrology, error) {
	utcTime, err := localToUTC(p)
	if err != nil {
		return output.Astrology{}, fmt.Errorf("convert birth time: %w", err)
	}
	jd := astronomy.ConvertUTCToJulianDay(utcTime)

	rawLongs := ephemeris.CalculatePositions(jd) // also initialises sweph
	sunLong := normalizeDeg(rawLongs["sun"])
	rawLongs["earth"] = normalizeDeg(sunLong + 180.0) // override SE_EARTH

	cusps := make([]float64, 13) // indices 1..12 used; 0 unused
	ascmc := make([]float64, 10)
	const placidus = int('P')
	swephgo.HousesEx(jd, sweph.SEFLG_SWIEPH|sweph.SEFLG_NONUT,
		p.Latitude, p.Longitude, placidus, cusps, ascmc)

	var cuspArr [12]float64
	for i := 0; i < 12; i++ {
		cuspArr[i] = normalizeDeg(cusps[i+1])
	}

	objects := make([]output.AstroObject, 0, len(canon.AstrologyObjectOrder))
	for _, id := range canon.AstrologyObjectOrder {
		long := normalizeDeg(rawLongs[id])
		objects = append(objects, output.AstroObject{
			ObjectID:  id,
			Longitude: output.Longitude(long),
			Sign:      SignFor(long),
			House:     HouseFor(long, cuspArr),
		})
	}

	cuspsOut := make([]output.HouseCusp, 12)
	for i := 0; i < 12; i++ {
		cuspsOut[i] = output.HouseCusp{
			House:     i + 1,
			Longitude: output.Longitude(cuspArr[i]),
			Sign:      SignFor(cuspArr[i]),
		}
	}

	asc := normalizeDeg(ascmc[0])
	mc := normalizeDeg(ascmc[1])

	return output.Astrology{
		System: output.AstroSystem{
			Zodiac:      "tropical",
			HouseSystem: "placidus",
			NodeType:    "mean",
		},
		Angles: output.Angles{
			Ascendant: output.SignedLongitude{
				Longitude: output.Longitude(asc),
				Sign:      SignFor(asc),
			},
			Midheaven: output.SignedLongitude{
				Longitude: output.Longitude(mc),
				Sign:      SignFor(mc),
			},
		},
		HouseCusps: cuspsOut,
		Objects:    objects,
	}, nil
}

// localToUTC parses the validated string forms of birth_date and
// birth_time, attaches the validated timezone, and converts to UTC.
// The validator has already proved the string format and zone are
// canonical, so the parse failures here would indicate a bug
// upstream – they are wrapped as descriptive errors anyway.
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

// normalizeDeg folds an arbitrary angular value into [0, 360),
// matching trinity.org line 234 ("360.0 normalizes to 0.0").  The
// canonical wrap is needed because Swiss Ephemeris returns
// longitudes in the same range, but cusp arithmetic and the
// earth = sun + 180 derivation can drift slightly above 360.
func normalizeDeg(x float64) float64 {
	r := math.Mod(x, 360)
	if r < 0 {
		r += 360
	}
	return r
}
