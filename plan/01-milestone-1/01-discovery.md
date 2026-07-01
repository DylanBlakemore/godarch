# 01.01 — Discovery

`internal/discovery`. Walk the project, classify files, parse `project.godot`, build the UID↔path
map. Produces the skeleton `model.Project` (nodes for files + global concepts) that extractors fill.

## Locate the project

- Project root = the directory containing `project.godot`. Accept a path; search upward if a
  subdir is given.
- Respect ignore globs from `internal/config` (`godarch.yml`): always ignore `.godot/`, `.git/`,
  and (configurably) `addons/` and export/build dirs.

## File classification

Walk the tree, classify by extension/content → emit a `Node` per file:

| Pattern | Kind | Notes |
|---|---|---|
| `*.gd` | script | GDScript |
| `*.cs` | script | C# (extracted in 99; node still created in M1) |
| `*.tscn`, `*.scn` | scene | `.scn` binary — flag, may need Godot-assisted read |
| `*.tres`, `*.res` | resource | `.res` binary — same caveat |
| `*.gdshader`, `*.gdshaderinc` | asset (shader) | |
| `*.png/.jpg/.svg/.ogg/.wav/.mp3/.glb/.gltf/.ttf/...` | asset | type recorded in identity |
| `*.import` | (not a node) | pairs with its asset → `imports` edge |
| `*.gdextension` | extension | native module registration (parsed in 99) |
| `*.md` and configured doc globs | doc | for the docs linter (M2 slice) |
| `project.godot` | (config, parsed below) | |
| `plugin.cfg` | (config) | editor plugin metadata |

## Parse `project.godot`

INI-like (use the same parser as `02-scene-resource-parser.md`). Extract and emit nodes/edges:

- `[autoload]` → one `autoload:<Name>` node each; edge `attaches_script`/`instances` to the target
  `res://` path. **This is the global-coupling backbone.**
- `[input]` → one `action:<name>` node per action; record the events (keys/buttons) in identity.
- `[layer_names]` (2d/3d physics/render) → `layer:<index>` nodes with names.
- `application/run/main_scene` → mark that scene node as the **scene-flow root**.
- `[global_group]` (Godot 4.3+) → predeclared `group:<name>` nodes.
- Engine version / features → `Project.GodotVersion`, `meta`.

## UID↔path map

- If `.godot/uid_cache.bin` exists, parse it (binary; document the format or fall back).
- Otherwise scan every `.tscn`/`.tres` header for `uid="uid://…"` and map to its path.
- Populate `Project.UIDMap`. Extractors resolve every `uid://` to a `res://` ID via this map.

## Output

A `model.Project` with: a node per file, per autoload, per action, per layer, per declared group;
`UIDMap`; `GodotVersion`; ignore-filtered file list ready for the extractors.

## Tasks

- [x] Project-root location + ignore-glob filtering.
- [x] File walker + classifier → file nodes (table above).
- [x] `project.godot` parsing → autoload/action/layer/group nodes + main-scene marker.
- [x] UID map: `.godot/uid_cache.bin` parse with header-scan fallback.
- [x] `.import`↔asset pairing recorded for the `imports` edge (emitted in 02).
- [x] Tests against `minimal/` (counts + the autoload/action/main-scene nodes exist).

## Definition of done

`discovery.Run(dir)` returns a `Project` whose node set matches the `minimal/` golden for files +
global concepts, with a correct UID map and identified main scene.
