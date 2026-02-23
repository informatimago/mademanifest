#!/usr/bin/env bash
set -euo pipefail

IMAGE_NAME=${IMAGE_NAME:-mademanifest-poc-v2}

exec docker run --rm -t \
  -v "$(pwd)":/workspace/poc_v2_bundle \
  -w /workspace/poc_v2_bundle \
  "$IMAGE_NAME" \
  bash -lc "make run diff"
