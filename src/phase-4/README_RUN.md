Proof of Capability V2
MadeManifest Deterministic Calculation Engine

Purpose
This Proof of Capability verifies whether an engineer can reproduce a fully deterministic calculation output exactly, using a pinned canon and a pinned environment.
Evaluation is strictly pass or fail.

What is provided
This bundle contains:
- Canonical calculation rules (canon/)
- The unchanged Golden Test Case (golden/)
- Engineering specifications (spec/)
- A pinned Swiss Ephemeris source and dataset (ephemeris/)
- A pinned Go environment (env/)
- Utility scripts for validation (tools/)

What you must do
1. Use Go only.
2. Use Swiss Ephemeris version 2.10.03 exactly as provided in this bundle.
3. Use only the ephemeris data files included in this bundle.
4. Use the canon files as the single source of truth.
5. Produce the output as a result of the calculation pipeline. Do not hardcode results.

Golden Test
The Golden Test Case is located at:
golden/GOLDEN_TEST_CASE_V1.json

It is the canonical reference for evaluation only.
It must not be used as a lookup table or shortcut.

How to run
You must provide:
- Source code written in Go
- One command that produces an output JSON file from the Golden Test input
- One command that diffs the produced output against the Golden Test expected output with no differences

Validation
Your submission must demonstrate:
- Bit exact numeric values
- Exact JSON structure and ordering
- No tolerance or rounding freedom
- Deterministic output on repeated runs

Prototype HTTP service
The phase-4 bundle now includes a thin HTTP wrapper around the engine.

Build the service container:
- `cd src/phase-4`
- `docker build -t mademanifest-phase-4 .`

Run it locally:
- `docker run --rm -p 8080:8080 mademanifest-phase-4`

Call it with the golden input:
- `src/phase-4/scripts/request_cloud_service.sh`

Kubernetes prototype
Prototype Kubernetes manifests live in:
- `src/phase-4/deploy/kubernetes`

Validate the manifests locally:
- `kubectl kustomize src/phase-4/deploy/kubernetes | kubectl apply --dry-run=client --validate=false -f -`
