package ephemeris

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mshafiee/swephgo"
	"mademanifest-engine/pkg/sweph"
)

var swe_initialized = false
const requiredSwissEphVersion = "2.10.03"

func resolveEphemerisPath() string {
	if ephePath := os.Getenv("SE_EPHE_PATH"); ephePath != "" {
		return ephePath
	}

	candidates := []string{
		"../ephemeris/data/REQUIRED_EPHEMERIS_FILES/",
		"/usr/local/share/swisseph/",
	}

	if _, filename, _, ok := runtime.Caller(0); ok {
		candidates = append([]string{
			filepath.Join(filepath.Dir(filename), "..", "..", "..", "ephemeris", "data", "REQUIRED_EPHEMERIS_FILES"),
		}, candidates...)
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	return "../ephemeris/data/REQUIRED_EPHEMERIS_FILES/"
}

func requireSwissEphVersion() {
	buf := make([]byte, 256)
	swephgo.Version(buf)
	n := bytes.IndexByte(buf, 0)
	if n < 0 {
		n = len(buf)
	}
	version := strings.TrimSpace(string(buf[:n]))
	if version == "" {
		log.Fatal("Swiss Ephemeris version check failed: empty version string")
	}
	if version != requiredSwissEphVersion {
		log.Fatalf("Swiss Ephemeris version mismatch: got %q, want %q", version, requiredSwissEphVersion)
	}
}

func longitude(julianDay float64, astre int) float64 {
	if !swe_initialized {
		ephePath := resolveEphemerisPath()
		swephgo.SetEphePath([]byte(ephePath + "\x00"))
		requireSwissEphVersion()
		swe_initialized = true
	}

	// Prepare output slices
	xx := make([]float64, 6)     // x[0]=longitude, x[1]=latitude, x[2]=distance, etc.
	serr := make([]byte, 256)
	// Call swephgo.Calc
	errCode := swephgo.Calc(julianDay, astre,
		sweph.SEFLG_SWIEPH,
		xx, serr)
	if errCode < 0 {
		// handle error if needed, e.g. log.Fatal or return NaN
		log.Printf("swephgo.Calc error: %+v", string(serr))
		panic("swephgo.Calc failed with error code " + fmt.Sprint(errCode))
	}

	return xx[0] // longitude in degrees
}

var asterConstants = []struct {
	Name     string
	Constant int
}{
	{"earth",    sweph.SE_EARTH},
	{"sun",      sweph.SE_SUN},
	{"moon",     sweph.SE_MOON},
	{"mercury",  sweph.SE_MERCURY},
	{"venus",    sweph.SE_VENUS},
	{"mars",     sweph.SE_MARS},
	{"jupiter",  sweph.SE_JUPITER},
	{"saturn",   sweph.SE_SATURN},
	{"uranus",   sweph.SE_URANUS},
	{"neptune",  sweph.SE_NEPTUNE},
	{"pluto",    sweph.SE_PLUTO},
	{"chiron",   sweph.SE_CHIRON},

	// Node policy by domain (trinity.org §"Node policy by domain"):
	//   * astrology   → SE_MEAN_NODE
	//   * human_design → SE_TRUE_NODE
	//   * gene_keys    → derived from human_design, also true
	//
	// The two distinct names below let callers pick the policy
	// explicitly without an environment-variable switch.  Phase 6
	// retired the SE_NODE_POLICY environment lookup that previously
	// made the same name return either MEAN or TRUE depending on
	// runtime state — that crosstalk broke the determinism the
	// canon mandates and is incompatible with deploying the engine
	// to multi-tenant environments where envvars cannot be trusted
	// to stay constant across requests.
	{"north_node_mean",  sweph.SE_MEAN_NODE},
	{"north_node",       sweph.SE_MEAN_NODE},  // legacy: same as mean for backward-compat
	{"north_node_true",  sweph.SE_TRUE_NODE},
}

func CalculatePositions(julianDay float64) map[string]float64 {
	// Using Swiss Ephemeris to compute positions of astronomical bodies
	positions := make(map[string]float64)
	for _, aster := range asterConstants {
		positions[aster.Name] = longitude(julianDay, aster.Constant)
	}
	return positions
}

func AsterConstantByName(name string) int {
	for _, a := range asterConstants {
		if a.Name == name {
			return a.Constant
		}
	}
	panic("unknown aster: " + name)
}

// GetPlanetLongAtTime returns the geocentric ecliptic longitude in
// degrees for the named body at the given Julian Day.  Node policy
// is selected by the body name:
//
//   * "north_node"      → SE_MEAN_NODE (legacy name; astrology default)
//   * "north_node_mean" → SE_MEAN_NODE
//   * "north_node_true" → SE_TRUE_NODE (Human Design canon)
//   * "south_node"      → MEAN-node + 180° (legacy)
//   * "south_node_true" → TRUE-node + 180° (Human Design canon)
//
// Phase 6 removed the SE_NODE_POLICY environment-variable switch
// that previously toggled "north_node" between mean and true at
// runtime; that switch was incompatible with the canon's
// per-domain policy (astrology must always be mean, HD must always
// be true) and broke determinism across deployments.
func GetPlanetLongAtTime(julianDay float64, astre string) float64 {
	switch astre {
	case "south_node":
		return math.Mod(180.0+longitude(julianDay, AsterConstantByName("north_node")), 360.0)
	case "south_node_true":
		return math.Mod(180.0+longitude(julianDay, AsterConstantByName("north_node_true")), 360.0)
	default:
		return longitude(julianDay, AsterConstantByName(astre))
	}
}
