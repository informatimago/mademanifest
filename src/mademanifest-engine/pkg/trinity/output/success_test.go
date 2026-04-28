package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"mademanifest-engine/pkg/trinity/input"
)

// canonicalPayload is the validated Trinity payload that drives the
// placeholder constructor in every Phase 3 test.
var canonicalPayload = input.Payload{
	BirthDate: "1990-04-09",
	BirthTime: "18:04",
	Timezone:  "Europe/Amsterdam",
	Latitude:  51.9167,
	Longitude: 4.4,
}

// TestSuccessEnvelopeTopLevelKeyOrder pins the key order at the
// envelope root.  encoding/json marshals struct fields in
// declaration order, so this test catches any future re-ordering
// of SuccessEnvelope's fields that would break wire compatibility.
func TestSuccessEnvelopeTopLevelKeyOrder(t *testing.T) {
	env := NewPlaceholderSuccess(canonicalPayload)
	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	wantOrder := []string{
		`"status"`,
		`"metadata"`,
		`"input_echo"`,
		`"astrology"`,
		`"human_design"`,
		`"gene_keys"`,
	}
	assertKeyOrder(t, raw, wantOrder)
}

// TestSuccessEnvelopeNestedKeyOrder pins the canonical sub-section
// orderings inside metadata, input_echo, astrology.system, and
// gene_keys.activations.
func TestSuccessEnvelopeNestedKeyOrder(t *testing.T) {
	env := NewPlaceholderSuccess(canonicalPayload)
	env.HumanDesign.System.DesignTimeUTC = DesignTime(
		time.Date(1990, 1, 1, 12, 0, 0, 0, time.UTC))
	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	cases := []struct {
		name  string
		order []string
	}{
		{"metadata", []string{
			`"engine_version"`, `"canon_version"`, `"source_stack_version"`,
			`"input_schema_version"`, `"mapping_version"`,
		}},
		{"input_echo", []string{
			`"birth_date"`, `"birth_time"`, `"timezone"`,
			`"latitude"`, `"longitude"`,
		}},
		{"astrology.system", []string{
			`"zodiac"`, `"house_system"`, `"node_type"`,
		}},
		{"human_design", []string{
			`"system"`, `"personality_activations"`, `"design_activations"`,
			`"channels"`, `"centers"`, `"definition"`, `"type"`,
			`"authority"`, `"profile"`, `"incarnation_cross"`,
		}},
		{"gene_keys.activations", []string{
			`"life_work"`, `"evolution"`, `"radiance"`, `"purpose"`,
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertKeyOrder(t, raw, tc.order)
		})
	}
}

// TestPlaceholderSuccessIsCanonShaped pins the contents of the
// placeholder envelope: status string, metadata block, input echo,
// system constants, centers list.  Phase 4-8 will mutate the
// remaining (currently empty) sub-fields; this test only protects
// the structural fields that Phase 3 owns.
func TestPlaceholderSuccessIsCanonShaped(t *testing.T) {
	env := NewPlaceholderSuccess(canonicalPayload)

	if env.Status != "success" {
		t.Errorf("status = %q, want success", env.Status)
	}
	if env.Metadata != CurrentMetadata() {
		t.Errorf("metadata = %+v, want %+v", env.Metadata, CurrentMetadata())
	}
	if env.InputEcho.BirthDate != "1990-04-09" ||
		env.InputEcho.BirthTime != "18:04" ||
		env.InputEcho.Timezone != "Europe/Amsterdam" {
		t.Errorf("input_echo strings drifted: %+v", env.InputEcho)
	}
	if float64(env.InputEcho.Latitude) != 51.9167 {
		t.Errorf("input_echo.latitude = %v, want 51.9167", env.InputEcho.Latitude)
	}
	if float64(env.InputEcho.Longitude) != 4.4 {
		t.Errorf("input_echo.longitude = %v, want 4.4", env.InputEcho.Longitude)
	}
	if env.Astrology.System.Zodiac != "tropical" ||
		env.Astrology.System.HouseSystem != "placidus" ||
		env.Astrology.System.NodeType != "mean" {
		t.Errorf("astrology.system not canon: %+v", env.Astrology.System)
	}
	if env.HumanDesign.System.NodeType != "true" {
		t.Errorf("human_design.system.node_type = %q, want true",
			env.HumanDesign.System.NodeType)
	}
	if got, want := len(env.HumanDesign.Centers), 9; got != want {
		t.Errorf("centers length = %d, want %d", got, want)
	}
	for i, c := range env.HumanDesign.Centers {
		if c.State != "undefined" {
			t.Errorf("center[%d].state = %q, want undefined", i, c.State)
		}
	}
	if env.GeneKeys.System.DerivationBasis != "human_design" {
		t.Errorf("gene_keys.system.derivation_basis = %q, want human_design",
			env.GeneKeys.System.DerivationBasis)
	}
}

// TestSuccessEnvelopeRoundTrip marshals and unmarshals the
// placeholder envelope and asserts deep equality.  This catches
// MarshalJSON / UnmarshalJSON drift on the typed scalar wrappers
// (Longitude, DesignTime).
func TestSuccessEnvelopeRoundTrip(t *testing.T) {
	original := NewPlaceholderSuccess(canonicalPayload)
	original.HumanDesign.System.DesignTimeUTC = DesignTime(
		time.Date(1990, 1, 11, 6, 4, 0, 0, time.UTC))

	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var roundTripped SuccessEnvelope
	if err := json.Unmarshal(raw, &roundTripped); err != nil {
		t.Fatalf("unmarshal: %v\nbody: %s", err, raw)
	}
	// DesignTime equality is by underlying time.Time, but
	// reflect.DeepEqual handles named time types correctly.
	reEnc, err := json.Marshal(roundTripped)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	if !bytes.Equal(raw, reEnc) {
		t.Errorf("round-trip not byte-equal:\n original: %s\n re-encoded: %s",
			raw, reEnc)
	}
}

// TestLongitudeMarshalSixDecimalPlaces locks in the formatting
// rule from trinity.org line 581.  Multiple representative inputs
// guard against drift in strconv.FormatFloat behaviour or accidental
// scientific-notation output.
func TestLongitudeMarshalSixDecimalPlaces(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0.0, "0.000000"},
		{4.4, "4.400000"},
		{51.9167, "51.916700"},
		{-122.4194, "-122.419400"},
		{179.999999, "179.999999"},
		{180.0, "180.000000"},
		{0.000001, "0.000001"},
		{1.234567891234, "1.234568"}, // banker's-style rounding by FormatFloat
	}
	for _, tc := range cases {
		got, err := json.Marshal(Longitude(tc.in))
		if err != nil {
			t.Errorf("marshal %v: %v", tc.in, err)
			continue
		}
		if string(got) != tc.want {
			t.Errorf("Longitude(%v) -> %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestDesignTimeMarshalRFC3339WholeSecondFloor pins the A3 / D22
// emission rule: any sub-second component is dropped (truncation),
// the timezone is forced to UTC, and the suffix is "Z".
func TestDesignTimeMarshalRFC3339WholeSecondFloor(t *testing.T) {
	cases := []struct {
		name string
		in   time.Time
		want string
	}{
		{
			name: "zero value",
			in:   time.Time{},
			want: `"0001-01-01T00:00:00Z"`,
		},
		{
			name: "exact second UTC",
			in:   time.Date(1990, 4, 9, 18, 4, 0, 0, time.UTC),
			want: `"1990-04-09T18:04:00Z"`,
		},
		{
			name: "sub-second floor",
			// 999_999_999 ns = 0.999999999 s; floor drops it.
			in:   time.Date(1990, 4, 9, 18, 4, 0, 999999999, time.UTC),
			want: `"1990-04-09T18:04:00Z"`,
		},
		{
			name: "non-UTC zone is converted to UTC",
			// 09:30+05:30 = 04:00 UTC.
			in: time.Date(1990, 4, 9, 9, 30, 0, 0,
				time.FixedZone("IST", 5*3600+30*60)),
			want: `"1990-04-09T04:00:00Z"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(DesignTime(tc.in))
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("DesignTime -> %s, want %s", got, tc.want)
			}
		})
	}
}

// TestInputEchoExcludesNonCanonicalFields verifies that a payload
// with ad-hoc extra fields (which the validator would reject – but
// here we test the type-level guarantee) cannot leak through into
// input_echo.  The InputEcho struct has exactly the canon's five
// fields; the type system enforces it.
func TestInputEchoExcludesNonCanonicalFields(t *testing.T) {
	echo := InputEcho{
		BirthDate: "1990-04-09",
		BirthTime: "18:04",
		Timezone:  "Europe/Amsterdam",
		Latitude:  51.9167,
		Longitude: 4.4,
	}
	raw, err := json.Marshal(echo)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, k := range []string{
		"birth_date", "birth_time", "timezone", "latitude", "longitude",
	} {
		if !strings.Contains(string(raw), `"`+k+`"`) {
			t.Errorf("input_echo missing key %q; body: %s", k, raw)
		}
	}
	// Negative assertions: PoC-era fields must not appear.
	for _, k := range []string{
		"case_id", "place_name", "engine_contract", "expected", "seconds_policy",
	} {
		if strings.Contains(string(raw), `"`+k+`"`) {
			t.Errorf("input_echo leaks PoC-era field %q; body: %s", k, raw)
		}
	}
}

// assertKeyOrder is a small helper that walks a JSON byte slice and
// asserts that the given key tokens appear in the given order.  Any
// missing key, or a key seen out of order, fails the test.
func assertKeyOrder(t *testing.T, raw []byte, keys []string) {
	t.Helper()
	last := 0
	for _, k := range keys {
		idx := bytes.Index(raw[last:], []byte(k))
		if idx < 0 {
			t.Errorf("key %q not found at or after offset %d; body: %s",
				k, last, raw)
			continue
		}
		last += idx + len(k)
	}
}
