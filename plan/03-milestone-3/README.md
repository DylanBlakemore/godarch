# 03 — Desktop app & distribution

Wrap the proven core (M1 graph + M2 integrity report) in a Wails desktop app a non-developer can
download, double-click, point at a project folder, and use (DESIGN §6.1, §6.2). No new analysis —
this milestone is about **usability, visualisation, and shipping a signed binary**.

## Goal

A signed, double-clickable app for macOS + Windows (+ Linux) that: opens a Godot project, runs the
pipeline with a progress indicator, and presents the integrity report, an interactive graph
explorer, and blast-radius — with "open in Godot" deep links.

## Scope

**In:**
- Wails shell wiring the core via the shared `Pipeline` (`01-wails-shell.md`).
- Interactive graph explorer (`02-graph-explorer-ui.md`).
- Integrity report UI — the primary view (`03-integrity-report-ui.md`).
- Blast-radius / "what depends on this" + editor-in-Godot deep links (`04-blast-radius-and-deeplinks.md`).
- Per-OS build, signing, packaging, auto-update story (`05-packaging-distribution.md`).

**Out:** coupling/domain views (M4), docs-drift UI (M5). The shell must be built so those drop in
as new views without rework.

## Deliverables

- Downloadable signed installers for macOS (.dmg/.app, notarized) and Windows (.exe/.msi).
- Open-folder → analyze → three working views (report, graph, blast radius).
- The UI reads from SQLite (it doesn't re-implement analysis).

## Master checklist

- [ ] Wails app opens a folder, runs `Pipeline.Run` with progress, handles errors (`01`)
- [ ] Core bound to frontend via Wails methods returning the JSON models (`01`)
- [ ] Integrity report view: filter by severity/rule, jump to file:line, show suggested fix (`03`)
- [ ] Graph explorer: render, filter by node/edge kind, search, focus a node (`02`)
- [ ] Editor↔code edges visually distinguished (origin = editor vs code) (`02`)
- [ ] Blast-radius view: forward/reverse reach from a selected node (`04`)
- [ ] "Open in Godot" / "open in editor" deep links (`04`)
- [ ] Signed + notarized macOS build; signed Windows build (`05`)
- [ ] Release CI matrix produces installers as artifacts (`05`)
- [ ] First-run UX: pick folder, no config required, sensible defaults (`01`)

## Exit criteria

1. A non-developer can install the app, open a Godot project, and read the integrity report without
   touching a terminal or config file.
2. The graph explorer renders a real mid-size project interactively (pan/zoom/filter stay smooth).
3. Installers are signed so the OS doesn't block them (no "unidentified developer" wall).

## Docs in this milestone

| Doc | Covers |
|---|---|
| [`01-wails-shell.md`](01-wails-shell.md) | App shell, core binding, progress, first-run UX |
| [`02-graph-explorer-ui.md`](02-graph-explorer-ui.md) | Interactive graph rendering & filtering |
| [`03-integrity-report-ui.md`](03-integrity-report-ui.md) | The primary report view |
| [`04-blast-radius-and-deeplinks.md`](04-blast-radius-and-deeplinks.md) | Reachability view + Godot deep links |
| [`05-packaging-distribution.md`](05-packaging-distribution.md) | Signing, notarization, installers, updates |

## Mockups

Static, self-contained mockups of every view live in [`mockups/`](mockups/) — rendered PNG exports
in `mockups/png/`, regenerable via `mockups/shot.sh`. They fix the design language and IA the build
targets; each focused doc embeds its view. They are design props (placeholder data, CDN assets), not
the shipped frontend — see [`mockups/README.md`](mockups/README.md).

| View | Mockup | Doc |
|---|---|---|
| App shell + Overview | [shell.png](mockups/png/shell.png) | `01` |
| Integrity report | [integrity.png](mockups/png/integrity.png) | `03` |
| Graph explorer | [graph.png](mockups/png/graph.png) | `02` |
| Blast radius | [blast.png](mockups/png/blast.png) | `04` |
| Domains map | [domains.png](mockups/png/domains.png) | **M4 preview** — the shell reserves its nav slot now |

The design language (Godot-blue accent, IBM Plex Mono as structural voice, the node-spine nav and
the editor↔code seam) is summarised in [`mockups/README.md`](mockups/README.md) and specified in
[`01-wails-shell.md`](01-wails-shell.md).
