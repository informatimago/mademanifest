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
	handler := New(canon.Paths{})
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
	handler := New(canon.Paths{})
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

// TestHandleManifestValidPayloadReturnsExecutionFailure documents
// the Phase 2 placeholder: a fully-valid Trinity payload returns
// execution_failure with HTTP 500 because the calculation pipeline
// is not yet wired.  Once Phases 3-8 land, this test must be
// updated to expect a success envelope.
func TestHandleManifestValidPayloadReturnsExecutionFailure(t *testing.T) {
	handler := New(canon.Paths{})
	req := httptest.NewRequest(http.MethodPost, "/manifest",
		strings.NewReader(canonicalBaseline))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.handleManifest(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body = %s", rec.Code, rec.Body.String())
	}
	var env output.ErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v\nbody: %s", err, rec.Body.String())
	}
	if env.Error.Type != output.ErrorExecutionFailure {
		t.Errorf("error_type = %q, want %q",
			env.Error.Type, output.ErrorExecutionFailure)
	}
	if !strings.Contains(env.Error.Message, "not yet implemented") {
		t.Errorf("placeholder message expected to mention \"not yet implemented\"; got %q",
			env.Error.Message)
	}
}

// TestHandleManifestProcessorErrorWrapsExecutionFailure swaps in a
// processor that returns an error and verifies the handler wraps
// the error as a Trinity execution_failure envelope.
func TestHandleManifestProcessorErrorWrapsExecutionFailure(t *testing.T) {
	handler := Handler{
		Process: func(_ io.Reader, _ canon.Paths) ([]byte, int, error) {
			return nil, 0, errors.New("synthetic processor failure")
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/manifest",
		strings.NewReader(`{}`))
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
		Process: func(_ io.Reader, _ canon.Paths) ([]byte, int, error) {
			panic("boom")
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/manifest",
		strings.NewReader(`{}`))
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
	handler := New(canon.Paths{})
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

// TestHandleVersionRejectsWrongMethod – Phase 1 invariant.
func TestHandleVersionRejectsWrongMethod(t *testing.T) {
	handler := New(canon.Paths{})
	req := httptest.NewRequest(http.MethodPost, "/version", nil)
	rec := httptest.NewRecorder()

	handler.handleVersion(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}
