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

Rules
- No clarification questions will be answered
- One submission only
- No revisions
- Any deviation from the Golden Test constitutes failure

If all requirements are met, the Proof of Capability is considered passed.
