package httpservice

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mademanifest-engine/pkg/canon"
	"mademanifest-engine/pkg/trinity/output"
)

// canonicalBaseline duplicates the validator-package baseline.  Any
// drift between the two is intentional: the handler-level test does
// not depend on the input package private state.
const canonicalBaseline = `{
  "birth_date": "1990-04-09",
  "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167,
  "longitude": 4.4
}`

// TestHandleManifestRejectsWrongMethod proves /manifest answers
// non-POST requests with 405.
func TestHandleManifestRejectsWrongMethod(t *testing.T) {
	handler := New()
	req := httptest.NewRequest(http.MethodGet, "/manifest", nil)
	rec := httptest.NewRecorder()

	handler.handleManifest(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

// TestHandleManifestValidatesAndReturnsErrorEnvelope drives the
// real (default) Trinity processor with a missing-field payload and
// verifies the response is a Trinity error envelope with
// incomplete_input + 400.
func TestHandleManifestValidatesAndReturnsErrorEnvelope(t *testing.T) {
	handler := New()
	body := `{"birth_date": "1990-04-09"}`
	req := httptest.NewRequest(http.MethodPost, "/manifest", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.handleManifest(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
	var env output.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v\nbody: %s", err, rec.Body.String())
	}
	if env.Status != output.StatusError {
		t.Errorf("envelope status = %q, want %q", env.Status, output.StatusError)
	}
	if env.Error.Type != output.ErrorIncompleteInput {
		t.Errorf("error_type = %q, want %q", env.Error.Type, output.ErrorIncompleteInput)
	}
	if env.Error.Message == "" {
		t.Errorf("envelope error message must not be empty")
	}
	if env.Metadata != output.CurrentMetadata() {
		t.Errorf("envelope metadata = %+v, want %+v",
			env.Metadata, output.CurrentMetadata())
	}
}

// TestHandleManifestValidPayloadReturnsSuccessEnvelope drives the
// Phase 3 success path: a fully-valid Trinity payload now returns
// HTTP 200 with the canonical SuccessEnvelope (placeholder content
// in the calculation sub-fields, real metadata + input_echo).
//
// Phase 4-8 will replace the placeholder calculation fields with
// real values.  Those phases must update the *content* assertions
// in this test but should not need to change the *shape* assertions
// (top-level keys, status, metadata, input_echo).
func TestHandleManifestValidPayloadReturnsSuccessEnvelope(t *testing.T) {
	handler := New()
	req := httptest.NewRequest(http.MethodPost, "/manifest",
		strings.NewReader(canonicalBaseline))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.handleManifest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
	var env output.SuccessEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v\nbody: %s", err, rec.Body.String())
	}
	if env.Status != output.StatusSuccess {
		t.Errorf("envelope status = %q, want %q",
			env.Status, output.StatusSuccess)
	}
	if env.Metadata != output.CurrentMetadata() {
		t.Errorf("envelope metadata = %+v, want %+v",
			env.Metadata, output.CurrentMetadata())
	}
	if env.InputEcho.BirthDate != "1990-04-09" ||
		env.InputEcho.BirthTime != "18:04" ||
		env.InputEcho.Timezone != "Europe/Amsterdam" {
		t.Errorf("input_echo string fields drifted: %+v", env.InputEcho)
	}
	if env.Astrology.System.Zodiac != "tropical" ||
		env.Astrology.System.HouseSystem != "placidus" ||
		env.Astrology.System.NodeType != "mean" {
		t.Errorf("astrology.system not canon: %+v", env.Astrology.System)
	}
	// Phase 4: the astrology placeholder is replaced with real
	// values.  Pin the structural shape – content (specific
	// longitudes) is asserted by the integration baseline test.
	if got, want := len(env.Astrology.Objects), 13; got != want {
		t.Errorf("astrology.objects length = %d, want %d", got, want)
	}
	if got, want := len(env.Astrology.HouseCusps), 12; got != want {
		t.Errorf("astrology.house_cusps length = %d, want %d", got, want)
	}
	if env.Astrology.Angles.Ascendant.Sign == "" {
		t.Errorf("astrology.angles.ascendant.sign empty")
	}
	if env.Astrology.Angles.Midheaven.Sign == "" {
		t.Errorf("astrology.angles.midheaven.sign empty")
	}
	if env.HumanDesign.System.NodeType != "true" {
		t.Errorf("human_design.system.node_type = %q, want true",
			env.HumanDesign.System.NodeType)
	}
	if env.GeneKeys.System.DerivationBasis != "human_design" {
		t.Errorf("gene_keys.system.derivation_basis = %q, want human_design",
			env.GeneKeys.System.DerivationBasis)
	}
}

// TestHandleManifestProcessorErrorWrapsExecutionFailure swaps in a
// processor that returns an error and verifies the handler wraps
// the error as a Trinity execution_failure envelope.
func TestHandleManifestProcessorErrorWrapsExecutionFailure(t *testing.T) {
	handler := Handler{
		Process: func(_ io.Reader) ([]byte, int, error) {
			return nil, 0, errors.New("synthetic processor failure")
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/manifest",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.handleManifest(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var env output.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v\nbody: %s", err, rec.Body.String())
	}
	if env.Error.Type != output.ErrorExecutionFailure {
		t.Errorf("error_type = %q, want %q",
			env.Error.Type, output.ErrorExecutionFailure)
	}
	if !strings.Contains(env.Error.Message, "synthetic processor failure") {
		t.Errorf("envelope message should propagate processor error; got %q",
			env.Error.Message)
	}
}

// TestHandleManifestRecoversFromPanic guarantees that a panic in the
// processor is caught and rendered as a Trinity execution_failure
// envelope, not as a partial response or an HTTP 200.
func TestHandleManifestRecoversFromPanic(t *testing.T) {
	handler := Handler{
		Process: func(_ io.Reader) ([]byte, int, error) {
			panic("boom")
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/manifest",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.handleManifest(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var env output.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v\nbody: %s", err, rec.Body.String())
	}
	if env.Error.Type != output.ErrorExecutionFailure {
		t.Errorf("error_type = %q, want %q",
			env.Error.Type, output.ErrorExecutionFailure)
	}
}

// TestHandleVersionReturnsCompiledInValues – Phase 1 invariant
// preserved through the Phase 2 handler refactor.
func TestHandleVersionReturnsCompiledInValues(t *testing.T) {
	handler := New()
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()

	handler.handleVersion(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("/version status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("/version content-type = %q", got)
	}
	var got canon.VersionInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode /version body: %v\nbody: %s", err, rec.Body.String())
	}
	if want := canon.Versions(); got != want {
		t.Errorf("/version payload = %+v\nwant          = %+v", got, want)
	}
}

// TestHandleVersionSurfacesEphePathResolved – Phase 9 invariant.
// /version must include the deployment-resolved ephemeris data
// path under the canonical key "ephe_path_resolved".  The value is
// a diagnostic, not a canon constant, so the field appears only in
// /version – never in the trinity success/error response metadata.
func TestHandleVersionSurfacesEphePathResolved(t *testing.T) {
	handler := New()
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()

	handler.handleVersion(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("/version status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var generic map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &generic); err != nil {
		t.Fatalf("decode /version: %v", err)
	}
	got, ok := generic["ephe_path_resolved"].(string)
	if !ok {
		t.Fatalf("ephe_path_resolved missing or not string: %v", generic)
	}
	if got == "" {
		t.Errorf("ephe_path_resolved is empty; want a non-empty path")
	}
}

// TestHandleVersionRejectsWrongMethod – Phase 1 invariant.
func TestHandleVersionRejectsWrongMethod(t *testing.T) {
	handler := New()
	req := httptest.NewRequest(http.MethodPost, "/version", nil)
	rec := httptest.NewRecorder()

	handler.handleVersion(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

// TestHandleHealthzAlwaysOK pins the Phase 1 invariant the Phase 10
// plan re-asserts: GET /healthz is liveness-only and never carries
// version info.  The body must be exactly {"status":"ok"}.
func TestHandleHealthzAlwaysOK(t *testing.T) {
	handler := New()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("/healthz status = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("/healthz content-type = %q", got)
	}
	body := strings.TrimSpace(rec.Body.String())
	if body != `{"status":"ok"}` {
		t.Errorf("/healthz body = %q, want {\"status\":\"ok\"}", body)
	}
	// Phase 10: /healthz must NOT leak version info.
	for _, banned := range []string{"engine_version", "canon_version", "swisseph_version"} {
		if strings.Contains(rec.Body.String(), banned) {
			t.Errorf("/healthz body leaks %q: %s", banned, rec.Body.String())
		}
	}
}

// TestHandleManifestRejectsMissingContentType is the Phase 10
// content-type sentinel: a POST /manifest with no Content-Type
// header at all gets HTTP 415 with a Trinity invalid_input
// envelope, before the body is even read.
func TestHandleManifestRejectsMissingContentType(t *testing.T) {
	handler := New()
	req := httptest.NewRequest(http.MethodPost, "/manifest",
		strings.NewReader(canonicalBaseline))
	// Intentionally do NOT set Content-Type.
	rec := httptest.NewRecorder()

	handler.handleManifest(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415; body = %s", rec.Code, rec.Body.String())
	}
	var env output.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if env.Error.Type != output.ErrorInvalidInput {
		t.Errorf("error_type = %q, want %q",
			env.Error.Type, output.ErrorInvalidInput)
	}
	if env.Metadata != output.CurrentMetadata() {
		t.Errorf("envelope metadata drifted")
	}
	if !strings.Contains(env.Error.Message, "Content-Type") {
		t.Errorf("error message should mention Content-Type; got %q",
			env.Error.Message)
	}
}

// TestHandleManifestRejectsWrongContentType – Phase 10.  POST with
// Content-Type: text/plain (or anything that is not application/json)
// must be rejected with 415 + invalid_input.
func TestHandleManifestRejectsWrongContentType(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
	}{
		{"text/plain", "text/plain"},
		{"application/x-www-form-urlencoded", "application/x-www-form-urlencoded"},
		{"application/xml", "application/xml"},
		{"application/json typo", "application/jso"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			handler := New()
			req := httptest.NewRequest(http.MethodPost, "/manifest",
				strings.NewReader(canonicalBaseline))
			req.Header.Set("Content-Type", c.contentType)
			rec := httptest.NewRecorder()

			handler.handleManifest(rec, req)

			if rec.Code != http.StatusUnsupportedMediaType {
				t.Fatalf("status = %d, want 415", rec.Code)
			}
			var env output.ErrorEnvelope
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("decode envelope: %v", err)
			}
			if env.Error.Type != output.ErrorInvalidInput {
				t.Errorf("error_type = %q, want %q",
					env.Error.Type, output.ErrorInvalidInput)
			}
		})
	}
}

// TestHandleManifestAcceptsContentTypeWithCharset – Phase 10.
// "application/json; charset=utf-8" is the canonical wire form for
// JSON requests sent by many clients; accept it.
func TestHandleManifestAcceptsContentTypeWithCharset(t *testing.T) {
	handler := New()
	req := httptest.NewRequest(http.MethodPost, "/manifest",
		strings.NewReader(canonicalBaseline))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rec := httptest.NewRecorder()

	handler.handleManifest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

// TestHandleManifestRejectsOversizeBody is the Phase 10 oversize
// sentinel.  A request body that exceeds MaxRequestBodyBytes is
// rejected with HTTP 413 and a Trinity unsupported_input envelope.
func TestHandleManifestRejectsOversizeBody(t *testing.T) {
	handler := New()
	// Build a body that's syntactically a JSON object but obviously
	// over the limit: open-brace + giant pad + close-brace.
	pad := strings.Repeat("x", MaxRequestBodyBytes+1024)
	body := `{"birth_date":"1990-04-09","pad":"` + pad + `"}`
	req := httptest.NewRequest(http.MethodPost, "/manifest",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.handleManifest(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413; body = %s", rec.Code, rec.Body.String())
	}
	var env output.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if env.Error.Type != output.ErrorUnsupportedInput {
		t.Errorf("error_type = %q, want %q",
			env.Error.Type, output.ErrorUnsupportedInput)
	}
	if env.Metadata != output.CurrentMetadata() {
		t.Errorf("envelope metadata drifted")
	}
	if !strings.Contains(env.Error.Message, "exceeds") {
		t.Errorf("error message should mention size limit; got %q",
			env.Error.Message)
	}
}

// TestHandleManifestRejectsMalformedJSON is the Phase 10 malformed-
// JSON sentinel.  A syntactically broken JSON body classifies as
// invalid_input + 400.  This is enforced by the validator (Phase 2);
// Phase 10 pins it through the HTTP surface.
func TestHandleManifestRejectsMalformedJSON(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"truncated", `{"birth_date":"1990-04-09"`},
		{"missing colon", `{"birth_date" "1990-04-09"}`},
		{"trailing comma", `{"birth_date":"1990-04-09",}`},
		{"single quote", `{'birth_date':'1990-04-09'}`},
		{"plain text", `not json at all`},
		{"empty body", ``},
		{"null payload", `null`},
		{"array payload", `["birth_date","1990-04-09"]`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			handler := New()
			req := httptest.NewRequest(http.MethodPost, "/manifest",
				strings.NewReader(c.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.handleManifest(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body = %s",
					rec.Code, rec.Body.String())
			}
			var env output.ErrorEnvelope
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("decode envelope: %v\nbody: %s", err, rec.Body.String())
			}
			if env.Error.Type != output.ErrorInvalidInput {
				t.Errorf("error_type = %q, want %q",
					env.Error.Type, output.ErrorInvalidInput)
			}
		})
	}
}
