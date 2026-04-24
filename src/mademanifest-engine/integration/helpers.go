// Package integration contains the shared test harness used by the Trinity
// implementation plan.  The harness exposes three ways to bring up the
// engine under test:
//
//   - StartLocalServer:   compile cmd/httpserver and run it as a subprocess
//                         on a loopback port (always compiled; driven by
//                         tests behind the integration_local build tag).
//   - StartDockerContainer: build the production image and docker run -d -p
//                         (always compiled; driven by tests behind the
//                         integration_docker build tag).
//   - StartKindCluster:   bring up a kind cluster, load the image, apply
//                         the repo kustomization with an image override,
//                         and kubectl port-forward to the service
//                         (always compiled; driven by tests behind the
//                         integration_k8s build tag).
//
// The helpers in this file are generic: they talk to any ServerHandle over
// HTTP and are verified against an httptest.NewServer dummy in
// helpers_test.go, so they can be exercised by the default "go test ./..."
// invocation that does not require docker, kind, or the compiled binary.
package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// ServerHandle is the contract every runtime launcher returns.
//
// BaseURL is the root URL (scheme://host:port) at which /healthz and
// /manifest are reachable.  Shutdown is idempotent and must always be
// called by the test (typically via t.Cleanup).
type ServerHandle struct {
	BaseURL  string
	Shutdown func()
}

// FreePort returns an unused loopback TCP port.  The listener is closed
// before the function returns, so there is a small race window during
// which another process on the host could grab the port.  This matches
// the standard Go idiom for test harnesses and is acceptable for our
// single-tenant CI runners.
func FreePort(t testing.TB) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pick free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// RepoRoot returns the absolute path of the repository root, resolved
// from the location of this source file so the result is independent of
// the working directory in which "go test" is invoked.
func RepoRoot(t testing.TB) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	// thisFile = <repo>/src/mademanifest-engine/integration/helpers.go
	// repo root = three directories up from the file's dir.
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
}

// PollHealthz sends GET baseURL+"/healthz" repeatedly until it receives a
// 2xx response or the deadline elapses.  It returns nil on success and a
// descriptive error otherwise.
func PollHealthz(baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	target, err := url.JoinPath(baseURL, "/healthz")
	if err != nil {
		return fmt.Errorf("build healthz URL: %w", err)
	}
	client := &http.Client{Timeout: 2 * time.Second}
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(target)
		if err != nil {
			lastErr = err
		} else {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
			lastErr = fmt.Errorf("healthz status %d", resp.StatusCode)
		}
		time.Sleep(100 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("deadline exceeded before first attempt")
	}
	return fmt.Errorf("%s not healthy within %s: %w", target, timeout, lastErr)
}

// PostJSON sends body (encoded as JSON unless it is already a []byte) to
// baseURL+path with Content-Type: application/json and returns the HTTP
// status code and response body bytes.  Transport errors are returned as
// err; non-2xx status codes are not errors – callers decide.
func PostJSON(baseURL, path string, body any) (int, []byte, error) {
	target, err := url.JoinPath(baseURL, path)
	if err != nil {
		return 0, nil, fmt.Errorf("build URL: %w", err)
	}
	var buf bytes.Buffer
	switch v := body.(type) {
	case nil:
		// empty body
	case []byte:
		buf.Write(v)
	case string:
		buf.WriteString(v)
	default:
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return 0, nil, fmt.Errorf("encode body: %w", err)
		}
	}
	req, err := http.NewRequest(http.MethodPost, target, &buf)
	if err != nil {
		return 0, nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, raw, fmt.Errorf("read body: %w", err)
	}
	return resp.StatusCode, raw, nil
}

// PostManifest is a thin wrapper around PostJSON that targets /manifest
// and, if out is non-nil, JSON-decodes the response body into it.  The
// HTTP status code is returned regardless of decode success so callers
// can distinguish transport-level, protocol-level, and payload-level
// failures.
func PostManifest(baseURL string, body any, out any) (int, []byte, error) {
	status, raw, err := PostJSON(baseURL, "/manifest", body)
	if err != nil {
		return status, raw, err
	}
	if out != nil && len(raw) > 0 {
		if decodeErr := json.Unmarshal(raw, out); decodeErr != nil {
			return status, raw, fmt.Errorf("decode manifest body: %w", decodeErr)
		}
	}
	return status, raw, nil
}

// GetJSON is the symmetric helper for read-only endpoints like /healthz
// or the forthcoming /version.
func GetJSON(baseURL, path string) (int, []byte, error) {
	target, err := url.JoinPath(baseURL, path)
	if err != nil {
		return 0, nil, fmt.Errorf("build URL: %w", err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(target)
	if err != nil {
		return 0, nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, raw, fmt.Errorf("read body: %w", err)
	}
	return resp.StatusCode, raw, nil
}
