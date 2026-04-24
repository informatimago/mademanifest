package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// LocalServerOptions configures StartLocalServer.  Empty fields fall back
// to repo-local defaults so the common-case caller can pass an empty
// struct.
type LocalServerOptions struct {
	// CanonDir overrides CANON_DIRECTORY.  Default: <repo>/src/canon.
	CanonDir string
	// EphemerisPath overrides SE_EPHE_PATH.
	// Default: <repo>/src/ephemeris/data/REQUIRED_EPHEMERIS_FILES.
	EphemerisPath string
	// ExtraEnv is appended to the subprocess environment after the
	// defaults.  Later entries override earlier ones.
	ExtraEnv []string
	// StartupTimeout caps how long PollHealthz is allowed to run.
	// Default: 15s.
	StartupTimeout time.Duration
	// BuildFlags are forwarded to "go build" during the one-time binary
	// compile.  Default: empty.
	BuildFlags []string
}

var (
	binaryBuildOnce sync.Once
	binaryBuildErr  error
	binaryPath      string
)

// buildHTTPServerBinary compiles cmd/httpserver to a temp file exactly
// once per test process.  The binary is cached in binaryPath and reused
// by every StartLocalServer call.
func buildHTTPServerBinary(t testing.TB, flags []string) (string, error) {
	t.Helper()
	binaryBuildOnce.Do(func() {
		tmp, err := os.CreateTemp("", "mademanifest-http-*")
		if err != nil {
			binaryBuildErr = fmt.Errorf("create temp file: %w", err)
			return
		}
		tmp.Close()
		if err := os.Remove(tmp.Name()); err != nil {
			binaryBuildErr = fmt.Errorf("remove temp placeholder: %w", err)
			return
		}
		binaryPath = tmp.Name()

		engineDir := filepath.Join(RepoRoot(t), "src", "mademanifest-engine")
		args := []string{"build"}
		args = append(args, flags...)
		args = append(args, "-o", binaryPath, "./cmd/httpserver")
		cmd := exec.Command("go", args...)
		cmd.Dir = engineDir
		cmd.Env = append(os.Environ(), "CGO_LDFLAGS=-lm")
		out, err := cmd.CombinedOutput()
		if err != nil {
			binaryBuildErr = fmt.Errorf("go build ./cmd/httpserver: %w\n%s", err, out)
		}
	})
	return binaryPath, binaryBuildErr
}

// StartLocalServer builds the HTTP server binary (once per process) and
// launches it on a free loopback port with repo-local canon and
// ephemeris paths.  The returned handle's Shutdown sends SIGINT first,
// falling back to SIGKILL after three seconds.
func StartLocalServer(t testing.TB, opts LocalServerOptions) ServerHandle {
	t.Helper()

	bin, err := buildHTTPServerBinary(t, opts.BuildFlags)
	if err != nil {
		t.Fatalf("build http server binary: %v", err)
	}

	root := RepoRoot(t)
	if opts.CanonDir == "" {
		opts.CanonDir = filepath.Join(root, "src", "canon")
	}
	if opts.EphemerisPath == "" {
		opts.EphemerisPath = filepath.Join(root, "src", "ephemeris", "data", "REQUIRED_EPHEMERIS_FILES")
	}
	if opts.StartupTimeout == 0 {
		opts.StartupTimeout = 15 * time.Second
	}

	port := FreePort(t)
	env := append(os.Environ(),
		fmt.Sprintf("PORT=%d", port),
		"CANON_DIRECTORY="+opts.CanonDir,
		"SE_EPHE_PATH="+opts.EphemerisPath,
	)
	env = append(env, opts.ExtraEnv...)

	cmd := exec.Command(bin)
	cmd.Env = env
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start local server: %v", err)
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	if err := PollHealthz(baseURL, opts.StartupTimeout); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("local server did not become healthy: %v", err)
	}

	shutdown := func() {
		if cmd.Process == nil {
			return
		}
		_ = cmd.Process.Signal(os.Interrupt)
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
	}

	return ServerHandle{
		BaseURL:  baseURL,
		Shutdown: shutdown,
	}
}
