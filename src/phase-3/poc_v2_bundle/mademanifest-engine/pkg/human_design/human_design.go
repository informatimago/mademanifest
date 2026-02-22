package human_design

import (
	"log"
	"fmt"
	"math"
	"github.com/mshafiee/swephgo"
	"mademanifest-engine/pkg/sweph"
	"mademanifest-engine/pkg/emit_golden"
	"mademanifest-engine/pkg/ephemeris"
)

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

// ... rest of file same as before, but modify LongitudesAt switch.

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

	log.Printf("julian birth time = %f\n",birthTime)

	// Personality
	personalityLong := longitudesAt(birthTime)

	// Design time
	designTime, err := SolveDesignTime(
		birthTime,
		func(t float64) float64 {
			return longitudesAt(t)["sun"]
		},
		dtsparam,
	)
	if err != nil {
		log.Fatalf("Cannot solve design time %s",err)
	}

	designLong := longitudesAt(designTime)

	log.Printf("personalityLong = %v\n",personalityLong)
	log.Printf("designLong = %v\n",designLong)

	// Mandala gate sequence (must be predefined)
	gateSeq := GateSequence64()

    return emit_golden.HumanDesign{
        ActivationObjectOrder: activationObjectOrder,
        Personality:           CalculateSnapshot(personalityLong, false, gateSeq[:], hdparam),
        Design:                CalculateSnapshot(designLong, true, gateSeq[:], hdparam),
	}
}

