# 02 — Resolution & the editor↔code integrity report

The v1 headline (DESIGN §5, §7). Resolve the raw edges from M1 into a connected graph, then run the
integrity checks that surface what no code-only tool can: editor wiring that disagrees with the code.

## Goal

`godarch check <project>` produces a precise, actionable integrity report — dangling editor signal
connections, dead `@export`s, broken groups/actions, unresolvable loads, orphaned assets/scripts —
plus a docs reference linter. Deterministic, no LLM, no domain inference required.

## Why this is the right first value

It's **verifiable** (every finding is a concrete graph fact), **unique** (the editor↔code seam is
invisible to the GDScript LSP and call-graph tools), **fast to demo**, and **largely unit-agnostic**
— findings are edge-local, so it doesn't depend on the still-open domain-inference question (M4).

## Scope

**In:**
- The resolution / stitching layer (`01-match-keys-resolution.md`) — reused by every later milestone.
- The integrity findings engine + rule catalog (`02-integrity-checks.md`).
- The docs reference linter — the cheap slice of docs-drift (`03-docs-reference-linter.md`).
- Report output: text + JSON, severities, the unresolved/diagnostic list (`04-report-output.md`).

**Out:** UI (M3 — this milestone is CLI), coupling/domain metrics (M4), LLM/prose docs drift (M5).

## Deliverables

- `godarch check <dir> [--json] [--severity error]` → the integrity report.
- A resolved graph in SQLite (edges flipped to `resolved=1`, `findings` table populated).
- `findings.json` goldens pass for `coupled/`.

## Master checklist

- [ ] Resolution layer: every edge type's resolver implemented (`01`)
- [ ] NodePath resolution against scene trees; `%UniqueName`; relative→`res://` (`01`)
- [ ] Signal emit→handler resolution via match keys + type lattice (`01`)
- [ ] Unresolved edges recorded with reasons (the diagnostic list) (`01`)
- [ ] Integrity rule catalog implemented (`02`)
- [ ] Severity model + suppression via `godarch.yml` (`02`, `04`)
- [ ] Docs frontmatter parser + anchor-resolution linter (`03`)
- [ ] Report formatters: text + JSON, stable ordering (`04`)
- [ ] `findings.json` goldens pass for `coupled/` (`02`/`04`)

## Exit criteria

1. Every smell seeded in `testdata/fixtures/coupled/` is detected, with correct file:line evidence
   and a useful message; **no false positives** on `minimal/` (a clean project → empty report).
2. The unresolved-edge list is complete and each entry has a reason (dynamic path, untyped target,
   missing symbol) — nothing silently dropped.
3. Running on a real project yields findings a human agrees are real on spot-check.

## Docs in this milestone

| Doc | Covers |
|---|---|
| [`01-match-keys-resolution.md`](01-match-keys-resolution.md) | The stitching layer: resolvers per edge type |
| [`02-integrity-checks.md`](02-integrity-checks.md) | The findings rule catalog |
| [`03-docs-reference-linter.md`](03-docs-reference-linter.md) | Doc frontmatter format + anchor resolution |
| [`04-report-output.md`](04-report-output.md) | Severity model, formatters, suppression |
