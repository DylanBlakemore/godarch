# godarch — Build Plan

This folder is the working plan for building godarch. It turns [`../DESIGN.md`](../DESIGN.md) into
an ordered, checklist-driven sequence of milestones.

Read [`../DESIGN.md`](../DESIGN.md) first — it holds the *why* (the archi model, the Godot edge
taxonomy, the conceptual mapping). This folder holds the *how* and the *in-what-order*.

## How this plan is organised

```
plan/
├── README.md          ← you are here (conventions + milestone map)
├── ROADMAP.md         ← the full arc, dependencies, and what each milestone unlocks
├── 00-foundation/     ← scaffolding & contracts everything else builds on (no analysis yet)
├── 01-milestone-1/    ← Project model & extraction → a populated typed graph
├── 02-milestone-2/    ← Resolution & the editor↔code integrity report (the v1 headline)
├── 03-milestone-3/    ← Desktop app (Wails) & distribution
├── 04-milestone-4/    ← Coupling & domain analysis (roadmap-level detail)
├── 05-milestone-5/    ← Documentation-drift detection (roadmap-level detail)
└── 99-future/         ← C#, GDExtension/C++, Godot-assisted resolution, CI gate
```

Each milestone folder contains:
- **`README.md`** — goal, scope (in/out), deliverables, the master checklist, and exit criteria.
- **Focused docs** (`01-*.md`, `02-*.md`, …) — one per coherent chunk of work, each ending with a
  task checklist and a "definition of done".

## Conventions

- **Checklists** use `- [ ]` / `- [x]`. A milestone is done when every box in its `README.md`
  master checklist is ticked and its exit criteria hold.
- **Terminology** follows `../DESIGN.md` and (by inheritance) archi's `GLOSSARY.md`: node, edge,
  boundary point (ingress/egress), match key, walk, unit/domain (a *projection*), flow pattern.
- **Stack is decided** (DESIGN §6.1): Go core, Wails shell, `tree-sitter-gdscript` via cgo,
  SQLite store, `gonum/graph` for analysis. Plans assume this; alternatives are not re-litigated.
- **Each milestone ships something runnable.** Even 00 produces a CLI skeleton that parses a
  project and prints stats. No milestone is purely internal.

## The decisions already locked (from DESIGN)

| Decision | Choice |
|---|---|
| Core language | Go |
| Desktop shell | Wails (Go backend + web frontend) |
| GDScript parser | `tree-sitter-gdscript` via `smacker/go-tree-sitter` (cgo) |
| Scene/config parser | hand-written Go (INI-like) |
| Store | SQLite + in-memory `gonum/graph` |
| v1 user-facing lead | **editor↔code integrity report** |
| Unit of analysis | file-grained graph always; **directory** is the default *projection*; inferred/declared deferred to M4 |

## The decisions still open (revisit at the noted milestone)

- **Inferred-domain projection** (Louvain) vs declared (`godarch.yml`) vs directory-only — settle in **M4**.
- **Godot-assisted resolution** (bundle a headless Godot binary for ground-truth) — optional booster, evaluate in **M2/M3**.
- **Module path / repo host** — placeholder `github.com/dylanblakemore/godarch` used throughout; adjust in 00.
