package human_design

import (
	"fmt"
	"log"
	"math"

	"github.com/mshafiee/swephgo"
	"mademanifest-engine/pkg/canon"
	"mademanifest-engine/pkg/emit_golden"
	"mademanifest-engine/pkg/ephemeris"
	"mademanifest-engine/pkg/sweph"
)

func GateSequence64() []int {
	return canon.GateSequenceV1
}

func normalizeDeg(x float64) float64 {
	r := math.Mod(x, 360)
	if r < 0 {
		r += 360
	}
	return r
}

func signedDiffDeg(a float64, b float64) float64 {
	diff := normalizeDeg(a - b)
	if diff >= 180.0 {
		diff -= 360.0
	}
	return diff
}

func SolveDesignTime(birthTime float64, sunFunc func(t float64) float64, dtsparams emit_golden.DesignTimeSolver) (float64, error) {
	if dtsparams.SunOffsetDeg <= 0 {
		return birthTime, nil
	}

	targetSun := normalizeDeg(sunFunc(birthTime) - dtsparams.SunOffsetDeg)
	bufferDays := 5.0
	start := birthTime - (dtsparams.SunOffsetDeg + bufferDays)
	end := birthTime - (dtsparams.SunOffsetDeg - bufferDays)
	if start > end {
		start, end = end, start
	}

	bracketed := false
	for i := 0; i < 10; i++ {
		diffStart := signedDiffDeg(sunFunc(start), targetSun)
		diffEnd := signedDiffDeg(sunFunc(end), targetSun)
		if diffStart == 0 {
			return start, nil
		}
		if diffEnd == 0 {
			return end, nil
		}
		if diffStart*diffEnd < 0 {
			bracketed = true
			break
		}
		start -= 2.0
		end += 2.0
	}
	if !bracketed {
		return 0, fmt.Errorf("design time bracket not found around offset %.3f", dtsparams.SunOffsetDeg)
	}

	stopIfTimeBracketBelowDays := float64(dtsparams.StopIfTimeBracketBelowSeconds) / 86400.0
	if stopIfTimeBracketBelowDays <= 0 {
		stopIfTimeBracketBelowDays = 1.0 / 86400.0
	}

	for (end - start) > stopIfTimeBracketBelowDays {
		mid := (start + end) / 2.0
		diff := signedDiffDeg(sunFunc(mid), targetSun)
		if math.Abs(diff) <= dtsparams.StopIfAbsSunDiffDegBelow {
			return mid, nil
		}
		if diff > 0 {
			end = mid
		} else {
			start = mid
		}
	}
	return (start + end) / 2.0, nil
}

func CalculateSnapshot(longs map[string]float64, gateSeq []int, hdparam emit_golden.HumanDesignMapping) map[string]string {
	out := make(map[string]string, len(activationObjectOrder))
	for _, obj := range activationObjectOrder {
		long := longs[obj]
		gate, line := mapToGateLine(long, hdparam, gateSeq)
		out[obj] = fmt.Sprintf("%.1f", float64(gate)+float64(line)/10.0)
	}
	return out
}
func mod360(x float64) float64 {
	r := math.Mod(x, 360)
	if r < 0 {
		r += 360
	}
	return r
}

func getTrueNodeLongAtTime(jd float64) float64 {
	xx := make([]float64, 6)
	serr := make([]byte, 256)
	errCode := swephgo.Calc(jd, sweph.SE_TRUE_NODE, sweph.SEFLG_SWIEPH, xx, serr)
	if errCode < 0 {
		log.Printf("swephgo.Calc error: %+v", string(serr))
		panic("swephgo.Calc failed with error code " + fmt.Sprint(errCode))
	}
	return xx[0]
}

// Fixed object order
var activationObjectOrder = []string{
	"sun",
	"earth", // must be after sun, see LongitudesAt
	"north_node",
	"south_node", // must be after south_node, see LongitudesAt
	"moon",
	"mercury",
	"venus",
	"mars",
	"jupiter",
	"saturn",
	"uranus",
	"neptune",
	"pluto",
}

func LongitudesAt(
	jd float64,
) map[string]float64 {
	out := make(map[string]float64, len(activationObjectOrder)+2)

	for _, body := range activationObjectOrder {
		switch body {
		case "earth":
			out["earth"] = mod360(out["sun"] + 180.0)
			continue
		case "north_node":
			out["north_node"] = getTrueNodeLongAtTime(jd)
			continue
		case "south_node":
			out["south_node"] = mod360(out["north_node"] + 180.0)
			continue
		default:
			out[body] = ephemeris.GetPlanetLongAtTime(jd, body)
			continue
		}
	}

	return out
}

// CalculateHumanDesign computes Human Design gates/lines for Personality and Design snapshots.

func CalculateHumanDesign(
	birthTime float64, // julian date of birth
	longitudesAt func(float64) map[string]float64,
	hdparam emit_golden.HumanDesignMapping,
	dtsparam emit_golden.DesignTimeSolver,
) emit_golden.HumanDesign {

	log.Printf("julian birth time = %f\n", birthTime)

	gateSeq := GateSequence64()
	if len(gateSeq) != 64 {
		panic(fmt.Sprintf("gate sequence must have 64 entries, got %d", len(gateSeq)))
	}

	// Personality
	personalityLong := longitudesAt(birthTime)

	// Design time
	designTime, err := SolveDesignTime(birthTime, func(t float64) float64 {
		return longitudesAt(t)["sun"]
	}, dtsparam)
	if err != nil {
		panic(err)
	}
	designLong := longitudesAt(designTime)

	return emit_golden.HumanDesign{
		ActivationObjectOrder: activationObjectOrder,
		Personality:           CalculateSnapshot(personalityLong, gateSeq, hdparam),
		Design:                CalculateSnapshot(designLong, gateSeq, hdparam),
	}
}

// mapToGateLine converts an absolute ecliptic longitude [0,360)
// into a Human Design gate (1–64) and line (1–6).
func mapToGateLine(absLon float64, hdparam emit_golden.HumanDesignMapping, gateSeq []int) (int, int) {
	r := normalizeDeg(absLon - hdparam.MandalaStartDeg)
	gateIndex := int(math.Floor(r / hdparam.GateWidthDeg))
	lineIndex := int(math.Floor(math.Mod(r, hdparam.GateWidthDeg) / hdparam.LineWidthDeg))

	if gateIndex < 0 {
		gateIndex = 0
	} else if gateIndex > 63 {
		gateIndex = 63
	}
	if lineIndex < 0 {
		lineIndex = 0
	} else if lineIndex > 5 {
		lineIndex = 5
	}

	line := lineIndex + 1
	gate := gateSeq[gateIndex]
	return gate, line
}
