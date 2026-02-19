package astrology

import (
	"testing"
)

func TestCalculateAstrology(t *testing.T) {
	// Sample planetary positions in degrees
	positions := map[string]float64{
		"sun":     19.0 + 32.0/60.0, // 19°32'
		"moon":    14.0 + 20.0/60.0, // 14°20'
		"mercury": 8.0 + 16.0/60.0,  // 8°16'
	}

	// Sample Julian day and location
	julianDay := 2460000.5       // example JD
	latitude := 48.8566          // Paris
	longitude := 2.3522

	// Call the function
	_ = CalculateAstrology(positions, julianDay, latitude, longitude)

}
