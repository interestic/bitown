#!/usr/bin/env python3
"""
Step5: pack variant sprites into a single atlas using trimmed bboxes.
"""

from __future__ import annotations

import json
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
SRC = ROOT / "assets" / "sprites-v1" / "variants"
OUT_DIR = ROOT / "assets" / "sprites-v1" / "atlas"
OUT_PNG = OUT_DIR / "sprites_v1_atlas.png"
OUT_JSON = OUT_DIR / "sprites_v1_atlas.json"

PADDING = 1
SHELF_WIDTH = 512


def trim_frame(img: Image.Image) -> tuple[Image.Image, int, int]:
    bbox = img.getbbox()
    if bbox is None:
        blank = Image.new("RGBA", (1, 1), (0, 0, 0, 0))
        # Preserve explicit zero anchors; renderer must not treat these as missing.
        return blank, 0, 0
    left, top, right, bottom = bbox
    cropped = img.crop(bbox)
    # Map the normalized canvas foot (bottom-center) into trimmed-frame space.
    # Do not use opaque-bbox center: asymmetric art would drift off the grid.
    anchor_x = img.width // 2 - left
    anchor_y = img.height - top
    return cropped, anchor_x, anchor_y


def pack_frames(frames: list[dict]) -> tuple[Image.Image, dict]:
    max_w = max((item["img"].width for item in frames), default=1)
    width = max(SHELF_WIDTH, max_w + 2 * PADDING)
    ordered = sorted(frames, key=lambda item: (-item["img"].height, item["key"]))
    x = PADDING
    y = PADDING
    row_h = 0
    packed: list[dict] = []
    for item in ordered:
        img = item["img"]
        w, h = img.size
        if x + w + PADDING > width:
            x = PADDING
            y += row_h + PADDING
            row_h = 0
        packed.append({**item, "x": x, "y": y})
        x += w + PADDING
        row_h = max(row_h, h)
    height = y + row_h + PADDING
    atlas = Image.new("RGBA", (width, max(height, 1)), (0, 0, 0, 0))
    meta_frames = {}
    for item in packed:
        img = item["img"]
        atlas.paste(img, (item["x"], item["y"]), img)
        meta_frames[item["key"]] = {
            "x": item["x"],
            "y": item["y"],
            "w": img.width,
            "h": img.height,
            "anchor_x": item["anchor_x"],
            "anchor_y": item["anchor_y"],
        }
    return atlas, meta_frames


def main() -> None:
    files = sorted(SRC.rglob("*.png"))
    if not files:
        for path in (OUT_PNG, OUT_JSON):
            if path.exists():
                path.unlink()
        raise SystemExit("no variant sprites found; removed stale atlas outputs")

    frames = []
    for path in files:
        img = Image.open(path).convert("RGBA")
        cropped, ax, ay = trim_frame(img)
        key = str(path.relative_to(SRC)).replace("\\", "/")
        frames.append({"key": key, "img": cropped, "anchor_x": ax, "anchor_y": ay})

    atlas, meta_frames = pack_frames(frames)
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    atlas.save(OUT_PNG)
    OUT_JSON.write_text(
        json.dumps(
            {
                "image": OUT_PNG.name,
                "count": len(files),
                "padding": PADDING,
                "frames": meta_frames,
            },
            ensure_ascii=False,
            indent=2,
        ),
        encoding="utf-8",
    )
    print(f"atlas generated: {OUT_PNG} ({atlas.size[0]}x{atlas.size[1]})")
    print(f"metadata generated: {OUT_JSON}")


if __name__ == "__main__":
    main()
