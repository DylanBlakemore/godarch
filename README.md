# godarch

Architecture intelligence for [Godot](https://godotengine.org/) projects.

godarch is a desktop app that statically analyses a Godot project — code **and**
scenes, resources, assets, and editor configuration — to build a typed
dependency graph of the whole game, then surfaces domain sprawl, coupling
hotspots, fragile wiring, and dead configuration.

See [`DESIGN.md`](DESIGN.md) for the architecture and [`plan/`](plan/) for the
milestone-by-milestone build plan.

## Status

Milestone 00 (foundation): repo skeleton, data model, storage, and a CLI that
proves the plumbing end to end. No analysis logic yet.

## Requirements

- Go 1.22+ (developed on 1.25).
- `CGO_ENABLED=1` and a C toolchain — tree-sitter (used for GDScript parsing in
  milestone 01) requires cgo. Because of cgo there is no effortless
  cross-compilation; each OS is built on its own CI runner.

## Layout

```
cmd/godarch/     CLI entrypoint
internal/        core packages (model, discovery, extract, resolve, graph,
                 analyze, store, report, docs, config)
ui/              Wails desktop app (milestone 03)
testdata/        fixture Godot projects + golden files
```

`internal/model` is the leaf package: it imports nothing from `internal/*`.
A guard test (`internal/model`) enforces this to keep the dependency graph
acyclic.

## Development

Dev tooling and task running are managed with [mise](https://mise.jdx.dev).
`mise install` provisions the pinned Go toolchain, `golangci-lint`, `gofumpt`,
and `goimports`. Tasks (`mise tasks` to list):

```sh
mise run build      # go build ./...
mise run test       # go test ./...
mise run test:race  # go test -race ./...
mise run lint       # golangci-lint run ./... (whole repo)
mise run fmt        # gofumpt + goimports, in place
mise run fmt:check  # fail if anything is unformatted
mise run ci         # fmt:check + lint + test:race — what CI gates on
```

Without mise the same gates run directly:

```sh
go build ./... && go vet ./... && go test -race ./...
golangci-lint run ./...
```

**Lint runs unscoped (whole repo) before every push**, not just changed files —
some checks depend on repo-wide context. CI (`.github/workflows/ci.yml`) runs
the same gates on a per-OS matrix (Linux, macOS, Windows) with `CGO_ENABLED=1`,
because cgo rules out cross-compilation. `go vet` plus the model-purity guard
test enforce the acyclic dependency direction.

## Wails (milestone 03)

The desktop UI is built with [Wails](https://wails.io). Before starting
milestone 03, install the CLI and confirm the local toolchain:

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails doctor   # must report a healthy environment (Go, npm, platform deps)
```

The buildable `ui/` skeleton is added in milestone 03; `mise run ui:dev` /
`mise run ui:build` wrap `wails dev` / `wails build` once it exists.

## CLI

```sh
godarch analyze <project-dir>
```

In milestone 00 `analyze` is recognised but does no real work yet.
