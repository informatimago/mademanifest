package output

// success.go declares the Trinity v1 success-response envelope and
// every nested type the canon enumerates.  Field declaration order is
// the canonical key order: encoding/json marshals struct fields in
// declaration order, so as long as the source matches trinity.org
// the JSON output does too.  The struct-tag JSON keys follow
// trinity.org §"Output Contract" lines 439-558.
//
// Phase 3 ships the type system and the placeholder constructor in
// placeholder.go.  Phases 4-8 populate the calculation values:
//   * Phase 4: Astrology.Angles, Astrology.HouseCusps, Astrology.Objects.
//   * Phase 5: HumanDesign.System.DesignTimeUTC.
//   * Phase 6: HumanDesign.PersonalityActivations / DesignActivations.
//   * Phase 7: HumanDesign.Channels / Centers / Definition / Type /
//     Authority / Profile / IncarnationCross.
//   * Phase 8: GeneKeys.Activations.
//
// Until those phases land, the placeholder constructor emits zero
// values for the calculation fields.

// SuccessEnvelope is the top-level Trinity success response.  See
// trinity.org §"Success Response" lines 440-449.
type SuccessEnvelope struct {
	Status      string         `json:"status"` // always "success"
	Metadata    Metadata       `json:"metadata"`
	InputEcho   InputEcho      `json:"input_echo"`
	Astrology   Astrology      `json:"astrology"`
	HumanDesign HumanDesignOut `json:"human_design"`
	GeneKeys    GeneKeysOut    `json:"gene_keys"`
}

// InputEcho re-emits the canonical input fields.  Per
// trinity.org §"Input Echo" lines 464-471 only the five canonical
// payload fields are echoed – nothing else from the original
// request body may leak into the response.
type InputEcho struct {
	BirthDate string    `json:"birth_date"`
	BirthTime string    `json:"birth_time"`
	Timezone  string    `json:"timezone"`
	Latitude  Longitude `json:"latitude"`
	Longitude Longitude `json:"longitude"`
}

// Astrology is the astrology section of the success envelope.
// See trinity.org §"Astrology Output" lines 473-503.
type Astrology struct {
	System     AstroSystem   `json:"system"`
	Angles     Angles        `json:"angles"`
	HouseCusps []HouseCusp   `json:"house_cusps"`
	Objects    []AstroObject `json:"objects"`
}

// AstroSystem pins the three system-block scalars (zodiac,
// house_system, node_type).  All three are canon constants: the
// placeholder constructor sets them to "tropical", "placidus",
// "mean" verbatim.
type AstroSystem struct {
	Zodiac      string `json:"zodiac"`
	HouseSystem string `json:"house_system"`
	NodeType    string `json:"node_type"`
}

// Angles bundles the ascendant and midheaven points that anchor the
// chart.  See trinity.org §"Astrology Output" lines 482-484.
type Angles struct {
	Ascendant SignedLongitude `json:"ascendant"`
	Midheaven SignedLongitude `json:"midheaven"`
}

// SignedLongitude is the {longitude, sign} pair used by the angles
// block – the canon does not require a house number on angles.
type SignedLongitude struct {
	Longitude Longitude `json:"longitude"`
	Sign      string    `json:"sign"`
}

// HouseCusp is one of the twelve placidus cusps emitted in canonical
// house order 1..12.
type HouseCusp struct {
	House     int       `json:"house"`
	Longitude Longitude `json:"longitude"`
	Sign      string    `json:"sign"`
}

// AstroObject is one entry in the astrology objects array.  The
// canon (Document 02) requires the four-field shape
// {object_id, longitude, sign, house}.
type AstroObject struct {
	ObjectID  string    `json:"object_id"`
	Longitude Longitude `json:"longitude"`
	Sign      string    `json:"sign"`
	House     int       `json:"house"`
}

// HumanDesignOut is the human_design section of the success
// envelope.  See trinity.org §"Human Design Output" lines 504-545.
//
// The "Out" suffix disambiguates the type from the existing PoC-era
// emit_golden.HumanDesign struct, which carries different fields.
type HumanDesignOut struct {
	System                 HDSystem            `json:"system"`
	PersonalityActivations []HDActivation      `json:"personality_activations"`
	DesignActivations      []HDActivation      `json:"design_activations"`
	Channels               []HDChannel         `json:"channels"`
	Centers                []HDCenter          `json:"centers"`
	Definition             string              `json:"definition"`
	Type                   string              `json:"type"`
	Authority              string              `json:"authority"`
	Profile                string              `json:"profile"`
	IncarnationCross       HDIncarnationCross  `json:"incarnation_cross"`
}

// HDSystem holds node_type and design_time_utc.  The canon pins
// node_type to "true" (Document 03 §"Node policy by domain", line
// 68); design_time_utc is computed by the bisection solver in
// Phase 5 and formatted by DesignTime.MarshalJSON.
type HDSystem struct {
	NodeType      string     `json:"node_type"`
	DesignTimeUTC DesignTime `json:"design_time_utc"`
}

// HDActivation is one row of the personality / design activation
// snapshot: object_id, gate, line.
type HDActivation struct {
	ObjectID string `json:"object_id"`
	Gate     int    `json:"gate"`
	Line     int    `json:"line"`
}

// HDChannel is the canonical channel emission shape: channel_id,
// the two gates, and the two centers it connects.  ChannelTable in
// pkg/canon is the source of truth.
type HDChannel struct {
	ChannelID string `json:"channel_id"`
	GateA     int    `json:"gate_a"`
	GateB     int    `json:"gate_b"`
	CenterA   string `json:"center_a"`
	CenterB   string `json:"center_b"`
}

// HDCenter is one row of the centers array: center_id and a state
// of "defined" or "undefined".  Order in the response array follows
// canon.CenterOrder (head, ajna, throat, g, ego, solar_plexus,
// sacral, spleen, root).
type HDCenter struct {
	CenterID string `json:"center_id"`
	State    string `json:"state"`
}

// HDIncarnationCross is the structural encoding of the four pillar
// activations: personality sun + earth, design sun + earth.  No
// human-readable cross name is required (trinity.org line 339).
type HDIncarnationCross struct {
	PersonalitySun   HDGateLine `json:"personality_sun"`
	PersonalityEarth HDGateLine `json:"personality_earth"`
	DesignSun        HDGateLine `json:"design_sun"`
	DesignEarth      HDGateLine `json:"design_earth"`
}

// HDGateLine is the {gate, line} sub-shape that the incarnation
// cross uses.  Same conceptual content as HDActivation minus the
// object_id, since the four cross slots are positional.
type HDGateLine struct {
	Gate int `json:"gate"`
	Line int `json:"line"`
}

// GeneKeysOut is the gene_keys section of the success envelope.
// See trinity.org §"Gene Keys Output" lines 547-558.
type GeneKeysOut struct {
	System      GKSystem    `json:"system"`
	Activations GKActivations `json:"activations"`
}

// GKSystem pins derivation_basis to "human_design".  No other
// fields appear in the canon.
type GKSystem struct {
	DerivationBasis string `json:"derivation_basis"`
}

// GKActivations holds the four canonical gene-keys positions.  Note
// the canonical key is "life_work" – the PoC emitter spelled it
// "lifes_work" and that drift is corrected here.
type GKActivations struct {
	LifeWork  GKActivation `json:"life_work"`
	Evolution GKActivation `json:"evolution"`
	Radiance  GKActivation `json:"radiance"`
	Purpose   GKActivation `json:"purpose"`
}

// GKActivation is the {key, line} pair per gene-keys position.
// "key" is the gate number, "line" is the line number.
type GKActivation struct {
	Key  int `json:"key"`
	Line int `json:"line"`
}
