package canon

// constants.go holds every pinned calculation constant and lookup
// table that the Trinity canon treats as fixed.  The values here are
// the single source of truth for later phases (astrology, human
// design, gene keys) – they must not depend on runtime configuration
// or environment variables.
//
// All numbers, sequences, and tables mirror
// specifications/trinity/trinity.org, which in turn summarises
// Documents 02, 05, 06, and 07 of the Trinity canon package.
//
// Divergence from src/canon/*.json (A8 – consistency pass):
//   The legacy JSON files under src/canon/ carry an older mandala
//   anchor of 313.25° (gate sequence starting with 13) inherited
//   from the pre-Trinity PoC bundle.  Trinity.org pins the canonical
//   anchor at 277.5° (sequence starting with 38) and explicitly
//   lists "rejected old gate anchor assumptions" among its
//   regression sentinels.  This package follows trinity.org.  The
//   PoC calculation path continues to load the JSON values; the two
//   will converge when Phase 9 removes the JSON-driven path.

// Mandala numeric constants.  See trinity.org lines 274-300.
const (
	// MandalaAnchorDeg is the start longitude of Gate 38 in the
	// canonical Human Design mandala.
	MandalaAnchorDeg = 277.5

	// GateWidthDeg is the angular span of each of the 64 gates.
	// 360 / 64 = 5.625.
	GateWidthDeg = 5.625

	// LineWidthDeg is the angular span of each of the 6 lines
	// within a gate.  5.625 / 6 = 0.9375.
	LineWidthDeg = 0.9375
)

// GateOrder is the canonical 64-gate sequence starting at
// MandalaAnchorDeg, ascending by GateWidthDeg per entry.  The first
// entry (38) starts at 277.5°; the fifth (41) starts at 300.0° as
// required by trinity.org lines 297-300.
var GateOrder = [64]int{
	38, 54, 61, 60, 41, 19, 13, 49,
	30, 55, 37, 63, 22, 36, 25, 17,
	21, 51, 42, 3, 27, 24, 2, 23,
	8, 20, 16, 35, 45, 12, 15, 52,
	39, 53, 62, 56, 31, 33, 7, 4,
	29, 59, 40, 64, 47, 6, 46, 18,
	48, 57, 32, 50, 28, 44, 1, 43,
	14, 34, 9, 5, 26, 11, 10, 58,
}

// SignOrder is the canonical tropical zodiac sign sequence in
// lowercase snake_case.  Each sign spans 30° starting at its index
// in this list multiplied by 30.  See trinity.org lines 351-355.
var SignOrder = [12]string{
	"aries", "taurus", "gemini", "cancer",
	"leo", "virgo", "libra", "scorpio",
	"sagittarius", "capricorn", "aquarius", "pisces",
}

// AstrologyObjectOrder is the canonical output ordering for the
// astrology object array.  See trinity.org lines 489-502.
var AstrologyObjectOrder = [13]string{
	"sun", "moon", "mercury", "venus", "mars",
	"jupiter", "saturn", "uranus", "neptune", "pluto",
	"chiron", "north_node_mean", "earth",
}

// HDSnapshotOrder is the canonical ordering for Human Design
// personality and design snapshot objects.  Same array applies to
// both snapshots.  See trinity.org lines 112-125.
var HDSnapshotOrder = [13]string{
	"sun", "earth", "north_node", "south_node", "moon",
	"mercury", "venus", "mars", "jupiter", "saturn",
	"uranus", "neptune", "pluto",
}

// CenterOrder is the canonical Human Design center ordering used
// wherever centers are enumerated in output.  See trinity.org
// lines 374-383.
var CenterOrder = [9]string{
	"head", "ajna", "throat", "g", "ego",
	"solar_plexus", "sacral", "spleen", "root",
}

// MotorCenters is the set of Human Design centers classified as
// motors.  Order matches CenterOrder; identity is set membership,
// not a sequence.  See trinity.org lines 384-389.
var MotorCenters = [4]string{
	"root", "sacral", "solar_plexus", "ego",
}

// Channel is a canonical Human Design channel definition.
//
// ID is the lowercase-dash-joined ascending gate pair ("1-8",
// "2-14", ...); GateA < GateB in every entry in ChannelTable.
// CenterA and CenterB are the two canon centers the channel
// connects; both must appear in CenterOrder.
type Channel struct {
	ID      string
	GateA   int
	GateB   int
	CenterA string
	CenterB string
}

// ChannelTable is the 36-entry canonical Human Design channel list.
// See trinity.org lines 390-428.
var ChannelTable = [36]Channel{
	{"1-8", 1, 8, "g", "throat"},
	{"2-14", 2, 14, "g", "sacral"},
	{"3-60", 3, 60, "sacral", "root"},
	{"4-63", 4, 63, "ajna", "head"},
	{"5-15", 5, 15, "sacral", "g"},
	{"6-59", 6, 59, "solar_plexus", "sacral"},
	{"7-31", 7, 31, "g", "throat"},
	{"9-52", 9, 52, "sacral", "root"},
	{"10-20", 10, 20, "g", "throat"},
	{"10-34", 10, 34, "g", "sacral"},
	{"10-57", 10, 57, "g", "spleen"},
	{"11-56", 11, 56, "ajna", "throat"},
	{"12-22", 12, 22, "throat", "solar_plexus"},
	{"13-33", 13, 33, "g", "throat"},
	{"16-48", 16, 48, "throat", "spleen"},
	{"17-62", 17, 62, "ajna", "throat"},
	{"18-58", 18, 58, "spleen", "root"},
	{"19-49", 19, 49, "root", "solar_plexus"},
	{"20-34", 20, 34, "throat", "sacral"},
	{"20-57", 20, 57, "throat", "spleen"},
	{"21-45", 21, 45, "ego", "throat"},
	{"23-43", 23, 43, "throat", "ajna"},
	{"24-61", 24, 61, "ajna", "head"},
	{"25-51", 25, 51, "g", "ego"},
	{"26-44", 26, 44, "ego", "spleen"},
	{"27-50", 27, 50, "sacral", "spleen"},
	{"28-38", 28, 38, "spleen", "root"},
	{"29-46", 29, 46, "sacral", "g"},
	{"30-41", 30, 41, "solar_plexus", "root"},
	{"32-54", 32, 54, "spleen", "root"},
	{"34-57", 34, 57, "sacral", "spleen"},
	{"35-36", 35, 36, "throat", "solar_plexus"},
	{"37-40", 37, 40, "solar_plexus", "ego"},
	{"39-55", 39, 55, "root", "solar_plexus"},
	{"42-53", 42, 53, "sacral", "root"},
	{"47-64", 47, 64, "ajna", "head"},
}
