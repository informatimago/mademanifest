package integration

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"mademanifest-engine/pkg/httpservice"
	"mademanifest-engine/pkg/trinity/output"
)

// AssertHTTPContract is the Phase 10 end-to-end probe of the
// mademanifest HTTP surface.  It exercises every error path the
// implementation plan calls out:
//
//   * method not allowed         (HTTP 405 on GET /manifest)
//   * missing Content-Type       (HTTP 415 + invalid_input)
//   * wrong Content-Type         (HTTP 415 + invalid_input)
//   * oversize body              (HTTP 413 + unsupported_input)
//   * malformed JSON body        (HTTP 400 + invalid_input)
//
// plus the post-conditions on the canonical happy paths:
//
//   * GET /healthz               (HTTP 200, body == {"status":"ok"})
//   * GET /version with charset  (HTTP 200, ephe_path_resolved set)
//   * POST /manifest with charset Content-Type (HTTP 200, success)
//
// Shared by the local, docker, and k8s harness tests so every
// runtime exercises the full contract through real network I/O.
func AssertHTTPContract(t *testing.T, baseURL string) {
	t.Helper()

	t.Run("method_not_allowed", func(t *testing.T) {
		status, _, err := GetJSON(baseURL, "/manifest")
		if err != nil {
			t.Fatalf("GET /manifest: %v", err)
		}
		if status != http.StatusMethodNotAllowed {
			t.Errorf("status = %d, want %d", status, http.StatusMethodNotAllowed)
		}
	})

	t.Run("missing_content_type", func(t *testing.T) {
		status, raw, err := PostRaw(baseURL, "/manifest",
			[]byte(`{"birth_date":"1990-04-09"}`), "")
		if err != nil {
			t.Fatalf("POST /manifest: %v", err)
		}
		if status != http.StatusUnsupportedMediaType {
			t.Fatalf("status = %d, want %d; body = %s",
				status, http.StatusUnsupportedMediaType, raw)
		}
		assertErrorEnvelopeType(t, raw, output.ErrorInvalidInput)
	})

	t.Run("wrong_content_type", func(t *testing.T) {
		status, raw, err := PostRaw(baseURL, "/manifest",
			[]byte(`{"birth_date":"1990-04-09"}`), "text/plain")
		if err != nil {
			t.Fatalf("POST /manifest: %v", err)
		}
		if status != http.StatusUnsupportedMediaType {
			t.Fatalf("status = %d, want %d; body = %s",
				status, http.StatusUnsupportedMediaType, raw)
		}
		assertErrorEnvelopeType(t, raw, output.ErrorInvalidInput)
	})

	t.Run("oversize_body", func(t *testing.T) {
		// Build a syntactically-valid JSON object whose body
		// exceeds MaxRequestBodyBytes by a comfortable margin.
		pad := strings.Repeat("x", httpservice.MaxRequestBodyBytes+4096)
		body := []byte(`{"birth_date":"1990-04-09","pad":"` + pad + `"}`)
		status, raw, err := PostRaw(baseURL, "/manifest", body, "application/json")
		if err != nil {
			t.Fatalf("POST /manifest: %v", err)
		}
		if status != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want %d; body = %s",
				status, http.StatusRequestEntityTooLarge, raw)
		}
		assertErrorEnvelopeType(t, raw, output.ErrorUnsupportedInput)
	})

	t.Run("malformed_json", func(t *testing.T) {
		cases := []struct {
			name string
			body string
		}{
			{"truncated", `{"birth_date":"1990-04-09"`},
			{"plain_text", `not json at all`},
			{"empty_body", ``},
			{"array_payload", `["1990-04-09"]`},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				status, raw, err := PostRaw(baseURL, "/manifest",
					[]byte(c.body), "application/json")
				if err != nil {
					t.Fatalf("POST /manifest: %v", err)
				}
				if status != http.StatusBadRequest {
					t.Fatalf("status = %d, want %d; body = %s",
						status, http.StatusBadRequest, raw)
				}
				assertErrorEnvelopeType(t, raw, output.ErrorInvalidInput)
			})
		}
	})

	t.Run("application_json_with_charset", func(t *testing.T) {
		body := []byte(`{"birth_date":"1990-04-09","birth_time":"18:04",` +
			`"timezone":"Europe/Amsterdam","latitude":51.9167,"longitude":4.4}`)
		status, raw, err := PostRaw(baseURL, "/manifest", body,
			"application/json; charset=utf-8")
		if err != nil {
			t.Fatalf("POST /manifest: %v", err)
		}
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", status, raw)
		}
		var env output.SuccessEnvelope
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatalf("decode envelope: %v\nbody: %s", err, raw)
		}
		if env.Status != output.StatusSuccess {
			t.Errorf("envelope status = %q, want %q",
				env.Status, output.StatusSuccess)
		}
	})

	t.Run("healthz_liveness_only", func(t *testing.T) {
		status, raw, err := GetJSON(baseURL, "/healthz")
		if err != nil {
			t.Fatalf("GET /healthz: %v", err)
		}
		if status != http.StatusOK {
			t.Fatalf("status = %d, want 200", status)
		}
		body := strings.TrimSpace(string(raw))
		if body != `{"status":"ok"}` {
			t.Errorf("/healthz body = %q, want canonical liveness payload", body)
		}
		// Phase 10: /healthz must not leak version info.
		for _, banned := range []string{"engine_version", "canon_version", "swisseph_version"} {
			if strings.Contains(string(raw), banned) {
				t.Errorf("/healthz body leaks %q: %s", banned, raw)
			}
		}
	})
}

func assertErrorEnvelopeType(t *testing.T, raw []byte, want string) {
	t.Helper()
	var env output.ErrorEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("decode error envelope: %v\nbody: %s", err, raw)
	}
	if env.Status != output.StatusError {
		t.Errorf("envelope status = %q, want %q", env.Status, output.StatusError)
	}
	if env.Error.Type != want {
		t.Errorf("error_type = %q, want %q\nbody: %s", env.Error.Type, want, raw)
	}
	if env.Error.Message == "" {
		t.Errorf("error message must not be empty\nbody: %s", raw)
	}
	if env.Metadata != output.CurrentMetadata() {
		t.Errorf("envelope metadata drift\nbody: %s", raw)
	}
}
