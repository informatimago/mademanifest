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

CalculateHumanDesign() in human_design.go needs to be updated.

- Read again human_design.go (I removed duplicate definitions).

- Note the panic error when running the program:

cd /Users/pjb/works/mademanifest/src/phase-3/poc_v2_bundle/mademanifest-engine ; ./proof-of-capability-2 ../golden/GOLDEN_TEST_CASE_V1.json out.json ; cat out.json
panic: open ../../canon/gate_sequence_v1.json: no such file or directory

goroutine 1 [running]:
mademanifest-engine/pkg/human_design.init.0()
	/Users/pjb/works/mademanifest/src/phase-3/poc_v2_bundle/mademanifest-engine/pkg/human_design/human_design.go:20 +0xc4

It comes from:

    data, err := os.ReadFile("../../canon/gate_sequence_v1.json")

the file cannot be found.

Therefore we must centralize all the paths to resource files, allow
them to be specified  by default as environment variables, or by
command-line options, or else keep a global default value. The paths
must be given as parameters at run-time (environment variables or
command-line options, never hardwired in the code. Only default values
for the data can be hardwired.

Do this for the 3 files/global variables  in canon/.
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
t
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

```
Task 2.4: add command-line options to specify the path to the canon files.

Current:

    Usage: ./proof-of-capability-2 $inputFile $outputFile

We want:

    Usage: ./proof-of-capability-2 \
               [--canon-directory|-cd        $canon_directory] \
               [--gate-sequence-file|-gs     $gate_sequence_file] \
	           [--mandala-constants-file|-mc $mandala_constants_file] \
	           [--node-policy-file|-np       $node_policy_file] \
               [--help|-h] \
               [--version|-v] \
               $inputFile $outputFile

The canon_directory argument may be an absolute path, or a relative
path. If it is a relative path it's looked up in the current working
directory. If it is absent, it's considered to be the subdirectory
named "canon/" in  current working directory.

The file arguments may be absolute path, or relative paths. If they're
relative paths, they're looked up in the canon_directory.

When a file argument is given, it specifies the corresponding file. If
no file is found at this path, an error is signaled and the program
exits. If a json file is found, it's loaded and overrides the
corresponding default global variable value.
When a file argument is not given, the default file name is used, and
searched in the canon directory. (adjust the default relative paths
for the files).

--help or -h prints the usage and exits.

--version or -v prints the program version (phase-3 poc-2 version 0.1)
and exits.

Otherwise the input and output file arguments are used and processed.

In main.go:

- Define an option structure with the parameters:

    canon_directory string
    gate_sequence_file string
    mandala_constants_file string
    node_policy_file string
    help boolean
    version boolean
    input_file string
    output_file string

- Implement a function to fill this structure with default values.

- Implement a function to parse the command-line options and fill this
  option structure.

- Implement a function that takes this option structure, and compute
  the absolute (or relative) pathnames to the 3 files.

- Add to the main function calls to these functions, tests of version
  and help boolean to issue the corresponding messages, and
  call the function to compute the absolute (or relative) pathnames to
  the 3 files otherwise. Then call the functions to load the data, and
  fill the global variables, before continuing with the main
  processing.

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

