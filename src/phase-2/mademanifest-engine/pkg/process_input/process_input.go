package process_input

import (
	"encoding/json"
	"mademanifest-engine/pkg/emit_golden"
)


// ProcessInput decodes the input JSON into the InputData structure
func ProcessInput(decoder *json.Decoder) (*emit_golden.GoldenCase, error) {
	var input emit_golden.GoldenCase
	err := decoder.Decode(&input)
	if err != nil {
		return nil, err
	}
	return &input, nil
}
