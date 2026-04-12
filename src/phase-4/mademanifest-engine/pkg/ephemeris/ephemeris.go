package ephemeris

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"os"
	"strings"

	"github.com/mshafiee/swephgo"
	"mademanifest-engine/pkg/sweph"
)

var swe_initialized = false
const requiredSwissEphVersion = "2.10.03"

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
		ephePath := os.Getenv("SE_EPHE_PATH")
		if ephePath == "" {
			ephePath = "../ephemeris/data/REQUIRED_EPHEMERIS_FILES/"
		}
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
	{"north_node_mean",  sweph.SE_MEAN_NODE},
	{"north_node",       sweph.SE_MEAN_NODE},
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
	panic("unknown aster PJB: " + name)
	return 0
}

func GetPlanetLongAtTime(julianDay float64, astre string) float64 {
	if astre == "south_node" {
		return math.Mod(180.0 +  longitude(julianDay,AsterConstantByName("north_node")), 360.0)
	} else if astre == "north_node" {
		// Node policy: mean node for astrology, true node for human_design
		if os.Getenv("SE_NODE_POLICY") == "true" {
			return longitude(julianDay, sweph.SE_TRUE_NODE)
		}
		return longitude(julianDay, sweph.SE_MEAN_NODE)
	} else {
		return longitude(julianDay,AsterConstantByName(astre))
	}
}
