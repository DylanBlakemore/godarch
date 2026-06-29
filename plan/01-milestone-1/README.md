# 01 — Project model & extraction

Turn a Godot project on disk into a populated, file-grained typed graph. This is the substrate the
integrity report (M2) and every later analysis read from.

## Goal

After M1, `godarch analyze <project>` produces a SQLite graph containing every node and **raw edge**
from DESIGN §3 — scripts, scenes, resources, assets, autoloads, named concepts, and the edges
between them. Edges may still be *unresolved* (target held as a match key); resolution is M2.

## Build order (important)

Per ROADMAP: **scene + project-config extractors first, GDScript extractor second.** The
editor-configured edges are where godarch's unique value lives, the INI parser is simpler, and
building it first proves the data model on easy ground before the tree-sitter work.

1. Discovery (`01-discovery.md`)
2. Scene/resource/config extractor (`02-scene-resource-parser.md`)
3. GDScript extractor (`03-gdscript-extractor.md`)
4. Graph assembly & persistence (`04-graph-assembly.md`)

## Scope

**In:** all extraction → raw nodes + raw edges + boundary points, persisted. CLI dump/query.

**Out:** resolution/stitching (M2), integrity findings (M2), any UI (M3), coupling/domains (M4).
GDScript symbol *resolution* across files is M2 — M1 extracts what each file declares and references.

## Deliverables

- `godarch analyze` populates `nodes`, `edges`, `boundaries` for a real project.
- `godarch graph --file res://x.gd` prints the node + its in/out edges (text + `--json`).
- Goldens for `minimal/` and `coupled/` fixtures pass for nodes/edges/boundaries.

## Master checklist

- [ ] Discovery walks the project, classifies files, builds the UID↔path map (`01`)
- [ ] `project.godot` parsed: autoloads, input map, layers, main scene, global groups (`01`)
- [ ] Scene/resource INI parser handles `.tscn`/`.tres`/`.import`/`plugin.cfg`/`.gdextension` (`02`)
- [ ] Scene extractor emits: instances, attaches_script, connections, groups, exports, node-paths, asset refs (`02`)
- [ ] tree-sitter-gdscript wired via cgo; grammar pinned (`03`)
- [ ] GDScript extractor emits: extends, class_name, signals, calls, emits, connects, loads, autoload access, node reach, groups, actions, rpc, exports (`03`)
- [ ] Boundary points (ingress/egress) emitted for both extractors (`03`, `02`)
- [ ] Graph assembled & persisted; CLI dump/query works (`04`)
- [ ] Goldens pass for `minimal/` and `coupled/` (nodes/edges/boundaries) (`04`)

## Exit criteria

1. Every DESIGN §3.2 edge type is emitted by *some* extractor (verified against `coupled/`).
2. Running on a real mid-size Godot project completes in seconds and the node/edge counts are
   plausible on manual spot-check.
3. No edge silently dropped — anything unparseable is recorded as a diagnostic, not lost.

## Docs in this milestone

| Doc | Covers |
|---|---|
| [`01-discovery.md`](01-discovery.md) | File walk, classification, `project.godot`, UID map |
| [`02-scene-resource-parser.md`](02-scene-resource-parser.md) | INI-format parser + editor-edge extraction |
| [`03-gdscript-extractor.md`](03-gdscript-extractor.md) | tree-sitter integration + code-edge extraction |
| [`04-graph-assembly.md`](04-graph-assembly.md) | Assemble raw graph, persist, CLI dump/query |
