# 00 — Foundation

Scaffolding and contracts that every later milestone depends on. **No analysis logic yet** — the
goal is a clean skeleton, the data model, the store, the test harness, and a CLI that proves the
plumbing works end to end on a real Godot project.

## Goal

A Go project that can: walk a Godot project directory, classify its files, persist a (still mostly
empty) graph to SQLite, and print a summary. Plus the type contracts and test fixtures that make
milestones 01+ fast and safe to build.

## Scope

**In:**
- Repo layout & Go module, package boundaries (`01-repo-layout.md`).
- The core data model: `Node`, `Edge`, `BoundaryPoint`, `MatchKey`, identifier schemes (`02-data-model.md`).
- Storage: SQLite schema + migrations + the in-memory `gonum` graph (`03-storage.md`).
- Tooling, build, lint, test, and the cgo/Wails CI matrix (`04-tooling-ci.md`).
- Test fixtures: a small sample Godot project + golden-test approach (`05-test-fixtures.md`).
- A CLI skeleton (`godarch analyze <path>`) that wires discovery → store → summary.

**Out:**
- Any extractor logic (M1), resolution (M2), UI (M3), analysis (M4+). Stubs/interfaces only.

## Deliverables

- `cmd/godarch` CLI: `godarch analyze <project-dir>` prints `{scripts, scenes, resources, assets,
  autoloads}` counts and writes a SQLite db.
- `internal/model` compiles with the full type contract and round-trips through `internal/store`.
- CI green on macOS + Windows + Linux (cgo builds).
- `testdata/fixtures/minimal/` — a tiny but valid Godot project used by tests.

## Master checklist

- [x] Repo layout & module created; `go build ./...` passes (`01`)
- [x] Package boundaries documented and enforced (no import cycles) (`01`)
- [ ] `Node` / `Edge` / `BoundaryPoint` / `MatchKey` types + identifier helpers (`02`)
- [ ] Identifier scheme documented and unit-tested (`02`)
- [ ] SQLite schema + migrations; `store.Save`/`store.Load` round-trip test (`03`)
- [ ] In-memory `gonum` graph builder from stored nodes/edges (`03`)
- [ ] `mise`/`Taskfile` (or `make`) targets: build, test, lint, fmt (`04`)
- [ ] CI matrix (macOS/Windows/Linux) green with cgo (`04`)
- [ ] Wails toolchain installs and `wails doctor` passes (skeleton only) (`04`)
- [ ] Minimal fixture Godot project committed; loads without error (`05`)
- [ ] Golden-test harness in place (`05`)
- [ ] `godarch analyze testdata/fixtures/minimal` prints correct counts

## Exit criteria

1. A fresh clone builds on all three OSes via CI.
2. `godarch analyze <real godot project>` runs in seconds and prints sane file counts.
3. The data model can represent every node/edge kind from DESIGN §3 (even though nothing populates
   most of them yet) — verified by a serialization round-trip test using hand-built fixtures.

## Docs in this milestone

| Doc | Covers |
|---|---|
| [`01-repo-layout.md`](01-repo-layout.md) | Go module, directory tree, package boundaries |
| [`02-data-model.md`](02-data-model.md) | Node/Edge/Boundary/MatchKey types, identifiers |
| [`03-storage.md`](03-storage.md) | SQLite schema, migrations, in-memory gonum graph |
| [`04-tooling-ci.md`](04-tooling-ci.md) | Build/test/lint, cgo, CI matrix, Wails toolchain |
| [`05-test-fixtures.md`](05-test-fixtures.md) | Sample projects, golden tests |
