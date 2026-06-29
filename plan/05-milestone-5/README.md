# 05 — Documentation drift detection

The full version of the docs feature (DESIGN §8a). M2 already shipped the deterministic slice (the
docs reference linter, 02.03); this milestone adds **drift over time** and **grounded LLM** checking
of prose claims. Roadmap-level detail.

## Goal

Detect when in-repo docs (dev log, design docs, level/system docs, ADRs) have drifted from the
actual project: anchors that broke (already in M2), claims contradicted by the graph, and intent
(from devlog/ADR) not reflected in current structure.

## Depends on

- The resolution layer + entity identifiers (M2) and the doc format / `documents` edges (02.03).
- A **snapshot/diff harness** (shared with the 99 CI gate; archi has one to copy): persist graph
  snapshots, diff entities/edges between two analyses/commits.

## The two mechanisms (DESIGN §8a)

1. **Identifier-anchored drift over time (mechanical, precise).** When a graph diff shows an entity
   renamed/removed/changed, find the docs whose `documents` edges reference it → drift candidates.
   No LLM. This is the high-precision core.
2. **Prose-claim drift (grounded LLM).** For natural-language assertions not tied to a hard anchor,
   an LLM extracts claims and checks them — **grounded** by being fed the doc *plus the relevant
   subgraph* (the covered entities + their current edges). RAG over the structural graph, never
   freeform. Output: claim, doc location, what the graph says, confidence, suggested edit.

Special flavour worth building: **intent verification** — a devlog/ADR entry asserts intent ("we
decoupled combat from UI"); check it against the *current* graph ("combat still has 3 edges into
ui"). A distinct, powerful drift type.

## LLM integration

- **Pluggable provider** (Claude / OpenAI / local) via an interface; API key in config. The analogue
  of archi's async enrichment layer.
- **Optional & offline-friendly**: mechanism 1 works with no LLM; the LLM is enrichment.
- Grounding is mandatory — the model only ever judges claims against a provided subgraph, reducing
  hallucination. Cache results keyed by (doc hash, subgraph hash) to control cost.
- Outputs feed the same `findings`/report surface (M2) + a dedicated "docs drift" UI view.

## Division of labor (the discipline)

The **graph + diff finds candidates** (precise, cheap); the **LLM only explains and drafts fixes** on
candidates. Never let the LLM hunt blind across all docs — that's the path to noise and cost.

## Master checklist

- [ ] Snapshot persistence + graph-diff harness (entities/edges added/removed/changed)
- [ ] Anchored drift-over-time: map diffs → affected docs via `documents` edges
- [ ] LLM provider interface + Claude/OpenAI/local adapters + config
- [ ] Grounded claim extraction + check (doc + subgraph context); result caching
- [ ] Intent-verification flavour (devlog/ADR vs current graph)
- [ ] Suggested-edit drafting (optional auto-PR of doc fixes)
- [ ] Docs-drift UI view + report integration

## Exit criteria

1. Renaming an entity that a doc references surfaces that doc as drifted **without** an LLM call.
2. A doc claim that contradicts the graph is caught, explained, and a correction is drafted — and
   the explanation cites the actual graph fact.
3. LLM usage is optional, grounded, cached, and never blind.

## To expand into focused docs (when work starts)

`01-snapshot-diff.md`, `02-anchored-drift.md`, `03-llm-grounded-claims.md`, `04-intent-verification.md`,
`05-docs-drift-ui.md`.
