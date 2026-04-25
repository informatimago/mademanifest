// Package canon holds the canonical Trinity v1 constants and the
// boot-time self-check helpers.  Phase 12 retired the legacy JSON
// loaders (LoadDefaults / LoadMandalaConstants / LoadNodePolicy /
// LoadGateSequenceV1) that once fed the PoC engine; the trinity
// runtime path consumes only the compiled-in values in
// constants.go and sources.go, validated at boot by SelfCheck()
// in selfcheck.go.
//
// The Phase 9 sanity-check function AssertGateSequenceFileMatchesGateOrder
// (selfcheck.go) is retained as an operator tool: deployment
// validators / CI hooks can opt in to verify a JSON gate-sequence
// fixture against the compiled canon, but no part of the engine's
// /manifest path depends on it.
package canon
