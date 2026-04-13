package emit_golden

import (
	"bytes"
	"fmt"
)

/*
   ============
   Data Model
   ============
*/


// BoolString allows "true"/"false" strings in JSON to unmarshal into a bool.
type BoolString bool

func (b *BoolString) UnmarshalJSON(data []byte) error {
    s := string(data)
    switch s {
    case `"true"`:
        *b = true
    case `"false"`:
        *b = false
    default:
        return fmt.Errorf("invalid BoolString value: %s", s)
    }
    return nil
}



type GoldenCase struct {
    CaseID         string         `json:"case_id"`
    Birth          Birth          `json:"birth"`
    EngineContract EngineContract `json:"engine_contract"`
    Expected       Expected       `json:"expected,omitempty"` // optional in input
}

type Birth struct {
    Date          string  `json:"date"`
    TimeHHMM      string  `json:"time_hh_mm"`
    SecondsPolicy string  `json:"seconds_policy"`
    PlaceName     string  `json:"place_name"`
    Latitude      float64 `json:"latitude"`
    Longitude     float64 `json:"longitude"`
    TimezoneIANA  string  `json:"timezone_iana"`
}

type EngineContract struct {
    Ephemeris          string          `json:"ephemeris"`
    Zodiac             string          `json:"zodiac"`
    Houses             string          `json:"houses"`
    NodePolicyBySystem NodePolicyBySystem `json:"node_policy_by_system"`
    HumanDesignMapping HumanDesignMapping `json:"human_design_mapping"`
    DesignTimeSolver   DesignTimeSolver   `json:"design_time_solver"`
}

type NodePolicyBySystem struct {
    Astrology   string `json:"astrology"`
    HumanDesign BoolString `json:"human_design"` // string "true"/"false"
    GeneKeys    BoolString `json:"gene_keys"`    // string "true"/"false"
}

type HumanDesignMapping struct {
    MandalaStartDeg float64 `json:"mandala_start_deg"`
    GateWidthDeg    float64 `json:"gate_width_deg"`
    LineWidthDeg    float64 `json:"line_width_deg"`
    IntervalRule    string  `json:"interval_rule"`
}

type DesignTimeSolver struct {
    SunOffsetDeg                  float64 `json:"sun_offset_deg"`
    StopIfAbsSunDiffDegBelow      float64 `json:"stop_if_abs_sun_diff_deg_below"`
    StopIfTimeBracketBelowSeconds int     `json:"stop_if_time_bracket_below_seconds"`
}

type Expected struct {
    Astrology   Astrology   `json:"astrology"`
    HumanDesign HumanDesign `json:"human_design"`
    GeneKeys    GeneKeys    `json:"gene_keys"`
}

type Astrology struct {
    Positions AstrologyPositions `json:"positions"`
}

type AstrologyPositions struct {
    Sun           Position `json:"sun"`
    Moon          Position `json:"moon"`
    Mercury       Position `json:"mercury"`
    Venus         Position `json:"venus"`
    Mars          Position `json:"mars"`
    Jupiter       Position `json:"jupiter"`
    Saturn        Position `json:"saturn"`
    Uranus        Position `json:"uranus"`
    Neptune       Position `json:"neptune"`
    Pluto         Position `json:"pluto"`
    Chiron        Position `json:"chiron"`
    NorthNodeMean Position `json:"north_node_mean"`
    Ascendant     Position `json:"ascendant"`
    MC            Position `json:"mc"`
}

type Position struct {
    Sign string `json:"sign"`
    Deg  int    `json:"deg"`
    Min  int    `json:"min"`
}

type HumanDesign struct {
    ActivationObjectOrder []string          `json:"activation_object_order"`
    Personality           map[string]string `json:"personality"`
    Design                map[string]string `json:"design"`
}

type GeneKeys struct {
    ActivationSequence ActivationSequence `json:"activation_sequence"`
}

type ActivationSequence struct {
    LifesWork ActivationKey `json:"lifes_work"`
    Evolution ActivationKey `json:"evolution"`
    Radiance  ActivationKey `json:"radiance"`
    Purpose   ActivationKey `json:"purpose"`
}


type ActivationKey struct {
    Key  int `json:"key"`
    Line int `json:"line"`
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
	w("\n}")

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
      "human_design": "%v",
      "gene_keys": "%v"
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

func emitPositionBlock(b *bytes.Buffer, name string, p Position) {
	fmt.Fprintf(b,
`        "%s": { "sign": "%s", "deg": %d, "min": %d }`,
		name, p.Sign, p.Deg, p.Min,
	)
}

func emitAstrologyPositions(w *bytes.Buffer, p AstrologyPositions) error {

	type entry struct {
		name string
		pos  Position
	}

	entries := []entry{
		{"sun", p.Sun},
		{"moon", p.Moon},
		{"mercury", p.Mercury},
		{"venus", p.Venus},
		{"mars", p.Mars},
		{"jupiter", p.Jupiter},
		{"saturn", p.Saturn},
		{"uranus", p.Uranus},
		{"neptune", p.Neptune},
		{"pluto", p.Pluto},
		{"chiron", p.Chiron},
		{"north_node_mean", p.NorthNodeMean},
		{"ascendant", p.Ascendant},
		{"mc", p.MC},
	}

	for i, e := range entries {
		if i > 0 {
			w.WriteString(",\n")
		}
		emitPositionBlock(w, e.name, e.pos)
	}

	return nil
}

func emitAstrology(b *bytes.Buffer, v Astrology) {
	fmt.Fprintf(b, `    "astrology": {
      "positions": {
`)
	emitAstrologyPositions(b,v.Positions)
	fmt.Fprintf(b, `
      }
    }`)
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
