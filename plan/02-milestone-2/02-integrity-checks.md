# 02.02 — Integrity rule catalog

`internal/analyze` (integrity rules). Each rule is a pure function over the resolved graph that
emits `report.Finding`s. Rules are independent and registered, so adding one is trivial (archi's
plugin-registry ethos). This catalog *is* the v1 product.

## Finding shape

```go
type Finding struct {
    Rule     string         // stable id, e.g. "dangling_connection"
    Severity string         // error | warning | info
    Title    string
    Message  string         // human sentence with the specifics
    NodeID   string
    EdgeID   int64
    Evidence model.Evidence // file:line (+ snippet)
    Detail   map[string]any // what the graph says; suggested fix; related ids
}
```

## Rule catalog

### Editor↔code integrity (the headline)

| Rule | Fires when | Severity | Why it matters |
|---|---|---|---|
| `dangling_connection` | a `.tscn` `[connection]` targets a method the `to` node's script doesn't define | error | the signal fires into nothing — silent runtime no-op |
| `connection_to_missing_signal` | `[connection]` references a signal the `from` node's type doesn't declare | error | broken wiring after a refactor |
| `dead_export` | `@export var` never assigned in any scene/inspector and never set in code | warning | dead config knob; confusion |
| `export_type_mismatch` | inspector-assigned `binds_export` value type ≠ declared `@export` type | warning | likely a stale scene after a type change |
| `undefined_action` | `Input.is_action_*("x")` where `x` ∉ `project.godot [input]` | error | input silently never triggers |
| `unused_action` | an input action defined but referenced nowhere | info | dead config |
| `empty_group_call` | `call_group("g")` / `get_nodes_in_group("g")` where nothing ever joins `g` | warning | broadcast into the void |
| `unused_group` | a group joined but never called/queried | info | possibly dead, or missing logic |
| `unresolved_load` | dynamic `load()`/`change_scene_to_file` path can't be resolved | info | can't verify the target exists — flag, don't fail |
| `missing_resource` | a static `res://`/`uid://` reference whose target file is absent | error | broken dependency |

### Signal / node hygiene

| Rule | Fires when | Severity |
|---|---|---|
| `unused_signal` | `signal` declared, never emitted and never connected | info |
| `unhandled_signal` | signal emitted but no connection/handler anywhere | info |
| `fragile_nodepath` | `get_node`/`$` reaches up/across the tree beyond a threshold distance | warning |
| `nodepath_unresolved` | a NodePath that can't be located in the owning scene tree | warning |

### Dead weight

| Rule | Fires when | Severity |
|---|---|---|
| `orphan_script` | a `.gd` attached to no scene_node, not an autoload, not extended, not preloaded | info |
| `orphan_scene` | a `.tscn` instanced nowhere and not the main scene / not a level entry | info |
| `orphan_asset` | an asset with no inbound `uses_asset` | info |

## Suppression & config

- `godarch.yml`: disable rules, set per-rule severity, ignore globs (e.g. `addons/**`), and
  per-finding suppression (rule + node id). Mirrors a linter's config.
- Some "orphans" are legitimately entry points (a level loaded only by name at runtime) — support an
  allowlist and lean toward `info` not `error` for orphan rules to keep signal-to-noise high.

## False-positive discipline (the make-or-break)

The product dies if it cries wolf. For every rule:
- Prefer `unresolved`/`info` over `error` when the analysis is uncertain (dynamic loads, untyped
  receivers). An honest "can't verify" beats a wrong "broken".
- Unit-test each rule with both a positive fixture (must fire) and a negative (must stay silent),
  drawn from `coupled/` and `minimal/` respectively.

## Tasks

- [ ] Define `report.Finding` + a rule registry interface.
- [ ] Implement each rule above with positive + negative fixture tests.
- [ ] Wire `godarch.yml` rule config/severity/suppression.
- [ ] Orphan-rule allowlist for runtime-only entry points.
- [ ] `findings.json` golden for `coupled/`; empty-report assertion for `minimal/`.

## Definition of done

Every catalog rule is implemented and tested both ways; `coupled/` produces exactly the expected
findings (golden); `minimal/` produces none; severities/suppression configurable.
