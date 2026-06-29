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

- [ ] Add task runner with the targets above.
- [ ] Configure `golangci-lint` (`.golangci.yml`) + `gofumpt`.
- [ ] Add the GitHub Actions CI matrix; get it green on all three OSes.
- [ ] Add the import-cycle / `model`-purity guard to CI.
- [ ] Install Wails; commit an empty buildable `ui/` skeleton; `wails doctor` documented in README.
- [ ] Pin tree-sitter binding + grammar versions in `go.mod` / vendor.

## Definition of done

CI is green on macOS, Windows, and Linux with cgo enabled; lint/fmt gates work; the Wails skeleton
builds locally on at least the primary dev OS.
