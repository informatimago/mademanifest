package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// DockerImageTag is the image reference that Phase 0 builds and drives
// through the Docker and Kubernetes harnesses.
const DockerImageTag = "mademanifest-engine:itest"

// DockerOptions configures StartDockerContainer.
type DockerOptions struct {
	// Image is the image reference to run.  Default: DockerImageTag.
	Image string
	// ExtraEnv is a list of KEY=VAL strings forwarded as "docker run -e".
	ExtraEnv []string
	// StartupTimeout caps how long PollHealthz is allowed to run.
	// Default: 30s.
	StartupTimeout time.Duration
	// SkipBuild bypasses BuildDockerImage and assumes Image already
	// exists locally (useful in CI where the image is prebuilt).
	SkipBuild bool
}

var (
	dockerBuildOnce sync.Once
	dockerBuildErr  error
)

// BuildDockerImage builds <repo>/src/Dockerfile once per test process,
// tagged as DockerImageTag.  It is safe to call from several tests
// concurrently.
func BuildDockerImage(t testing.TB) error {
	t.Helper()
	dockerBuildOnce.Do(func() {
		if _, err := exec.LookPath("docker"); err != nil {
			dockerBuildErr = fmt.Errorf("docker binary not on PATH: %w", err)
			return
		}
		srcDir := filepath.Join(RepoRoot(t), "src")
		cmd := exec.Command("docker", "build", "-f", "Dockerfile", "-t", DockerImageTag, ".")
		cmd.Dir = srcDir
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			dockerBuildErr = fmt.Errorf("docker build: %w", err)
		}
	})
	return dockerBuildErr
}

// StartDockerContainer builds the image (first call per process) and
// launches a container publishing container port 8080 to a free loopback
// port on the host.  The returned Shutdown removes the container.
func StartDockerContainer(t testing.TB, opts DockerOptions) ServerHandle {
	t.Helper()
	if opts.Image == "" {
		opts.Image = DockerImageTag
	}
	if opts.StartupTimeout == 0 {
		opts.StartupTimeout = 30 * time.Second
	}
	if !opts.SkipBuild {
		if err := BuildDockerImage(t); err != nil {
			t.Fatalf("build docker image: %v", err)
		}
	}

	port := FreePort(t)
	args := []string{"run", "-d", "--rm", "-p", fmt.Sprintf("127.0.0.1:%d:8080", port)}
	for _, e := range opts.ExtraEnv {
		args = append(args, "-e", e)
	}
	args = append(args, opts.Image)

	var stdout, stderr bytes.Buffer
	runCmd := exec.Command("docker", args...)
	runCmd.Stdout = &stdout
	runCmd.Stderr = &stderr
	if err := runCmd.Run(); err != nil {
		t.Fatalf("docker run: %v\nstderr: %s", err, stderr.String())
	}
	containerID := strings.TrimSpace(stdout.String())
	if containerID == "" {
		t.Fatalf("docker run returned empty container id")
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	if err := PollHealthz(baseURL, opts.StartupTimeout); err != nil {
		dumpDockerLogs(t, containerID)
		_ = exec.Command("docker", "rm", "-f", containerID).Run()
		t.Fatalf("docker container %s did not become healthy: %v", containerID, err)
	}

	shutdown := func() {
		_ = exec.Command("docker", "rm", "-f", containerID).Run()
	}
	return ServerHandle{
		BaseURL:  baseURL,
		Shutdown: shutdown,
	}
}

func dumpDockerLogs(t testing.TB, containerID string) {
	t.Helper()
	out, err := exec.Command("docker", "logs", containerID).CombinedOutput()
	if err != nil {
		t.Logf("docker logs %s: %v", containerID, err)
		return
	}
	t.Logf("docker logs %s:\n%s", containerID, out)
}
