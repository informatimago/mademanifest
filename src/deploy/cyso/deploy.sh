#!/usr/bin/env bash
# deploy.sh — build, push, and deploy mademanifest-engine to the
# Cyso Managed Kubernetes cluster (project Trinity / cluster
# trinityengine / region ams2).
#
# Designed to be run repeatedly: every invocation performs a fresh
# build, pushes a uniquely tagged image, and applies the kustomize
# overlay so kubectl rolls out the new pod template.
#
# Usage:
#   KUBECONFIG=~/.kube/cyso-trinity.yaml \
#   REGISTRY=docker.io/<your-account> \
#     ./deploy.sh [--no-build] [--no-push] [--no-apply] [--no-smoke]
#
# Required environment:
#   KUBECONFIG  Path to the Cyso kubeconfig.  Cyso-issued kubeconfigs
#               are limited to one day of validity; if the script
#               fails on the first kubectl call with an auth error,
#               request a fresh kubeconfig from the cluster owner
#               (or set up a permanent service-account token).
#   REGISTRY    Image registry the cluster can pull from
#               (e.g. docker.io/madeMANIFEST, ghcr.io/<org>).
#               The script pushes <REGISTRY>/mademanifest-engine:<tag>.
#
# Optional environment:
#   IMAGE_NAME  Override the image name component (default:
#               mademanifest-engine).
#   TAG         Override the computed tag (default: engine version
#               from canon.EngineVersion + git short SHA + dirty
#               marker).
#   NAMESPACE   Override the deploy namespace (default: mademanifest).
#   ROLLOUT_TIMEOUT  kubectl rollout status timeout (default: 5m).
#   SMOKE_TIMEOUT    Port-forward smoke deadline (default: 60s).
#
# Flags (all default-on):
#   --no-build   skip docker build (use a previously-built local
#                image; tag must already exist locally as
#                mademanifest-engine:<TAG>).
#   --no-push    skip docker push (useful when pushing manually or
#                when the cluster pulls from a local registry mirror).
#   --no-apply   skip kubectl apply (build/push only — useful for
#                pre-warming a registry).
#   --no-smoke   skip the post-rollout smoke test.
#   -h, --help   show this help text and exit.
#
# Exit codes:
#   0   success (build + push + apply + smoke all green)
#   1   precondition failed (missing tools / env / kubeconfig)
#   2   build failed
#   3   push failed
#   4   apply / rollout failed
#   5   smoke test failed

set -euo pipefail

usage() { sed -n '2,/^# Exit codes:/p' "$0" | sed 's/^# \?//'; }

build=1; push=1; apply=1; smoke=1
for arg in "$@"; do
    case "$arg" in
        --no-build)  build=0 ;;
        --no-push)   push=0 ;;
        --no-apply)  apply=0 ;;
        --no-smoke)  smoke=0 ;;
        -h|--help)   usage; exit 0 ;;
        *) echo "unknown flag: $arg" >&2; usage; exit 1 ;;
    esac
done

# ---- 1. Preconditions ------------------------------------------------------

die()   { echo "ERROR: $*" >&2; exit "${2:-1}"; }
note()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }

note "Checking preconditions"

for bin in docker kubectl git awk sha256sum; do
    command -v "$bin" >/dev/null 2>&1 || \
        command -v "${bin/sha256sum/shasum}" >/dev/null 2>&1 || \
        die "missing required binary: $bin"
done

[[ -n "${KUBECONFIG:-}" ]] || die "KUBECONFIG env var is not set; point it at the Cyso kubeconfig"
[[ -r "$KUBECONFIG" ]]     || die "KUBECONFIG=$KUBECONFIG is not readable"
[[ -n "${REGISTRY:-}" ]]   || die "REGISTRY env var is not set (e.g. docker.io/<account>)"

# Resolve repo paths relative to the script location so the script
# works no matter where it is invoked from.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOY_DIR="$SCRIPT_DIR"
SRC_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
REPO_ROOT="$(cd "$SRC_DIR/.." && pwd)"
DOCKERFILE="$SRC_DIR/Dockerfile"

[[ -f "$DOCKERFILE" ]] || die "Dockerfile not found at $DOCKERFILE"

# ---- 2. Compute tag --------------------------------------------------------

if [[ -n "${TAG:-}" ]]; then
    tag="$TAG"
else
    # Engine version comes from the canonical compiled-in constant
    # (canon.EngineVersion).  Cross-check against the git tree state
    # so each push is uniquely identifiable.
    eng_ver=$(grep -E '^\s*EngineVersion\s*=' "$SRC_DIR/mademanifest-engine/pkg/canon/sources.go" \
              | head -1 | awk -F'"' '{print $2}')
    [[ -n "$eng_ver" ]] || die "could not parse EngineVersion from canon/sources.go"
    git_sha=$(git -C "$REPO_ROOT" rev-parse --short=10 HEAD)
    dirty=""
    if ! git -C "$REPO_ROOT" diff --quiet HEAD -- "$SRC_DIR" 2>/dev/null; then
        dirty="-dirty"
    fi
    tag="${eng_ver}-${git_sha}${dirty}"
fi

image_name="${IMAGE_NAME:-mademanifest-engine}"
local_ref="${image_name}:${tag}"
remote_ref="${REGISTRY}/${image_name}:${tag}"
namespace="${NAMESPACE:-mademanifest}"
rollout_timeout="${ROLLOUT_TIMEOUT:-5m}"
smoke_timeout="${SMOKE_TIMEOUT:-60s}"

note "Plan:"
printf '    %-20s %s\n' \
    "engine version:" "$eng_ver" \
    "tag:"            "$tag" \
    "local image:"    "$local_ref" \
    "remote image:"   "$remote_ref" \
    "namespace:"      "$namespace" \
    "kubeconfig:"     "$KUBECONFIG"

# ---- 3. Build --------------------------------------------------------------

if [[ "$build" -eq 1 ]]; then
    note "Building image $local_ref"
    docker build \
        -f "$DOCKERFILE" \
        -t "$local_ref" \
        -t "${image_name}:latest" \
        "$SRC_DIR" \
        || die "docker build failed" 2
else
    note "Skipping build (--no-build)"
fi

# ---- 4. Push ---------------------------------------------------------------

if [[ "$push" -eq 1 ]]; then
    note "Tagging $local_ref as $remote_ref"
    docker tag "$local_ref" "$remote_ref" || die "docker tag failed" 3
    note "Pushing $remote_ref"
    docker push "$remote_ref" || die "docker push failed (registry login? network?)" 3
else
    note "Skipping push (--no-push)"
fi

# ---- 5. kubectl apply ------------------------------------------------------

KUBECTL=(kubectl --kubeconfig "$KUBECONFIG")

if [[ "$apply" -eq 1 ]]; then
    note "Verifying cluster reachability"
    "${KUBECTL[@]}" version --output=yaml >/dev/null 2>&1 \
        || die "kubectl cannot reach the cluster — check kubeconfig validity (Cyso tokens expire after 24 h)" 4

    # The kustomization.yaml under cyso/ contains placeholder strings
    # for the image; we render the manifests via a temp directory so
    # the committed file stays untouched and `git diff` is clean
    # after the script runs.
    work=$(mktemp -d)
    trap 'rm -rf "$work"' EXIT
    cp -R "$DEPLOY_DIR/." "$work/"
    cp -R "$SRC_DIR/deploy/kubernetes" "$work/kubernetes"
    # Rewrite the resources reference now that the base lives in a
    # sibling directory inside the temp tree.
    sed -i.bak 's|\.\./kubernetes|kubernetes|' "$work/kustomization.yaml"
    sed -i.bak \
        -e "s|REPLACE_BY_DEPLOY_SCRIPT/mademanifest-engine|${REGISTRY}/${image_name}|" \
        -e "s|newTag: REPLACE_BY_DEPLOY_SCRIPT|newTag: ${tag}|" \
        "$work/kustomization.yaml"
    rm -f "$work/kustomization.yaml.bak"

    note "Rendered kustomize:"
    "${KUBECTL[@]}" kustomize "$work" | sed 's/^/    /' | head -40
    echo "    ..."

    note "Applying to namespace $namespace"
    "${KUBECTL[@]}" apply -k "$work" || die "kubectl apply failed" 4

    note "Waiting for rollout (timeout $rollout_timeout)"
    "${KUBECTL[@]}" -n "$namespace" rollout status deployment/mademanifest \
        --timeout="$rollout_timeout" \
        || die "rollout did not complete in $rollout_timeout" 4
else
    note "Skipping kubectl apply (--no-apply)"
fi

# ---- 6. Smoke test ---------------------------------------------------------

if [[ "$smoke" -eq 1 && "$apply" -eq 1 ]]; then
    note "Running smoke test via port-forward"
    pf_log=$(mktemp)
    "${KUBECTL[@]}" -n "$namespace" port-forward svc/mademanifest 18080:80 \
        >"$pf_log" 2>&1 &
    pf_pid=$!
    trap 'kill "$pf_pid" 2>/dev/null || true; rm -rf "$work" "$pf_log"' EXIT

    # Wait until the port-forward is ready (or the deadline elapses).
    deadline=$((SECONDS + ${smoke_timeout%s}))
    until curl -fsS http://127.0.0.1:18080/healthz >/dev/null 2>&1; do
        if [[ $SECONDS -ge $deadline ]]; then
            echo "port-forward log:" >&2
            sed 's/^/    /' "$pf_log" >&2
            die "smoke /healthz did not respond within $smoke_timeout" 5
        fi
        sleep 1
    done

    healthz=$(curl -fsS http://127.0.0.1:18080/healthz)
    version=$(curl -fsS http://127.0.0.1:18080/version)
    note "Smoke responses:"
    printf '    /healthz  %s\n' "$healthz"
    printf '    /version  %s\n' "$version"

    # Cross-check the deployed engine version against the local tag
    # so a stale image cache or a missed push surfaces immediately.
    deployed_engine=$(printf '%s' "$version" | python3 -c 'import json,sys; print(json.load(sys.stdin)["engine_version"])')
    [[ "$deployed_engine" = "$eng_ver" ]] \
        || die "smoke engine_version=$deployed_engine != expected $eng_ver (stale image?)" 5

    kill "$pf_pid" 2>/dev/null || true
    wait "$pf_pid" 2>/dev/null || true
elif [[ "$smoke" -eq 0 ]]; then
    note "Skipping smoke test (--no-smoke)"
fi

note "Done."
