package canon

// sources.go holds the pinned version strings that identify the
// canon revision, the calculation source stack, and the engine
// implementation.  Each of these string values propagates into the
// deterministic Trinity response metadata so reproducibility can be
// verified across environments and time.
//
// Canon ambiguities A1..A7 are now RESOLVED (Document 12 decisions
// D20..D26, folded back into Documents 03/04/05/07/08/09/10).  The
// pins below reflect those rulings; comments keep the A-tag for
// traceability with prior commits.

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
	// A5 (invalid vs unsupported boundary) and A6 (IANA canonical
	// names only) are now folded back into Document 04 / D24.
	InputSchemaVersion = "trinity-v1-rev-0"

	// SourceStackVersion is the combined revision of the
	// authoritative external sources (Swiss Ephemeris + IANA tzdb).
	// A1 (tzdb release pin) is now RESOLVED – Document 03 pins
	// IANA tzdb 2026a; see TZDBVersion below.
	SourceStackVersion = "trinity-v1-rev-0"

	// EngineVersion is this build's own implementation revision.
	// Bumps on any production code change, not just canon revisions.
	EngineVersion = "v1.0.0-trinity"
)

const (
	// SwissEphVersion is the pinned Swiss Ephemeris release.  Canon:
	// Document 03 / D14 ("Swiss Ephemeris version pin: 2.10.03").
	// Cross-checked at runtime by pkg/ephemeris at first ephemeris
	// call; a mismatch there aborts the process.
	SwissEphVersion = "2.10.03"

	// TZDBVersion is the canon-pinned IANA Time Zone Database
	// release for Trinity Engine v1.
	//
	// A1 (RESOLVED, Document 03 / D20): the authoritative IANA tzdb
	// release is 2026a.  Per Document 12's ruling on A1, the runtime
	// must not rely on whatever tzdata the Go runtime or host OS
	// happens to ship unless that release is confirmed to match.
	//
	// The production Docker image vendors IANA 2026a explicitly:
	// src/tzdata/ holds the IANA source tarball and pinned SHA-512;
	// the Dockerfile builder stage compiles it with `zic` into
	// /usr/local/share/zoneinfo, drops the upstream `version` file as
	// `+VERSION`, and the runtime stage exports
	// ZONEINFO=/usr/local/share/zoneinfo.  At boot,
	// AssertTZDBVersion (tzdata.go) reads `<ZONEINFO>/+VERSION` and
	// aborts the process if it does not equal TZDBVersion.
	//
	// Local non-Docker `go test` runs leave ZONEINFO unset and fall
	// back to the host's system zoneinfo (Go consults
	// /usr/share/zoneinfo, etc.).  The production binary no longer
	// embeds `time/tzdata` (see pkg/astronomy/time.go for the
	// rationale), so an unset ZONEINFO at production runtime would
	// fail LoadLocation cleanly rather than silently resolving
	// against an unverified release.  AssertTZDBVersion is a no-op
	// in the unset-ZONEINFO case so dev / CI still boots.
	// Reproducing canonical results outside the container requires
	// `make -C src/tzdata zoneinfo` and
	// `ZONEINFO=$PWD/src/tzdata/zoneinfo` in the test command.
	TZDBVersion = "2026a"
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
