package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"mademanifest-engine/pkg/trinity/output"
)

// TrinityCase is a single end-to-end probe against the live engine.
// Body is sent verbatim to POST /manifest.  Exactly one of
// WantSuccess and WantErrorType is set: WantSuccess=true asserts a
// SuccessEnvelope shape; WantErrorType names the expected
// error_type for an ErrorEnvelope.  WantStatus pins the HTTP
// response status code in either case.
type TrinityCase struct {
	Name          string
	Body          []byte
	WantStatus    int
	WantSuccess   bool
	WantErrorType string
}

// TrinityRejectionCases is the canonical end-to-end probe matrix
// shared by the local, docker, and kubernetes harnesses.  It
// exercises the Phase 2 input contract: every rejection category
// from trinity.org §"Validation Rules" plus a representative
// success-path probe (which still returns execution_failure during
// Phase 2 because the calculation pipeline is not yet wired).
//
// The list is intentionally a subset of the validator unit tests –
// the goal of the integration matrix is to confirm the shape of the
// HTTP envelope and the status-code policy through the real network
// surface, not to re-prove the validator's logic.
func TrinityRejectionCases() []TrinityCase {
	const baseline = `{
  "birth_date": "1990-04-09",
  "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167,
  "longitude": 4.4
}`
	return []TrinityCase{
		{
			Name:        "valid baseline -> 200 success envelope",
			Body:        []byte(baseline),
			WantStatus:  http.StatusOK,
			WantSuccess: true,
		},
		{
			Name: "missing birth_date -> incomplete_input",
			Body: []byte(`{
  "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": 4.4
}`),
			WantStatus:    http.StatusBadRequest,
			WantErrorType: output.ErrorIncompleteInput,
		},
		{
			Name: "latitude as string -> invalid_input",
			Body: []byte(`{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": "51.9167", "longitude": 4.4
}`),
			WantStatus:    http.StatusBadRequest,
			WantErrorType: output.ErrorInvalidInput,
		},
		{
			Name: "non-existent date 1990-02-30 -> invalid_input",
			Body: []byte(`{
  "birth_date": "1990-02-30", "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": 4.4
}`),
			WantStatus:    http.StatusBadRequest,
			WantErrorType: output.ErrorInvalidInput,
		},
		{
			Name: "seconds-present birth_time -> unsupported_input (A5)",
			Body: []byte(`{
  "birth_date": "1990-04-09", "birth_time": "18:04:00",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": 4.4
}`),
			WantStatus:    http.StatusUnprocessableEntity,
			WantErrorType: output.ErrorUnsupportedInput,
		},
		{
			Name: "timezone abbreviation CET -> invalid_input",
			Body: []byte(`{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "CET",
  "latitude": 51.9167, "longitude": 4.4
}`),
			WantStatus:    http.StatusBadRequest,
			WantErrorType: output.ErrorInvalidInput,
		},
		{
			Name: "timezone link name US/Eastern -> invalid_input (A6)",
			Body: []byte(`{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "US/Eastern",
  "latitude": 51.9167, "longitude": 4.4
}`),
			WantStatus:    http.StatusBadRequest,
			WantErrorType: output.ErrorInvalidInput,
		},
		{
			Name: "latitude > 90 -> invalid_input",
			Body: []byte(`{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 91.0, "longitude": 4.4
}`),
			WantStatus:    http.StatusBadRequest,
			WantErrorType: output.ErrorInvalidInput,
		},
		{
			Name: "unknown field place_name -> invalid_input",
			Body: []byte(`{
  "birth_date": "1990-04-09", "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167, "longitude": 4.4,
  "place_name": "Schiedam"
}`),
			WantStatus:    http.StatusBadRequest,
			WantErrorType: output.ErrorInvalidInput,
		},
	}
}

// AssertTrinityRejectionMatrix runs every case in
// TrinityRejectionCases against baseURL and checks that the response
// status matches WantStatus and the response body decodes as a
// Trinity error envelope with the matching error_type plus a
// non-empty message and the canonical metadata block.
//
// Per A4 (canonical error messages not yet pinned), only error_type
// is asserted – Message text is checked for non-emptiness only.
//
// The signature takes *testing.T (rather than testing.TB) because
// each case is wrapped in t.Run for sub-test isolation, which is
// only available on *testing.T.
func AssertTrinityRejectionMatrix(t *testing.T, baseURL string) {
	t.Helper()
	for _, c := range TrinityRejectionCases() {
		t.Run(c.Name, func(t *testing.T) {
			runTrinityCase(t, baseURL, c)
		})
	}
}

func runTrinityCase(t *testing.T, baseURL string, c TrinityCase) {
	t.Helper()
	status, raw, err := PostManifest(baseURL, c.Body, nil)
	if err != nil {
		t.Fatalf("POST /manifest: %v", err)
	}
	if status != c.WantStatus {
		t.Fatalf("status = %d, want %d (body: %s)", status, c.WantStatus, raw)
	}
	if c.WantSuccess {
		assertSuccessEnvelope(t, raw)
		return
	}
	assertErrorEnvelope(t, raw, c.WantErrorType)
}

func assertErrorEnvelope(t *testing.T, raw []byte, wantType string) {
	t.Helper()
	var env output.ErrorEnvelope
	if decErr := json.Unmarshal(raw, &env); decErr != nil {
		t.Fatalf("decode envelope: %v\nbody: %s", decErr, raw)
	}
	if env.Status != output.StatusError {
		t.Errorf("envelope status = %q, want %q\nbody: %s",
			env.Status, output.StatusError, raw)
	}
	if env.Error.Type != wantType {
		t.Errorf("error_type = %q, want %q\nbody: %s",
			env.Error.Type, wantType, raw)
	}
	if env.Error.Message == "" {
		t.Errorf("envelope error.message must not be empty\nbody: %s", raw)
	}
	if env.Metadata != output.CurrentMetadata() {
		t.Errorf("envelope metadata = %+v\nwant %+v\nbody: %s",
			env.Metadata, output.CurrentMetadata(), raw)
	}
}

// assertSuccessEnvelope validates the Phase 3 SuccessEnvelope
// *shape* end-to-end: top-level keys present, status string,
// metadata block, input_echo round-tripped, and the system
// constants pinned.  Phase 4-8 will add content assertions for
// the calculation sub-fields; this helper deliberately stays
// shape-only so it remains the canonical invariant under those
// later phase changes.
func assertSuccessEnvelope(t *testing.T, raw []byte) {
	t.Helper()
	var env output.SuccessEnvelope
	if decErr := json.Unmarshal(raw, &env); decErr != nil {
		t.Fatalf("decode success envelope: %v\nbody: %s", decErr, raw)
	}
	if env.Status != output.StatusSuccess {
		t.Errorf("envelope status = %q, want %q\nbody: %s",
			env.Status, output.StatusSuccess, raw)
	}
	if env.Metadata != output.CurrentMetadata() {
		t.Errorf("envelope metadata = %+v\nwant %+v\nbody: %s",
			env.Metadata, output.CurrentMetadata(), raw)
	}
	if env.InputEcho.BirthDate == "" || env.InputEcho.BirthTime == "" ||
		env.InputEcho.Timezone == "" {
		t.Errorf("input_echo string fields empty: %+v\nbody: %s",
			env.InputEcho, raw)
	}
	if env.Astrology.System.Zodiac != "tropical" ||
		env.Astrology.System.HouseSystem != "placidus" ||
		env.Astrology.System.NodeType != "mean" {
		t.Errorf("astrology.system not canon: %+v\nbody: %s",
			env.Astrology.System, raw)
	}
	if env.HumanDesign.System.NodeType != "true" {
		t.Errorf("human_design.system.node_type = %q, want true\nbody: %s",
			env.HumanDesign.System.NodeType, raw)
	}
	if env.GeneKeys.System.DerivationBasis != "human_design" {
		t.Errorf("gene_keys.system.derivation_basis = %q, want human_design\nbody: %s",
			env.GeneKeys.System.DerivationBasis, raw)
	}
}
