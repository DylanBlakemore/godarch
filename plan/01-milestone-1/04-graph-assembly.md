# 01.04 — Graph assembly & persistence

Combine discovery + both extractors into one `model.Project`, persist to SQLite, and expose CLI
inspection. Closes M1.

## Assembly

In the shared `Pipeline` (00.01):

1. `discovery.Run` → skeleton `Project` (file + concept nodes, UID map, main scene).
2. Run extractors over the relevant files, **merging** their emitted nodes/edges/boundaries into the
   project. Extractors run concurrently across files (goroutines + a bounded worker pool); merge is
   single-threaded to keep the node map race-free.
3. De-duplicate nodes by ID (a `class:Foo` may be referenced before its declaring script is parsed —
   first reference creates a stub, declaration enriches it).
4. Leave edges **unresolved** where targets are match keys — that's M2's job. Record everything
   unparseable in `Project`-level diagnostics.

## Persistence

- `store.SaveProject(project)` (00.03), transactional, chunked bulk insert.
- Write `meta`: godot version, godarch version, analyzed-at, project root, schema version.

## CLI surface (v0 of the real tool)

```
godarch analyze <dir> [--db path] [--ignore glob...]
    → runs the pipeline, writes <dir>/.godarch.db, prints a summary:
      nodes by kind, edges by type, boundary counts, # unresolved, # diagnostics

godarch graph --file res://player/player.gd [--json]
    → prints the node + its inbound/outbound edges (with origin & resolved flags)

godarch stats [--json]
    → totals + top fan-in nodes (degree only; real metrics come in M4)
```

## Sanity & determinism

- Summary counts must be stable across runs (sort everything).
- Spot-check on a real project: do autoload fan-in, scene instance counts, and signal counts look
  plausible? (Manual, but catches whole classes of extractor bugs early.)

## Tasks

- [x] Implement the `Pipeline.Run` assembly (discovery → concurrent extract → merge). _(merge is sequential — extractors mutate the shared project in place; a goroutine worker pool is deferred until the extractors emit into per-file collections)_
- [x] Node de-dup/stub-enrichment logic with tests.
- [x] Wire `store.SaveProject`; write `meta`. _(adds godarch_version + analyzed_at; `store.Meta` reads run-level keys)_
- [x] `godarch analyze` summary output (text + counts).
- [x] `godarch graph --file` and `godarch stats`.
- [x] End-to-end goldens for `minimal/` + `coupled/` (nodes/edges/boundaries).
- [ ] Perf check on a mid-size real project. _(manual; needs a real project checkout — not runnable in this environment)_

## Definition of done

`godarch analyze` produces a persisted graph whose nodes/edges/boundaries match the fixture goldens;
`godarch graph --file` and `stats` work; a real project analyses in seconds with plausible counts.
This is the M1 ship.
