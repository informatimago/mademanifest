package canon

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode"
)

// TestVersionConstantsNonEmpty enforces that every pinned version
// string has content.  A blank pin would silently produce
// unreproducible metadata in Trinity responses.
func TestVersionConstantsNonEmpty(t *testing.T) {
	cases := map[string]string{
		"CanonVersion":       CanonVersion,
		"MappingVersion":     MappingVersion,
		"InputSchemaVersion": InputSchemaVersion,
		"SourceStackVersion": SourceStackVersion,
		"EngineVersion":      EngineVersion,
		"SwissEphVersion":    SwissEphVersion,
		"TZDBVersion":        TZDBVersion,
	}
	for name, v := range cases {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			t.Errorf("%s is empty", name)
		}
		if trimmed != v {
			t.Errorf("%s contains surrounding whitespace: %q", name, v)
		}
		for _, r := range v {
			if unicode.IsSpace(r) {
				t.Errorf("%s = %q contains internal whitespace", name, v)
				break
			}
		}
	}
}

// TestSwissEphVersionIsCanon locks the pin to trinity.org line 61.
// If the canon changes the Swiss Ephemeris version, this constant
// must move in lock-step.
func TestSwissEphVersionIsCanon(t *testing.T) {
	if SwissEphVersion != "2.10.03" {
		t.Errorf("SwissEphVersion = %q, want 2.10.03 (trinity.org line 61)", SwissEphVersion)
	}
}

// TestVersionsReturnsCompiledInValues guards the mapping from
// constants to the JSON-shaped struct.  Anyone who adds a new pinned
// constant must extend Versions() at the same time – this test
// fails if they don't.
func TestVersionsReturnsCompiledInValues(t *testing.T) {
	v := Versions()
	if v.EngineVersion != EngineVersion {
		t.Errorf("EngineVersion mismatch: %q vs %q", v.EngineVersion, EngineVersion)
	}
	if v.CanonVersion != CanonVersion {
		t.Errorf("CanonVersion mismatch: %q vs %q", v.CanonVersion, CanonVersion)
	}
	if v.MappingVersion != MappingVersion {
		t.Errorf("MappingVersion mismatch: %q vs %q", v.MappingVersion, MappingVersion)
	}
	if v.InputSchemaVersion != InputSchemaVersion {
		t.Errorf("InputSchemaVersion mismatch: %q vs %q", v.InputSchemaVersion, InputSchemaVersion)
	}
	if v.SourceStackVersion != SourceStackVersion {
		t.Errorf("SourceStackVersion mismatch: %q vs %q", v.SourceStackVersion, SourceStackVersion)
	}
	if v.SwissEphVersion != SwissEphVersion {
		t.Errorf("SwissEphVersion mismatch: %q vs %q", v.SwissEphVersion, SwissEphVersion)
	}
	if v.TZDBVersion != TZDBVersion {
		t.Errorf("TZDBVersion mismatch: %q vs %q", v.TZDBVersion, TZDBVersion)
	}
}

// TestVersionsJSONKeysAreCanonical pins the JSON shape so any rename
// is caught before it reaches the response envelope.  Trinity
// Document 07 lists these keys explicitly (engine_version,
// canon_version, source_stack_version, input_schema_version,
// mapping_version).  SwissEph and TZDB are our additions for
// /version; they use snake_case for consistency.
func TestVersionsJSONKeysAreCanonical(t *testing.T) {
	raw, err := json.Marshal(Versions())
	if err != nil {
		t.Fatalf("marshal Versions(): %v", err)
	}
	var decoded map[string]string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal Versions(): %v", err)
	}
	wantKeys := []string{
		"engine_version",
		"canon_version",
		"mapping_version",
		"input_schema_version",
		"source_stack_version",
		"swisseph_version",
		"tzdb_version",
	}
	if got, want := len(decoded), len(wantKeys); got != want {
		t.Errorf("Versions JSON key count = %d, want %d; got %v", got, want, decoded)
	}
	for _, k := range wantKeys {
		if _, ok := decoded[k]; !ok {
			t.Errorf("Versions JSON missing key %q", k)
		}
	}
}
