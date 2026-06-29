# godarch — Roadmap

The arc from empty repo to a shippable desktop tool, then to the broader archi-style analysis.
Each milestone is independently shippable and unlocks the next.

## The arc

```
00 Foundation ──► 01 Extraction ──► 02 Resolution & Integrity ──► 03 Desktop app
   (contracts)      (typed graph)       (the v1 headline)            (ship it)
                                              │
                                              ▼
                                  04 Coupling & Domains ──► 05 Docs drift
                                  (archi's core pitch)       (grounded LLM)
                                              │
                                              ▼
                                        99 Future (C#, C++, CI gate, Godot-assist)
```

## What each milestone delivers and unlocks

| # | Milestone | Delivers (user-visible) | Unlocks |
|---|---|---|---|
| 00 | Foundation | CLI that walks a project and prints file/entity counts | The data model, store, and test harness everything else needs |
| 01 | Project model & extraction | A populated typed graph (CLI: dump nodes/edges, query by file) | Anything that reasons over structure |
| 02 | Resolution & integrity report | **The editor↔code integrity report** + docs reference linter (CLI) | The headline value; the resolution layer reused everywhere |
| 03 | Desktop app & distribution | A signed, double-clickable Wails app: graph explorer + integrity report + blast radius | Non-dev usability; the shell for all later views |
| 04 | Coupling & domains | Domain map, coupling metrics, cycles, autoload fan-in, scene-flow graph | archi's "domain sprawl" story; the unit-projection model |
| 05 | Documentation drift | Docs-vs-graph reconciliation with grounded LLM | The in-repo docs use case |
| 99 | Future | C# extractor, GDExtension/C++, Godot-assisted resolution, CI coupling-gate | Multi-language reach, automation |

## Hard dependencies between milestones

- **02 depends on 01**: resolution needs both the scene/config edges *and* the GDScript symbols to
  match against. (Within 01, build the **scene + project-config extractors before GDScript** — the
  integrity report's most striking findings come from resolving `.tscn`/`@export` against symbols,
  and those extractors are simpler, de-risking the data model first.)
- **03 depends on 02**: the desktop app's first real view is the integrity report.
- **04 depends on 01** (the graph) but only *lightly* on 02. It introduces the **unit projection**
  abstraction — directory projection is already available from 00; inferred/declared land here.
- **05 depends on 04** (entity identifiers + the snapshot/diff harness) and the graph.
- **99** items are mostly independent extractors that emit into the *same* graph (DESIGN §6.3).

## Sequencing principle

Ship the unique, verifiable value first (integrity report — needs no domain inference, no LLM, no
ground-truth ambiguity), then layer on the analytically harder and the AI-assisted work once the
graph and its identifiers are proven. This front-loads de-risking: the noisy/fuzzy parts (domain
inference, prose-claim drift) come *after* the deterministic core is solid.
