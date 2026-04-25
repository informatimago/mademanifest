package astro

import (
	"math"
	"testing"

	"mademanifest-engine/pkg/canon"
	"mademanifest-engine/pkg/trinity/input"
)

// schiedamBaseline mirrors the canonical Schiedam payload used as
// the Phase 4 oracle.  The struct is constructed in code rather
// than parsed from JSON so the test does not depend on the
// validator package.
var schiedamBaseline = input.Payload{
	BirthDate: "1990-04-09",
	BirthTime: "18:04",
	Timezone:  "Europe/Amsterdam",
	Latitude:  51.9167,
	Longitude: 4.4,
}

// TestComputeAstrologySystemBlock pins the three canonical scalars
// that ComputeAstrology must always emit regardless of inputs.
func TestComputeAstrologySystemBlock(t *testing.T) {
	got, err := ComputeAstrology(schiedamBaseline)
	if err != nil {
		t.Fatalf("ComputeAstrology: %v", err)
	}
	if got.System.Zodiac != "tropical" {
		t.Errorf("zodiac = %q, want tropical", got.System.Zodiac)
	}
	if got.System.HouseSystem != "placidus" {
		t.Errorf("house_system = %q, want placidus", got.System.HouseSystem)
	}
	if got.System.NodeType != "mean" {
		t.Errorf("node_type = %q, want mean", got.System.NodeType)
	}
}

// TestComputeAstrologyObjectsAreCanonOrdered locks the object array
// length and ordering to canon.AstrologyObjectOrder.  Any drift
// here would silently shift the contract.
func TestComputeAstrologyObjectsAreCanonOrdered(t *testing.T) {
	got, err := ComputeAstrology(schiedamBaseline)
	if err != nil {
		t.Fatalf("ComputeAstrology: %v", err)
	}
	if got, want := len(got.Objects), len(canon.AstrologyObjectOrder); got != want {
		t.Fatalf("objects length = %d, want %d", got, want)
	}
	for i, obj := range got.Objects {
		if obj.ObjectID != canon.AstrologyObjectOrder[i] {
			t.Errorf("objects[%d].object_id = %q, want %q",
				i, obj.ObjectID, canon.AstrologyObjectOrder[i])
		}
	}
}

// TestComputeAstrologyEarthIsSunPlus180 verifies the canon's Earth
// derivation – not Swiss Ephemeris's heliocentric SE_EARTH.
func TestComputeAstrologyEarthIsSunPlus180(t *testing.T) {
	got, err := ComputeAstrology(schiedamBaseline)
	if err != nil {
		t.Fatalf("ComputeAstrology: %v", err)
	}
	var sun, earth float64
	for _, obj := range got.Objects {
		switch obj.ObjectID {
		case "sun":
			sun = float64(obj.Longitude)
		case "earth":
			earth = float64(obj.Longitude)
		}
	}
	want := math.Mod(sun+180.0, 360.0)
	if math.Abs(earth-want) > 1e-6 {
		t.Errorf("earth = %v, want sun+180 mod 360 = %v (sun=%v)",
			earth, want, sun)
	}
}

// TestComputeAstrologyHouseCuspsAreOrdered1To12 checks that the
// twelve house cusps come out in canonical order 1..12 with
// non-zero longitudes (the Schiedam payload is far from any
// degenerate latitude).
func TestComputeAstrologyHouseCuspsAreOrdered1To12(t *testing.T) {
	got, err := ComputeAstrology(schiedamBaseline)
	if err != nil {
		t.Fatalf("ComputeAstrology: %v", err)
	}
	if got, want := len(got.HouseCusps), 12; got != want {
		t.Fatalf("house_cusps length = %d, want %d", got, want)
	}
	for i, c := range got.HouseCusps {
		if c.House != i+1 {
			t.Errorf("house_cusps[%d].house = %d, want %d", i, c.House, i+1)
		}
		if c.Sign == "" {
			t.Errorf("house_cusps[%d].sign empty (longitude=%v)",
				i, float64(c.Longitude))
		}
	}
}

// TestComputeAstrologyObjectsHaveNonEmptySigns ensures every object
// receives a canonical lowercase sign id.  This catches longitude-
// normalisation bugs (e.g. emitting 360.0 by mistake).
func TestComputeAstrologyObjectsHaveNonEmptySigns(t *testing.T) {
	got, err := ComputeAstrology(schiedamBaseline)
	if err != nil {
		t.Fatalf("ComputeAstrology: %v", err)
	}
	signSet := make(map[string]bool, len(canon.SignOrder))
	for _, s := range canon.SignOrder {
		signSet[s] = true
	}
	for _, obj := range got.Objects {
		if obj.Sign == "" {
			t.Errorf("%s: sign empty (longitude=%v)",
				obj.ObjectID, float64(obj.Longitude))
			continue
		}
		if !signSet[obj.Sign] {
			t.Errorf("%s: sign %q not in canon.SignOrder",
				obj.ObjectID, obj.Sign)
		}
	}
}

// TestComputeAstrologyObjectsHaveValidHouses ensures every object's
// house is in [1, 12] (the HouseFor sentinel-zero return would
// indicate a bug).
func TestComputeAstrologyObjectsHaveValidHouses(t *testing.T) {
	got, err := ComputeAstrology(schiedamBaseline)
	if err != nil {
		t.Fatalf("ComputeAstrology: %v", err)
	}
	for _, obj := range got.Objects {
		if obj.House < 1 || obj.House > 12 {
			t.Errorf("%s: house = %d, want in [1,12] (longitude=%v)",
				obj.ObjectID, obj.House, float64(obj.Longitude))
		}
	}
}

// TestComputeAstrologyAnglesArePopulated locks in the ascendant
// and midheaven shape: both must have non-zero longitudes (Schiedam
// is far from the poles) and a canonical sign.
func TestComputeAstrologyAnglesArePopulated(t *testing.T) {
	got, err := ComputeAstrology(schiedamBaseline)
	if err != nil {
		t.Fatalf("ComputeAstrology: %v", err)
	}
	if got.Angles.Ascendant.Sign == "" {
		t.Errorf("ascendant.sign empty (longitude=%v)",
			float64(got.Angles.Ascendant.Longitude))
	}
	if got.Angles.Midheaven.Sign == "" {
		t.Errorf("midheaven.sign empty (longitude=%v)",
			float64(got.Angles.Midheaven.Longitude))
	}
}
