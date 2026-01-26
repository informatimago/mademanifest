package gene_keys

import (
	"testing"
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
	humanDesignData := map[string]interface{}{
		"activation_object_order": []string{
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
		},

		"personality": map[string]string{
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
		},

		"design": map[string]string{
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
		},
	}

	result := DeriveGeneKeys(humanDesignData)

	// Verify that result map is not nil (function exists)
	if result == nil {
		t.Error("DeriveGeneKeys should return a valid result map")
	}

	// Basic verification that function signature works
	if len(result) < 1 {
		t.Logf("Gene keys result is not empty")
	}

	t.Logf("result = %v",result)
}
