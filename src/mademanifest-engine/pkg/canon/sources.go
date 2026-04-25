package canon

// sources.go holds the pinned version strings that identify the
// canon revision, the calculation source stack, and the engine
// implementation.  Each of these string values propagates into the
// deterministic Trinity response metadata so reproducibility can be
// verified across environments and time.
//
// Canon ambiguities (A1..A8) that affect pins here are noted inline;
// the companion src/doc/version-pins.org catalogs the full set and
// tracks their resolution state.

const (
	// CanonVersion is the revision of the Trinity canon implemented
	// by this build.  Bumps on any change to scope, calculation,
	// mapping, output, precedence, formatting, or input behaviour
	// (per Document 08 § versioning).
	CanonVersion = "trinity-v1-rev-0"

	// MappingVersion is the revision of the mapping canon (gate
	// order, channel table, center list, channel-to-center map,
	// identifier schemes).  Bumps separately from CanonVersion so
	// pure-mapping changes do not invalidate broader regressions.
	MappingVersion = "trinity-v1-rev-0"

	// InputSchemaVersion is the revision of the canonical input
	// contract: field names, types, formats, ranges, validation.
	// Tracks ambiguities A5 (unsupported_input boundary) and A6
	// (IANA link names) that still gate final acceptance.
	InputSchemaVersion = "trinity-v1-rev-0"

	// SourceStackVersion is the combined revision of the
	// authoritative external sources (Swiss Ephemeris + IANA tzdb).
	// Tracks ambiguity A1 (tzdb release not yet pinned in canon).
	SourceStackVersion = "trinity-v1-rev-0"

	// EngineVersion is this build's own implementation revision.
	// Bumps on any production code change, not just canon revisions.
	EngineVersion = "v0.1.0-phase-10"
)

const (
	// SwissEphVersion is the pinned Swiss Ephemeris release.  Canon:
	// trinity.org line 61 ("Swiss Ephemeris version pin: 2.10.03").
	// Cross-checked at runtime by pkg/ephemeris at first ephemeris
	// call; a mismatch there aborts the process.
	SwissEphVersion = "2.10.03"

	// TZDBVersion is the working-assumption IANA tzdb release.
	//
	// A1 (UNRESOLVED) – trinity.org pins the IANA Time Zone Database
	// as authoritative but does not name an exact tzdb release.  We
	// inherit whatever tzdata the Go toolchain embeds via
	// time/tzdata; Go 1.22.x ships tzdata 2023c.  The canon owner
	// must confirm or override this choice before final acceptance.
	TZDBVersion = "2023c"
)

// VersionInfo is the JSON shape returned by GET /version.  It is
// also embedded into the Trinity success / error response metadata
// in later phases.  Keys are pinned (JSON tag order is preserved by
// the serializer used in httpservice.handleVersion).
type VersionInfo struct {
	EngineVersion      string `json:"engine_version"`
	CanonVersion       string `json:"canon_version"`
	MappingVersion     string `json:"mapping_version"`
	InputSchemaVersion string `json:"input_schema_version"`
	SourceStackVersion string `json:"source_stack_version"`
	SwissEphVersion    string `json:"swisseph_version"`
	TZDBVersion        string `json:"tzdb_version"`
}

// Versions returns the compiled-in pinned versions as a VersionInfo.
// Callers that need JSON-stable output should marshal the result of
// this function rather than hand-building a map.
func Versions() VersionInfo {
	return VersionInfo{
		EngineVersion:      EngineVersion,
		CanonVersion:       CanonVersion,
		MappingVersion:     MappingVersion,
		InputSchemaVersion: InputSchemaVersion,
		SourceStackVersion: SourceStackVersion,
		SwissEphVersion:    SwissEphVersion,
		TZDBVersion:        TZDBVersion,
	}
}
