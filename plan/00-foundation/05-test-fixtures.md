# 00.05 — Test fixtures & golden tests

Extraction and resolution are only trustworthy if pinned against real Godot files. Fixtures are the
ground truth; golden tests catch regressions (archi's snapshot harness is the model).

## Fixture projects

Under `testdata/fixtures/`:

- **`minimal/`** — the smallest valid Godot 4 project: `project.godot`, one scene, one attached
  script, one autoload, one input action, one editor signal connection, one `@export`. Hand-written,
  fully understood, every entity/edge enumerated in a `expected.json` golden. Used by unit tests.
- **`coupled/`** — deliberately exercises the smells: a god-autoload touched by many scripts, a
  dangling editor connection (method removed), a dead `@export`, a group called with no members, an
  undefined input action, a dynamic `load()` path. Drives the M2 integrity-report golden tests.
- **`real/`** *(gitignored, optional)* — a path to a checked-out real open-source Godot game for
  smoke/perf testing locally (don't commit; document a couple of suggested repos).

## Golden-test approach

```
testdata/fixtures/<name>/
├── <the godot project files>
└── golden/
    ├── nodes.json        # expected nodes (sorted, stable)
    ├── edges.json        # expected edges (sorted, stable)
    ├── boundaries.json   # expected boundary points
    └── findings.json     # expected integrity findings (M2+)
```

- Tests run the pipeline over the fixture and diff against `golden/*`.
- An `UPDATE_GOLDEN=1` env regenerates goldens (review the diff like archi's snapshot review).
- Keep output **deterministic**: sort nodes/edges by ID; never emit absolute paths, wall-clock
  times, or map-iteration-ordered lists.

## Match-key & resolution fixtures

`testdata/matchkey_fixtures.yml` and `testdata/resolution_fixtures.yml` — small, targeted cases that
lock normalization and stitching behaviour (the archi `link_identity_fixtures.yml` pattern). Each
entry: an input snippet/edge + the expected match key / resolved target / resolution strategy.

## Determinism rules (write once, enforce forever)

- Sort every collection before serialising.
- Strip the project root prefix; emit only `res://…` and repo-relative paths.
- No timestamps in golden files.
- Pin Godot fixture format version (`format=3`, Godot 4.x).

## Tasks

- [x] Build `testdata/fixtures/minimal/` (valid, loadable in Godot 4) + its `golden/`. _(project.godot + main.tscn + player.gd (@export, signal, editor connection) + game_state.gd autoload + icon.svg asset + jump input action; `golden/nodes.json` pins discovery output)_
- [ ] Build `testdata/fixtures/coupled/` with one instance of each M2 smell + `golden/findings.json`. _(deferred to M2: the smells (dangling connection, dead export, …) are only observable once the M1 extractors + M2 integrity engine exist; no analysis logic is in scope for 00. Lands with `plan/02-*`.)_
- [x] Implement the golden-diff test helper + `UPDATE_GOLDEN` regeneration. _(`internal/golden.AssertJSON`: deterministic indented-JSON diff, regenerates on `UPDATE_GOLDEN=1`, creates parent dirs)_
- [ ] Seed `matchkey_fixtures.yml` (filled out as M1/M2 land). _(the file already exists at `internal/model/testdata/matchkey_fixtures.yml` from `02`; further entries land with the extractors in M1/M2)_
- [x] Document the "regenerate & review goldens" workflow in `plan/` or repo docs. _(see "Regenerating goldens" below)_

## Regenerating goldens

Golden files are committed ground truth. After an intentional change to discovery
(or, later, extraction) output:

```
UPDATE_GOLDEN=1 go test ./...   # rewrites every golden/*.json in place
git diff -- testdata             # review the change like a snapshot review
```

Commit the regenerated goldens only once the diff is understood and expected. A
bare `go test ./...` (no env var) asserts against the committed goldens and fails
on any drift.

## Definition of done

`minimal/` round-trips to a stable golden; the golden-diff helper works and regenerates cleanly; the
fixture loads without error in a real Godot 4 editor (sanity check the fixture is legitimate).
