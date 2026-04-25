#!/usr/bin/env bash
#
# Bring up a local Kubernetes dev environment running the mademanifest
# HTTP service and open a port-forward to it.  Runs in the foreground;
# Ctrl-C tears everything down (the kind cluster is deleted unless it
# pre-existed or --keep-cluster was passed).
#
# Usage:
#   src/scripts/k8s-local-up.sh [--keep-cluster] [--cluster NAME] [--port PORT]
#
# Environment overrides:
#   CLUSTER   kind cluster name (default: trinity-dev)
#   IMAGE     local image tag    (default: mademanifest:dev)
#   NAMESPACE target namespace   (default: default)
#   PORT      host port for forward (default: 8080)
#
# On success the service is reachable at http://127.0.0.1:$PORT and the
# script blocks on the port-forward.  Run src/scripts/k8s-local-test.sh
# in another terminal to exercise it with curl.

set -euo pipefail

CLUSTER=${CLUSTER:-trinity-dev}
IMAGE=${IMAGE:-mademanifest:dev}
NAMESPACE=${NAMESPACE:-default}
PORT=${PORT:-8080}
KEEP_CLUSTER=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --keep-cluster) KEEP_CLUSTER=true; shift ;;
    --port)         PORT="$2"; shift 2 ;;
    --cluster)      CLUSTER="$2"; shift 2 ;;
    --image)        IMAGE="$2"; shift 2 ;;
    --namespace)    NAMESPACE="$2"; shift 2 ;;
    -h|--help)
      sed -n '2,22p' "$0" | sed 's/^# \{0,1\}//'
      exit 0 ;;
    *) echo "unknown arg: $1" >&2; exit 1 ;;
  esac
done

for bin in docker kind kubectl; do
  if ! command -v "$bin" >/dev/null 2>&1; then
    echo "error: '$bin' not found on PATH" >&2
    exit 1
  fi
done

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

echo "==> Building Docker image $IMAGE"
docker build -f "$ROOT_DIR/Dockerfile" -t "$IMAGE" "$ROOT_DIR"

CLUSTER_CREATED=false
if kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; then
  echo "==> Reusing existing kind cluster $CLUSTER"
else
  echo "==> Creating kind cluster $CLUSTER"
  kind create cluster --name "$CLUSTER" --wait 60s
  CLUSTER_CREATED=true
fi

echo "==> Loading $IMAGE into cluster"
kind load docker-image "$IMAGE" --name "$CLUSTER"

# Render the base kustomization with an in-line image override rather
# than materialising a temp overlay directory.  `sed` rewrites the
# placeholder image reference emitted by the base Deployment before
# kubectl applies it.
rendered=$(kubectl kustomize "$ROOT_DIR/deploy/kubernetes" \
  | sed "s|registry.example.com/mademanifest:latest|$IMAGE|g")

echo "==> Applying manifests to namespace $NAMESPACE"
printf '%s\n' "$rendered" | kubectl apply --namespace "$NAMESPACE" -f -

echo "==> Waiting for rollout of deployment/mademanifest"
kubectl rollout status deployment/mademanifest --namespace "$NAMESPACE" --timeout=120s

cleanup_ran=false
cleanup() {
  # Trap fires on both INT and EXIT; guard so we only do the work once.
  [[ "$cleanup_ran" == true ]] && return
  cleanup_ran=true
  echo
  echo "==> Tearing down"
  printf '%s\n' "$rendered" \
    | kubectl delete --namespace "$NAMESPACE" -f - --ignore-not-found=true --wait=false \
    || true
  if [[ "$KEEP_CLUSTER" != true && "$CLUSTER_CREATED" == true ]]; then
    kind delete cluster --name "$CLUSTER" || true
  elif [[ "$KEEP_CLUSTER" == true ]]; then
    echo "    keeping cluster $CLUSTER (use: kind delete cluster --name $CLUSTER)"
  else
    echo "    cluster $CLUSTER was pre-existing; leaving it in place"
  fi
}
trap cleanup EXIT INT TERM

echo
echo "==> Port-forwarding svc/mademanifest to 127.0.0.1:$PORT"
echo "    Ctrl-C to stop.  Try it from another terminal with:"
echo "      src/scripts/k8s-local-test.sh"
echo "    Or open the browser test client (form + result tables):"
echo "      open src/scripts/client.html"
echo

# NOT `exec`: the cleanup trap belongs to this shell, so we must keep
# the shell in the process tree.  Using exec would replace it with
# kubectl and strip the trap, so Ctrl-C would leave the cluster behind.
kubectl port-forward svc/mademanifest "$PORT:80" \
  --namespace "$NAMESPACE" --address 127.0.0.1
