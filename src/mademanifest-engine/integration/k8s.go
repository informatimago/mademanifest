package integration

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// K8sOptions configures StartKindCluster.
type K8sOptions struct {
	// ClusterName is the kind cluster name.  Default: trinity-itest.
	ClusterName string
	// ImageTag is the image to load into the cluster and pin in the
	// deployment.  Default: DockerImageTag.
	ImageTag string
	// Namespace targets a specific namespace.  Default: default.
	Namespace string
	// DeploymentName is the deployment to wait for.  Default: mademanifest.
	DeploymentName string
	// ServiceName is the service to port-forward.  Default: mademanifest.
	ServiceName string
	// KustomizationDir overrides the path to the kustomize base.
	// Default: <repo>/src/deploy/kubernetes.
	KustomizationDir string
	// StartupTimeout caps the end-to-end readiness poll.  Default: 180s.
	StartupTimeout time.Duration
	// SkipImageBuild bypasses BuildDockerImage (the image is assumed to
	// already exist locally).
	SkipImageBuild bool
	// ReuseExistingCluster tells the helper to use a pre-existing
	// cluster of the same name without deleting it on shutdown.  The
	// cluster is reused automatically when it is already registered; set
	// this flag to force reuse even for a freshly created one (useful
	// for iterative local debugging).
	ReuseExistingCluster bool
}

// StartKindCluster brings up a kind cluster (unless one with the same
// name already exists), loads ImageTag into it, applies the repo
// kustomize base with an image override pointing at the loaded image,
// waits for the deployment to roll out, and opens kubectl port-forward
// to the service.  The returned ServerHandle.Shutdown disposes of every
// resource this call created, including the cluster itself – but only
// if this call is the one that created it.
//
// This function does *not* use testing.TB so it can be invoked from
// TestMain, where tests typically want to share a single cluster across
// many Test* functions.  Callers that want the test to fail-fast should
// wrap it with MustStartKindCluster.
func StartKindCluster(opts K8sOptions) (ServerHandle, error) {
	if opts.ClusterName == "" {
		opts.ClusterName = "trinity-itest"
	}
	if opts.ImageTag == "" {
		opts.ImageTag = DockerImageTag
	}
	if opts.Namespace == "" {
		opts.Namespace = "default"
	}
	if opts.DeploymentName == "" {
		opts.DeploymentName = "mademanifest"
	}
	if opts.ServiceName == "" {
		opts.ServiceName = "mademanifest"
	}
	if opts.KustomizationDir == "" {
		root, err := repoRootFromSource()
		if err != nil {
			return ServerHandle{}, fmt.Errorf("resolve repo root: %w", err)
		}
		opts.KustomizationDir = filepath.Join(root, "src", "deploy", "kubernetes")
	}
	if opts.StartupTimeout == 0 {
		opts.StartupTimeout = 180 * time.Second
	}

	for _, bin := range []string{"kind", "kubectl", "docker"} {
		if _, err := exec.LookPath(bin); err != nil {
			return ServerHandle{}, fmt.Errorf("required tool not found on PATH: %s (%w)", bin, err)
		}
	}

	// Cleanup stack: every successful step pushes its undo action here.
	// If a later step fails we unwind in reverse, so partial setups
	// don't leave dangling clusters or port-forwards.
	var cleanups []func()
	addCleanup := func(fn func()) { cleanups = append(cleanups, fn) }
	runCleanupsOnError := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}

	// 1. Docker image build.
	if !opts.SkipImageBuild {
		if err := buildDockerImageNoTB(); err != nil {
			return ServerHandle{}, fmt.Errorf("build docker image: %w", err)
		}
	}

	// 2. Cluster: reuse pre-existing one, or create a fresh one and
	// register its teardown.
	preExisting, err := kindClusterExists(opts.ClusterName)
	if err != nil {
		return ServerHandle{}, fmt.Errorf("list kind clusters: %w", err)
	}
	weCreatedCluster := false
	switch {
	case preExisting:
		// Reuse regardless of ReuseExistingCluster; never delete
		// someone else's cluster on shutdown.
	default:
		if err := createKindCluster(opts.ClusterName); err != nil {
			return ServerHandle{}, fmt.Errorf("create kind cluster: %w", err)
		}
		weCreatedCluster = true
		if !opts.ReuseExistingCluster {
			addCleanup(func() {
				_ = exec.Command("kind", "delete", "cluster", "--name", opts.ClusterName).Run()
			})
		}
	}

	// 3. Load the local image into the cluster's containerd.
	loadCmd := exec.Command("kind", "load", "docker-image", opts.ImageTag, "--name", opts.ClusterName)
	loadCmd.Stdout, loadCmd.Stderr = os.Stderr, os.Stderr
	if err := loadCmd.Run(); err != nil {
		runCleanupsOnError()
		return ServerHandle{}, fmt.Errorf("kind load docker-image: %w", err)
	}

	// 4. Write a kustomize overlay that references the base via a
	// relative path (kustomize rejects absolute paths in resources:)
	// and overrides the production registry image with the one we just
	// loaded.
	overlayDir, err := writeKustomizeOverlay(opts.KustomizationDir, opts.ImageTag)
	if err != nil {
		runCleanupsOnError()
		return ServerHandle{}, fmt.Errorf("write kustomize overlay: %w", err)
	}
	addCleanup(func() { _ = os.RemoveAll(overlayDir) })

	// 5. Apply the overlay and register its teardown.
	applyCmd := exec.Command("kubectl", "apply", "-k", overlayDir, "--namespace", opts.Namespace)
	applyCmd.Stdout, applyCmd.Stderr = os.Stderr, os.Stderr
	if err := applyCmd.Run(); err != nil {
		runCleanupsOnError()
		return ServerHandle{}, fmt.Errorf("kubectl apply -k: %w", err)
	}
	addCleanup(func() {
		del := exec.Command("kubectl", "delete", "-k", overlayDir,
			"--namespace", opts.Namespace, "--ignore-not-found=true",
			"--wait=false")
		del.Stdout, del.Stderr = os.Stderr, os.Stderr
		_ = del.Run()
	})

	// 6. Wait for the deployment to roll out.  kubectl rollout has its
	// own timeout; mirror ours.
	waitCmd := exec.Command("kubectl", "rollout", "status",
		"deployment/"+opts.DeploymentName,
		"--namespace", opts.Namespace,
		"--timeout="+fmt.Sprintf("%ds", int(opts.StartupTimeout.Seconds())),
	)
	waitCmd.Stdout, waitCmd.Stderr = os.Stderr, os.Stderr
	if err := waitCmd.Run(); err != nil {
		dumpKubectlDiagnostics(opts.Namespace, opts.DeploymentName)
		runCleanupsOnError()
		return ServerHandle{}, fmt.Errorf("kubectl rollout status: %w", err)
	}

	// 7. Open a port-forward and wait for /healthz.
	pfPort, err := freePortNoTB()
	if err != nil {
		runCleanupsOnError()
		return ServerHandle{}, fmt.Errorf("allocate port-forward port: %w", err)
	}
	pfCmd := exec.Command("kubectl", "port-forward",
		"svc/"+opts.ServiceName,
		fmt.Sprintf("%d:80", pfPort),
		"--namespace", opts.Namespace,
		"--address", "127.0.0.1",
	)
	pfOut, err := pfCmd.StdoutPipe()
	if err != nil {
		runCleanupsOnError()
		return ServerHandle{}, fmt.Errorf("port-forward stdout pipe: %w", err)
	}
	pfCmd.Stderr = os.Stderr
	if err := pfCmd.Start(); err != nil {
		runCleanupsOnError()
		return ServerHandle{}, fmt.Errorf("start kubectl port-forward: %w", err)
	}
	go io.Copy(io.Discard, pfOut)
	addCleanup(func() {
		if pfCmd.Process != nil {
			_ = pfCmd.Process.Kill()
			_ = pfCmd.Wait()
		}
	})

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", pfPort)
	if err := PollHealthz(baseURL, opts.StartupTimeout); err != nil {
		dumpKubectlDiagnostics(opts.Namespace, opts.DeploymentName)
		runCleanupsOnError()
		return ServerHandle{}, fmt.Errorf("kubernetes service did not become healthy: %w", err)
	}

	_ = weCreatedCluster // kept for future logging / diagnostics

	shutdown := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}
	return ServerHandle{
		BaseURL:  baseURL,
		Shutdown: shutdown,
	}, nil
}

// kindClusterExists returns whether a kind cluster with the given name
// is currently registered.
func kindClusterExists(name string) (bool, error) {
	out, err := exec.Command("kind", "get", "clusters").Output()
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == name {
			return true, nil
		}
	}
	return false, nil
}

// createKindCluster creates a fresh cluster, waiting up to 60s for the
// control-plane to report Ready before returning.
func createKindCluster(name string) error {
	create := exec.Command("kind", "create", "cluster", "--name", name, "--wait", "60s")
	create.Stdout, create.Stderr = os.Stderr, os.Stderr
	return create.Run()
}

// writeKustomizeOverlay materialises a temp directory with a
// kustomization.yaml that references the repo base by a *relative* path
// (kustomize refuses absolute paths in resources:) and overrides the
// placeholder image.  Both paths are canonicalised via EvalSymlinks
// first – macOS's default $TMPDIR is inside /var, which is a symlink to
// /private/var; kustomize resolves symlinks while walking the relative
// path and otherwise ends up looking for non-existent directories like
// /private/Users/....
//
// Phase 9: the overlay also hard-codes a SE_NODE_POLICY=true env on
// the mademanifest container.  After Phase 6 retired the env-driven
// node policy shim, that variable must have no effect; injecting it
// here and asserting the canonical Phase 4-8 oracles still match
// makes every k8s integration test an env-immunity sentinel.  The
// docker harness covers the same invariant via DockerOptions.ExtraEnv,
// and the local harness via LocalServerOptions.ExtraEnv – this is
// the third leg.
func writeKustomizeOverlay(basePath, imageTag string) (string, error) {
	dir, err := os.MkdirTemp("", "trinity-k8s-overlay-*")
	if err != nil {
		return "", fmt.Errorf("mkdir temp: %w", err)
	}
	canonicalDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("canonicalise overlay dir: %w", err)
	}
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("abs base: %w", err)
	}
	canonicalBase, err := filepath.EvalSymlinks(absBase)
	if err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("canonicalise base path %q: %w", absBase, err)
	}
	relBase, err := filepath.Rel(canonicalDir, canonicalBase)
	if err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("relative base: %w", err)
	}
	newName, newTag, ok := strings.Cut(imageTag, ":")
	if !ok {
		newTag = "latest"
	}

	// Strategic-merge patch that adds SE_NODE_POLICY=true to the
	// mademanifest container.  The env is the Phase 9 env-immunity
	// probe described in the function docstring.
	patchPath := "envimmunity-patch.yaml"
	patchBody := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: mademanifest
spec:
  template:
    spec:
      containers:
        - name: mademanifest
          env:
            - name: SE_NODE_POLICY
              value: "true"
`
	if err := os.WriteFile(filepath.Join(dir, patchPath), []byte(patchBody), 0644); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("write env-immunity patch: %w", err)
	}

	content := fmt.Sprintf(`apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - %s
patches:
  - path: %s
images:
  - name: registry.example.com/mademanifest
    newName: %s
    newTag: %s
`, relBase, patchPath, newName, newTag)
	if err := os.WriteFile(filepath.Join(dir, "kustomization.yaml"), []byte(content), 0644); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("write overlay: %w", err)
	}
	return dir, nil
}

// dumpKubectlDiagnostics attaches best-effort kubectl output to stderr
// when readiness fails.  Uses stderr rather than a testing.TB because
// we are potentially called from TestMain.
func dumpKubectlDiagnostics(namespace, deployment string) {
	cmds := [][]string{
		{"kubectl", "get", "pods", "-o", "wide", "--namespace", namespace},
		{"kubectl", "describe", "deployment", deployment, "--namespace", namespace},
		{"kubectl", "logs", "deployment/" + deployment, "--namespace", namespace, "--tail=200"},
	}
	for _, c := range cmds {
		out, err := exec.Command(c[0], c[1:]...).CombinedOutput()
		fmt.Fprintf(os.Stderr, "==> %s\n", strings.Join(c, " "))
		if err != nil {
			fmt.Fprintf(os.Stderr, "    (error: %v)\n", err)
		}
		os.Stderr.Write(out)
		fmt.Fprintln(os.Stderr)
	}
}

// buildDockerImageNoTB is the TB-free sibling of BuildDockerImage so
// the kind helper can be called from TestMain.  The sync.Once in
// docker.go ensures the build only runs once per process.
func buildDockerImageNoTB() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker binary not on PATH: %w", err)
	}
	root, err := repoRootFromSource()
	if err != nil {
		return err
	}
	var outerErr error
	dockerBuildOnce.Do(func() {
		cmd := exec.Command("docker", "build", "-f", "Dockerfile", "-t", DockerImageTag, ".")
		cmd.Dir = filepath.Join(root, "src")
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			dockerBuildErr = fmt.Errorf("docker build: %w", err)
		}
	})
	if outerErr == nil {
		outerErr = dockerBuildErr
	}
	return outerErr
}

// freePortNoTB is the TB-free sibling of FreePort.
func freePortNoTB() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("listen on 127.0.0.1:0: %w", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// repoRootFromSource walks up from this source file (not the caller's)
// to the repo root.
func repoRootFromSource() (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("runtime.Caller(0) failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..")), nil
}
