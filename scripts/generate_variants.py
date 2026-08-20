#!/usr/bin/env python3
"""
Step4: generate color variants from normalized sprites.
"""

from pathlib import Path
import colorsys
from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
SRC = ROOT / "assets" / "sprites-v1" / "normalized"
DST = ROOT / "assets" / "sprites-v1" / "variants"

HUE_SHIFTS = [0, 18, 36, -24]


def shift_hue(img: Image.Image, shift: int) -> Image.Image:
    rgba = img.convert("RGBA")
    out = Image.new("RGBA", rgba.size)
    px_in = rgba.load()
    px_out = out.load()
    for y in range(rgba.height):
        for x in range(rgba.width):
            r, g, b, a = px_in[x, y]
            if a == 0:
                px_out[x, y] = (r, g, b, a)
                continue
            hf, sf, vf = colorsys.rgb_to_hsv(r / 255.0, g / 255.0, b / 255.0)
            hf = (hf + (shift / 360.0)) % 1.0
            rr_f, gg_f, bb_f = colorsys.hsv_to_rgb(hf, sf, vf)
            rr, gg, bb = int(rr_f * 255), int(gg_f * 255), int(bb_f * 255)
            px_out[x, y] = (rr, gg, bb, a)
    return out


def main() -> None:
    DST.mkdir(parents=True, exist_ok=True)
    for stale in DST.rglob("*.png"):
        stale.unlink()
    count = 0
    for src in SRC.rglob("*.png"):
        rel = src.relative_to(SRC)
        base = Image.open(src).convert("RGBA")
        for idx, shift in enumerate(HUE_SHIFTS):
            dst = DST / rel.parent / f"{rel.stem}_v{idx:02d}.png"
            dst.parent.mkdir(parents=True, exist_ok=True)
            img = shift_hue(base, shift)
            img.save(dst)
            count += 1
    print(f"generated {count} variant file(s) into {DST}")


if __name__ == "__main__":
    main()
