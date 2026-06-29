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

## Build & test

```sh
go build ./...
go vet ./...
go test ./...
```

## CLI

```sh
godarch analyze <project-dir>
```

In milestone 00 `analyze` is recognised but does no real work yet.
