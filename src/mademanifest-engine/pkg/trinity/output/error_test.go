package output

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"mademanifest-engine/pkg/canon"
)

// TestMetadataMatchesCanonVersions confirms that CurrentMetadata()
// reads the five-field subset from pkg/canon.  Any drift between the
// two packages is caught here before it reaches a response envelope.
func TestMetadataMatchesCanonVersions(t *testing.T) {
	m := CurrentMetadata()
	if m.EngineVersion != canon.EngineVersion {
		t.Errorf("EngineVersion = %q, want %q", m.EngineVersion, canon.EngineVersion)
	}
	if m.CanonVersion != canon.CanonVersion {
		t.Errorf("CanonVersion = %q, want %q", m.CanonVersion, canon.CanonVersion)
	}
	if m.SourceStackVersion != canon.SourceStackVersion {
		t.Errorf("SourceStackVersion = %q, want %q", m.SourceStackVersion, canon.SourceStackVersion)
	}
	if m.InputSchemaVersion != canon.InputSchemaVersion {
		t.Errorf("InputSchemaVersion = %q, want %q", m.InputSchemaVersion, canon.InputSchemaVersion)
	}
	if m.MappingVersion != canon.MappingVersion {
		t.Errorf("MappingVersion = %q, want %q", m.MappingVersion, canon.MappingVersion)
	}
}

// TestMetadataKeysAreTheCanonicalFive locks the JSON key set so a
// future struct change that adds swisseph_version or tzdb_version is
// rejected at test time (canon permits only 5 metadata fields).
func TestMetadataKeysAreTheCanonicalFive(t *testing.T) {
	raw, err := json.Marshal(CurrentMetadata())
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	var decoded map[string]string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	want := []string{
		"engine_version", "canon_version", "source_stack_version",
		"input_schema_version", "mapping_version",
	}
	if got, w := len(decoded), len(want); got != w {
		t.Errorf("metadata key count = %d, want %d; got %v", got, w, decoded)
	}
	for _, k := range want {
		if _, ok := decoded[k]; !ok {
			t.Errorf("metadata missing key %q", k)
		}
	}
}

// TestErrorEnvelopeShape pins the JSON shape of a fresh error
// envelope: status, metadata, error keys at the top level; error
// holds error_type + message.
func TestErrorEnvelopeShape(t *testing.T) {
	env := NewError(ErrorInvalidInput, "bad lat")
	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal error envelope: %v", err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, k := range []string{"status", "metadata", "error"} {
		if _, ok := decoded[k]; !ok {
			t.Errorf("top-level missing key %q; body=%s", k, raw)
		}
	}
	var inner map[string]string
	if err := json.Unmarshal(decoded["error"], &inner); err != nil {
		t.Fatalf("unmarshal error sub-object: %v", err)
	}
	if inner["error_type"] != ErrorInvalidInput {
		t.Errorf("error_type = %q, want %q", inner["error_type"], ErrorInvalidInput)
	}
	if inner["message"] != "bad lat" {
		t.Errorf("message = %q, want bad lat", inner["message"])
	}
	// Status string must be exactly the canon literal.
	if !strings.Contains(string(decoded["status"]), `"error"`) {
		t.Errorf("status field not canonical: %s", decoded["status"])
	}
}

// TestStatusCodeForErrorTypeMatchesPhase3Policy pins the mapping
// documented in trinity-implementation-plan.org Phase 3.  This is a
// regression barrier – future refactors must update both the policy
// doc and this test in lock-step.
func TestStatusCodeForErrorTypeMatchesPhase3Policy(t *testing.T) {
	cases := []struct {
		errType string
		status  int
	}{
		{ErrorInvalidInput, http.StatusBadRequest},
		{ErrorIncompleteInput, http.StatusBadRequest},
		{ErrorUnsupportedInput, http.StatusUnprocessableEntity},
		{ErrorCanonConflict, http.StatusInternalServerError},
		{ErrorExecutionFailure, http.StatusInternalServerError},
		{"not_a_canonical_type", http.StatusInternalServerError}, // safe default
	}
	for _, tc := range cases {
		t.Run(tc.errType, func(t *testing.T) {
			if got := StatusCodeForErrorType(tc.errType); got != tc.status {
				t.Errorf("StatusCodeForErrorType(%q) = %d, want %d",
					tc.errType, got, tc.status)
			}
		})
	}
}

// TestCanonicalErrorTypeConstants guards against accidental drift
// of the error_type literal strings.  These strings appear verbatim
// in fixtures and in API consumers; any rename is a contract break.
func TestCanonicalErrorTypeConstants(t *testing.T) {
	want := map[string]string{
		"invalid_input":     ErrorInvalidInput,
		"incomplete_input":  ErrorIncompleteInput,
		"unsupported_input": ErrorUnsupportedInput,
		"canon_conflict":    ErrorCanonConflict,
		"execution_failure": ErrorExecutionFailure,
	}
	for literal, constVal := range want {
		if literal != constVal {
			t.Errorf("constant for %q has drifted to %q", literal, constVal)
		}
	}
}
