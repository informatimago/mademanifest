package gene_keys

import (
	"log"
	"testing"
	"mademanifest-engine/pkg/emit_golden"
)

// Fixed object order
var activationObjectOrder = []string{
	"sun",
	"earth",
	"north_node",
	"south_node",
	"moon",
	"mercury",
	"venus",
	"mars",
	"jupiter",
	"saturn",
	"uranus",
	"neptune",
	"pluto",
}


func TestDeriveGeneKeys(t *testing.T) {
	// Test that the function can be called and returns a map
	var humanDesignData emit_golden.HumanDesign

	humanDesignData.ActivationObjectOrder = []string{
			"sun",
			"earth",
			"north_node",
			"south_node",
			"moon",
			"mercury",
			"venus",
			"mars",
			"jupiter",
			"saturn",
			"uranus",
			"neptune",
			"pluto",
		}

	humanDesignData.Personality = map[string]string{
			"sun":         "15.3",
			"earth":       "195.3",
			"north_node":  "120.5",
			"south_node":  "300.5",
			"moon":        "45.7",
			"mercury":     "10.2",
			"venus":       "88.1",
			"mars":        "270.9",
			"jupiter":     "102.4",
			"saturn":      "210.6",
			"uranus":      "33.3",
			"neptune":     "355.8",
			"pluto":       "123.4",
		}

	humanDesignData.Design = map[string]string{
			"sun":         "14.9",
			"earth":       "194.9",
			"north_node":  "121.0",
			"south_node":  "301.0",
			"moon":        "44.8",
			"mercury":     "9.8",
			"venus":       "87.5",
			"mars":        "271.2",
			"jupiter":     "103.0",
			"saturn":      "211.1",
			"uranus":      "34.0",
			"neptune":     "356.1",
			"pluto":       "124.0",
		}

	result := DeriveGeneKeys(humanDesignData)
	log.Printf("result = %v",result)
	// Basic verification that function signature works
	assert(t,result.ActivationSequence.LifesWork.Key == 15, "result.ActivationSequence.LifesWork.Key != 15")
	assert(t,result.ActivationSequence.LifesWork.Line == 3, "result.ActivationSequence.LifesWork.Line != 3")
	assert(t,result.ActivationSequence.Evolution.Key == 195, "result.ActivationSequence.Evolution.Key != 195")
	assert(t,result.ActivationSequence.Evolution.Line == 3, "result.ActivationSequence.Evolution.Line != 3")
	assert(t,result.ActivationSequence.Radiance.Key == 14, "result.ActivationSequence.Radiance.Key != 14")
	assert(t,result.ActivationSequence.Radiance.Line == 9, "result.ActivationSequence.Radiance.Line != 9")
	assert(t,result.ActivationSequence.Purpose.Key == 194, "result.ActivationSequence.Radiance.Key != 194")
	assert(t,result.ActivationSequence.Purpose.Line == 9, "result.ActivationSequence.Radiance.Line != 9")


	t.Logf("result = %v",result)
}


func assert(t *testing.T,cond bool, msg string) {
    if !cond {
		t.Fatalf("%s",msg)
    }
}
