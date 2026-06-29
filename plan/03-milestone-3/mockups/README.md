# godarch UI mockups

Static, self-contained mockups of the desktop app's views. They are **design props** — not the
real frontend — used to pin down layout, the design language, and the information architecture
before the Wails build. The real frontend (M3) re-implements these against live data.

## Files

| File | View | Milestone |
|---|---|---|
| [`shell.html`](shell.html) | App shell + Overview | M3 |
| [`integrity.html`](integrity.html) | Editor↔code integrity report | M3 |
| [`graph.html`](graph.html) | Graph explorer (focus/ego view) | M3 |
| [`blast.html`](blast.html) | Blast radius | M3 |
| [`domains.html`](domains.html) | Domains map (dependency matrix) | **M4 preview** |

`png/` holds rendered exports (light), plus `*-dark.png` samples.

## Viewing

Open any `.html` directly in a browser. Append `?dark` for the dark theme
(e.g. `shell.html?dark`). Fonts (IBM Plex) and icons (Tabler) load from CDN, so view online.

## Regenerating the PNGs

```sh
./shot.sh          # renders png/*.png with headless Chrome, trims margins via ImageMagick
```

Override the Chrome path with `CHROME=/path/to/chrome ./shot.sh` if needed.

## What's real vs. placeholder

- **Real:** the design language (see [`tokens.css`](tokens.css) and `../01-app-shell.md`), the IA,
  the per-view layout and interaction model, and the *kinds* of data each view shows.
- **Placeholder:** all numbers and entity names (`forgehollow`, `GameState`, `player.gd`…) are
  fabricated to look like a plausible mid-size Godot project. The graph layout is hand-placed;
  the real explorer runs a force layout. Blast-radius dots are a representative sample, not 312
  real nodes.

## Design language (summary)

- **Accent:** Godot blue, deepened for contrast — `#3D7DB0` / strong `#1F5C8C`.
- **Type:** IBM Plex Mono is the *structural voice* (wordmark, labels, all identifiers and
  match-keys); IBM Plex Sans for readable prose. Two weights (400/500).
- **Signatures:** the node-spine nav (views as nodes on a connector line; collapses to an icon
  rail) and the editor↔code *seam* (shown literally, with the break called out).
- **Status:** semantic roles — error red, warning amber, success green, info neutral. Node kinds
  get categorical ramps (scene=blue, script=purple, autoload=coral, signal=teal, action=amber,
  node=gray).
- Built on CDS-style surface/role tokens so every view works in light and dark.
