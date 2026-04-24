#!/usr/bin/env bash
#
# Tear down the kind cluster created by k8s-local-up.sh.  Usually you
# don't need this – k8s-local-up.sh deletes the cluster on Ctrl-C unless
# --keep-cluster was passed.  Use this script when you did pass
# --keep-cluster, or when a previous run crashed without cleaning up.

set -euo pipefail

CLUSTER=${CLUSTER:-trinity-dev}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --cluster) CLUSTER="$2"; shift 2 ;;
    -h|--help)
      echo "Usage: $0 [--cluster NAME]"
      echo "Default cluster: $CLUSTER (override with \$CLUSTER or --cluster)"
      exit 0 ;;
    *) echo "unknown arg: $1" >&2; exit 1 ;;
  esac
done

if ! kind get clusters 2>/dev/null | grep -qx "$CLUSTER"; then
  echo "no kind cluster named '$CLUSTER' – nothing to do"
  exit 0
fi

echo "==> Deleting kind cluster $CLUSTER"
kind delete cluster --name "$CLUSTER"
