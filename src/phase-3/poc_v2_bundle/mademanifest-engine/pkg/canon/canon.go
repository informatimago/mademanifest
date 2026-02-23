package canon

import (
    "encoding/json"
    "fmt"
    "os"
    // "path/filepath"
)

// loadJSONIntoMap reads a JSON file and decodes it into a map[string]any.
func loadJSONIntoMap(path string) (map[string]any, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var m map[string]any
    if err := json.Unmarshal(data, &m); err != nil {
        return nil, fmt.Errorf("%w: %s", err, path)
    }
    return m, nil
}

// LoadMandalaConstants loads canon/mandala_constants.json and returns the
// mapping that should be merged under human_design_mapping in the engine input.
func LoadMandalaConstants(path string) (map[string]any, error) {
    m, err := loadJSONIntoMap(path)
    if err != nil {
        return nil, err
    }
    mapped := map[string]any{
        "mandala_start_deg": m["start_longitude_deg"],
        "gate_width_deg":    m["gate_width_deg"],
        "line_width_deg":    m["line_width_deg"],
        "interval_rule":     "start_inclusive_end_exclusive",
    }
    return map[string]any{"engine_contract": map[string]any{"human_design_mapping": mapped}}, nil
}

// LoadNodePolicy loads canon/node_policy.json and returns the mapping that
// should be merged under engine_contract.node_policy_by_system.
func LoadNodePolicy(path string) (map[string]any, error) {
    m, err := loadJSONIntoMap(path)
    if err != nil {
        return nil, err
    }
    mapped := map[string]any{
        "astrology":      m["astrology_nodes"],
        "human_design":   m["human_design_nodes"],
        "gene_keys":      "true",
    }
    return map[string]any{"engine_contract": map[string]any{"node_policy_by_system": mapped}}, nil
}

// LoadDefaults loads all canon defaults and returns a map ready for merging.
func LoadDefaults() (map[string]any, error) {
    // Use relative paths from the repository root
    const (
        mandalaPath = "canon/mandala_constants.json"
        nodePath    = "canon/node_policy.json"
    )
    mandala, err := LoadMandalaConstants(mandalaPath)
    if err != nil {
        return nil, err
    }
    node, err := LoadNodePolicy(nodePath)
    if err != nil {
        return nil, err
    }
    defaults := make(map[string]any)
    mergeMap(defaults, mandala)
    mergeMap(defaults, node)
    return defaults, nil
}

// mergeMap merges src into dst.  dst is modified.  It performs a shallow
// merge for non-map values and a recursive merge for map values.
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
