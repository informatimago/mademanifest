#!/usr/bin/env bash
#
# Drive the locally port-forwarded mademanifest service with curl.
#
#   Terminal 1:  src/scripts/k8s-local-up.sh        # leave running
#   Terminal 2:  src/scripts/k8s-local-test.sh      # this script
#
# Exercises:
#   GET  /healthz   – expects 200 {"status":"ok"}
#   POST /manifest  – expects 200 with the JSON calculation envelope,
#                     payload taken from golden/GOLDEN_TEST_CASE_V1.json
#
# Overrides:
#   URL       service base URL (default: http://127.0.0.1:8080)
#   FIXTURE   path to the POST payload
#             (default: <repo>/src/golden/GOLDEN_TEST_CASE_V1.json)
#   OUTPUT    where to save the /manifest response body
#             (default: <tempfile>; printed on exit)

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

URL=${URL:-http://127.0.0.1:8080}
FIXTURE=${FIXTURE:-$ROOT_DIR/golden/GOLDEN_TEST_CASE_V1.json}
OUTPUT=${OUTPUT:-$(mktemp -t mademanifest-response-XXXXXX.json)}

if [[ ! -f "$FIXTURE" ]]; then
  echo "fixture not found: $FIXTURE" >&2
  exit 2
fi

fail=0

print_result() {
  local label=$1 status=$2 body=$3
  printf '    status: %s\n' "$status"
  printf '    body:   '
  if [[ ${#body} -gt 400 ]]; then
    printf '%s…\n' "${body:0:400}"
  else
    printf '%s\n' "$body"
  fi
  if [[ "$status" != 2?? ]]; then
    printf '    FAIL (%s)\n' "$label" >&2
    fail=1
  fi
}

echo "==> GET $URL/healthz"
resp=$(curl --silent --show-error --max-time 5 \
  --write-out '\nHTTPSTATUS:%{http_code}' \
  "$URL/healthz" || printf '\nHTTPSTATUS:000')
status=${resp##*HTTPSTATUS:}
body=${resp%$'\n'HTTPSTATUS:*}
print_result healthz "$status" "$body"

echo
echo "==> POST $URL/manifest"
echo "    payload: $FIXTURE"
resp=$(curl --silent --show-error --max-time 30 \
  -X POST \
  -H 'Content-Type: application/json' \
  --data-binary @"$FIXTURE" \
  --write-out '\nHTTPSTATUS:%{http_code}' \
  "$URL/manifest" || printf '\nHTTPSTATUS:000')
status=${resp##*HTTPSTATUS:}
body=${resp%$'\n'HTTPSTATUS:*}
printf '%s' "$body" > "$OUTPUT"
print_result manifest "$status" "$body"
echo "    full response saved to: $OUTPUT"

if [[ $fail -ne 0 ]]; then
  echo
  echo "one or more probes failed" >&2
  exit 1
fi

echo
echo "all probes OK"
