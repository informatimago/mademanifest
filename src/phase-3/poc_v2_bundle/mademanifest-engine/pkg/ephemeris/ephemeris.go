package ephemeris

import (
	//	"os"
	"log"
	"fmt"
	"math"
    "github.com/mshafiee/swephgo"
	"mademanifest-engine/pkg/sweph"
)


var swe_initialized = false

func longitude(julianDay float64, astre int) float64 {
	if !swe_initialized {
		swephgo.SetEphePath([]byte("/usr/local/share/swisseph/\x00"))
		// swephgo.SetJplFile([]byte("/usr/local/share/swisseph/de200.eph"))
		swe_initialized = true
	}

    // Prepare output slices
    xx := make([]float64, 6)     // x[0]=longitude, x[1]=latitude, x[2]=distance, etc.
    // var serr []byte               // optional error buffer; nil if you don't need errors
	// var serr []byte //
	serr := make([]byte, 256)
	// log.Printf("SE_EPHE_PATH: %v", os.Getenv("SE_EPHE_PATH"))
	// log.Printf("julianDay %v, astre %v, sweph.SEFLG_SWIEPH %v, xx %v, serr %v", julianDay, astre, sweph.SEFLG_SWIEPH, xx, serr)
    // Call swephgo.Calc
    errCode := swephgo.Calc(julianDay, astre,
		sweph.SEFLG_SWIEPH,
		// sweph.SEFLG_SWIEPH | sweph.SEFLG_TRUEPOS | sweph.SEFLG_NONUT,
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


// CalculatePositions computes positions of celestial bodies using Swiss Ephemeris
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
	} else {
		return longitude(julianDay,AsterConstantByName(astre))
	}
}

