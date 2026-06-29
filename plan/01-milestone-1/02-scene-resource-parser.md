# 01.02 — Scene / resource / config extractor

`internal/extract/scene`. The highest-value extractor: it surfaces the **editor-configured edges**
no code-only tool sees (DESIGN §2, §3.3). Build this **before** the GDScript extractor.

## The format

`.tscn`, `.tres`, `.import`, `project.godot`, `plugin.cfg`, `.gdextension` are all the same
INI-like text: `[section key=value …]` headers followed by `key = value` property lines, where
values are Godot literals (`ExtResource("id")`, `SubResource("id")`, `NodePath("…")`,
`Vector3(…)`, arrays, dicts, strings, numbers). Build one tolerant parser for all of them.

### Parser design

- Tokenise headers into `(section_type, attrs map)` and bodies into `key → raw_value`.
- A small **value parser** for the literal grammar: `ExtResource`/`SubResource` refs, `NodePath`,
  arrays `[…]`, dicts `{…}`, typed ctors (capture name + args), strings, numbers, bools.
- Be **lenient**: unknown sections/types are kept as opaque; never fail the whole file on one
  unparseable property — emit a diagnostic and continue (M1 exit criterion: no silent drops).

## What to extract from a `.tscn`

Resolve `ExtResource("id")` / `SubResource("id")` against the file's resource tables first, then:

| Source | Edge / output |
|---|---|
| `[ext_resource type="Script" path=…]` referenced as a node's `script =` | `attaches_script` (scene_node → script), origin=editor |
| `[ext_resource type="PackedScene" …]` used as `[node … instance=ExtResource(id)]` | `instances` (scene → scene), origin=editor |
| `[ext_resource type="Texture2D"/Audio/Font/...]` used in a property | `uses_asset` (scene/resource → asset), origin=editor |
| `[node … groups=["a","b"]]` | `in_group` (scene_node → `group:a`), origin=editor |
| property `= NodePath("…")` (incl. `@export` targets) | `references_node` / `binds_export`, origin=editor; record raw path for fragile-reach analysis |
| `[connection signal=… from=… to=… method=… flags=… binds=…]` | `connects_signal` (origin=editor) **and** an `editor_connection` ingress on the target method's match key |
| `[editable path="…"]` | property-override marker on an instanced sub-scene (record on the `instances` edge) |
| `[node … script=… ]` exported property values | `binds_export` (the inspector-assigned value of an `@export` var) |
| each `[node]` | a `scene_node` node with `name`, `type`, `parent`, owning scene |

Also emit, per scene: the internal node tree (parent edges) so NodePaths can later resolve against
it (M2). The scene root + `main_scene` marker feed the scene-flow graph (M4).

## What to extract from a `.tres`/`.res`

- `[ext_resource type="Script" …]` → the custom Resource's backing script (`attaches_script`).
- Nested `ExtResource`/`SubResource` → `loads_resource`/`uses_asset` edges (resource→resource/asset).
- A `.tres` whose `script` is a `class_name`'d Resource → link to that class.

## What to extract from `.import`

- `[deps] source_file=…`, `[remap] importer=…` → `imports` edge (asset ↔ `.import`); record importer
  + key params in identity (for import-settings-drift checks later).

## `plugin.cfg` / `.gdextension`

- `plugin.cfg` → mark the addon script as `@tool`/editor-plugin (identity flag).
- `.gdextension` → an `extension` node + the entry/library mapping (classes parsed in 99).

## Boundary points from scenes

- Every `[connection] … to=X method=M` → an **`editor_connection` ingress** on `X`'s script method
  `M`, with the signal as match key. This is the link M2 resolves against the actual GDScript method
  (dangling-connection detection).

## Tasks

- [ ] INI tokeniser + Godot value-literal parser (lenient, diagnostic-emitting).
- [ ] Resource-table resolution (`ExtResource`/`SubResource` id → path/subresource).
- [ ] `.tscn` extraction: instances, attaches_script, connections, groups, exports/binds, node-paths, asset refs, node tree.
- [ ] `.tres`/`.res` extraction: backing script, nested resource/asset refs.
- [ ] `.import` pairing → `imports` edges with importer params.
- [ ] `plugin.cfg` / `.gdextension` minimal extraction.
- [ ] `editor_connection` ingress emission.
- [ ] Goldens for `minimal/` + `coupled/` scene edges.

## Definition of done

The scene extractor emits every editor-origin edge type for the fixtures, the node tree per scene is
captured, and `[connection]`s become `editor_connection` ingress points — all matching golden.
