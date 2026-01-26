package astrology

import (
	"github.com/mshafiee/swephgo"
)

// CalculateAstrology computes complete astrology data using position data
func CalculateAstrology(positions map[string]float64, julianDay float64, latitude, longitude float64) map[string]interface{} {
	// Initialize the result map
	result := make(map[string]interface{})

	// Calculate house cusps using Placidus system

	cusps := make([]float64, 13)  // 12 house cusps + index 0 unused
	ascmc := make([]float64, 10)  // 0=ascendant, 1=MC, 2=ARMC, etc.

	err := swephgo.Houses(julianDay, latitude, longitude, 'P', cusps, ascmc)
	if err != 0 {
		// handle error
	}

	// Retrieve Ascendant and MC from ascmc slice
	ascendant := ascmc[0]
	mc := ascmc[1]

	// Convert ASC and MC to degrees and minutes
	ascDeg, ascMin := convertLongitudeToDegMin(ascendant)
	mcDeg, mcMin := convertLongitudeToDegMin(mc)

	// Add ASC and MC to the result
	result["ascendant"] = map[string]interface{}{
		// "house":  1, // ASC is always the 1st house cusp
		"sign":   getZodiacSign(ascendant), // Use "sign" instead of "house"
		"degree": ascDeg,
		"minute": ascMin,
	}
	result["mc"] = map[string]interface{}{
		// "house":  10, // MC is always the 10th house cusp
		"sign":   getZodiacSign(mc), // Use "sign" instead of "house"
		"degree": mcDeg,
		"minute": mcMin,
	}

	// Process each planet
	for key, lon := range positions {
		// Convert longitude to degrees and minutes
		deg, min := convertLongitudeToDegMin(lon)

		// Determine the house
		// house := getHouse(lon, cusps)

		// Determine the zodiac sign
		sign := getZodiacSign(lon)

		// Add to the result
		result[key] = map[string]interface{}{
			// "house":  house,
			"sign":   sign,
			"degree": deg,
			"minute": min,
		}
	}

	return result
}

// convertLongitudeToDegMin converts a float longitude to degrees and minutes
func convertLongitudeToDegMin(longitude float64) (int, int) {
	degrees := int(longitude)
	minutesFloat := (longitude - float64(degrees)) * 60
	minutes := int(minutesFloat)
	return degrees, minutes
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
