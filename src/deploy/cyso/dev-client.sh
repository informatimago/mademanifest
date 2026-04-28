#!/usr/bin/env bash
# dev-client.sh — open src/scripts/client.html against the engine
# running on the Cyso cluster.
#
# The production deploy keeps CORS off (it is gratuitous attack
# surface in front of a real ingress) and exposes the engine only
# as a ClusterIP service.  This helper bridges both gaps for
# interactive browser testing:
#
#   1. Sets TRINITY_DEV_CORS=1 on the deployment, waits for the
#      pod rollout to finish.
#   2. Starts `kubectl port-forward svc/mademanifest 8080:80` so
#      the engine is reachable at http://127.0.0.1:8080.
#   3. Opens src/scripts/client.html in the system default browser
#      (macOS: open ; Linux: xdg-open).
#   4. On Ctrl-C: tears down port-forward and disables CORS again
#      (rollout to the production-clean configuration).
#
# Usage:
#   KUBECONFIG=~/.kube/cyso-trinity.yaml ./dev-client.sh
#       [--no-cors] [--no-open] [--port=8080]
#
# Flags:
#   --no-cors   skip the CORS toggle (useful when you have already
#               turned it on, or when serving client.html from the
#               same origin as the engine — e.g. a future ingress).
#   --no-open   skip the browser launch (just print the URL).
#   --port=N    forward to local port N (default 8080).

set -euo pipefail

cors=1; open_browser=1; port=8080
for arg in "$@"; do
    case "$arg" in
        --no-cors)   cors=0 ;;
        --no-open)   open_browser=0 ;;
        --port=*)    port="${arg#--port=}" ;;
        -h|--help)   sed -n '2,/^# Flags:/,/^$/p' "$0" | sed 's/^# \?//'; exit 0 ;;
        *) echo "unknown flag: $arg" >&2; exit 1 ;;
    esac
done

die()  { echo "ERROR: $*" >&2; exit 1; }
note() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }

[[ -n "${KUBECONFIG:-}" ]] || die "KUBECONFIG env var must point at the Cyso kubeconfig"
[[ -r "$KUBECONFIG" ]]     || die "KUBECONFIG=$KUBECONFIG is not readable"
command -v kubectl >/dev/null 2>&1 || die "kubectl not on PATH"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
CLIENT_HTML="$SRC_DIR/scripts/client.html"
[[ -f "$CLIENT_HTML" ]] || die "client.html not found at $CLIENT_HTML"

namespace="${NAMESPACE:-mademanifest}"
KUBECTL=(kubectl --kubeconfig "$KUBECONFIG" -n "$namespace")

# ---- 1. Toggle CORS on -----------------------------------------------------

cors_on() {
    note "Enabling TRINITY_DEV_CORS=1 on deployment/mademanifest"
    "${KUBECTL[@]}" set env deployment/mademanifest TRINITY_DEV_CORS=1 >/dev/null
    "${KUBECTL[@]}" rollout status deployment/mademanifest --timeout=2m >/dev/null
}
cors_off() {
    note "Disabling TRINITY_DEV_CORS on deployment/mademanifest"
    "${KUBECTL[@]}" set env deployment/mademanifest TRINITY_DEV_CORS- >/dev/null 2>&1 || true
    "${KUBECTL[@]}" rollout status deployment/mademanifest --timeout=2m >/dev/null 2>&1 || true
}

# ---- 2. Port-forward + cleanup ---------------------------------------------

pf_pid=""
cleanup() {
    [[ -n "$pf_pid" ]] && kill "$pf_pid" 2>/dev/null || true
    [[ "$cors" -eq 1 ]] && cors_off
}
trap cleanup EXIT INT TERM

[[ "$cors" -eq 1 ]] && cors_on

note "Port-forwarding svc/mademanifest 80 -> 127.0.0.1:$port"
"${KUBECTL[@]}" port-forward svc/mademanifest "$port:80" >/dev/null 2>&1 &
pf_pid=$!

# Wait for the forward to accept connections.
deadline=$((SECONDS + 30))
until curl -fsS "http://127.0.0.1:$port/healthz" >/dev/null 2>&1; do
    if [[ $SECONDS -ge $deadline ]]; then
        die "port-forward never became reachable on 127.0.0.1:$port"
    fi
    sleep 1
done
note "Engine reachable at http://127.0.0.1:$port (preflight OK)"

# ---- 3. Open client.html ---------------------------------------------------

if [[ "$open_browser" -eq 1 ]]; then
    case "$(uname -s)" in
        Darwin)  open "$CLIENT_HTML" ;;
        Linux)   xdg-open "$CLIENT_HTML" >/dev/null 2>&1 || \
                 note "couldn't auto-open; visit file://$CLIENT_HTML manually" ;;
        *)       note "open client.html manually: file://$CLIENT_HTML" ;;
    esac
else
    note "Skipping browser open (--no-open).  URL: file://$CLIENT_HTML"
fi

note "Press Ctrl-C to tear down port-forward and disable CORS."
wait "$pf_pid"
