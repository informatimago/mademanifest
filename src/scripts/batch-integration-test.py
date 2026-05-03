#!/usr/bin/env python3
# batch-integration-test.py — drive every fixture under
# src/golden/trinity/<category>/<name>/ through a running mademanifest
# engine, compare each response against the matching expected.json,
# and emit a pass/fail report plus useful diffs on failure.
#
# Designed to be run identically against:
#   * a local engine subprocess         (--mode=local)
#   * a Cyso (or any other) base URL    (--mode=remote --url=...)
#   * the existing kubectl port-forward (--mode=portforward [--namespace=...])
#
# The local mode is convenient for development; the remote / port-
# forward mode is what Jaimie's review milestone calls for ("cluster
# batch tests runnable").
#
# Comparison rules mirror the in-tree pkg/golden runner exactly:
#
#   * success cases use semantic JSON equality, ignoring the
#     "metadata" block (which depends on the running build's
#     EngineVersion);
#   * error cases compare the error.error_type field only and require
#     a non-empty error.message plus a canonical envelope shape
#     (status: "error", metadata block present); A4 / D23.
#
# Output:
#   stdout         per-fixture PASS / FAIL line, summary at the end.
#   --report=PATH  optional JSON report covering /version metadata,
#                  per-fixture status, per-failure diffs, totals.
#
# Exit status:
#   0  all fixtures matched
#   1  one or more fixtures failed (or precondition failed)

from __future__ import annotations

import argparse
import json
import os
import shlex
import shutil
import signal
import subprocess
import sys
import time
import urllib.error
import urllib.request
from dataclasses import dataclass, field, asdict
from pathlib import Path
from typing import Any, Optional


# ---- repo layout ---------------------------------------------------

SCRIPT_DIR = Path(__file__).resolve().parent
SRC_DIR = SCRIPT_DIR.parent
REPO_ROOT = SRC_DIR.parent
GOLDEN_DIR = SRC_DIR / "golden" / "trinity"

# Categories the canon enumerates and the minimum fixture counts
# (Document 10 §"Required test categories", mirrored in pkg/golden).
CATEGORIES = [
    ("valid_baseline",      3),
    ("valid_edge",          5),
    ("invalid_input",       5),
    ("incomplete_input",    5),
    ("unsupported_input",   2),
    ("regression_sentinel", 3),
]
SUCCESS_CATEGORIES = {"valid_baseline", "valid_edge", "regression_sentinel"}


# ---- HTTP helpers --------------------------------------------------


def http_get_json(url: str, timeout: float = 10.0) -> Any:
    req = urllib.request.Request(url, method="GET")
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.loads(resp.read().decode("utf-8"))


def http_post_manifest(base_url: str, body: bytes,
                       timeout: float = 30.0) -> "tuple[int, dict]":
    req = urllib.request.Request(
        base_url.rstrip("/") + "/manifest",
        data=body,
        method="POST",
        headers={"Content-Type": "application/json"},
    )
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status, json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        # Trinity error responses are JSON envelopes with non-2xx
        # status codes — read the body as JSON regardless of status.
        try:
            return e.code, json.loads(e.read().decode("utf-8"))
        except Exception:
            return e.code, {"_decode_error": str(e)}


def wait_for_healthz(base_url: str, deadline_seconds: float = 30.0) -> bool:
    start = time.monotonic()
    while time.monotonic() - start < deadline_seconds:
        try:
            r = http_get_json(base_url.rstrip("/") + "/healthz", timeout=2.0)
            if r.get("status") == "ok":
                return True
        except Exception:
            pass
        time.sleep(0.5)
    return False


# ---- semantic JSON equality (mirrors pkg/golden.CompareSuccess) ----


def jpath_strip_metadata(d: Any) -> Any:
    """Return a deep copy of d with the top-level "metadata" key
    removed, so the comparison ignores build-specific version pins."""
    if isinstance(d, dict):
        return {k: v for k, v in d.items() if k != "metadata"}
    return d


def semantic_diff(got: Any, want: Any, path: str = "") -> list:
    """Return a list of human-readable drift descriptions, empty when
    got == want under canonical comparison rules."""
    if isinstance(want, dict) and isinstance(got, dict):
        diffs = []
        for k in sorted(set(want.keys()) | set(got.keys())):
            sub = f"{path}.{k}" if path else k
            if k not in got:
                diffs.append(f"{sub}: missing  (want {want[k]!r})")
            elif k not in want:
                diffs.append(f"{sub}: unexpected  (got {got[k]!r})")
            else:
                diffs.extend(semantic_diff(got[k], want[k], sub))
        return diffs
    if isinstance(want, list) and isinstance(got, list):
        if len(got) != len(want):
            return [f"{path}: length got {len(got)} want {len(want)}"]
        diffs = []
        for i, (g, w) in enumerate(zip(got, want)):
            diffs.extend(semantic_diff(g, w, f"{path}[{i}]"))
        return diffs
    if got != want:
        return [f"{path}: got {got!r} want {want!r}"]
    return []


# ---- engine targets ------------------------------------------------


@dataclass
class EngineTarget:
    base_url: str
    cleanup: Optional[object] = None  # callable; typed loosely for older Python


def start_local_engine(port: int) -> EngineTarget:
    bin_dir = SRC_DIR / "mademanifest-engine"
    if not (bin_dir / "go.mod").exists():
        raise SystemExit(f"engine module not found at {bin_dir}")
    env = os.environ.copy()
    env.setdefault("SE_EPHE_PATH", "/usr/local/share/swisseph")
    env["PORT"] = str(port)
    print(f"[batch] building engine in {bin_dir} ...", flush=True)
    build = subprocess.run(
        ["go", "build", "-o", str(bin_dir / "mademanifest-engine"), "./cmd/httpserver"],
        cwd=bin_dir,
        env={**env, "CGO_LDFLAGS": "-lm"},
    )
    if build.returncode != 0:
        raise SystemExit("local engine build failed")
    print(f"[batch] starting engine on :{port} ...", flush=True)
    proc = subprocess.Popen(
        [str(bin_dir / "mademanifest-engine")],
        cwd=bin_dir,
        env=env,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.STDOUT,
    )
    base_url = f"http://127.0.0.1:{port}"
    if not wait_for_healthz(base_url, deadline_seconds=30):
        proc.send_signal(signal.SIGTERM)
        raise SystemExit("local engine never became healthy")

    def stop():
        proc.send_signal(signal.SIGTERM)
        try:
            proc.wait(timeout=10)
        except subprocess.TimeoutExpired:
            proc.kill()

    return EngineTarget(base_url=base_url, cleanup=stop)


def start_portforward(namespace: str, kubeconfig, port: int) -> EngineTarget:
    if not shutil.which("kubectl"):
        raise SystemExit("kubectl not on PATH")
    cmd = ["kubectl"]
    if kubeconfig:
        cmd += ["--kubeconfig", kubeconfig]
    cmd += ["-n", namespace, "port-forward", "svc/mademanifest", f"{port}:80"]
    print(f"[batch] kubectl port-forward {' '.join(shlex.quote(c) for c in cmd)} ...", flush=True)
    proc = subprocess.Popen(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    base_url = f"http://127.0.0.1:{port}"
    if not wait_for_healthz(base_url, deadline_seconds=30):
        proc.send_signal(signal.SIGTERM)
        raise SystemExit("port-forward never became healthy")

    def stop():
        proc.send_signal(signal.SIGTERM)
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()

    return EngineTarget(base_url=base_url, cleanup=stop)


# ---- fixture discovery + comparison --------------------------------


@dataclass
class FixtureResult:
    category: str
    name: str
    passed: bool
    status_code: int = 0
    diffs: list = field(default_factory=list)
    error: str = ""


def discover_fixtures():
    found = {}
    for cat, _ in CATEGORIES:
        cat_dir = GOLDEN_DIR / cat
        if not cat_dir.is_dir():
            found[cat] = []
            continue
        found[cat] = sorted([p for p in cat_dir.iterdir() if p.is_dir()])
    return found


def evaluate_fixture(target: EngineTarget, category: str, fixture_dir: Path) -> FixtureResult:
    name = fixture_dir.name
    try:
        body = (fixture_dir / "input.json").read_bytes()
    except FileNotFoundError:
        return FixtureResult(category, name, False, error="input.json missing")
    try:
        expected = json.loads((fixture_dir / "expected.json").read_text())
    except FileNotFoundError:
        return FixtureResult(category, name, False, error="expected.json missing")
    try:
        status, got = http_post_manifest(target.base_url, body)
    except Exception as e:
        return FixtureResult(category, name, False, error=f"HTTP error: {e}")

    if category in SUCCESS_CATEGORIES:
        if expected.get("status") != "success":
            return FixtureResult(category, name, False, status_code=status,
                                 error="expected.json declares non-success "
                                       "status in a success-category fixture")
        diffs = semantic_diff(jpath_strip_metadata(got), jpath_strip_metadata(expected))
        if diffs:
            return FixtureResult(category, name, False, status_code=status, diffs=diffs)
        if status != 200:
            return FixtureResult(category, name, False, status_code=status,
                                 error=f"success path returned HTTP {status}")
        return FixtureResult(category, name, True, status_code=status)

    # Error categories: compare error_type only.
    expected_type = (expected.get("error") or {}).get("error_type", "")
    if not expected_type:
        return FixtureResult(category, name, False, status_code=status,
                             error="expected.json error.error_type missing")
    got_type = (got.get("error") or {}).get("error_type", "")
    got_message = (got.get("error") or {}).get("message", "")
    if got.get("status") != "error":
        return FixtureResult(category, name, False, status_code=status,
                             error=f"envelope status got {got.get('status')!r} want 'error'")
    if got_type != expected_type:
        return FixtureResult(category, name, False, status_code=status,
                             diffs=[f"error.error_type: got {got_type!r} want {expected_type!r}"])
    if not got_message:
        return FixtureResult(category, name, False, status_code=status,
                             error="error.message empty (must be present per canon)")
    return FixtureResult(category, name, True, status_code=status)


# ---- main ----------------------------------------------------------


def main() -> int:
    ap = argparse.ArgumentParser(
        description="Batch-drive Trinity golden fixtures through a running engine "
                    "and produce a pass/fail report.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    ap.add_argument("--mode", choices=("local", "remote", "portforward"),
                    default="local",
                    help="how to reach the engine "
                         "(default: local subprocess; remote: --url; "
                         "portforward: kubectl)")
    ap.add_argument("--url", help="base URL when --mode=remote")
    ap.add_argument("--port", type=int, default=18181,
                    help="local port for local/portforward modes (default 18181)")
    ap.add_argument("--namespace", default="mademanifest",
                    help="namespace for portforward mode (default mademanifest)")
    ap.add_argument("--kubeconfig", help="kubeconfig for portforward mode")
    ap.add_argument("--report", help="write JSON report to this path")
    ap.add_argument("--max-failures", type=int, default=10,
                    help="cap diffs printed for each failure (default 10)")
    args = ap.parse_args()

    # ---- target ----
    if args.mode == "remote":
        if not args.url:
            ap.error("--url required when --mode=remote")
        target = EngineTarget(base_url=args.url.rstrip("/"))
        if not wait_for_healthz(target.base_url, deadline_seconds=10):
            print(f"[batch] WARNING: {target.base_url}/healthz not responding "
                  "yet — proceeding anyway", file=sys.stderr)
    elif args.mode == "portforward":
        target = start_portforward(args.namespace, args.kubeconfig, args.port)
    else:
        target = start_local_engine(args.port)

    try:
        # ---- /version capture ----
        version_blob = {}
        try:
            version_blob = http_get_json(target.base_url.rstrip("/") + "/version")
            print("[batch] /version:")
            for k, v in version_blob.items():
                print(f"        {k:<22} {v}")
        except Exception as e:
            print(f"[batch] WARNING: /version unreachable: {e}", file=sys.stderr)

        # ---- discover fixtures ----
        fixtures = discover_fixtures()
        results = []
        category_failed = False
        print()
        for cat, minimum in CATEGORIES:
            entries = fixtures.get(cat, [])
            print(f"[batch] {cat}  ({len(entries)} fixtures, canon minimum {minimum})")
            if len(entries) < minimum:
                print(f"        category below canon minimum — TEST RUN INVALID")
                category_failed = True
            for fx in entries:
                r = evaluate_fixture(target, cat, fx)
                results.append(r)
                tag = "PASS" if r.passed else "FAIL"
                line = f"        {tag}  {cat}/{r.name}"
                if not r.passed:
                    if r.error:
                        line += f"   [{r.error}]"
                    if r.diffs:
                        line += f"   ({len(r.diffs)} drift(s))"
                print(line)
                if not r.passed:
                    for d in r.diffs[: args.max_failures]:
                        print(f"            - {d}")
                    if len(r.diffs) > args.max_failures:
                        print(f"            ... (+{len(r.diffs) - args.max_failures} more)")

        passed = sum(1 for r in results if r.passed)
        failed = sum(1 for r in results if not r.passed)
        print()
        print(f"[batch] summary: {passed} passed, {failed} failed, "
              f"{len(results)} total fixtures")

        if args.report:
            report = {
                "version": version_blob,
                "passed": passed,
                "failed": failed,
                "total":  len(results),
                "fixtures": [asdict(r) for r in results],
            }
            Path(args.report).write_text(json.dumps(report, indent=2) + "\n")
            print(f"[batch] report written to {args.report}")

        if failed > 0 or category_failed:
            return 1
        return 0
    finally:
        if target.cleanup:
            target.cleanup()


if __name__ == "__main__":
    sys.exit(main())
