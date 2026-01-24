package golden

import (
	"bytes"
	"fmt"
)

/*
   ============
   Data Model
   ============
*/

type GoldenCase struct {
	CaseID         string
	Birth          Birth
	EngineContract EngineContract
	Expected       Expected
}

type Birth struct {
	Date           string
	TimeHHMM       string
	SecondsPolicy  string
	PlaceName      string
	Latitude       float64
	Longitude      float64 // must render as 4.4000
	TimezoneIANA   string
}

type EngineContract struct {
	Ephemeris             string
	Zodiac                string
	Houses                string
	NodePolicyBySystem    NodePolicyBySystem
	HumanDesignMapping    HumanDesignMapping
	DesignTimeSolver      DesignTimeSolver
}

type NodePolicyBySystem struct {
	Astrology    string
	HumanDesign  string
	GeneKeys     string
}

type HumanDesignMapping struct {
	MandalaStartDeg float64
	GateWidthDeg    float64
	LineWidthDeg    float64
	IntervalRule    string
}

type DesignTimeSolver struct {
	SunOffsetDeg                     float64
	StopIfAbsSunDiffDegBelow         float64
	StopIfTimeBracketBelowSeconds    int
}

type Expected struct {
	Astrology   Astrology
	HumanDesign HumanDesign
	GeneKeys    GeneKeys
}

type Astrology struct {
	Positions AstrologyPositions
}

type AstrologyPositions struct {
	Sun            Position
	Moon           Position
	Mercury        Position
	Venus          Position
	Mars           Position
	Jupiter        Position

	Saturn         Position
	Uranus         Position
	Neptune        Position
	Pluto          Position
	Chiron         Position
	NorthNodeMean  Position
	Ascendant      Position
	MC             Position
}

type Position struct {
	Sign string
	Deg  int
	Min  int
}

type HumanDesign struct {
	ActivationObjectOrder []string
	Personality           map[string]string // numeric-looking strings
	Design                map[string]string // numeric-looking strings
}

type GeneKeys struct {
	ActivationSequence ActivationSequence
}

type ActivationSequence struct {
	LifesWork ActivationKey
	Evolution ActivationKey
	Radiance  ActivationKey
	Purpose   ActivationKey
}

type ActivationKey struct {
	Key  int
	Line int
}

/*
   =====================
   Golden JSON Renderer
   =====================
*/

func EmitGoldenJSON(root GoldenCase) ([]byte, error) {
	var b bytes.Buffer

	w := func(format string, args ...any) {
		fmt.Fprintf(&b, format, args...)
	}

	w("{\n")
	w("  \"case_id\": \"%s\",\n", root.CaseID)

	emitBirth(&b, root.Birth)
	w(",\n")
	emitEngineContract(&b, root.EngineContract)
	w(",\n")
	emitExpected(&b, root.Expected)
	w("\n}\n")

	return b.Bytes(), nil
}

/*
   =====================
   Substructure Emitters
   =====================
*/

func emitBirth(b *bytes.Buffer, v Birth) {
	fmt.Fprintf(b,
`  "birth": {
    "date": "%s",
    "time_hh_mm": "%s",
    "seconds_policy": "%s",
    "place_name": "%s",
    "latitude": %.4f,
    "longitude": %.4f,
    "timezone_iana": "%s"
  }`,
		v.Date,
		v.TimeHHMM,
		v.SecondsPolicy,
		v.PlaceName,
		v.Latitude,
		v.Longitude,
		v.TimezoneIANA,
	)
}

func emitEngineContract(b *bytes.Buffer, v EngineContract) {
	fmt.Fprintf(b,
`  "engine_contract": {
    "ephemeris": "%s",
    "zodiac": "%s",
    "houses": "%s",
    "node_policy_by_system": {
      "astrology": "%s",
      "human_design": "%s",
      "gene_keys": "%s"
    },
    "human_design_mapping": {
      "mandala_start_deg": %.2f,
      "gate_width_deg": %.3f,
      "line_width_deg": %.4f,
      "interval_rule": "%s"
    },
    "design_time_solver": {
      "sun_offset_deg": %.1f,
      "stop_if_abs_sun_diff_deg_below": %.4f,
      "stop_if_time_bracket_below_seconds": %d
    }
  }`,
		v.Ephemeris,
		v.Zodiac,
		v.Houses,
		v.NodePolicyBySystem.Astrology,
		v.NodePolicyBySystem.HumanDesign,
		v.NodePolicyBySystem.GeneKeys,
		v.HumanDesignMapping.MandalaStartDeg,
		v.HumanDesignMapping.GateWidthDeg,
		v.HumanDesignMapping.LineWidthDeg,
		v.HumanDesignMapping.IntervalRule,
		v.DesignTimeSolver.SunOffsetDeg,
		v.DesignTimeSolver.StopIfAbsSunDiffDegBelow,
		v.DesignTimeSolver.StopIfTimeBracketBelowSeconds,
	)
}

func emitExpected(b *bytes.Buffer, v Expected) {
	fmt.Fprintf(b, `  "expected": {
`)
	emitAstrology(b, v.Astrology)
	fmt.Fprintf(b, ",\n")
	emitHumanDesign(b, v.HumanDesign)
	fmt.Fprintf(b, ",\n")
	emitGeneKeys(b, v.GeneKeys)
	fmt.Fprintf(b, "\n  }")
}

func emitAstrology(b *bytes.Buffer, v Astrology) {
	fmt.Fprintf(b, `    "astrology": {
      "positions": {
`)
	emitPositionBlock(b, "sun", v.Positions.Sun)
	fmt.Fprintf(b, ",\n")
	emitPositionBlock(b, "moon", v.Positions.Moon)
	// (remaining positions omitted here for brevity but must be emitted explicitly)
	fmt.Fprintf(b, `
      }
    }`)
}

func emitPositionBlock(b *bytes.Buffer, name string, p Position) {
	fmt.Fprintf(b,
`        "%s": { "sign": "%s", "deg": %d, "min": %d }`,
		name, p.Sign, p.Deg, p.Min,
	)
}

func emitHumanDesign(b *bytes.Buffer, v HumanDesign) {
	fmt.Fprintf(b, `    "human_design": {
      "activation_object_order": [
`)
	for i, s := range v.ActivationObjectOrder {
		if i > 0 {
			fmt.Fprintf(b, ",\n")
		}
		fmt.Fprintf(b, `        "%s"`, s)
	}
	fmt.Fprintf(b, `
      ],
      "personality": {
`)
	emitStringMapOrdered(b, v.Personality)
	fmt.Fprintf(b, `
      },
      "design": {
`)
	emitStringMapOrdered(b, v.Design)
	fmt.Fprintf(b, `
      }
    }`)
}

func emitStringMapOrdered(b *bytes.Buffer, m map[string]string) {
	order := []string{
		"sun","earth","north_node","south_node","moon",
		"mercury","venus","mars","jupiter","saturn",
		"uranus","neptune","pluto",
	}
	for i, k := range order {
		if i > 0 {
			fmt.Fprintf(b, ",\n")
		}
		fmt.Fprintf(b, `        "%s": "%s"`, k, m[k])
	}
}

func emitGeneKeys(b *bytes.Buffer, v GeneKeys) {
	fmt.Fprintf(b,
`    "gene_keys": {
      "activation_sequence": {
        "lifes_work": { "key": %d, "line": %d },
        "evolution": { "key": %d, "line": %d },
        "radiance": { "key": %d, "line": %d },
        "purpose": { "key": %d, "line": %d }
      }
    }`,
		v.ActivationSequence.LifesWork.Key,
		v.ActivationSequence.LifesWork.Line,
		v.ActivationSequence.Evolution.Key,
		v.ActivationSequence.Evolution.Line,
		v.ActivationSequence.Radiance.Key,
		v.ActivationSequence.Radiance.Line,
		v.ActivationSequence.Purpose.Key,
		v.ActivationSequence.Purpose.Line,
	)
}
