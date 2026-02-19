package astrology

import (
	"log"
	"reflect"
	"math"
	"github.com/mshafiee/swephgo"
	"mademanifest-engine/pkg/emit_golden"
	"mademanifest-engine/pkg/sweph"
)


func setByJSONTag(ptr any, tag string, val any) bool {
    v := reflect.ValueOf(ptr)
    if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
        panic("ptr must be pointer to struct")
    }

    v = v.Elem()
    t := v.Type()

    for i := 0; i < t.NumField(); i++ {
        field := t.Field(i)
        if field.Tag.Get("json") == tag {
            fv := v.Field(i)
            if !fv.CanSet() {
                return false
            }
            fv.Set(reflect.ValueOf(val))
            return true
        }
    }
    return false
}


// CalculateAstrology computes complete astrology data using position data
func CalculateAstrology(positions map[string]float64, julianDay float64, latitude, longitude float64) emit_golden.Astrology {
	// Initialize the result map
	var result emit_golden.Astrology

	// Calculate house cusps using Placidus system

	cusps := make([]float64, 13)  // 12 house cusps + index 0 unused
	ascmc := make([]float64, 10)  // 0=ascendant, 1=MC, 2=ARMC, etc.

	err := swephgo.HousesEx(julianDay,
		sweph.SEFLG_SWIEPH | sweph.SEFLG_TRUEPOS | sweph.SEFLG_NONUT,
		latitude, longitude, 'P', cusps, ascmc)
	if err != 0 {
		// handle error
	}

	// Retrieve Ascendant and MC from ascmc slice
	ascendant := ascmc[0]
	mc := ascmc[1]

	// Convert ASC and MC to degrees and minutes
	ascDeg, ascMin := convertLongitudeToDegMinAstro(ascendant)
	mcDeg, mcMin := convertLongitudeToDegMinAstro(mc)

	// Add ASC and MC to the result
	// result.Positions.Ascendant.House // "house":  1, // ASC is always the 1st
	result.Positions.Ascendant.Sign = getZodiacSign(ascendant)
	result.Positions.Ascendant.Deg = ascDeg
	result.Positions.Ascendant.Min = ascMin

	// "house":  10, // MC is always the 10th house cusp
	result.Positions.MC.Sign = getZodiacSign(mc)
	result.Positions.MC.Deg = mcDeg
	result.Positions.MC.Min = mcMin

	// Process each planet
	for key, lon := range positions {
		// Convert longitude to degrees and minutes
		deg, min := convertLongitudeToDegMin(lon)

		// Determine the house
		// house := getHouse(lon, cusps)

		// Determine the zodiac sign
		sign := getZodiacSign(lon)

		// Add to the result
		var pos emit_golden.Position
		pos.Sign = sign
		pos.Deg = deg
		pos.Min = min

		ok := setByJSONTag(&result.Positions, key, pos)
		if !ok {
			log.Printf("ok = %v  key = %v",ok,key)
			switch key {
			case "earth":
				// derived from sun elsewhere, ignore here
				continue

			case "north_node":
				result.Positions.NorthNodeMean = pos
				continue

			case "south_node":
				// derived later from north node
				continue
			default:
				log.Fatalf("Unknown Astrology key %s",key)
				continue
			}
		}

	}

	return result
}


// convertLongitudeToDegMin converts a float longitude to degrees and minutes
func convertLongitudeToDegMinAstro(longitude float64) (int, int) {
	lon := math.Mod(longitude, 360.0)
	if lon < 0 {
		lon += 360.0
	}

	z := math.Mod(lon, 30.0)

	totalMinutes := z * 60.0
	deg := int(totalMinutes / 60.0)
	min := int(math.Floor(totalMinutes - float64(deg*60) - 0.5))

	if min == 60 {
		min = 0
		deg++
		if deg == 30 {
			deg = 0
		}
	}

	return deg, min
}


// convertLongitudeToDegMin converts a float longitude to degrees and minutes
func convertLongitudeToDegMin(longitude float64) (int, int) {
    // Normalize to [0, 360)
    lon := math.Mod(longitude, 360.0)
    if lon < 0 {
        lon += 360.0
    }

    // Zodiac-relative longitude [0, 30)
    z := math.Mod(lon, 30.0)

    const eps = 1e-9

    deg := int(math.Floor(z + eps))
	min := int(math.Floor((z - float64(deg)) * 60 + eps))

    return deg, min
}


// getHouse determines the house of a planet based on its longitude and house cusps
func getHouse(longitude float64, houseCusps [13]float64) int {
	for i := 1; i <= 12; i++ {
		if longitude >= houseCusps[i] && longitude < houseCusps[i+1] {
			return i
		}
	}
	return 12 // Fallback to the 12th house
}


// getZodiacSign returns the zodiac sign for a given longitude
func getZodiacSign(longitude float64) string {
	// Normalize longitude to 0-360
	lon := longitude
	for lon < 0 {
		lon += 360
	}
	for lon >= 360 {
		lon -= 360
	}

	// Determine the zodiac sign
	switch {
	case lon >= 0 && lon < 30:
		return "Aries"
	case lon >= 30 && lon < 60:
		return "Taurus"
	case lon >= 60 && lon < 90:
		return "Gemini"
	case lon >= 90 && lon < 120:
		return "Cancer"
	case lon >= 120 && lon < 150:
		return "Leo"
	case lon >= 150 && lon < 180:
		return "Virgo"
	case lon >= 180 && lon < 210:
		return "Libra"
	case lon >= 210 && lon < 240:
		return "Scorpio"
	case lon >= 240 && lon < 270:
		return "Sagittarius"
	case lon >= 270 && lon < 300:
		return "Capricorn"
	case lon >= 300 && lon < 330:
		return "Aquarius"
	case lon >= 330 && lon < 360:
		return "Pisces"
	default:
		return "Unknown"
	}
}
