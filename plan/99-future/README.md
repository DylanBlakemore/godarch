# 99 — Future

Items that extend reach once the core product (M0–M3) and the analysis story (M4–M5) are solid.
Each is largely independent and emits into the **same graph** (DESIGN §6.3), so they slot in without
disturbing the core. Not yet sequenced; promote to a numbered milestone when prioritised.

## C# extractor

- Godot supports C# (Mono/.NET) with the same node model: `[Signal]`/`EmitSignal`, `GetNode`,
  `[Export]`, `Connect`, `partial class : Node`.
- From a Go core: extract via `tree-sitter-c-sharp` (consistent with the GDScript path), or shell
  out to a small Roslyn helper for full semantic fidelity.
- Crucially, C# edges land in the **same graph** and stitch with GDScript: a C# `[Export]` assigned
  in a `.tscn`, a GDScript signal handled in C#. The scene/config extractors already work regardless
  of script language.
- [ ] C# extractor emitting the model's edge/boundary types
- [ ] Cross-language resolution (C#↔GDScript) in the resolver
- [ ] Mixed-language fixture + goldens

## GDExtension / C/C++

- Parse `.gdextension` + the registered class table (and optionally headers) → `extension` nodes +
  native `class:` nodes usable from GDScript/C#.
- Calls into native classes from script become resolvable edges.
- [ ] `.gdextension` parsing + native class registration
- [ ] Resolve script→native references
- [ ] (Optional) header scan for richer native API

## Godot-assisted resolution (promote from spike)

- If the M2 spike proved valuable: a first-class mode that drives a headless Godot for ground-truth
  symbol/UID/dependency data and binary `.scn`/`.res` reading.
- [ ] Detect installed Godot / optional bundled headless binary
- [ ] Resolution fallback path using Godot's dependency DB
- [ ] Binary scene/resource reading

## CI mode / coupling-regression gate

- archi has a snapshot/diff harness worth copying. A `godarch ci` command that fails a build on new
  integrity errors or a coupling regression (instability/cycle count worsens vs baseline).
- Shares the snapshot/diff harness with M5.
- [ ] `godarch ci` with baseline + thresholds (errors, new cycles, instability deltas)
- [ ] Graph-diff output (added/removed entities & edges) for PR comments
- [ ] GitHub Action wrapper

## MCP / agent integration

- archi exposes (or plans to) an MCP surface. Expose godarch's graph to an LLM/agent for Q&A and
  refactor suggestions ("what depends on `CombatState`?", "propose a decoupling").
- [ ] MCP server exposing graph queries (node detail, blast radius, findings, flow patterns)
- [ ] Grounded refactor-suggestion tooling

## Other candidates (unsorted)

- Performance surface: `_process`/`_physics_process` hotspots + what they touch (DESIGN §5).
- AnimationPlayer/AnimationTree method-track extraction (invisible behaviour edges, DESIGN §3.2 `animates`).
- Theme/shader/material asset dependency depth analysis.
- Diff/trend view across commits (graph metrics over time).
- Localization key coverage (`.po`/`.csv` keys referenced vs defined).
