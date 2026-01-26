package human_design

import (
	"fmt"
	"math"
	"mademanifest-engine/pkg/process_input"
)

// Fixed object order
var activationObjectOrder = []string{
	"sun",
	"earth",
	"north_node",
	"south_node",
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

// CalculateHumanDesign computes Human Design gates/lines for Personality and Design snapshots.
// positions: Swiss Ephemeris longitudes in degrees for sun, moon, mercury, venus, mars, jupiter, saturn, uranus, neptune, pluto, north_node, etc.
// sunOffsetDeg: the design offset, usually 88 degrees before birth.
func CalculateHumanDesign(positions map[string]float64, hdparam process_input.HumanDesignParameters, sunOffsetDeg float64) map[string]interface{} {
	personality := make(map[string]string)
	design := make(map[string]string)

	// Derived objects
	positions["earth"] = math.Mod(positions["sun"]+180.0, 360.0)
	positions["south_node"] = math.Mod(positions["north_node"]+180.0, 360.0)

	// Compute Personality snapshot
	for _, obj := range activationObjectOrder {
		long := positions[obj]
		gate, line := mapToGateLine(long,hdparam)
		personality[obj] = fmt.Sprintf("%.1f", gate+float64(line-1)*hdparam.LineWidthDeg)
	}

	// Compute Design snapshot
	for _, obj := range activationObjectOrder {
		var long float64
		if obj == "sun" {
			long = math.Mod(positions["sun"]-sunOffsetDeg+360.0, 360.0)
		} else if obj == "earth" {
			long = math.Mod(positions["earth"]-sunOffsetDeg+360.0, 360.0)
		} else {
			// Other planets and nodes: use same as personality (spec: exact same calculation, no offset)
			long = positions[obj]
		}
		gate, line := mapToGateLine(long,hdparam)
		design[obj] = fmt.Sprintf("%.1f", gate+float64(line-1)*hdparam.LineWidthDeg)
	}

	result:= make(map[string]interface{})
	result["activation_object_order"]=activationObjectOrder
	result["personality"]=personality
	result["design"]=design

    return result
}

// mapToGateLine converts a longitude (0-360) into gate number (1-64) and line (1-6)
func mapToGateLine(long float64, hdparam process_input.HumanDesignParameters) (float64, int) {
	r := math.Mod(long-hdparam.MandalaStartDeg+360.0, 360.0)
	gateIndex := int(math.Floor(r / hdparam.GateWidthDeg))
	lineIndex := int(math.Floor(math.Mod(r, hdparam.GateWidthDeg) / hdparam.LineWidthDeg))
	if gateIndex >= 64 {
		gateIndex = 63
	}
	if lineIndex >= 6 {
		lineIndex = 5
	}
	gate := float64(gateIndex + 1)
	line := lineIndex + 1
	return gate, line
}

