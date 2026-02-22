# Claude Code guide — MadeManifest Phase 3

This repository is a deterministic calculation engine (Astrology, Human Design, Gene Keys). It is designed to be *golden-test driven*: any output mismatch is a failure.

## Repo layout

- `mademanifest-engine/` — Go module and executable
  - `cmd/main.go` — CLI entrypoint (reads JSON input, writes JSON output)
  - `pkg/` — engine packages (`ephemeris`, `astrology`, `human_design`, `gene_keys`, …)
- `golden/` — golden input+expected fixture (`GOLDEN_TEST_CASE_V1.json`)
- `canon/` — static canonical data (defaults + tables). Phase-3 work is to make the engine *canon-driven*.
- `ephemeris/` — Swiss Ephemeris source archive, build notes, and the *required* ephemeris data files
  - `ephemeris/swisseph/FLAGS.md` — authoritative runtime rules for Swiss Ephemeris usage
  - `ephemeris/data/REQUIRED_EPHEMERIS_FILES/` — the only data files that are allowed
- `spec/*.pdf` — authoritative specifications (see especially “Engineering Implementation Specification”)
- `Makefile` — build/test workflow

## How to build and test

From repo root:

- `make all` — build/install SwissEph + build Go binary + run + diff against golden
- `make test` — unit tests + integration golden diff
- `make test-unit` — `go test ./pkg/...`
- `make test-integration` — runs the binary and diffs output vs golden

Notes:
- The integration diff uses `diff -twb` (whitespace/newlines tolerant). Some fixtures may have DOS newlines.

## Golden contract and invariants

The current CLI expects:

- `birth.seconds_policy == "assume_00"`
- `engine_contract.ephemeris == "swiss_ephemeris"`
- `engine_contract.zodiac == "tropical"`
- `engine_contract.houses == "placidus"`
- `engine_contract.node_policy_by_system.astrology == "mean"`
- `engine_contract.node_policy_by_system.human_design == "true"`
- `engine_contract.node_policy_by_system.gene_keys == "true"`
- `engine_contract.human_design_mapping.interval_rule == "start_inclusive_end_exclusive"`

Treat the above as hard requirements until the spec introduces versioned variations.

## Phase-3 goals

We must implement three upgrades:

1. **Swiss Ephemeris runtime rules**
   - Implement `ephemeris/swisseph/FLAGS.md`.
   - Key requirements:
     - Tropical zodiac only (no sidereal flags / ayanamsha)
     - No topocentric corrections
     - No heliocentric mode
     - No nutation suppression
     - Use Swiss Ephemeris precision defaults
     - **Ephemeris data must be loaded only from** `ephemeris/data/REQUIRED_EPHEMERIS_FILES/`
     - Node policy:
       - Mean Node for Astrology outputs
       - True Node for Human Design outputs

2. **Canon-driven defaults**
   - Load canonical defaults from `canon/` and apply them before/while processing input.
   - Input file values must be able to override canon defaults.

3. **Human Design mandala gate sequence**
   - Implement the normal mandala mapping algorithm using a 64-element gate sequence loaded from canon.
   - Populate `canon/gate_sequence_v1.json` with the correct 64-gate sequence for PoC V2.
   - Remove any “golden override” logic once the canonical algorithm matches golden deterministically.

## Engineering constraints

- Deterministic output: no randomness, no time-dependent behavior.
- Keep the calculation layer free of interpretation/text.
- Prefer small, reviewable diffs: one task per PR/commit.
- Every behavior change must come with:
  - a unit test (if feasible), or
  - a golden/integration assertion that would have failed before.

## Suggested implementation approach

- Add a new package (suggested): `pkg/canon` for loading `canon/*.json` and merging with input.
- Add a SwissEph initialization layer (suggested): `pkg/ephemeris/runtime` or `pkg/ephemeris/config` that:
  - resolves ephemeris data path (prefer repo-relative, allow override via env like `SE_EPHE_PATH`)
  - provides flag sets for each subsystem (Astrology vs Human Design)
  - exposes explicit functions to compute:
    - tropical longitudes for planets
    - mean node longitude (astrology)
    - true node longitude (human design)

