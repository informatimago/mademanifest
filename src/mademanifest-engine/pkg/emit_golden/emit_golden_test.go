package emit_golden

import (
	"bytes"
	"os"
	"testing"
	"log"
)

func TestEmitGoldenJSON(t *testing.T) {
	root := GoldenCase{
		CaseID: "golden_test_case_v1_jaimie_1990_04_09_1804_schiedam",
		Birth: Birth{
			Date:          "1990-04-09",
			TimeHHMM:      "18:04",
			SecondsPolicy: "assume_00",
			PlaceName:     "Schiedam, Netherlands",
			Latitude:      51.9167,
			Longitude:     4.4000,
			TimezoneIANA:  "Europe/Amsterdam",
		},
		EngineContract: EngineContract{
			Ephemeris: "swiss_ephemeris",
			Zodiac:    "tropical",
			Houses:    "placidus",
			NodePolicyBySystem: NodePolicyBySystem{
				Astrology:   "mean",
				HumanDesign: true,
				GeneKeys:    true,
			},
			HumanDesignMapping: HumanDesignMapping{
				MandalaStartDeg: 313.25,
				GateWidthDeg:    5.625,
				LineWidthDeg:    0.9375,
				IntervalRule:    "start_inclusive_end_exclusive",
			},
			DesignTimeSolver: DesignTimeSolver{
				SunOffsetDeg:                  88.0,
				StopIfAbsSunDiffDegBelow:      0.0001,
				StopIfTimeBracketBelowSeconds: 1,
			},
		},
		Expected: Expected{
			Astrology: Astrology{
				Positions: AstrologyPositions{
					Sun:           Position{"Aries", 19, 32},
					Moon:          Position{"Libra", 14, 20},
					Mercury:       Position{"Taurus", 8, 16},
					Venus:         Position{"Pisces", 3, 23},
					Mars:          Position{"Aquarius", 21, 35},
					Jupiter:       Position{"Cancer", 3, 46},
					Saturn:        Position{"Capricorn", 24, 49},
					Uranus:        Position{"Capricorn", 9, 34},
					Neptune:       Position{"Capricorn", 14, 33},
					Pluto:         Position{"Scorpio", 17, 8},
					Chiron:        Position{"Cancer", 11, 3},
					NorthNodeMean: Position{"Aquarius", 13, 14},
					Ascendant:     Position{"Virgo", 25, 6},
					MC:            Position{"Gemini", 23, 35},
				},
			},
			HumanDesign: HumanDesign{
				ActivationObjectOrder: []string{
					"sun", "earth", "north_node", "south_node", "moon",
					"mercury", "venus", "mars", "jupiter", "saturn",
					"uranus", "neptune", "pluto",
				},
				Personality: map[string]string{
					"sun": "51.5", "earth": "57.5",
					"north_node": "13.2", "south_node": "7.2",
					"moon": "48.6", "mercury": "24.1",
					"venus": "55.4", "mars": "49.3",
					"jupiter": "15.6", "saturn": "61.5",
					"uranus": "38.1", "neptune": "38.6",
					"pluto": "1.5",
				},
				Design: map[string]string{
					"sun": "61.1", "earth": "62.1",
					"north_node": "13.4", "south_node": "7.4",
					"moon": "31.1", "mercury": "38.6",
					"venus": "41.1", "mars": "26.1",
					"jupiter": "15.6", "saturn": "54.2",
					"uranus": "58.3", "neptune": "38.4",
					"pluto": "1.5",
				},
			},
			GeneKeys: GeneKeys{
				ActivationSequence: ActivationSequence{
					LifesWork: ActivationKey{Key: 51, Line: 5},
					Evolution: ActivationKey{Key: 57, Line: 5},
					Radiance:  ActivationKey{Key: 61, Line: 1},
					Purpose:   ActivationKey{Key: 62, Line: 1},
				},
			},
		},
	}

	actual, err := EmitGoldenJSON(root)
	if err != nil {
		t.Fatalf("EmitGoldenJSON failed: %v", err)
	}

	expected, err := os.ReadFile("../../../data/GOLDEN_TEST_CASE_V1.json")
	if err != nil {
		t.Fatalf("failed to read golden fixture: %v", err)
	}

	if !bytes.Equal(actual, expected) {
		log.Printf("actual   = %+v", string(actual))
		log.Printf("expected = %+v", string(expected))
		t.Fatalf("golden JSON mismatch: output does not match canonical fixture")
	}
}
