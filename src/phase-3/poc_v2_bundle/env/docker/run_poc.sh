#!/usr/bin/env bash
set -euo pipefail

IMAGE_NAME=${IMAGE_NAME:-mademanifest-poc-v2}
IMAGE_WORK_DIR=/workspace/mademanifest/src/phase-3/poc_v2_bundle

exec docker run --rm -t \
  -v "$(pwd)":${IMAGE_WORK_DIR} \
  -w /workspace/mademanifest/src/phase-3/poc_v2_bundle \
  "$IMAGE_NAME" \
  bash -lc "make run diff"
