package human_design

import (
	"strconv"
	"testing"
	"mademanifest-engine/pkg/emit_golden"
)

func TestCalculateHumanDesign(t *testing.T) {

	// Sample HumanDesignParameters
	hdparams := emit_golden.HumanDesignMapping{
		MandalaStartDeg: 313.25,
		GateWidthDeg:    5.625, // 360 / 64 gates
		LineWidthDeg:    0.9375, // 5.625 / 6 lines
		IntervalRule:    "start_inclusive_end_exclusive",
	}

	dtsparams := emit_golden.DesignTimeSolver{
		SunOffsetDeg: 88.0,
		StopIfAbsSunDiffDegBelow: 0.0001,
		StopIfTimeBracketBelowSeconds: 1,
	}

	julianDate := 2447991.169444

	// Call the function
	result := CalculateHumanDesign(julianDate, LongitudesAt, hdparams, dtsparams)

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
