package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"mademanifest-engine/pkg/trinity/output"
)

// TrinityCase is a single end-to-end probe against the live engine.
// Body is sent verbatim to POST /manifest; WantStatus and
// WantErrorType are the canonical status code / error_type expected
// in the response.
type TrinityCase struct {
	Name          string
	Body          []byte
	WantStatus    int
	WantErrorType string // empty means a success envelope is expected
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
			Name:          "valid baseline -> execution_failure (placeholder, Phase 3+)",
			Body:          []byte(baseline),
			WantStatus:    http.StatusInternalServerError,
			WantErrorType: output.ErrorExecutionFailure,
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
	var env output.ErrorEnvelope
	if decErr := json.Unmarshal(raw, &env); decErr != nil {
		t.Fatalf("decode envelope: %v\nbody: %s", decErr, raw)
	}
	if env.Status != output.StatusError {
		t.Errorf("envelope status = %q, want %q\nbody: %s",
			env.Status, output.StatusError, raw)
	}
	if env.Error.Type != c.WantErrorType {
		t.Errorf("error_type = %q, want %q\nbody: %s",
			env.Error.Type, c.WantErrorType, raw)
	}
	if env.Error.Message == "" {
		t.Errorf("envelope error.message must not be empty\nbody: %s", raw)
	}
	if env.Metadata != output.CurrentMetadata() {
		t.Errorf("envelope metadata = %+v\nwant %+v\nbody: %s",
			env.Metadata, output.CurrentMetadata(), raw)
	}
}
