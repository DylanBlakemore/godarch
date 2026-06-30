# 00.04 — Tooling & CI

## Local task runner

Reuse the `mise` habit from archi, or a `Taskfile.yml` (go-task). Targets:

```
build      go build ./...
test       go test ./...
test:race  go test -race ./...
lint       golangci-lint run
fmt        gofumpt -w . && goimports -w .
ci         fmt-check + lint + test:race
ui:dev     wails dev          (M3)
ui:build   wails build        (M3)
```

## Lint / format

- `golangci-lint` with a sane preset (`govet`, `staticcheck`, `errcheck`, `ineffassign`,
  `revive`). Mirror the global preference from archi: **lint runs unscoped (whole repo) before any
  push**, not just changed files.
- `gofumpt` + `goimports` for formatting; `fmt-check` fails CI on drift.

## The cgo reality

tree-sitter requires `CGO_ENABLED=1` and a C toolchain. Consequences (DESIGN §6.1):
- **No effortless cross-compilation.** Build each OS on its own runner.
- CI and release both run a **per-OS matrix**: `macos-latest`, `windows-latest`, `ubuntu-latest`.
- Pin the tree-sitter Go binding + the `tree-sitter-gdscript` grammar commit; vendor or `go.mod`-lock.
- Windows: ensure a C compiler is available (the GitHub Windows runner has one; document it).

## CI (GitHub Actions sketch)

```yaml
strategy:
  matrix:
    os: [ubuntu-latest, macos-latest, windows-latest]
steps:
  - checkout
  - setup-go (with cache)
  - run fmt-check
  - run golangci-lint            # whole-repo
  - run go build ./...
  - run go test -race ./...
  - (import-cycle / model-purity guard)
```

Release workflow (M3) extends this matrix with `wails build`, code-signing, and artifact upload.

## Wails toolchain (skeleton only in 00)

- Install Wails CLI; `wails doctor` must pass on the dev machine.
- Decide frontend framework now so M3 isn't blocked: recommend **Svelte or vanilla TS + Vite**
  (light), with **Cytoscape.js** (or Sigma.js for very large graphs) for graph rendering.
- Create an empty `ui/` Wails project that builds and shows a "hello" window; don't wire core yet.

## Tasks

- [x] Add task runner with the targets above. _(`mise.toml`: build, test, test:race, lint, fmt, fmt:check, ci, plus ui:dev/ui:build stubs; `mise run ci` is green locally)_
- [x] Configure `golangci-lint` (`.golangci.yml`) + `gofumpt`. _(golangci-lint v2 config: default set + curated revive; gofumpt + goimports as v2 formatters; the `std-error-handling` exclusion covers CLI `fmt.Fprint*` writes)_
- [x] Add the GitHub Actions CI matrix; get it green on all three OSes. _(workflow at `.github/workflows/ci.yml` — 3-OS matrix, cgo, fmt-check, lint, build, vet, race; confirmed green on `windows-latest`, `macos-latest`, and `ubuntu-latest` on the latest `main` push — run 28431852698.)_
- [x] Add the import-cycle / `model`-purity guard to CI. _(CI runs `go vet ./...` + `go test -race ./...`, which executes `internal/model/purity_test.go`)_
- [ ] Install Wails; commit an empty buildable `ui/` skeleton; `wails doctor` documented in README. _(the `wails doctor` workflow is documented in README; the CLI install + buildable `ui/` skeleton are deferred — the `wails` CLI isn't present in this environment and a "hello" window can't be verified headlessly. M3-facing.)_
- [ ] Pin tree-sitter binding + grammar versions in `go.mod` / vendor. _(deferred to M1: no tree-sitter dependency exists yet, so pinning would add an unused module. Lands with the GDScript extractor in `plan/01-milestone-1/03-gdscript-extractor.md`.)_

## Definition of done

CI is green on macOS, Windows, and Linux with cgo enabled; lint/fmt gates work; the Wails skeleton
builds locally on at least the primary dev OS.
