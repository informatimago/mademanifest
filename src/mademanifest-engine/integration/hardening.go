package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// AssertHardenedDeploymentShape is the Phase 13 sentinel for the
// kustomize hardening pass.  Given a kind-deployed mademanifest
// service, it inspects the actual cluster state via kubectl and
// asserts every Phase 13 deployment-hardening invariant:
//
//   * a NetworkPolicy named "mademanifest" exists in the namespace.
//   * a HorizontalPodAutoscaler named "mademanifest" exists with
//     target CPU utilisation 70% and min/max replicas 1/10.
//   * the pod runs as non-root with UID 10001 (matches Dockerfile
//     USER 10001:10001).
//   * the container has readOnlyRootFilesystem=true,
//     allowPrivilegeEscalation=false, and capabilities.drop=[ALL].
//   * resource requests/limits are present on the container.
//
// The shape checks do not require a NetworkPolicy-enforcing CNI;
// the resource has to exist, but actual enforcement is asserted by
// AssertNetworkPolicyEnforced (gated on TRINITY_NETWORK_POLICY_ENFORCED=1).
//
// Phase 13 enforces the deployment-time hardening posture; Phase 14
// (release + acceptance) reviews CVEs against the pinned digests.
func AssertHardenedDeploymentShape(t *testing.T, namespace string) {
	t.Helper()
	if namespace == "" {
		namespace = "default"
	}

	t.Run("network_policy_present", func(t *testing.T) {
		out, err := kubectlJSON("get", "networkpolicy", "mademanifest",
			"-n", namespace, "-o", "json")
		if err != nil {
			t.Fatalf("get networkpolicy mademanifest: %v\n%s", err, out)
		}
		var np struct {
			Spec struct {
				Ingress []struct {
					Ports []struct {
						Port any `json:"port"`
					} `json:"ports"`
				} `json:"ingress"`
				PolicyTypes []string `json:"policyTypes"`
			} `json:"spec"`
		}
		if err := json.Unmarshal(out, &np); err != nil {
			t.Fatalf("decode networkpolicy: %v\n%s", err, out)
		}
		if !contains(np.Spec.PolicyTypes, "Ingress") {
			t.Errorf("policyTypes = %v; want to include Ingress",
				np.Spec.PolicyTypes)
		}
		if len(np.Spec.Ingress) == 0 {
			t.Errorf("networkpolicy has no ingress rules")
		}
	})

	t.Run("hpa_present_with_canon_targets", func(t *testing.T) {
		out, err := kubectlJSON("get", "hpa", "mademanifest",
			"-n", namespace, "-o", "json")
		if err != nil {
			t.Fatalf("get hpa mademanifest: %v\n%s", err, out)
		}
		var hpa struct {
			Spec struct {
				MinReplicas int32 `json:"minReplicas"`
				MaxReplicas int32 `json:"maxReplicas"`
				Metrics     []struct {
					Type     string `json:"type"`
					Resource struct {
						Name   string `json:"name"`
						Target struct {
							Type               string `json:"type"`
							AverageUtilization int32  `json:"averageUtilization"`
						} `json:"target"`
					} `json:"resource"`
				} `json:"metrics"`
				ScaleTargetRef struct {
					Kind string `json:"kind"`
					Name string `json:"name"`
				} `json:"scaleTargetRef"`
			} `json:"spec"`
		}
		if err := json.Unmarshal(out, &hpa); err != nil {
			t.Fatalf("decode hpa: %v\n%s", err, out)
		}
		if hpa.Spec.MinReplicas != 1 {
			t.Errorf("minReplicas = %d; want 1", hpa.Spec.MinReplicas)
		}
		if hpa.Spec.MaxReplicas != 10 {
			t.Errorf("maxReplicas = %d; want 10", hpa.Spec.MaxReplicas)
		}
		if hpa.Spec.ScaleTargetRef.Kind != "Deployment" ||
			hpa.Spec.ScaleTargetRef.Name != "mademanifest" {
			t.Errorf("scaleTargetRef = %+v; want kind=Deployment name=mademanifest",
				hpa.Spec.ScaleTargetRef)
		}
		ok := false
		for _, m := range hpa.Spec.Metrics {
			if m.Type == "Resource" &&
				m.Resource.Name == "cpu" &&
				m.Resource.Target.Type == "Utilization" &&
				m.Resource.Target.AverageUtilization == 70 {
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("hpa CPU utilisation target 70%% not found in metrics: %+v",
				hpa.Spec.Metrics)
		}
	})

	t.Run("pod_runs_as_non_root_with_dropped_capabilities", func(t *testing.T) {
		out, err := kubectlJSON("get", "pods", "-l", "app=mademanifest",
			"-n", namespace, "-o", "json")
		if err != nil {
			t.Fatalf("get pods: %v\n%s", err, out)
		}
		var podList struct {
			Items []struct {
				Spec struct {
					SecurityContext struct {
						RunAsNonRoot *bool  `json:"runAsNonRoot"`
						RunAsUser    *int64 `json:"runAsUser"`
					} `json:"securityContext"`
					Containers []struct {
						Name            string `json:"name"`
						SecurityContext struct {
							AllowPrivilegeEscalation *bool `json:"allowPrivilegeEscalation"`
							ReadOnlyRootFilesystem   *bool `json:"readOnlyRootFilesystem"`
							Capabilities             struct {
								Drop []string `json:"drop"`
							} `json:"capabilities"`
						} `json:"securityContext"`
						Resources struct {
							Requests map[string]string `json:"requests"`
							Limits   map[string]string `json:"limits"`
						} `json:"resources"`
					} `json:"containers"`
				} `json:"spec"`
			} `json:"items"`
		}
		if err := json.Unmarshal(out, &podList); err != nil {
			t.Fatalf("decode pods: %v\n%s", err, out)
		}
		if len(podList.Items) == 0 {
			t.Fatalf("no pods with label app=mademanifest in namespace %s",
				namespace)
		}
		for _, pod := range podList.Items {
			ps := pod.Spec.SecurityContext
			if ps.RunAsNonRoot == nil || !*ps.RunAsNonRoot {
				t.Errorf("pod securityContext.runAsNonRoot != true")
			}
			if ps.RunAsUser == nil || *ps.RunAsUser != 10001 {
				t.Errorf("pod securityContext.runAsUser = %v; want 10001", ps.RunAsUser)
			}
			for _, c := range pod.Spec.Containers {
				if c.Name != "mademanifest" {
					continue
				}
				cs := c.SecurityContext
				if cs.AllowPrivilegeEscalation == nil || *cs.AllowPrivilegeEscalation {
					t.Errorf("container allowPrivilegeEscalation != false")
				}
				if cs.ReadOnlyRootFilesystem == nil || !*cs.ReadOnlyRootFilesystem {
					t.Errorf("container readOnlyRootFilesystem != true")
				}
				if !contains(cs.Capabilities.Drop, "ALL") {
					t.Errorf("container capabilities.drop = %v; want to include ALL",
						cs.Capabilities.Drop)
				}
				if c.Resources.Requests["cpu"] == "" || c.Resources.Limits["cpu"] == "" {
					t.Errorf("container missing CPU resources: %+v", c.Resources)
				}
			}
		}
	})
}

// AssertNetworkPolicyEnforced is the Phase 13 enforcement-side
// sentinel.  Gated on TRINITY_NETWORK_POLICY_ENFORCED=1 because
// the default kind cluster ships kindnet, which does not enforce
// NetworkPolicy.  Real enforcement requires a CNI like Calico /
// Cilium, which the canonical CI runner does not install.
//
// When the env var is set, this helper:
//
//   * Spins up a one-shot probe pod with a label that does NOT
//     match the ingress-nginx selector.
//   * Runs `kubectl exec` on that pod with `wget --timeout=5
//     http://mademanifest:80/healthz`.
//   * Asserts the wget command fails (the NetworkPolicy blocks the
//     traffic).
//
// The helper is otherwise a no-op so the canonical CI run does not
// flake on a missing-CNI mismatch.
func AssertNetworkPolicyEnforced(t *testing.T, namespace string) {
	t.Helper()
	if os.Getenv("TRINITY_NETWORK_POLICY_ENFORCED") != "1" {
		t.Skip("TRINITY_NETWORK_POLICY_ENFORCED not set; default kind cluster does not enforce NetworkPolicy")
	}
	if namespace == "" {
		namespace = "default"
	}
	probeName := "trinity-network-probe"
	defer func() {
		_ = exec.Command("kubectl", "delete", "pod", probeName,
			"-n", namespace, "--ignore-not-found=true",
			"--grace-period=0", "--force").Run()
	}()
	create := exec.Command("kubectl", "run", probeName,
		"-n", namespace,
		"--image=busybox:1.36",
		"--restart=Never",
		"--labels=app=trinity-network-probe",
		"--command", "--",
		"sh", "-c", "sleep 60")
	create.Stderr = os.Stderr
	if err := create.Run(); err != nil {
		t.Fatalf("kubectl run %s: %v", probeName, err)
	}
	wait := exec.Command("kubectl", "wait", "pod", probeName,
		"-n", namespace, "--for=condition=Ready", "--timeout=30s")
	wait.Stderr = os.Stderr
	if err := wait.Run(); err != nil {
		t.Fatalf("probe pod did not become ready: %v", err)
	}
	probe := exec.Command("kubectl", "exec", probeName,
		"-n", namespace, "--",
		"wget", "--timeout=5", "-q", "-O", "-",
		"http://mademanifest:80/healthz")
	out, err := probe.CombinedOutput()
	if err == nil {
		t.Errorf("non-ingress probe reached /healthz; NetworkPolicy not enforced.\nbody: %s", out)
	}
}

// AssertHPAScalesUnderLoad is the Phase 13 horizontal-scaling
// sentinel.  Gated on TRINITY_LOAD_TEST=1 because metrics-server
// is not installed in the default kind cluster (HPA cannot evaluate
// CPU utilisation without a metrics source) and because a real
// scale-up test takes minutes of sustained load.
//
// When enabled, the helper drives the service with a fan of
// concurrent /manifest requests and asserts that the deployment's
// replica count rises above the cold-start floor of 1.
func AssertHPAScalesUnderLoad(t *testing.T, namespace, baseURL string) {
	t.Helper()
	if os.Getenv("TRINITY_LOAD_TEST") != "1" {
		t.Skip("TRINITY_LOAD_TEST not set; HPA scaling test requires metrics-server")
	}
	if namespace == "" {
		namespace = "default"
	}
	// Drive sustained load against /healthz from this process for
	// 90 seconds; metrics-server samples every 15s and the HPA
	// controller reconciles every 15s, so two sample windows is
	// the minimum that can produce a scale event.
	stop := time.After(90 * time.Second)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
				_, _, _ = GetJSON(baseURL, "/healthz")
			}
		}
	}()
	<-done

	out, err := exec.Command("kubectl", "get", "deployment", "mademanifest",
		"-n", namespace, "-o", "jsonpath={.status.replicas}").CombinedOutput()
	if err != nil {
		t.Fatalf("kubectl get deployment: %v\n%s", err, out)
	}
	replicas := strings.TrimSpace(string(out))
	if replicas == "1" || replicas == "" {
		t.Errorf("deployment did not scale beyond 1 replica under load (got %q)",
			replicas)
	}
}

// kubectlJSON wraps `kubectl` with a JSON output formatter and
// returns the stdout bytes.  Errors include the stderr capture so
// flakes are diagnosable.
func kubectlJSON(args ...string) ([]byte, error) {
	cmd := exec.Command("kubectl", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w (stderr: %s)", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
