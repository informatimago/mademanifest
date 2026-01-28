package human_design

import (
	"strconv"
	"testing"
	"mademanifest-engine/pkg/emit_golden"
)

func TestCalculateHumanDesign(t *testing.T) {
	// Sample positions (degrees)
	positions := map[string]float64{
		"sun":     19.0 + 32.0/60.0, // 19°32'
		"moon":    14.0 + 20.0/60.0, // 14°20'
		"mercury": 8.0 + 16.0/60.0,  // 8°16'
		"venus":   5.0,
		"mars":    28.0,
		"jupiter": 120.0,
		"saturn":  250.0,
		"uranus":  300.0,
		"neptune": 350.0,
		"pluto":   15.0,
		"north_node": 10.0,
	}

	// Sample HumanDesignParameters
	hdParams := emit_golden.HumanDesignMapping{
		MandalaStartDeg: 313.25,
		GateWidthDeg:    5.625, // 360 / 64 gates
		LineWidthDeg:    0.9375, // 5.625 / 6 lines
		IntervalRule:    "start_inclusive_end_exclusive",
	}

	sunOffsetDeg := 88.0

	// Call the function
	result := CalculateHumanDesign(positions, hdParams, sunOffsetDeg)

	// Verify that all activation objects exist
	for _, obj := range activationObjectOrder {
		if _, ok := result.Personality[obj]; !ok {
			t.Errorf("personality missing object %s", obj)
		}
		if _, ok := result.Design[obj]; !ok {
			t.Errorf("design missing object %s", obj)
		}
	}

	// Spot check a value format
	for obj, val := range result.Personality {
		if _, err := strconv.ParseFloat(val, 64); err != nil {
			t.Errorf("personality[%s] is not a valid float string: %s", obj, val)
		}
	}
	for obj, val := range result.Design {
		if _, err := strconv.ParseFloat(val, 64); err != nil {
			t.Errorf("design[%s] is not a valid float string: %s", obj, val)
		}
	}
}
