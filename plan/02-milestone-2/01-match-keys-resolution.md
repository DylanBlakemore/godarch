# 02.01 — Resolution / stitching layer

`internal/resolve`. Turn M1's raw edges (targets held as match keys / raw expressions) into resolved
edges pointing at real node IDs. The archi analogue is the V2 stitcher (DESIGN §1, §4). This layer is
reused by integrity (M2), coupling (M4), and docs drift (M5) — get it right and general.

## Inputs / outputs

- **In:** the M1 `Project` (nodes, raw edges, boundary points, UID map, scene node trees).
- **Out:** edges with `Resolved=true` + concrete `TargetID` and a `Confidence`/`Strategy`; plus
  `Project.Unresolved` — every edge that couldn't resolve, each with a machine-readable reason.

## Resolution index

Build lookup indexes once, then resolve in a pass (archi's match-key index pattern):

- `byClassName: class:Foo → script node`
- `byResPath: res://… → file node` (+ UID map already applied)
- `bySignal: (owning script, signal name) → declares_signal node`
- `byMethod: (script, method name/arity) → symbol node`
- `byAutoload / byAction / byGroup / byLayer: name → concept node`
- per-scene `sceneTree: NodePath → scene_node` (with `%UniqueName` and `owner` rules)

## Resolvers (one per edge family)

| Edge | Resolution | Strategy / confidence |
|---|---|---|
| `attaches_script`, `instances`, `uses_asset`, `preloads`, `imports` | path/UID → file node | exact (1.0) |
| `loads_resource` (static string) | path → node | exact; **dynamic string → unresolved** (reason: dynamic_path) |
| `extends` | `class:X` → script, or path → script | exact; built-in/native class → keep as `class:` node (resolved-to-engine) |
| `calls` | (type lattice + method index) → symbol | exact when type known; else best-effort or unresolved (reason: untyped_receiver) |
| `emits_signal` → `connects_signal`/handler | match `signal:T:name` to declared signal + connected handlers | exact when emitter type known; variant when `*`; record fan-out |
| `connects_signal` (editor `[connection]`) | resolve `from`/`to` NodePaths to scene_nodes; resolve `method` against the `to` node's attached script | **the dangling-connection check** lives on failure here |
| `references_node` / editor NodePath | NodePath → scene_node via `sceneTree`; compute reach distance | exact / unresolved (reason: nodepath_unresolved); record cross-tree distance |
| `binds_export` ↔ `exports_var` | match scene-assigned `@export` value to the script's declared var | unmatched export var = dead-export candidate |
| `accesses_autoload`, `uses_action`, `in_group`/`calls_group`, `uses_layer` | name → concept node | exact; missing concept → unresolved (reason: undefined_<kind>) |
| `rpc_call` → `rpc_endpoint` | `rpc:T:method` match | exact when type known; else variant |
| `changes_scene_to` | path → scene node | exact; dynamic → unresolved |

## Unresolved reasons (the diagnostic taxonomy)

`dynamic_path`, `untyped_receiver`, `nodepath_unresolved`, `missing_method`, `missing_signal`,
`undefined_action`, `undefined_group`, `undefined_autoload`, `missing_scene`, `binary_resource`
(`.scn`/`.res` not text-readable without Godot-assist). Each unresolved edge carries one.

## Optional Godot-assisted resolution (evaluate here)

For binary resources and hard symbol cases, optionally shell out to a headless Godot to read the
resolved dependency/UID data (DESIGN §6.1). Gate behind a flag; never required. Decide in this
milestone whether it earns its keep or stays a 99 item.

## Overrides

Honour `godarch.yml` (DESIGN §6.4): manual `load` path mappings, custom signal/event-bus
conventions, ignore globs. Plus `resolution_fixtures.yml` as locked ground truth.

## Tasks

- [ ] Build the resolution indexes.
- [ ] Implement each resolver family above; attach strategy + confidence.
- [ ] NodePath resolver (relative, absolute, `%UniqueName`, `owner`) with tests.
- [ ] Signal emit→handler + editor-connection→method resolution.
- [ ] `@export`↔`binds_export` matching.
- [ ] Unresolved-reason taxonomy + population of `Project.Unresolved`.
- [ ] Wire `godarch.yml` overrides + `resolution_fixtures.yml`.
- [ ] (Spike) Godot-assisted resolution; decide keep/defer.

## Definition of done

Resolution flips all resolvable fixture edges to resolved with correct targets; every unresolved
edge has a correct reason; signal/connection/export resolution pass their fixtures; results are
deterministic.
