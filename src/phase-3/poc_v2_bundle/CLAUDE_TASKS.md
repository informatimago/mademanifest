# Phase-3 implementation plan for Claude Code

This file decomposes the three requested tasks into Claude-executable subtasks, with explicit acceptance criteria and recommended prompts.

---

## Task 1 — Enforce Swiss Ephemeris runtime rules (FLAGS.md)

### Goal
Make all Swiss Ephemeris calls comply with `ephemeris/swisseph/FLAGS.md` and the Engineering Implementation Spec.

### Current state (as-is)
- Ephemeris path is hardcoded to `/usr/local/share/swisseph/`.
- Calculation flag is `SEFLG_SWIEPH` only.
- Nodes:
  - `ephemeris` exposes only mean node under both `north_node_mean` and `north_node`.
  - Human Design currently receives mean-node longitudes, which violates the node policy.

### Subtasks

#### 1.1 Add an explicit SwissEph init/config layer
**Implementation idea**
- Create `mademanifest-engine/pkg/ephemeris/runtime.go` (or `pkg/ephemeris/config.go`) that:
  - Resolves the ephemeris data path:
    1) if env `SE_EPHE_PATH` set: use it
    2) else default to repo-relative `../ephemeris/data/REQUIRED_EPHEMERIS_FILES/` (because the binary is executed from `mademanifest-engine/` in Makefile)
  - Calls `swephgo.SetEphePath()` once (null-terminated bytes)
  - Exposes `InitOnce()`

**Acceptance**
- No hardcoded `/usr/local/share/swisseph` remains in engine code.
- Running `make test` succeeds without requiring system ephemeris data beyond the three files in `ephemeris/data/REQUIRED_EPHEMERIS_FILES/`.

#### 1.2 Enforce calculation flags
**Implementation idea**
- Define a single `calcFlagsTropical()` that returns exactly:
  - `SEFLG_SWIEPH` (and optionally `SEFLG_SPEED` only if spec later requires it; currently it does not)
- Ensure the following are **not** used:
  - `SEFLG_SIDEREAL`, `swe_set_sid_mode`
  - `SEFLG_TOPOCTR`, `swe_set_topo`
  - `SEFLG_HELCTR`
  - `SEFLG_NONUT`

**Acceptance**
- Flags used in every `swephgo.Calc` call match `FLAGS.md` requirements.

#### 1.3 Add true node support and apply node policy
**Implementation idea**
- In `pkg/ephemeris`:
  - keep `north_node_mean` = `SE_MEAN_NODE`
  - add `north_node_true` = `SE_TRUE_NODE`
- Expose accessor(s):
  - `GetNodeLongitude(jd, policy)` or two explicit functions
- In `pkg/human_design.LongitudesAt`:
  - request `north_node_true` (or `north_node` resolved via policy)
- In Astrology:
  - continue to use mean node.

**Acceptance**
- Human Design snapshot uses true node.
- Astrology uses mean node.
- Golden test still passes.

#### 1.4 Add regression tests
**Implementation idea**
- Unit test in `pkg/ephemeris` asserting mean vs true node differ at the golden JD (or at least are both computable and not equal).
- Integration golden test remains the main guard.

**Acceptance**
- `make test-unit` passes.

### Claude prompt (Task 1)

Use this prompt in repo root:

```
You are working in this repo. Implement Task 1 from CLAUDE_TASKS.md.

Requirements:
- Follow ephemeris/swisseph/FLAGS.md exactly.
- Remove hardcoded /usr/local/share/swisseph path; resolve ephemeris data path via SE_EPHE_PATH env var, else default to ../ephemeris/data/REQUIRED_EPHEMERIS_FILES/ relative to mademanifest-engine.
- Ensure flags are tropical-only and do not enable sidereal/topocentric/heliocentric/nonut.
- Implement node policy: mean node for astrology, true node for human_design.

Do:
1) Explain the changes you will make (files/functions).
2) Make the edits.
3) Run: make test
4) If anything fails, iterate until make test is green.

Constraints:
- Keep diffs minimal.
- Do not change golden JSON fixtures.
```

---

## Task 2 — Load canon defaults from `canon/` and allow input overrides

### Goal
Introduce canonical defaults sourced from `canon/*.json`, applied first, then overridden by user input JSON when fields are present.

### Canon files
- `canon/mandala_constants.json` — default HD mapping constants
- `canon/node_policy.json` — default node policy per subsystem
- `canon/gate_sequence_v1.json` — gate sequence table (will be filled in Task 3)

### Key design decision
The current Go structs do not distinguish “missing field” vs “zero value”. To correctly support overrides:
- Decode input JSON into `map[string]any`
- Load canon JSON into `map[string]any`
- Deep-merge canon → input (input wins)
- Marshal merged map back into `emit_golden.GoldenCase`

### Subtasks

#### 2.1 Create a `pkg/canon` loader
**Implementation idea**
- `pkg/canon/canon.go`:
  - `LoadMandalaConstants(path string) (map[string]any, error)`
  - `LoadNodePolicy(path string) (map[string]any, error)`
  - `LoadGateSequence(path string) (… later in Task 3 …)`
  - `LoadDefaults() (map[string]any, error)` returning a canonical JSON subtree shaped like the input contract (e.g. under `engine_contract`)

**Acceptance**
- Canon loader reads files and returns deterministic values.

#### 2.2 Merge defaults into input in `ProcessInput`
**Implementation idea**
- Replace direct decode into struct with:
  1) decode input JSON into `map[string]any`
  2) load canon defaults into `map[string]any`
  3) deep merge
  4) marshal+unmarshal into `emit_golden.GoldenCase`

**Acceptance**
- Golden test still passes unchanged.
- Add a unit test: create an input JSON missing `human_design_mapping` and confirm it is filled from canon.

#### 2.3 Decide the precedence rules explicitly
**Rule**
- Canon provides defaults.
- Input file overrides canon if the JSON key exists.
- Engine-internal derived values (e.g. Earth = Sun+180) still override everything.

**Acceptance**
- Document the merge behavior in `pkg/process_input` or `pkg/canon` docstring.

### Claude prompt (Task 2)

```
Implement Task 2 from CLAUDE_TASKS.md.

Requirements:
- Load canon defaults from canon/mandala_constants.json and canon/node_policy.json.
- Apply them as defaults to the decoded input before engine asserts run.
- Input JSON values must override canon values when present.
- Use a map-decode + deep-merge approach so we can distinguish missing vs zero-values.

Do:
1) Add pkg/canon to load defaults.
2) Update pkg/process_input.ProcessInput to deep-merge canon->input.
3) Add at least one unit test proving defaults apply when fields are missing.
4) Run: make test

Constraints:
- Do not change golden fixtures.
- Keep diffs minimal and readable.
```

---

## Task 3 — Implement normal mandala gate sequence from canon (and populate canon table)

### Goal
Use the standard mandala mapping algorithm (already in spec) with a 64-element gate sequence loaded from `canon/gate_sequence_v1.json`. Populate that JSON file with the correct sequence.

### Current state (as-is)
- `pkg/human_design` contains:
  - multiple candidate hardcoded sequences
  - `GateSequence64()` currently returns an iota sequence (1..64)
  - a *GoldenPersonalityTable* override that forces expected values for the golden test

### Subtasks

#### 3.1 Populate `canon/gate_sequence_v1.json`
**Implementation idea**
- Fill `gate_sequence` with the authoritative 64-gate integer array for PoC V2.
- The sequence must be treated as data, not computed.

**Acceptance**
- File contains exactly 64 integers.

#### 3.2 Load the gate sequence via `pkg/canon`
**Implementation idea**
- Extend `pkg/canon` with `LoadGateSequenceV1()` returning `[]int` length 64.
- Add validation: length == 64, all entries are in [1..64], all unique.

**Acceptance**
- Bad canon file fails fast with a clear error.

#### 3.3 Remove golden overrides
**Implementation idea**
- Delete or disable `GoldenPersonalityTable` and `GetGateAndLineAster` special casing.
- Use `ComputeGateLine()` with the loaded gate sequence.

**Acceptance**
- Golden test passes without any special-case mapping.

#### 3.4 Add unit tests for mapping boundaries
**Implementation idea**
- Add tests to ensure interval rule “start inclusive, end exclusive” holds at:
  - exactly START
  - START + GATE_WIDTH
  - START - epsilon

**Acceptance**
- `go test ./pkg/human_design` passes.

### Claude prompt (Task 3)

```
Implement Task 3 from CLAUDE_TASKS.md.

Requirements:
- Populate canon/gate_sequence_v1.json with the correct 64-gate sequence for PoC V2.
- Load gate sequence from canon at runtime (via pkg/canon).
- Use the canonical mapping algorithm from the spec.
- Remove GoldenPersonalityTable and any gate/line overrides.

Do:
1) Fill gate_sequence_v1.json with 64 integers and add validation.
2) Wire the gate sequence into human_design.CalculateHumanDesign.
3) Add mapping boundary unit tests.
4) Run: make test

Constraints:
- No interpretation text.
- Keep diffs minimal.
```

