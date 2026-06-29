# 00.01 — Repo layout & package boundaries

## Module

- Module path (placeholder): `github.com/dylanblakemore/godarch` — adjust to the real repo host.
- Go version: latest stable (1.22+). cgo **required** (tree-sitter) — `CGO_ENABLED=1`.

## Directory tree

```
godarch/
├── cmd/
│   └── godarch/            # CLI entrypoint (cobra or stdlib flags)
│       └── main.go
├── internal/
│   ├── discovery/          # walk project, classify files, parse project.godot, UID↔path map
│   ├── model/              # core types: Node, Edge, BoundaryPoint, MatchKey, IDs (no deps on other internal pkgs)
│   ├── extract/            # extractors (M1)
│   │   ├── scene/          #   .tscn/.tres/.import/project.godot INI parser + edge extraction
│   │   └── gdscript/       #   tree-sitter-gdscript integration + edge extraction
│   ├── resolve/            # match-key resolution / stitching (M2)
│   ├── graph/              # build gonum graph from model; queries, projections, blast radius
│   ├── analyze/            # findings engines: integrity (M2), coupling/domains (M4)
│   ├── store/              # SQLite persistence + migrations
│   ├── report/             # findings model + formatters (text/json)
│   ├── docs/               # doc frontmatter parse + reference linter (M2 slice / M5)
│   └── config/             # godarch.yml overrides & ignore globs
├── ui/                     # Wails app (M3) — wraps internal/* ; frontend/ lives here
├── testdata/
│   └── fixtures/           # sample Godot projects + golden files
├── docs/                   # godarch's own developer docs (architecture notes)
├── DESIGN.md
├── plan/
├── go.mod
└── Taskfile.yml            # or mise tasks / Makefile
```

## Package boundaries (dependency direction)

Strict one-way dependencies — no cycles. `model` is the leaf everything imports.

```
model  ←  discovery, extract/*, resolve, graph, store, report, docs, config, analyze
graph  ←  resolve, analyze, report
store  ←  cmd, ui              (store imports model only)
resolve ← analyze
cmd/godarch → discovery, extract/*, resolve, graph, analyze, store, report, config
ui          → same set as cmd (Wails binds the same core)
```

Rules:
- `model` imports **nothing** from `internal/*`. Pure types + identifier helpers.
- Extractors depend on `model` only; they **emit** nodes/edges, they don't resolve or persist.
- `resolve` consumes raw extractor output + `model`; produces resolved edges + diagnostics.
- `graph` is the gonum adapter; analysis (`analyze`) and reporting query it.
- `cmd` and `ui` are the only orchestrators — they wire the pipeline. Keep them thin.

## Pipeline orchestration (the seam cmd & ui share)

Define one `Pipeline` type (in a small `internal/pipeline` package, or in `analyze`) that both the
CLI and Wails call:

```go
type Pipeline struct { /* config */ }
func (p Pipeline) Run(ctx context.Context, projectDir string) (*model.Project, error)
// Stages: Discover → Extract → Resolve → BuildGraph → (Analyze on demand)
```

Keeping this seam single-sourced is what lets the Wails UI (M3) be "just another front-end" with
zero core logic of its own.

## Tasks

- [ ] `go mod init`; commit `go.mod`, `.gitignore`, `LICENSE`, top-level `README.md`.
- [ ] Create the package directories with a `doc.go` stating each package's responsibility.
- [ ] Add `cmd/godarch/main.go` with a no-op `analyze` subcommand.
- [ ] Add an import-cycle guard to CI (`go vet` + a check that `model` imports no `internal/*`).
- [ ] Decide CLI framework (recommend `cobra` for subcommands; stdlib `flag` is fine for v0).

## Definition of done

`go build ./...` and `go vet ./...` pass; the package tree exists with documented responsibilities;
no import cycles; `godarch analyze` is a recognised (if empty) command.
