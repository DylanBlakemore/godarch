# 03.05 — Packaging & distribution

The "non-dev can run it" goal lives or dies here (DESIGN §6.1). cgo (tree-sitter) means per-OS
builds and real code-signing, or the OS blocks the download.

## Build

- `wails build` per OS on the CI matrix (00.04) — `CGO_ENABLED=1` with each platform's C toolchain.
- Targets: macOS (universal `arm64`+`amd64` if feasible), Windows `amd64`, Linux `amd64`.
- Bundle the frontend + the `tree-sitter-gdscript` grammar into the binary; no runtime downloads.

## Signing & notarization (don't skip)

- **macOS**: Developer ID signing + notarization + stapling → no Gatekeeper "unidentified developer"
  wall. Produce a `.dmg` or `.app` in a zip.
- **Windows**: Authenticode signing (a code-signing cert) → avoids SmartScreen friction. Produce an
  installer (`.msi`/NSIS) or a signed portable `.exe`.
- **Linux**: AppImage (or `.deb`/`.tar.gz`); signing optional.
- Store signing secrets in CI secrets; never in the repo.

## Distribution & updates

- v1: GitHub Releases with the signed artifacts per OS; a simple download page.
- Auto-update: evaluate Wails-compatible updaters (or a "new version available" check against the
  releases API) — nice-to-have, not blocking for v1.
- Telemetry: **off by default**; if added later, opt-in and clearly disclosed.

## Optional Godot-assist bundling

If 02.01's Godot-assisted resolution is kept: either detect a user-installed Godot, or bundle a
headless Godot binary (size/licensing trade-off). Decide here; default to "detect, optional".

## Tasks

- [ ] `wails build` working per OS on CI; grammar + frontend bundled.
- [ ] macOS Developer ID sign + notarize + staple pipeline.
- [ ] Windows Authenticode signing + installer.
- [ ] Linux AppImage.
- [ ] Release workflow: tag → build matrix → signed artifacts → GitHub Release.
- [ ] Download/install docs for non-devs (per OS, with screenshots).
- [ ] Decide Godot-assist bundling (detect vs bundle vs none).

## Definition of done

A tagged release yields signed, installable artifacts for macOS and Windows that open without OS
security blocks; a non-developer can follow the install docs and be looking at their project's
integrity report within minutes. **M3 ships here — godarch is a real desktop product.**
