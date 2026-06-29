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

- [ ] Build `testdata/fixtures/minimal/` (valid, loadable in Godot 4) + its `golden/`.
- [ ] Build `testdata/fixtures/coupled/` with one instance of each M2 smell + `golden/findings.json`.
- [ ] Implement the golden-diff test helper + `UPDATE_GOLDEN` regeneration.
- [ ] Seed `matchkey_fixtures.yml` (filled out as M1/M2 land).
- [ ] Document the "regenerate & review goldens" workflow in `plan/` or repo docs.

## Definition of done

`minimal/` round-trips to a stable golden; the golden-diff helper works and regenerates cleanly; the
fixture loads without error in a real Godot 4 editor (sanity check the fixture is legitimate).
