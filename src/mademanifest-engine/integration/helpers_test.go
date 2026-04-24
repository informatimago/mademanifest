package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newDummyServer wires a minimal /healthz + /manifest behind
// httptest.NewServer so we can exercise the HTTP helpers without
// compiling the real engine binary.
func newDummyServer(t *testing.T) (*httptest.Server, *int32) {
	t.Helper()
	var manifestHits int32
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/manifest", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&manifestHits, 1)
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusUnsupportedMediaType)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var parsed map[string]any
		if err := json.Unmarshal(body, &parsed); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"status":"error","error":{"error_type":"invalid_input","message":"bad JSON"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","echo":` + string(body) + `}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &manifestHits
}

func TestPollHealthzSucceedsOnLiveServer(t *testing.T) {
	srv, _ := newDummyServer(t)
	if err := PollHealthz(srv.URL, 2*time.Second); err != nil {
		t.Fatalf("PollHealthz returned: %v", err)
	}
}

func TestPollHealthzReturnsErrorOnDeadServer(t *testing.T) {
	// Use a loopback port that nothing is bound to.  FreePort returns
	// a port that was momentarily bound and then closed; there's a
	// tiny race but it is vanishingly unlikely to produce a live
	// listener before the poll deadline.
	port := FreePort(t)
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	err := PollHealthz(url, 500*time.Millisecond)
	if err == nil {
		t.Fatal("expected error for dead server, got nil")
	}
	if !strings.Contains(err.Error(), "not healthy") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostJSONAndPostManifestRoundTrip(t *testing.T) {
	srv, hits := newDummyServer(t)

	type echo struct {
		Status string         `json:"status"`
		Echo   map[string]any `json:"echo"`
	}
	var got echo
	status, raw, err := PostManifest(srv.URL, map[string]any{"birth_date": "1990-04-09"}, &got)
	if err != nil {
		t.Fatalf("PostManifest returned: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", status, string(raw))
	}
	if got.Status != "success" {
		t.Fatalf("echo.Status = %q, want success", got.Status)
	}
	if got.Echo["birth_date"] != "1990-04-09" {
		t.Fatalf("echo payload lost: got %v", got.Echo)
	}
	if *hits != 1 {
		t.Fatalf("manifest hit counter = %d, want 1", *hits)
	}
}

func TestPostJSONPreservesRawBytesWhenBodyIsBytes(t *testing.T) {
	srv, _ := newDummyServer(t)
	status, raw, err := PostJSON(srv.URL, "/manifest", []byte(`{"ok":true}`))
	if err != nil {
		t.Fatalf("PostJSON: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, raw = %s", status, raw)
	}
	if !strings.Contains(string(raw), `"ok":true`) {
		t.Fatalf("raw body did not echo payload: %s", raw)
	}
}

func TestPostJSONReportsNon2xxWithoutError(t *testing.T) {
	srv, _ := newDummyServer(t)
	// Send invalid JSON so the dummy returns 400 with an error envelope.
	status, raw, err := PostJSON(srv.URL, "/manifest", []byte("not-json"))
	if err != nil {
		t.Fatalf("PostJSON returned transport error: %v", err)
	}
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", status, raw)
	}
	if !strings.Contains(string(raw), `"invalid_input"`) {
		t.Fatalf("expected error envelope, got: %s", raw)
	}
}

func TestGetJSONReadsHealthz(t *testing.T) {
	srv, _ := newDummyServer(t)
	status, raw, err := GetJSON(srv.URL, "/healthz")
	if err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if !strings.Contains(string(raw), `"status":"ok"`) {
		t.Fatalf("unexpected body: %s", raw)
	}
}

func TestFreePortReturnsDistinctPorts(t *testing.T) {
	seen := make(map[int]bool, 8)
	for i := 0; i < 8; i++ {
		p := FreePort(t)
		if p <= 0 || p > 65535 {
			t.Fatalf("FreePort returned %d", p)
		}
		if seen[p] {
			t.Fatalf("FreePort returned duplicate port %d", p)
		}
		seen[p] = true
	}
}

func TestRepoRootPointsAtCheckout(t *testing.T) {
	root := RepoRoot(t)
	// The repo root must contain src/mademanifest-engine/go.mod –
	// otherwise the path math is wrong.
	modPath := filepath.Join(root, "src", "mademanifest-engine", "go.mod")
	if _, err := os.Stat(modPath); err != nil {
		t.Fatalf("RepoRoot=%s does not point at a checkout: %v", root, err)
	}
}

