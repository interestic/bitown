#!/usr/bin/env python3
"""
Step3: normalize raw sprite images into deterministic RGBA PNGs.

Places each sprite foot-aligned on a fixed canvas without clipping the
source when it fits. Oversized frames are nearest-neighbor scaled down so
the atlas stays map-usable. The old 32x32 center paste clipped buildings.

Requires Pillow:
  pip install pillow
"""

from pathlib import Path
from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
RAW = ROOT / "assets" / "sprites-v1" / "raw"
NORMALIZED = ROOT / "assets" / "sprites-v1" / "normalized"
CANVAS = 96


def normalize_png(src: Path, dst: Path) -> None:
    img = Image.open(src).convert("RGBA")
    w, h = img.size
    if w > CANVAS or h > CANVAS:
        scale = min(CANVAS / w, CANVAS / h)
        nw = max(1, int(round(w * scale)))
        nh = max(1, int(round(h * scale)))
        img = img.resize((nw, nh), Image.Resampling.NEAREST)
        w, h = img.size

    canvas = Image.new("RGBA", (CANVAS, CANVAS), (0, 0, 0, 0))
    x = (CANVAS - w) // 2
    y = CANVAS - h  # keep feet on the bottom edge
    canvas.paste(img, (x, y), img)
    dst.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(dst)


def main() -> None:
    NORMALIZED.mkdir(parents=True, exist_ok=True)
    for stale in NORMALIZED.rglob("*.png"):
        stale.unlink()
    count = 0
    for src in RAW.rglob("*.png"):
        rel = src.relative_to(RAW)
        dst = NORMALIZED / rel
        normalize_png(src, dst)
        count += 1
    print(f"normalized {count} file(s) into {NORMALIZED} (canvas={CANVAS})")


if __name__ == "__main__":
    main()
