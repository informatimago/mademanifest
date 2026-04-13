#!/usr/bin/env bash
set -euo pipefail

IMAGE_NAME=${IMAGE_NAME:-mademanifest-poc-v2}
IMAGE_WORK_DIR=/workspace/mademanifest/src

exec docker run --rm -t \
  -v "$(pwd)":${IMAGE_WORK_DIR} \
  -w /workspace/mademanifest/src \
  "$IMAGE_NAME" \
  bash -lc "make run diff"
