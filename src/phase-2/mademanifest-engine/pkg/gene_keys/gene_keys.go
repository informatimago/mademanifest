package gene_keys

import (
	"fmt"
	"math"
)

// DeriveGeneKeys computes Gene Key values using Human Design output (string-formatted)
func DeriveGeneKeys(humanDesignData map[string]interface{}) map[string]int {
	result := make(map[string]int)

	personality := humanDesignData["personality"].(map[string]string)
	design := humanDesignData["design"].(map[string]string)

	// Gene Keys mapping per specification
	result["lifes_work"] = parseGate(personality["sun"])
	result["evolution"] = parseGate(personality["earth"])
	result["radiance"] = parseGate(design["sun"])
	result["purpose"] = parseGate(design["earth"])

	return result
}

// parseGate converts a Human Design string value ("61.1") into its integer gate number using floor
func parseGate(val string) int {
	var f float64
	fmt.Sscanf(val, "%f", &f)
	return int(math.Floor(f))
}
