package gene_keys

import (
	"fmt"
	"mademanifest-engine/pkg/emit_golden"
)


func DeriveGeneKeys(humanDesignData emit_golden.HumanDesign) emit_golden.GeneKeys {

	var result emit_golden.GeneKeys
	result.ActivationSequence.LifesWork = parseActivationKey(humanDesignData.Personality["sun"])
	result.ActivationSequence.Evolution = parseActivationKey(humanDesignData.Personality["earth"])
	result.ActivationSequence.Radiance = parseActivationKey(humanDesignData.Design["sun"])
	result.ActivationSequence.Purpose = parseActivationKey(humanDesignData.Design["earth"])
	return result
}

// parseActivationKey converts a Human Design value like "61.1"
// into Gene Keys ActivationKey {Key: 61, Line: 1}
func parseActivationKey(val string) emit_golden.ActivationKey {
	var key, line int

	// Strictly parse "gate.line"
	_, err := fmt.Sscanf(val, "%d.%d", &key, &line)
	if err != nil {
		panic(fmt.Sprintf("invalid Human Design value: %v", val))
	}

	return emit_golden.ActivationKey{
		Key:  key,
		Line: line,
	}
}
