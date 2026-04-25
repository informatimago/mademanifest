---
title: MadeManifest Calculation Engine — User Manual
date: 2026-04-24
---

# MadeManifest Calculation Engine User Manual

This manual documents how to run the deterministic calculation engine
in this bundle: the HTTP service surface, environment variables,
computation details, and output format.

The engine implements the Trinity v1 runtime contract defined in
[`specifications/trinity/`](../../specifications/trinity/).  Phase 12
of the implementation plan retired the pre-Trinity "Golden PoC"
file-based CLI; the HTTP service is now the only runtime surface.
See the companion docs for the transition history and the pinned
versions:

- [`trinity-implementation-plan.org`](trinity-implementation-plan.org) — 15-phase
  migration plan with per-phase tests.
- [`version-pins.org`](version-pins.org) — every pinned version and
  constant, with its canonical source and the open ambiguity register
  (A1..A8).
- [`requirement-tracking.org`](requirement-tracking.org) — detailed
  gap analysis between the original PoC and the Trinity canon.

## Overview

The engine calculates objective, deterministic outputs for:

- Astrology
- Human Design
- Gene Keys (derived from Human Design)

It uses the pinned Swiss Ephemeris version (`2.10.03`), the pinned
canon constants compiled into `pkg/canon`, and the IANA tzdata
embedded in the Go 1.22 toolchain.  The output is bit-exactly
reproducible given identical input and version pins.

## API Surface

The single binary `cmd/httpserver` (built into
`src/mademanifest-engine/mademanifest-engine` by `make -C src compile`)
exposes a small HTTP API:

- `GET  /healthz`  — liveness probe.  Body is exactly
  `{"status":"ok"}`; never carries version information.
- `GET  /version`  — pinned canon version block plus the Phase 9
  diagnostic field `ephe_path_resolved`.
- `POST /manifest` — submit a Trinity payload and receive a Trinity
  success or error envelope (Phase 10 contract).

## Quick Start

From the repo root:

```bash
make -C src swisseph-install swisseph-install-data   # one-time
make -C src compile                                  # build cmd/httpserver
make -C src test-integration-local                   # full Trinity golden pack
```

To start the service manually:

```bash
cd src/mademanifest-engine
PORT=8080 \
SE_EPHE_PATH=/usr/local/share/swisseph \
./mademanifest-engine
```

Or containerised + on Kubernetes — see the top-level
[`README.org`](../../README.org) for the full build/run recipe and
the `src/scripts/k8s-local-{up,down,test}.sh` dev loop.

> **Phase 12 transition note.**  The legacy `cmd/main.go` file-based
> CLI that took `--canon-directory` / `--gate-sequence-file` /
> `--mandala-constants-file` / `--node-policy-file` flags and read
> the PoC `case_id` / `engine_contract` / `expected` shape has been
> removed.  The Makefile aliases `run`, `diff`, and
> `test-integration` now delegate to `test-integration-local`,
> which runs the Phase 11 golden pack against a freshly-built
> local subprocess.

## HTTP Service Usage

```bash
GET  http://<host>:<port>/healthz
GET  http://<host>:<port>/version
POST http://<host>:<port>/manifest   (Content-Type: application/json)
```

`POST /manifest` enforces (Phase 10):

- `Content-Type: application/json` is required (charset suffixes
  like `; charset=utf-8` are accepted).
- Body size cap = `MaxRequestBodyBytes` = 1 MiB.

Status code policy:

| HTTP code | When                                                     | Trinity error_type    |
|-----------|----------------------------------------------------------|-----------------------|
| 200       | Valid payload; canonical success envelope is returned.   | -                     |
| 400       | Missing required field, type/format violation, malformed JSON, IANA timezone alias, range violation. | `incomplete_input` / `invalid_input` |
| 405       | Wrong HTTP method on any of the three endpoints.         | -                     |
| 413       | Body exceeds `MaxRequestBodyBytes`.                      | `unsupported_input`   |
| 415       | Missing or non-`application/json` Content-Type.          | `invalid_input`       |
| 422       | Structurally valid input outside Trinity v1 scope (sub-minute precision, multi-person, etc.). | `unsupported_input`  |
| 500       | Internal calculation failure or handler panic.           | `execution_failure`   |

`GET /healthz` is liveness-only: the body is exactly
`{"status":"ok"}` and never contains version information (the
liveness probe does not change scope across phases).

`GET /version` returns the canonical version block plus the Phase 9
diagnostic field `ephe_path_resolved`.

Environment variables read at startup:

| Variable        | Default               | Purpose                                                          |
|-----------------|-----------------------|------------------------------------------------------------------|
| `PORT`          | `8080`                | Port the server binds to.                                        |
| `SE_EPHE_PATH`  | resolved at runtime   | Directory containing Swiss Ephemeris `.se1` data files.          |

Phase 12 retired `CANON_DIRECTORY` (the trinity HTTP path consumes
only the compiled-in canon constants).  Phase 6 retired
`SE_NODE_POLICY` (per-domain node policy is fixed by the canon).
The engine reads no other environment variables at runtime.

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

The resolved absolute path is surfaced under `ephe_path_resolved`
in `GET /version` (Phase 9 diagnostic).  `pkg/ephemeris.ValidateEphePath()`
runs at boot and aborts startup if the resolved path does not
exist or is not a directory.

### `PORT`

HTTP listen port (default `8080`).

### `CANON_DIRECTORY`  *(deprecated — Phase 12 retired)*

Phase 12 retired the legacy canon JSON loaders; the trinity HTTP
path consumes only the compiled-in `pkg/canon/constants.go`.  The
variable can still be set for backward compat with older deployment
manifests but has no effect.

## Input Contract

The Trinity v1 input is a JSON object with exactly five fields and
no others:

| field        | type   | format / rule                  |
|--------------|--------|--------------------------------|
| `birth_date` | string | `YYYY-MM-DD`                   |
| `birth_time` | string | `HH:MM`, 24-hour, minute only  |
| `timezone`   | string | IANA `Area/Location` identifier |
| `latitude`   | number | decimal degrees, `-90..90`     |
| `longitude`  | number | decimal degrees, `-180..180`   |

```json
{
  "birth_date": "1990-04-09",
  "birth_time": "18:04",
  "timezone": "Europe/Amsterdam",
  "latitude": 51.9167,
  "longitude": 4.4
}
```

`pkg/trinity/input.Validate` (Phase 2) classifies any deviation as
one of three Trinity error types:

- `incomplete_input` — required field missing.
- `invalid_input`    — wrong type, malformed format, out-of-range
                       value, IANA abbreviation (e.g. `CET`), IANA
                       link name (e.g. `US/Eastern`), unknown field,
                       malformed JSON, or wrong Content-Type
                       (Phase 10 also adds 415 + invalid_input for
                       Content-Type errors).
- `unsupported_input` — structurally valid input outside Trinity v1
                       scope (e.g. sub-minute time precision).

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

> **Transition note (Phase 4 — landed; Phase 12 — retired
> the legacy alternative).**  The Trinity canonical
> `{object_id, longitude, sign, house}` output is now emitted by
> `POST /manifest`, with explicit house cusps 1–12 and Earth as a
> first-class astrology object.  The legacy `{sign, deg, min}` PoC
> shape was removed in Phase 12.

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

> **Transition note (Phase 5 — landed; Phase 12 — legacy path
> retired).**  Trinity calls emit
> `human_design.system.design_time_utc` from
> `pkg/trinity/hd.ComputeDesignTime`, which calls the canonical
> bisection solver in `pkg/hd/calc.SolveDesignTime`.  The legacy
> input-driven `DesignTimeSolver` PoC parameters and
> `pkg/human_design.SolveDesignTime` were removed in Phase 12.
> Exact-second rounding is tracked as ambiguity *A3* in
> `version-pins.org`.

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

> **Transition note (A8 — landed via Phase 6; Phase 12 — legacy
> JSON canon removed).**  The compiled Trinity canon in
> `pkg/canon/constants.go` uses the canonical mandala anchor
> **277.5°** (sequence starting with gate 38) specified by
> `trinity.org`.  Phase 12 removed the legacy 313.25° JSON files
> under `src/canon/`; the trinity HTTP path always reads the
> compiled canon directly.

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

The Trinity success envelope returned by `POST /manifest` has the
canonical six-key shape (`trinity.org` §"Success Response"):

```json
{
  "status": "success",
  "metadata":    { "engine_version": "...", "canon_version": "...", ... },
  "input_echo":  { "birth_date": "...", "birth_time": "...", ... },
  "astrology":   { "system": {...}, "angles": {...}, "house_cusps": [...], "objects": [...] },
  "human_design":{ "system": {...}, "personality_activations": [...], ... },
  "gene_keys":   { "system": {...}, "activations": {...} }
}
```

The error envelope has three top-level keys:

```json
{
  "status": "error",
  "metadata": { ... },
  "error":    { "error_type": "invalid_input", "message": "..." }
}
```

`error_type` is one of `invalid_input`, `incomplete_input`,
`unsupported_input`, `canon_conflict`, `execution_failure`.

### Field highlights

- `metadata` contains the five canon version pins
  (`engine_version`, `canon_version`, `source_stack_version`,
  `input_schema_version`, `mapping_version`).  It does **not**
  include `swisseph_version`, `tzdb_version`, or
  `ephe_path_resolved` — those live only in `GET /version`.
- `astrology.objects[]` is 13 entries in canon order
  `[sun, moon, mercury, venus, mars, jupiter, saturn, uranus,
    neptune, pluto, chiron, north_node_mean, earth]`.
- `astrology.house_cusps[]` is 12 entries, houses 1..12.
- `human_design.personality_activations[]` and `design_activations[]`
  are 13 entries each in canon order `[sun, earth, north_node,
    south_node, moon, mercury, venus, mars, jupiter, saturn,
    uranus, neptune, pluto]`.
- `human_design.channels[]` is sorted lexicographically by
  `channel_id`.
- `human_design.centers[]` is 9 entries in canon order
  `[head, ajna, throat, g, ego, solar_plexus, sacral, spleen, root]`.
- `gene_keys.activations` carries `{life_work, evolution, radiance,
  purpose}`, each as `{key, line}`.  The legacy PoC name
  `lifes_work` is retired (Phase 8).

### Formatting rules

- Longitude fields are rounded to 6 decimal places.
- `human_design.system.design_time_utc` is RFC 3339 UTC with
  whole-second precision and a trailing `Z` (A3 floor rule).
- Object/array ordering is canon-pinned and asserted by the
  Phase 11 golden pack runner.

## Versioning

Since Phase 1 the engine exposes its pinned versions in three places.
All three return the same `VersionInfo` structure.

### `GET /version` (HTTP)

```bash
curl -s http://localhost:8080/version
```

```json
{
  "engine_version": "v0.1.0-phase-12",
  "canon_version": "trinity-v1-rev-0",
  "mapping_version": "trinity-v1-rev-0",
  "input_schema_version": "trinity-v1-rev-0",
  "source_stack_version": "trinity-v1-rev-0",
  "swisseph_version": "2.10.03",
  "tzdb_version": "2023c",
  "ephe_path_resolved": "/usr/local/share/swisseph"
}
```

The `ephe_path_resolved` field is a Phase 9 deployment diagnostic:
it surfaces the absolute filesystem path the engine resolved
`SE_EPHE_PATH` to, so operators can confirm at runtime which
ephemeris bundle the binary loaded.  The field appears in
`/version` only; it never appears in the trinity success/error
response metadata block (`metadata` is reserved for canon version
pins per `trinity.org` lines 451-462).

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
| `ephe_path_resolved`   | Deployment-resolved Swiss Ephemeris path; appears only in `/version`, never in metadata. |

See [`version-pins.org`](version-pins.org) for the full A-register.

## Golden Test Pack

Phase 11 ships the canonical Trinity Golden Test Pack under
`src/golden/trinity/`.  The directory is organised by canon
category:

```
src/golden/trinity/
  valid_baseline/
    schiedam_1990_04_09/
      input.json
      expected.json
    new_york_1985_07_21/...
    tokyo_2000_01_01/...
  valid_edge/                  (5 cases)
  invalid_input/               (5 cases)
  incomplete_input/            (5 cases)
  unsupported_input/           (2 cases)
  regression_sentinel/         (3 cases)
```

`pkg/golden.LoadFixtures` enforces the canon minimums (3 / 5 / 5 /
5 / 2 / 3) at load time.  The runner asserts:

- **Success cases** (`valid_baseline`, `valid_edge`,
  `regression_sentinel`): the engine response must equal the frozen
  `expected.json` field-by-field.  The metadata block is *excluded*
  from the comparison and asserted separately against
  `output.CurrentMetadata()` — so phase bumps that change
  `engine_version` do not invalidate fixtures.
- **Error cases** (`invalid_input`, `incomplete_input`,
  `unsupported_input`): the engine must return the canon-mapped
  HTTP status code (400 / 400 / 422) and an error envelope whose
  `error_type` matches the fixture.  Per ambiguity *A4*, the
  `message` field is informational and is not compared.

`TestTrinityGoldenPack` in each integration harness (local
subprocess, Docker container, kind cluster) iterates the entire
pack and runs each fixture as a `t.Run` sub-test, so a single drift
surfaces with the exact `<category>/<name>` path.

To capture a fresh fixture set against a different (or
re-engineered) engine build, start `cmd/httpserver` and post the
existing `input.json` files; freeze the resulting body (with the
`metadata` key removed) as `expected.json`.

## Determinism Requirements

- The trinity HTTP path consumes only the compiled-in
  `pkg/canon/constants.go`.  Phase 12 removed the legacy
  `src/canon/*.json` files; the trinity binary no longer ships any
  canon JSON files.
- The engine refuses to boot when its compiled constants fail
  `pkg/canon.SelfCheck()` or when the resolved ephemeris path fails
  `pkg/ephemeris.ValidateEphePath()`.  Phase 9 pins this invariant.
- Use only Swiss Ephemeris version `2.10.03` and the bundled
  ephemeris data.  The runtime aborts if it detects a different
  library version.
- Do not hardcode results; run the full computation pipeline.
- Output must be semantically identical to the Trinity expected
  output (Phase 11+).
- No hidden state: no mutable caches, no implicit environment
  defaults, no third-party service responses, no stored run history.
- The only runtime environment variables consulted by the engine
  are deployment settings — `SE_EPHE_PATH` (Swiss Ephemeris data
  directory) and `PORT` (HTTP listen port).  Phase 6 retired the
  `SE_NODE_POLICY` env shim; Phase 12 retired `CANON_DIRECTORY`.
  Phase 9 verifies env-immunity end-to-end through local
  subprocess, Docker container, and kind cluster (the kustomize
  overlay injects `SE_NODE_POLICY=true` and the canonical Phase 4-8
  oracles must remain bit-identical).
