#!/usr/bin/env bash
set -euo pipefail

# Step2: deterministic export layout contract for manually extracted files.
# This script prepares directories and validates expected input placement.

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RAW_DIR="$ROOT_DIR/assets/sprites-v1/raw"
SOURCE_DIR="$ROOT_DIR/assets/sprites-v1/source"
SWF_PATH="$SOURCE_DIR/gfx.swf"

mkdir -p "$RAW_DIR/terrain" "$RAW_DIR/residential" "$RAW_DIR/landmarks"
mkdir -p "$SOURCE_DIR"

echo "Prepared raw sprite directories:"
echo "  - $RAW_DIR/terrain"
echo "  - $RAW_DIR/residential"
echo "  - $RAW_DIR/landmarks"

if [[ ! -f "$SWF_PATH" ]]; then
  cat <<EOF
[WARN] Missing source SWF: $SWF_PATH

Place gfx.swf there, then run FFDec export manually:
  1) Open gfx.swf in FFDec
  2) Export image assets as PNG
  3) Copy selected outputs into $RAW_DIR/{terrain,residential,landmarks}
EOF
  exit 0
fi

cat <<EOF
[OK] Source SWF found: $SWF_PATH
Next: export PNG assets via FFDec and copy into $RAW_DIR.
EOF
