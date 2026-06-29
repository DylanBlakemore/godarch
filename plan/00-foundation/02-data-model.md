# 00.02 — Data model

The contract every extractor emits and every analysis consumes. Mirrors archi's generic
node/edge/typed-properties core (DESIGN §1, §4). Lives in `internal/model`, pure types only.

## Identifiers

A single, deterministic, human-readable ID scheme. IDs are the join keys for resolution and the
anchors docs reference (DESIGN §8a). All paths are normalised to `res://…`.

| Entity | ID form | Example |
|---|---|---|
| Script / scene / resource / asset (a file) | the `res://` path | `res://player/player.gd` |
| Symbol inside a script | `<file>::<symbol>` | `res://player/player.gd::take_damage` |
| Signal declared in a script | `<file>::signal:<name>` | `res://player/player.gd::signal:died` |
| Scene-internal node | `<scene>::<NodePath>` | `res://ui/hud.tscn::HBox/HealthBar` |
| Autoload | `autoload:<Name>` | `autoload:GameState` |
| Input action | `action:<name>` | `action:jump` |
| Group | `group:<name>` | `group:enemies` |
| Collision layer | `layer:<index>` (+ name prop) | `layer:3` |
| Class (class_name / built-in / native) | `class:<Name>` | `class:Player` |
| GDExtension | `ext:<path>` | `ext:res://bin/my_ext.gdextension` |
| Doc | the repo-relative path | `docs/systems/combat.md` |

**UID handling:** Godot scene/resource headers carry `uid://…`. Discovery builds a `UID↔res://`
map (from `.godot/uid_cache.bin` if present, else by scanning headers). All `uid://` references are
resolved to `res://` IDs at extraction time so the rest of the system never sees UIDs.

## `Node`

```go
type Kind string // script, scene, scene_node, resource, autoload, asset,
                  // action, group, layer, signal, class, extension, doc

type Node struct {
    ID         string         // canonical, per the table above
    Kind       Kind
    Path       string         // res:// path or repo-relative for docs ("" for pure concepts)
    Line       int            // 0 if N/A
    Identity   map[string]any // kind-specific: {name, node_type, parent_path, arity, ...}
    Properties map[string]any // derived/analytic: {fan_in, in_cycle, complexity, ...}
}
```

## `Edge`

```go
type EdgeType string // see the ~22 types in DESIGN §3.2:
// instances, attaches_script, extends, connects_signal, emits_signal, declares_signal,
// calls, references_node, exports_var, binds_export, accesses_autoload, loads_resource,
// changes_scene_to, in_group, calls_group, uses_action, uses_layer, rpc_call, rpc_endpoint,
// animates, uses_asset, imports, uses_shader, uses_material, uses_theme

type Origin string // code | editor | config | docs

type Edge struct {
    Type       EdgeType
    SourceID   string
    TargetID   string   // may be a *match key* (unresolved) until resolve runs
    Origin     Origin   // where it was declared — the C/E column from DESIGN §3.2
    Resolved   bool     // did TargetID resolve to a real Node?
    Confidence float64  // 1.0 exact; <1 for variant/fuzzy/dynamic
    Evidence   Evidence // file + line where this edge was observed
    Properties map[string]any // {signal_name, method, flags, binds, path_expr, ...}
}

type Evidence struct { File string; Line int; Snippet string }
```

## `BoundaryPoint` (ingress / egress)

Boundary points are the typed entry/exit semantics, attached to a script symbol (DESIGN §4).

```go
type Direction string // ingress | egress

type BoundaryType string
// ingress: lifecycle, signal_handler, editor_connection, rpc_endpoint, group_target,
//          input_handler, notification
// egress:  signal_emit, signal_connect, resource_load, scene_change, autoload_access,
//          node_reach, group_call, rpc_call, file_io

type BoundaryPoint struct {
    Direction Direction
    Type      BoundaryType
    NodeID    string     // the owning symbol (script::method)
    MatchKey  MatchKey   // egress targets; ingress public surface
    Evidence  Evidence
    Meta      map[string]any
}
```

## `MatchKey` — the normalization contract (the secret sauce)

A deterministic string; two boundary points / an edge+target link iff keys are equal (DESIGN §4).

```go
type MatchKey string
```

| Concept | Form | Notes |
|---|---|---|
| Signal | `signal:<emitter_type_or_*>:<name>` | `*` when emitter type can't be statically resolved |
| Resource/scene | `res:<normalized_path>` | `uid://` and relative paths canonicalised first |
| Action | `action:<name>` | from `Input.is_action_*` and `project.godot [input]` |
| Group | `group:<name>` | from `add_to_group`/`groups=[…]` and `call_group` |
| Autoload | `autoload:<Name>` | global name from `project.godot [autoload]` |
| RPC | `rpc:<class_or_*>:<method>` | matches `@rpc` endpoint to `rpc()/rpc_id()` call |
| Node path | resolved to a scene-node ID where possible, else `nodepath:<expr>` | fragile-reach analysis uses the raw expr |

Normalisation rules (document and unit-test each, archi-style fixtures): path canonicalisation,
relative→`res://`, NodePath resolution against the owning scene tree, `%UniqueName` resolution,
case sensitivity. These rules are the part most likely to need override files later (DESIGN §6.4).

## Container

```go
type Project struct {
    Root        string
    Nodes       map[string]*Node
    Edges       []*Edge
    Boundaries  []*BoundaryPoint
    Unresolved  []*Edge          // edges whose TargetID never resolved (the diagnostic list)
    UIDMap      map[string]string // uid:// → res://
    GodotVersion string
}
```

## Tasks

- [ ] Define all types above in `internal/model`.
- [ ] Implement ID constructors + parsers (`model.ScriptID`, `model.SymbolID`, `model.SceneNodeID`, …) with tests.
- [ ] Implement `MatchKey` constructors per the table, with a documented normalisation function each.
- [ ] Round-trip serialization (JSON) test covering every Kind / EdgeType / BoundaryType.
- [ ] A fixtures file (`testdata/matchkey_fixtures.yml`) locking expected match-key outputs — archi-style ground truth.

## Definition of done

The model can represent every node, edge, and boundary kind from DESIGN §3, IDs round-trip, and the
match-key fixtures pass. Nothing in `internal/model` imports another `internal/*` package.
