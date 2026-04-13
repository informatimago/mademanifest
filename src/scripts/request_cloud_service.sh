#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
SERVICE_URL=${1:-${SERVICE_URL:-http://localhost:8080/manifest}}
OUTPUT_FILE=${2:-${OUTPUT_FILE:-$ROOT_DIR/out/cloud-service-output.json}}
INPUT_FILE=${INPUT_FILE:-$ROOT_DIR/golden/GOLDEN_TEST_CASE_V1.json}

mkdir -p "$(dirname "$OUTPUT_FILE")"

curl --fail --silent --show-error \
  -X POST \
  -H 'Content-Type: application/json' \
  --data-binary @"$INPUT_FILE" \
  "$SERVICE_URL" \
  -o "$OUTPUT_FILE"

printf 'Saved response to %s\n' "$OUTPUT_FILE"
