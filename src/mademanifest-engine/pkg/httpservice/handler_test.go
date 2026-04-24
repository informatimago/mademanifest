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
)

func TestHandleManifestReturnsOutput(t *testing.T) {
	handler := Handler{
		CanonPaths: canon.Paths{},
		Process: func(bodyReader io.Reader, _ canon.Paths) ([]byte, error) {
			payload, err := io.ReadAll(bodyReader)
			if err != nil {
				return nil, err
			}
			if string(payload) != `{"hello":"world"}` {
				t.Fatalf("unexpected body %q", string(payload))
			}
			return []byte(`{"ok":true}`), nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/manifest", strings.NewReader(`{"hello":"world"}`))
	rec := httptest.NewRecorder()

	handler.handleManifest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content-type: %s", got)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"ok":true}` {
		t.Fatalf("unexpected body: %s", got)
	}
}

func TestHandleManifestRejectsWrongMethod(t *testing.T) {
	handler := New(canon.Paths{})
	req := httptest.NewRequest(http.MethodGet, "/manifest", nil)
	rec := httptest.NewRecorder()

	handler.handleManifest(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestHandleManifestReturnsBadRequestOnError(t *testing.T) {
	handler := Handler{
		Process: func(_ io.Reader, _ canon.Paths) ([]byte, error) {
			return nil, errors.New("bad input")
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/manifest", strings.NewReader(`{"bad":true}`))
	rec := httptest.NewRecorder()

	handler.handleManifest(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "bad input") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestHandleManifestRecoversFromPanic(t *testing.T) {
	handler := Handler{
		Process: func(_ io.Reader, _ canon.Paths) ([]byte, error) {
			panic("boom")
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/manifest", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	handler.handleManifest(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

// TestHandleVersionReturnsCompiledInValues verifies the /version
// endpoint echoes the pinned constants from pkg/canon without
// mutation.  Any future drift between the endpoint and the compiled
// VersionInfo is caught here before it reaches an integration test.
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

func TestHandleVersionRejectsWrongMethod(t *testing.T) {
	handler := New(canon.Paths{})
	req := httptest.NewRequest(http.MethodPost, "/version", nil)
	rec := httptest.NewRecorder()

	handler.handleVersion(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}
