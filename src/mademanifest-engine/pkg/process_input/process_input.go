package process_input

import (
	"encoding/json"
	"mademanifest-engine/pkg/canon"
	"mademanifest-engine/pkg/emit_golden"
)

// ProcessInput decodes the input JSON into the InputData structure.
// It first decodes into a generic map so that missing fields can be
// distinguished from zero values.  Canon defaults are loaded via
// canon.LoadDefaults and merged into the input map – input values win.
// The merged map is then marshalled back to JSON and finally
// unmarshalled into the strong GoldenCase type.
func ProcessInput(decoder *json.Decoder, canonPaths canon.Paths) (*emit_golden.GoldenCase, error) {
	var inputMap map[string]any
	if err := decoder.Decode(&inputMap); err != nil {
		return nil, err
	}
	defaults, err := canon.LoadDefaults(canonPaths)
	if err != nil {
		return nil, err
	}
	mergeMap(defaults, inputMap)
	mergedJSON, err := json.Marshal(defaults)
	if err != nil {
		return nil, err
	}
	var result emit_golden.GoldenCase
	if err := json.Unmarshal(mergedJSON, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// mergeMap is a shallow/recursive merge helper used by ProcessInput.
func mergeMap(dst, src map[string]any) {
	for k, v := range src {
		if existing, ok := dst[k]; ok {
			em, eok := existing.(map[string]any)
			vm, vok := v.(map[string]any)
			if eok && vok {
				mergeMap(em, vm)
				continue
			}
		}
		dst[k] = v
	}
}
