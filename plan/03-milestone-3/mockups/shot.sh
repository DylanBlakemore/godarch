#!/usr/bin/env bash
# Render the mockup HTML files to PNG with headless Chrome.
# Usage: ./shot.sh           (regenerates png/*.png)
# Requires Google Chrome. Fonts/icons load from CDN, so run online.
set -euo pipefail

CHROME="${CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
DIR="$(cd "$(dirname "$0")" && pwd)"
OUT="$DIR/png"
mkdir -p "$OUT"

shoot() { # <html-file> <out-name> <query>
  "$CHROME" --headless=new --disable-gpu --hide-scrollbars --force-color-profile=srgb \
    --default-background-color=00000000 --virtual-time-budget=7000 --window-size=980,860 \
    --screenshot="$OUT/$2.png" "file://$DIR/$1$3" >/dev/null 2>&1
  # trim uniform page-bg margins for a tighter crop
  if command -v magick >/dev/null 2>&1; then magick "$OUT/$2.png" -trim +repage "$OUT/$2.png"; fi
}

for v in shell integrity graph domains blast; do
  echo "rendering $v ..."
  shoot "$v.html" "$v" ""
done

echo "rendering dark samples ..."
shoot "shell.html"     "shell-dark"     "?dark"
shoot "integrity.html" "integrity-dark" "?dark"

echo "done -> $OUT"
