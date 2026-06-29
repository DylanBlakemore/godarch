# 02.04 — Report output

`internal/report`. Turn findings into output a human (CLI now, UI in M3) and a machine (CI later)
can act on. Closes M2.

## CLI

```
godarch check <dir> [--db path] [--json] [--severity error|warning|info]
                    [--rule r1,r2] [--no-docs] [--exit-code]
```

- Default: human-readable, grouped by severity then rule, each finding as
  `SEVERITY  rule  res://file.gd:42  message`, with an indented `→ detail/suggested fix`.
- `--json`: the full `Finding` list (stable order) for tooling/CI/the UI.
- `--severity`: minimum level to show.
- `--exit-code`: non-zero if any finding ≥ a threshold (sets up the CI gate in 99).
- Always print a footer: counts by severity + `N unresolved edges (use --json for the list)`.

## Severity model

- `error` — provably broken (dangling connection, missing resource, undefined action).
- `warning` — likely wrong / risky (dead export, fragile nodepath, empty group call).
- `info` — hygiene / maybe-intentional (orphans, unused signals, doc anchor issues).
- Defaults chosen for **high signal-to-noise**; everything overridable in `godarch.yml`.

## Determinism & evidence

- Stable ordering: by severity, then rule, then `res://` path, then line.
- Every finding carries `file:line` and, where useful, a one-line snippet + a concrete suggested fix
  (e.g. "method `_on_pressed` not found on `res://ui/menu.gd`; did you mean `_on_play_pressed`?").
- Persist findings to the `findings` table (00.03) so the UI (M3) reads them without re-running.

## `godarch.yml` (the override/config file)

```yaml
ignore: ["addons/**", "**/generated/**"]
rules:
  orphan_asset: { severity: info, enabled: true }
  fragile_nodepath: { max_distance: 3 }
suppress:
  - { rule: dead_export, node: "res://ui/debug.gd::show_fps" }
resolve:
  loads: { "level_%d": "res://levels/level_*.tscn" }   # dynamic-path hints
docs:
  globs: ["docs/**/*.md", "*.md"]
```

## Tasks

- [ ] `report.Finding` formatters: text (grouped) + JSON (stable).
- [ ] `godarch check` flags: `--json`, `--severity`, `--rule`, `--no-docs`, `--exit-code`.
- [ ] Persist findings to SQLite; footer summary.
- [ ] `godarch.yml` loader (ignore, rules, suppress, resolve hints, docs globs) — shared with 02.01/02.02.
- [ ] Suggested-fix text for the high-value rules (dangling_connection, undefined_action, dead_export).
- [ ] Golden for `coupled/` report (text + json).

## Definition of done

`godarch check` emits a deterministic, well-grouped report (text + json) with file:line + suggested
fixes, honours `godarch.yml`, persists findings, and sets a CI-usable exit code. **M2 ships here** —
godarch now delivers its unique value from the command line.
