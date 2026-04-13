package human_design

import (
	"mademanifest-engine/pkg/canon"
	"mademanifest-engine/pkg/emit_golden"
	"strconv"
	"testing"
)

func testGateSequence() []int {
	seq := make([]int, 64)
	for i := 0; i < 64; i++ {
		seq[i] = i + 1
	}
	return seq
}

func TestCalculateHumanDesign(t *testing.T) {
	canon.GateSequenceV1 = testGateSequence()

	// Sample HumanDesignParameters
	hdparams := emit_golden.HumanDesignMapping{
		MandalaStartDeg: 313.25,
		GateWidthDeg:    5.625,  // 360 / 64 gates
		LineWidthDeg:    0.9375, // 5.625 / 6 lines
		IntervalRule:    "start_inclusive_end_exclusive",
	}

	dtsparams := emit_golden.DesignTimeSolver{
		SunOffsetDeg:                  88.0,
		StopIfAbsSunDiffDegBelow:      0.0001,
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

func TestMapToGateLineBoundaries(t *testing.T) {
	hdparams := emit_golden.HumanDesignMapping{
		MandalaStartDeg: 313.25,
		GateWidthDeg:    5.625,
		LineWidthDeg:    0.9375,
		IntervalRule:    "start_inclusive_end_exclusive",
	}
	gateSeq := testGateSequence()
	start := hdparams.MandalaStartDeg

	gate, line := mapToGateLine(start, hdparams, gateSeq)
	if gate != 1 || line != 1 {
		t.Fatalf("start boundary expected gate 1 line 1, got %d.%d", gate, line)
	}

	gate, line = mapToGateLine(start+hdparams.GateWidthDeg, hdparams, gateSeq)
	if gate != 2 || line != 1 {
		t.Fatalf("gate boundary expected gate 2 line 1, got %d.%d", gate, line)
	}

	eps := 1e-6
	gate, line = mapToGateLine(start-eps, hdparams, gateSeq)
	if gate != 64 || line != 6 {
		t.Fatalf("end boundary expected gate 64 line 6, got %d.%d", gate, line)
	}
}
