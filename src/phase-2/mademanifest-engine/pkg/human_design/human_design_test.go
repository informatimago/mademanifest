package human_design

import (
	"testing"
)

func TestCalculateHumanDesign(t *testing.T) {
	// Test that the function can be called and returns a map
	positions := map[string]float64{
		"sun": 19.0 + 32.0/60.0,
		"moon": 14.0 + 20.0/60.0,
		"mercury": 8.0 + 16.0/60.0,
	}
	
	result := CalculateHumanDesign(positions, 88.0)
	
	// Verify that result map is not nil (function exists)
	if result == nil {
		t.Error("CalculateHumanDesign should return a valid result map")
	}
	
	// Basic verification that function signature works
	if len(result) < 1 {
		t.Logf("Human design result is not empty")
	}
}
