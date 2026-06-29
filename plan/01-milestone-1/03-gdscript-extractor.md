# 01.03 — GDScript extractor

`internal/extract/gdscript`. Parse `.gd` files with tree-sitter and emit the code-origin nodes,
edges, and boundary points. The archi analogue is the Elixir/Ruby AST analyzer (DESIGN §1).

## Parser wiring

- `smacker/go-tree-sitter` + the `tree-sitter-gdscript` grammar (PrestonKnopp), both pinned.
- Parse each file to a CST; run **tree-sitter queries** (`.scm` patterns) to capture the constructs
  below. Keep queries in `internal/extract/gdscript/queries/` so they're reviewable and testable.
- One parse per file; collect declarations + references in a single walk where practical.

## Declarations → nodes / identity

| Construct | Output |
|---|---|
| file | `script` node (created in discovery; enrich identity here) |
| `class_name Foo` | `class:Foo` node + `is class_name` flag on the script |
| `extends X` | `extends` edge (script → `class:X` or `res://base.gd`) |
| inner `class Bar:` | nested class in identity |
| `func name(args) -> T:` | symbol node `file::name`; record arity, return type, `static`, virtual (`_ready`, `_process`, …) |
| `signal sig(args)` | `declares_signal` → `file::signal:sig` node |
| `@export var x: T` | `exports_var` edge; record type → resolved against scene `binds_export` in M2 |
| `@onready var n = $Path` | node-reach (see below) at ready time |
| `const`, `enum`, member vars | identity (used by call/type resolution in M2) |
| `@rpc(...) func f` | `rpc_endpoint` ingress + `rpc:<class_or_*>:f` match key |

## References → edges / egress (the call-site detectors)

Modelled as archi's pluggable egress detectors — a registry of matchers over call expressions:

| Pattern | Edge (origin=code) | Boundary |
|---|---|---|
| `emit_signal("s", …)` / `s.emit(…)` | `emits_signal` → `signal:*:s` | egress `signal_emit` |
| `x.connect(c)` / `sig.connect(c)` / `connect("s", …)` | `connects_signal` | egress `signal_connect` |
| `func _on_*` / connected handler | — | ingress `signal_handler` |
| `_ready/_process/_physics_process/_input/_notification/...` | — | ingress `lifecycle` |
| `preload("res://…")` | `preloads` → `res:<path>` (static, high confidence) | egress `resource_load` |
| `load(expr)` / `ResourceLoader.load(expr)` | `loads_resource`; **dynamic string → unresolved**, confidence<1 | egress `resource_load` |
| `get_node("p")` / `$p` / `%Unique` / `get_node_or_null` | `references_node` → `nodepath:<expr>` | egress `node_reach` |
| `<AutoloadName>.member` (name in autoload set) | `accesses_autoload` → `autoload:<Name>` | egress `autoload_access` |
| `change_scene_to_file("res://…")` / `change_scene_to_packed` | `changes_scene_to` | egress `scene_change` |
| `add_to_group("g")` | `in_group` → `group:g` | — |
| `get_tree().call_group("g", …)` / `get_nodes_in_group` | `calls_group` → `group:g` | egress `group_call`; ingress `group_target` on members |
| `Input.is_action_*("a")` / `get_action_strength` | `uses_action` → `action:a` | ingress `input_handler` (enclosing) |
| `rpc("m", …)` / `rpc_id(id, "m", …)` / `m.rpc(…)` | `rpc_call` → `rpc:*:m` | egress `rpc_call` |
| `FileAccess.open` / `ConfigFile` / `user://…` | `file_io` | egress `file_io` |
| call to `Other.method()` / `class_name` type method | `calls` (resolved in M2) | — |

**Match-key construction** uses `internal/model` constructors (00.02). Emitter type for signals is
`*` unless statically inferable (typed var, `self`, known `class_name`).

## Fidelity levers (DESIGN §6.1)

- Use `class_name`, `extends`, typed vars, and `@export` types to build a best-effort type lattice —
  enough to resolve most `emit`→handler and `Type.method` calls in M2.
- Where a target is a dynamic string or untyped `Variant`, emit the edge **unresolved** with the raw
  expression in `Evidence`/`properties` — it becomes a diagnostic, not a silent loss.

## Complexity metric (cheap, do it here)

While walking each `func`, compute cyclomatic complexity (count decision points) and store on the
symbol node's properties — reproduces `gdradon`'s value without the Python dep (DESIGN §6.1).

## Tasks

- [ ] Wire tree-sitter-gdscript; smoke-test parse on `minimal/` scripts.
- [ ] Author tree-sitter queries for declarations (class_name, extends, func, signal, @export, @rpc).
- [ ] Implement the call-site detector registry for every reference row above.
- [ ] Emit ingress/egress boundary points.
- [ ] Cyclomatic complexity per func.
- [ ] Mark dynamic/untyped targets as unresolved with evidence.
- [ ] Goldens for `minimal/` + `coupled/` script edges & boundaries.

## Definition of done

Every code-origin edge type and boundary type from the tables is emitted for the fixtures (matching
golden), dynamic targets are flagged unresolved, and per-function complexity is recorded.
