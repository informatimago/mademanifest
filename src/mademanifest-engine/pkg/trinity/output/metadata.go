// Package output holds the Trinity response envelopes (success and
// error) and the metadata block they share.  Phase 2 introduces only
// the metadata and the error envelope – enough to answer rejected
// requests with the canonical shape.  Phase 3 extends the package
// with the success envelope, the key-order-stable serializer, and
// the full HTTP status-code policy.
package output

import (
	"mademanifest-engine/pkg/canon"
)

// Metadata is the five-field deterministic-reproducibility block
// defined in trinity.org §"Output Contract" lines 451-462.  It is
// not the same struct as canon.VersionInfo – the Trinity metadata
// does *not* carry swisseph_version or tzdb_version (those live in
// canon.VersionInfo for diagnostic purposes and are returned by
// GET /version, not by the engine's response metadata).
type Metadata struct {
	EngineVersion      string `json:"engine_version"`
	CanonVersion       string `json:"canon_version"`
	SourceStackVersion string `json:"source_stack_version"`
	InputSchemaVersion string `json:"input_schema_version"`
	MappingVersion     string `json:"mapping_version"`
}

// CurrentMetadata returns the compiled-in metadata block.  Every
// response envelope must include exactly this block; tests in later
// phases will pin the JSON key order against the trinity canon.
func CurrentMetadata() Metadata {
	return Metadata{
		EngineVersion:      canon.EngineVersion,
		CanonVersion:       canon.CanonVersion,
		SourceStackVersion: canon.SourceStackVersion,
		InputSchemaVersion: canon.InputSchemaVersion,
		MappingVersion:     canon.MappingVersion,
	}
}
