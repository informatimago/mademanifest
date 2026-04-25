package output

import (
	"mademanifest-engine/pkg/canon"
	"mademanifest-engine/pkg/trinity/input"
)

// NewPlaceholderSuccess builds a canonically-shaped SuccessEnvelope
// from a validated Trinity payload.  It fills in every structural
// field that Phase 3 can determine without running the calculation
// pipeline:
//
//   * status / metadata / input_echo are fully populated.
//   * astrology.system pins zodiac=tropical, house_system=placidus,
//     node_type=mean (canon constants from Document 03).
//   * human_design.system pins node_type=true; design_time_utc
//     defaults to the zero time and will be replaced by Phase 5.
//   * gene_keys.system pins derivation_basis=human_design.
//   * centers is populated with all nine canonical centers in
//     canon.CenterOrder, each undefined.
//   * activations / channels / cusps / objects / incarnation_cross
//     are zero values until the calculation phases land.
//
// The placeholder still answers HTTP 200 because the Trinity
// envelope's status field is "success": clients see a structurally
// complete response, but the calculation values are not yet
// canonical.  Phases 4-8 replace the placeholder values one section
// at a time.  Each phase that fills a section must update the
// matching golden fixtures and bump the relevant *Version constant.
func NewPlaceholderSuccess(p input.Payload) SuccessEnvelope {
	centers := make([]HDCenter, 0, len(canon.CenterOrder))
	for _, id := range canon.CenterOrder {
		centers = append(centers, HDCenter{
			CenterID: id,
			State:    "undefined",
		})
	}
	return SuccessEnvelope{
		Status:    StatusSuccess,
		Metadata:  CurrentMetadata(),
		InputEcho: InputEcho{
			BirthDate: p.BirthDate,
			BirthTime: p.BirthTime,
			Timezone:  p.Timezone,
			Latitude:  Longitude(p.Latitude),
			Longitude: Longitude(p.Longitude),
		},
		Astrology: Astrology{
			System: AstroSystem{
				Zodiac:      "tropical",
				HouseSystem: "placidus",
				NodeType:    "mean",
			},
			HouseCusps: []HouseCusp{},   // Phase 4
			Objects:    []AstroObject{}, // Phase 4
		},
		HumanDesign: HumanDesignOut{
			System: HDSystem{
				NodeType: "true",
				// DesignTimeUTC is the zero value; Phase 5
				// replaces it with the bisection result.
			},
			PersonalityActivations: []HDActivation{}, // Phase 6
			DesignActivations:      []HDActivation{}, // Phase 6
			Channels:               []HDChannel{},    // Phase 7
			Centers:                centers,          // Phase 7 will compute states
			Definition:             "none",           // Phase 7
			Type:                   "reflector",      // Phase 7 (reflector is canon-allowed when no centers defined)
			Authority:              "lunar",          // Phase 7 (lunar is canon-allowed for reflector)
			Profile:                "1/1",            // Phase 7 (placeholder; lines 1-6 are canon-allowed)
			// IncarnationCross is the zero value;
			// Phase 7 fills the four pillar activations.
		},
		GeneKeys: GeneKeysOut{
			System: GKSystem{
				DerivationBasis: "human_design",
			},
			// Activations is the zero-value GKActivations
			// (all keys/lines = 0); Phase 8 fills it.
		},
	}
}
