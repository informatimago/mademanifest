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
	result := CalculateAstrology(positions, julianDay, latitude, longitude)

	if result == nil {
		t.Fatal("CalculateAstrology returned nil")
	}

	// Check that ASC and MC exist
	if _, ok := result["ascendant"].(map[string]interface{}); !ok {
		t.Error("ascendant missing or wrong type")
	}
	if _, ok := result["mc"].(map[string]interface{}); !ok {
		t.Error("mc missing or wrong type")
	}

	// Verify planets
	for planet := range positions {
		val, ok := result[planet].(map[string]interface{})
		if !ok {
			t.Errorf("planet %s missing or wrong type", planet)
			continue
		}
		// Check sign, degree, minute keys exist
		if _, ok := val["sign"]; !ok {
			t.Errorf("%s: missing 'sign'", planet)
		}
		if _, ok := val["degree"]; !ok {
			t.Errorf("%s: missing 'degree'", planet)
		}
		if _, ok := val["minute"]; !ok {
			t.Errorf("%s: missing 'minute'", planet)
		}
	}

	// Optional: spot check value types
	for _, key := range []string{"ascendant", "mc"} {
		val := result[key].(map[string]interface{})
		if _, ok := val["sign"].(string); !ok {
			t.Errorf("%s.sign is not a string", key)
		}
		if _, ok := val["degree"].(int); !ok {
			t.Errorf("%s.degree is not an int", key)
		}
		if _, ok := val["minute"].(int); !ok {
			t.Errorf("%s.minute is not an int", key)
		}
	}
}
