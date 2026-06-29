# 04 — Coupling & domain analysis

archi's core pitch, applied to Godot (DESIGN §5). The integrity report (M2) is edge-local; this
milestone is about **structure**: how the project clusters, where coupling concentrates, and what's
sprawling. Roadmap-level detail — to be expanded into focused docs when M3 ships.

## Goal

From the resolved graph, compute and present: unit/domain groupings, coupling metrics, dependency
cycles, hub/god-object detection, and the scene-flow graph — so a user can see "is my inventory
system tangled with combat?" at a glance.

## The unit-of-analysis model (the open decision, settled here)

DESIGN §8 / the unit discussion: a "unit" is a **projection over the file-grained graph**, not a
stored thing. Implement projections as `graph.Project(by ProjectionFunc)`:

- **Directory projection** (default, available since 00) — fold nodes by folder.
- **Inferred-domain projection** — Louvain/modularity community detection on the resolved coupling
  graph, **seeded by directory structure**. Build or vendor Louvain (DESIGN §6.1).
- **Declared projection** — `godarch.yml` glob→domain map (archi's `service_registry.yml` analogue).

The headline sprawl signal: **divergence between the directory projection and the inferred
projection** — "your files cluster differently than your folders claim."

## Analyses to compute (post-resolution passes, archi-style)

| Analysis | Definition | Output |
|---|---|---|
| Coupling metrics | per unit: afferent `Ca`, efferent `Ce`, instability `Ce/(Ca+Ce)` | a table + per-node properties |
| Dependency cycles | Tarjan SCC over unit edges (gonum) | cycle list + membership |
| Autoload fan-in | scripts (and units) touching each singleton | ranked list; the #1 sprawl finding |
| Scene instancing fan-in | scenes instanced widely = de-facto globals | ranked list |
| Signal fan-out | emitters with many listeners (esp. via autoload event buses) | ranked list |
| Fragile-reach map | cross-tree NodePath reach distances aggregated by unit | heatmap |
| Scene-flow graph | `changes_scene_to` graph from the main scene | navigable flow diagram |
| Walks & flow patterns | rooted DFS ingress→egress; group by signature (DESIGN §1) | flow-pattern list |
| Cohesion / sprawl score | dir-vs-inferred divergence + scattered-member detection | per-domain score + report |

## UI (new views in the M3 shell)

- **Domain map**: units as nodes, cross-unit coupling as weighted edges; toggle projection
  (dir/inferred/declared).
- **Metrics dashboard**: instability, fan-in/out leaderboards, cycle list.
- **Scene-flow view**: the game's navigation graph.
- **Sprawl report**: ranked, explained findings ("CombatState autoload is touched by 7 of 9
  domains").

## Master checklist

- [ ] `graph.Project` projection abstraction + directory projection
- [ ] Louvain (or vendored) inferred-domain projection, dir-seeded
- [ ] Declared projection from `godarch.yml`
- [ ] Coupling metrics (Ca/Ce/instability) per unit
- [ ] Cycles (Tarjan) over unit edges + membership
- [ ] Autoload fan-in, scene-instance fan-in, signal fan-out leaderboards
- [ ] Fragile-reach aggregation
- [ ] Scene-flow graph from main scene
- [ ] Walks + flow-pattern grouping
- [ ] Dir-vs-inferred divergence sprawl score
- [ ] M3 UI: domain map, metrics dashboard, scene-flow, sprawl report

## Exit criteria

1. On a real project, the inferred domains and coupling leaderboards match a human's intuition about
   where the tangles are.
2. The dir-vs-inferred divergence highlights at least one genuine "this lives in the wrong place".
3. Projections are switchable live in the UI without re-analysis.

## To expand into focused docs (when work starts)

`01-projections-and-domains.md`, `02-coupling-metrics.md`, `03-walks-and-flow-patterns.md`,
`04-scene-flow.md`, `05-domain-ui.md`.
