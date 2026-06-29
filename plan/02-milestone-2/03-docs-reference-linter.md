# 02.03 — Docs reference linter (the cheap slice of docs drift)

`internal/docs`. The mechanical, no-LLM half of documentation drift (DESIGN §8a): docs carry
frontmatter anchors (entity references); the linter resolves those anchors against the graph and
flags the ones that no longer exist. Same check family as the integrity report — ships alongside it.

This milestone delivers **only** the deterministic anchor-resolution piece. Prose-claim checking and
LLM grounding are M5.

## Doc format (define now — adoption compounds over time)

Lightweight YAML frontmatter on in-repo markdown. The `covers` anchors are the join keys to the
graph (same identifiers as `model` IDs / match keys):

```markdown
---
type: system            # system | level | design | devlog | adr
domain: combat          # optional; a unit/projection name (M4)
covers:                 # entity anchors — resolved against the graph
  - res://combat/weapons/weapon.gd
  - uid://b8x2k...
  - signal:Weapon:fired
  - autoload:CombatState
  - action:attack
updated: 2026-06-20
---
prose…
```

- Anchors accept any `model` ID form (00.02) or `uid://`. UIDs resolve via `Project.UIDMap`.
- `type` and `covers` are the only required fields for linting; the rest support M4/M5.
- Each doc becomes a `doc` node; each anchor becomes a `documents` edge (doc → entity) — so "what
  docs cover this entity?" is a graph query, which M5's drift detection reuses.

## Lint rules

| Rule | Fires when | Severity |
|---|---|---|
| `doc_broken_anchor` | a `covers` anchor resolves to no graph node (deleted/renamed entity) | warning |
| `doc_malformed_frontmatter` | frontmatter missing/invalid, or `type` unknown | info |
| `doc_unknown_domain` | `domain` names a unit that doesn't exist (once M4 lands) | info |
| `doc_uncovered_entity` *(opt-in)* | a high-importance node (e.g. high-fan-in autoload) has no `documents` edge | info |

`doc_broken_anchor` is the valuable one: it's exactly "the doc references something the code no
longer has" — verifiable drift, no LLM.

## Tasks

- [ ] Frontmatter parser + the doc-type schema; validate `type`/`covers`.
- [ ] Emit `doc` nodes + `documents` edges into the graph (resolved like any other reference in 02.01).
- [ ] Implement `doc_broken_anchor` + `doc_malformed_frontmatter` (+ stub the others).
- [ ] Author 2–3 example docs under `testdata/fixtures/coupled/docs/` (one with a broken anchor) + golden.
- [ ] Write the doc-format spec into repo docs (so users can adopt it before M5).

## Definition of done

Docs with valid frontmatter produce `documents` edges; a doc whose anchor points at a
deleted/renamed entity yields `doc_broken_anchor` with file:line; the format is documented for users.
This is the foundation M5 builds the LLM-grounded prose checking on.
