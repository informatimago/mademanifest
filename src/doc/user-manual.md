---
title: MadeManifest Calculation Engine — User Manual
date: 2026-04-24
---

# MadeManifest Calculation Engine User Manual

This manual documents how to run the deterministic calculation engine
in this bundle, including its two API surfaces (file-based CLI and
HTTP service), command-line options, environment variables,
computation details, and output format.

The engine is currently in transition from the pre-Trinity "Golden PoC"
contract to the Trinity v1 runtime contract defined in
[`specifications/trinity/`](../../specifications/trinity/).  See the
companion docs for the transition plan and the pinned versions:

- [`trinity-implementation-plan.org`](trinity-implementation-plan.org) — 15-phase
  migration plan with per-phase tests.
- [`version-pins.org`](version-pins.org) — every pinned version and
  constant, with its canonical source and the open ambiguity register
  (A1..A8).
- [`requirement-tracking.org`](requirement-tracking.org) — detailed
  gap analysis between the current code and the Trinity canon.

## Overview

The engine calculates objective, deterministic outputs for:

- Astrology
- Human Design
- Gene Keys (derived from Human Design)

It uses the pinned Swiss Ephemeris version (`2.10.03`) and the pinned
canon constants.  The output is bit-exactly reproducible given
identical input, canon state, and version pins.

## API Surfaces

There are two ways to drive the engine:

### 1. File-based CLI (pre-Trinity PoC)

The binary `mademanifest-engine/mademanifest-engine` (built by
`make -C src compile`) takes one input JSON file and one output JSON
file.  It implements the Golden PoC contract — this is the only
runtime path still wired for the legacy `case_id` /
`engine_contract` / `expected` shape.

### 2. HTTP service

The binary `mademanifest-engine/cmd/httpserver` exposes a small HTTP
API:

- `GET  /healthz`  — liveness probe.
- `GET  /version`  — pinned version info as JSON (since Phase 1).
- `POST /manifest` — submit a calculation payload and get the result.

The HTTP service is the surface the Trinity input / output contract
will land on in Phases 2–10.  Today `POST /manifest` still returns the
PoC response shape; `GET /version` already returns the canonical
Trinity-compatible `VersionInfo` envelope.

## Quick Start

From the repo root:

```bash
make -C src swisseph-install swisseph-install-data   # one-time
make -C src compile                                  # build the PoC CLI
make -C src run                                      # produce out.json
make -C src diff                                     # strict diff vs golden
```

For the HTTP service:

```bash
cd src/mademanifest-engine
CGO_LDFLAGS="-lm" go build -o mademanifest-http ./cmd/httpserver
CANON_DIRECTORY=$(pwd)/../canon \
SE_EPHE_PATH=/usr/local/share/swisseph \
PORT=8080 \
./mademanifest-http
```

Or containerised + on Kubernetes — see the top-level
[`README.org`](../../README.org) for the full build/run recipe and
the `src/scripts/k8s-local-{up,down,test}.sh` dev loop.

## CLI Command-Line Usage

```bash
mademanifest-engine [options] <input.json> <output.json>
```

The program requires exactly two positional arguments:

- `input.json` — JSON input case file.
- `output.json` — output file to write.

### Options

| Flag                                | Default                   | Meaning                                                               |
|-------------------------------------|---------------------------|-----------------------------------------------------------------------|
| `--canon-directory` / `-cd`         | `canon`                   | Base directory for canon files.  Resolved from CWD if relative.       |
| `--gate-sequence-file` / `-gs`      | `gate_sequence_v1.json`   | Canon gate sequence file.  Resolved from `--canon-directory`.         |
| `--mandala-constants-file` / `-mc`  | `mandala_constants.json`  | Canon mandala constants file.                                         |
| `--node-policy-file` / `-np`        | `node_policy.json`        | Canon node policy file.                                               |
| `--dos`                             | off                       | Write output with CRLF line endings.                                  |
| `--help` / `-h`                     |                           | Print usage and exit.                                                 |
| `--version` / `-v`                  |                           | Print the pinned `VersionInfo` as JSON and exit (see *Versioning*).  |

### Example

From `src/mademanifest-engine/`:

```bash
./mademanifest-engine -cd ../canon ../golden/GOLDEN_TEST_CASE_V1.json out.json
```

## HTTP Service Usage

```bash
GET  http://<host>:<port>/healthz
GET  http://<host>:<port>/version
POST http://<host>:<port>/manifest   (Content-Type: application/json)
```

Body-size cap on `POST /manifest` is 10 MiB.  The handler returns:

- `200 OK` with the calculation result on success.
- `400 Bad Request` on malformed input or processing failure (to be
  replaced with the Trinity error envelope in Phase 3).
- `405 Method Not Allowed` for non-POST on `/manifest` and non-GET on
  `/version`.
- `500 Internal Server Error` on a handler panic.

Environment variables read at startup:

| Variable             | Default                       | Purpose                                                          |
|----------------------|-------------------------------|------------------------------------------------------------------|
| `PORT`               | `8080`                        | Port the server binds to.                                        |
| `CANON_DIRECTORY`    | `/app/canon` (in container)   | Directory containing the three canon JSON files.                |
| `SE_EPHE_PATH`       | resolved at runtime           | Directory containing Swiss Ephemeris `.se1` data files.          |

Helper scripts for interactive dev loops:

- `src/scripts/request_cloud_service.sh` — POST the golden fixture
  to a running service.
- `src/scripts/k8s-local-up.sh` / `k8s-local-test.sh` /
  `k8s-local-down.sh` — spin a kind cluster with the service, drive
  it with curl, tear it down.

The `--version` / `-v` flag on `cmd/httpserver` prints the same JSON
as `GET /version` and exits without starting the listener.

## Environment Variables

### `SE_EPHE_PATH`

Path to Swiss Ephemeris data files.  If unset, the engine tries:

1. The relative repo path `../ephemeris/data/REQUIRED_EPHEMERIS_FILES/`.
2. `/usr/local/share/swisseph/`.

### `SE_NODE_POLICY`  *(deprecated — slated for removal in Phase 9)*

Controls the node used by `GetPlanetLongAtTime` for `north_node`
lookups in the PoC astrology path.  If set to `true`, the true node
is used; otherwise the mean node is used.  Trinity policy is fixed
(astrology = mean, Human Design = true) and no longer depends on
this variable.

### `PORT`, `CANON_DIRECTORY`

Used only by `cmd/httpserver`.  See *HTTP Service Usage* above.

## Input Contract

The current PoC input shape is a JSON document with three sections:

- `case_id`
- `birth`
- `engine_contract`

The engine merges canon defaults into the input before processing;
input values override canon defaults.

> **Transition note.**  Phase 2 of the Trinity plan replaces this
> shape with the canonical Trinity payload
> `{birth_date, birth_time, timezone, latitude, longitude}` and adds
> a strict boundary validator that classifies rejections as
> `incomplete_input`, `invalid_input`, or `unsupported_input`.

### `birth` fields (PoC)

- `date`: `YYYY-MM-DD`
- `time_hh_mm`: `HH:MM` (seconds assumed `00`)
- `seconds_policy`: must be `assume_00`
- `place_name`: text name (not used for computation)
- `latitude`: decimal degrees
- `longitude`: decimal degrees
- `timezone_iana`: IANA timezone identifier

### `engine_contract` fields (PoC)

The engine asserts the following contract values:

- `ephemeris`: `swiss_ephemeris`
- `zodiac`: `tropical`
- `houses`: `placidus`
- `node_policy_by_system.astrology`: `mean`
- `node_policy_by_system.human_design`: `true`
- `node_policy_by_system.gene_keys`: `true`
- `human_design_mapping.interval_rule`: `start_inclusive_end_exclusive`

The remaining fields are provided via canon defaults:

- `human_design_mapping.mandala_start_deg`
- `human_design_mapping.gate_width_deg`
- `human_design_mapping.line_width_deg`
- `design_time_solver.sun_offset_deg`
- `design_time_solver.stop_if_abs_sun_diff_deg_below`
- `design_time_solver.stop_if_time_bracket_below_seconds`

## Computations Performed

This section summarises the computation pipeline implemented in code.

### 1. Time conversion

- Parse local `birth.date` and `birth.time_hh_mm`.
- Convert local time to UTC using the IANA tzdb (including DST rules)
  — tzdata version tracked as *A1* in `version-pins.org`.
- Convert UTC time to Julian Day (UT).

### 2. Ephemeris longitudes

Using Swiss Ephemeris (pinned to `2.10.03`) and tropical zodiac:

- Compute ecliptic longitudes for Sun, Moon, Mercury, Venus, Mars,
  Jupiter, Saturn, Uranus, Neptune, Pluto, Chiron, and Mean North
  Node.
- Derived values:
  - Earth longitude = Sun + 180° (mod 360).
  - South Node longitude = North Node + 180° (mod 360).

### 3. Astrology module

- House system: Placidus (`swephgo.HousesEx` with `'P'`).
- Ascendant and Midheaven (MC) come from the `ascmc` output.
- For each object: convert longitude into sign and degree/minute
  within the sign.
- The `north_node_mean` field in astrology output is the mean node.

> **Transition note (Phase 4 — landed).**  The Trinity canonical
> `{object_id, longitude, sign, house}` output is now emitted by
> `POST /manifest`, with explicit house cusps 1–12 and Earth as a
> first-class astrology object.  The legacy `{sign, deg, min}` shape
> survives only in the Golden PoC contract path and is retired in
> Phase 12.

### 4. Human Design module

Two snapshots:

- **Personality** at birth.
- **Design** at the prior moment when the Sun longitude equals
  `birth_sun − sun_offset_deg`.

Design-time solver (Trinity / Phase 5):

- Target Sun longitude = `Sun(birth) − 88.0°` (normalised 0–360°).
  The 88° offset is canon-pinned in
  `pkg/hd/calc.SunOffsetDeg`.
- Search direction: backward only.
- Initial bracket: `birth − (88 ± 5)` days.  The bracket is widened
  on either side in 2-day steps if the canonical 10-day window does
  not bracket the root (this is a defensive fallback; the canonical
  case starts already bracketed because the Sun moves ~0.985°/day).
- Pure bisection (no secant fallback, no fixed-day shortcut, no
  precomputed lookup).
- Stop conditions:
  - `|Sun(t) − target| < 0.0001°`  (canon: `StopAbsSunDiffDeg`), or
  - bracket width < 1 second (canon: `StopBracketSeconds`).
- The midpoint of the final bracket is returned as the design moment;
  Phase 3's `output.DesignTime` marshaler floors that value to whole
  seconds before emitting `human_design.system.design_time_utc`.

> **Transition note (Phase 5 — landed).**  Trinity calls now emit
> `human_design.system.design_time_utc` from
> `pkg/trinity/hd.ComputeDesignTime`, which calls the canonical
> bisection solver in `pkg/hd/calc.SolveDesignTime`.  The legacy
> input-driven `DesignTimeSolver` parameters in the Golden PoC
> contract still feed `pkg/human_design.SolveDesignTime` for the
> file-based PoC CLI; that path is retired in Phase 12.  Exact-second
> rounding is tracked as ambiguity *A3* in `version-pins.org`.

Mapping to gates and lines (Trinity / Phase 6):

- Canon numeric constants come from `pkg/canon/constants.go`:
  - `MandalaAnchorDeg` = **277.5°** (gate 38 starts here)
  - `GateWidthDeg`     = 5.625°
  - `LineWidthDeg`     = 0.9375°
  - `GateOrder`        = 64-entry canonical gate sequence (starts with 38)
- Algorithm (`pkg/hd/calc.MapToGateLine`):
  - `r = (longitude − MandalaAnchorDeg) mod 360`
  - `gate_index = floor(r / GateWidthDeg)`
  - `line_index = floor((r mod GateWidthDeg) / LineWidthDeg)`
  - return `(GateOrder[gate_index], line_index + 1)`
- Interval rule: start inclusive, end exclusive — a longitude
  exactly on a gate or line boundary belongs to the *new* segment.
- The function consumes only the longitude argument and the
  compiled-in canon constants; no environment variables, no input
  parameters.
- Activation snapshots use Swiss Ephemeris with the canon's
  per-domain node policy (Document 03 §"Node policy by domain"):
  - astrology: `north_node_mean` = SE_MEAN_NODE
  - human_design: `north_node` = SE_TRUE_NODE (Phase 6+)
  - gene_keys: derived from human_design (also true)
- Output: `human_design.personality_activations` and
  `human_design.design_activations` are 13-entry arrays of
  `{object_id, gate, line}` in `canon.HDSnapshotOrder`.

> **Transition note (A8 — landed via Phase 6).**  The compiled
> Trinity canon in `pkg/canon/constants.go` uses the canonical
> mandala anchor **277.5°** (sequence starting with gate 38)
> specified by `trinity.org`.  The legacy anchor 313.25° in
> `src/canon/mandala_constants.json` is explicitly marked "rejected"
> in the Trinity regression sentinels.  The Trinity HTTP path uses
> the compiled canon today; the legacy JSON-driven PoC CLI path
> still uses the legacy anchor and will be retired in Phase 12.

> **Transition note (Phase 6 — landed).**  The
> `SE_NODE_POLICY` environment-variable switch in
> `pkg/ephemeris.GetPlanetLongAtTime` was retired in Phase 6.  Each
> caller selects the node policy explicitly by body name:
> `"north_node"`/`"north_node_mean"` for SE_MEAN_NODE,
> `"north_node_true"` for SE_TRUE_NODE.  Trinity HD activations
> always use the true policy; Trinity astrology always uses mean.

Structural derivations (Trinity / Phase 7) are computed by
`pkg/hd/structure.Compute` from the combined personality + design
activation set:

- **Channels.**  Walk `canon.ChannelTable`; emit each channel whose
  two gates both appear in the active set; output sorted
  lexicographically by `channel_id`.
- **Centers.**  Each center is `defined` if it participates in at
  least one active channel, `undefined` otherwise.  The output array
  always has 9 entries in `canon.CenterOrder` (head, ajna, throat, g,
  ego, solar_plexus, sacral, spleen, root).
- **Definition.**  Run union-find over the active channels (each
  channel = an edge between two centers).  Count connected
  components made of *defined* centers and pick the canonical class:
  - 0 -> `none`
  - 1 -> `single`
  - 2 -> `split`
  - 3 -> `triple_split`
  - 4 -> `quadruple_split`
- **Type** decision tree (`trinity.org` lines 318-325, first
  matching rule wins):
  1. `definition == none` -> `reflector`
  2. sacral defined:
     - motor-to-throat path in the same component -> `manifesting_generator`
     - otherwise -> `generator`
  3. non-sacral motor connects to throat -> `manifestor`
  4. otherwise -> `projector`
- **Authority** priority list (`trinity.org` lines 326-334, first
  matching rule wins):
  1. `emotional` if solar_plexus defined
  2. `sacral` if type in {generator, manifesting_generator}
  3. `splenic` if spleen defined
  4. `ego_manifested` if type=manifestor and ego defined
  5. `ego_projected` if type=projector and ego defined
  6. `self_projected` if type=projector and g defined
  7. `mental` if type=projector
  8. `lunar` if type=reflector
- **Profile** = `personality_sun_line/design_sun_line` (e.g. `"1/3"`).
- **Incarnation cross** = `{personality_sun, personality_earth,
  design_sun, design_earth}`, each as `{gate, line}` pairs; no
  human-readable cross name is required (`trinity.org` line 339).

### 5. Gene Keys module

Gene Keys are derived directly from Human Design output by
`pkg/trinity/genekeys.Compute`.  The block has exactly two top-level
sub-objects: `system` and `activations`.

- `system.derivation_basis` is the literal string `"human_design"`.
- `activations` carries the four canonical positions, each a
  `{key, line}` pair where `key` is the HD gate number and `line` is
  the HD line number:
  - `life_work` = Personality Sun
  - `evolution` = Personality Earth
  - `radiance`  = Design Sun
  - `purpose`   = Design Earth

The derivation is a pure function of the four HD pillar activations;
no astronomical computation, no node policy, no canon constants.

Out of scope for v1 (must not appear in the wire output):

- shadow / gift / essence text
- sequence prose
- semantic-state fields

> **Transition note (Phase 8 -- landed).**  The PoC field name
> `lifes_work` is retired; the canonical name is `life_work`.  The
> rename is enforced by the Phase 3 output type
> `output.GKActivations.LifeWork`, which has had the canonical JSON
> tag since the envelope types landed.  Phase 8 only adds the
> derivation that finally fills those four `{key, line}` pairs.

## Output Format

The current output is the PoC document; Phases 3–8 replace it with
the Trinity success/error envelope.

### Top-level structure (PoC)

- `case_id`
- `birth`
- `engine_contract`
- `expected`

### `expected.astrology.positions`

Position objects with `sign`, `deg`, and `min`:

- `sun`, `moon`, `mercury`, `venus`, `mars`, `jupiter`, `saturn`,
  `uranus`, `neptune`, `pluto`, `chiron`, `north_node_mean`,
  `ascendant`, `mc`

### `expected.human_design`

- `activation_object_order` — fixed array
  `[sun, earth, north_node, south_node, moon, mercury, venus, mars,
    jupiter, saturn, uranus, neptune, pluto]`.
- `personality` — map keyed by the same objects, values formatted as
  `gate.line` with one decimal place.
- `design` — same as `personality` at the design-time snapshot.

### `expected.gene_keys.activation_sequence`

- `lifes_work`, `evolution`, `radiance`, `purpose`.
- Each is `{ "key": <int>, "line": <int> }`.

### Formatting rules

- Output JSON is rendered in a fixed key order and spacing.
- `--dos` switches line endings to CRLF; otherwise LF.
- Floating-point values in the emitted JSON use fixed precision as
  defined in the renderer.

## Versioning

Since Phase 1 the engine exposes its pinned versions in three places.
All three return the same `VersionInfo` structure.

### `GET /version` (HTTP)

```bash
curl -s http://localhost:8080/version
```

```json
{
  "engine_version": "v0.1.0-phase-8",
  "canon_version": "trinity-v1-rev-0",
  "mapping_version": "trinity-v1-rev-0",
  "input_schema_version": "trinity-v1-rev-0",
  "source_stack_version": "trinity-v1-rev-0",
  "swisseph_version": "2.10.03",
  "tzdb_version": "2023c"
}
```

### `--version` (CLI and HTTP server)

Both `mademanifest-engine --version` and
`cmd/httpserver --version` emit the same JSON to stdout and exit.

### `pkg/canon.Versions()` (Go API)

Code that embeds the engine as a library can read the same struct
directly:

```go
info := canon.Versions()
fmt.Println(info.CanonVersion, info.SwissEphVersion)
```

### Field meanings

| Field                  | Bumps when ...                                                                        |
|------------------------|---------------------------------------------------------------------------------------|
| `engine_version`       | Any production implementation change, even refactors.                                 |
| `canon_version`        | Any change to scope, calculation, mapping, output, precedence, formatting, or input.  |
| `mapping_version`      | Gate order, channel table, center list, or identifier scheme changes.                 |
| `input_schema_version` | Field/type/format/range/validation changes.                                          |
| `source_stack_version` | Swiss Ephemeris or tzdb release changes.                                              |
| `swisseph_version`     | Pinned by trinity.org — verified at first ephemeris call; mismatch aborts the engine. |
| `tzdb_version`         | Tracked as ambiguity *A1*; currently inherits Go 1.22 embedded tzdata (2023c).        |

See [`version-pins.org`](version-pins.org) for the full A-register.

## Determinism Requirements

- Use only the canon files provided; once Phase 9 lands, the canon
  constants are compiled into the binary and the JSON loader is
  retired.
- Use only Swiss Ephemeris version `2.10.03` and the bundled
  ephemeris data.  The runtime aborts if it detects a different
  library version.
- Do not hardcode results; run the full computation pipeline.
- Output must be bit-exact identical to the golden output (PoC) or
  semantically identical to the Trinity expected output (Phase 11+).
- No hidden state: no mutable caches, no implicit environment
  defaults, no third-party service responses, no stored run history.
  Remaining environment-variable overrides (`SE_NODE_POLICY`,
  `SE_EPHE_PATH` override) are tracked for removal in Phase 9.
