# Sprites v1 Step5: Atlas

Pack generated variants into a single atlas and metadata map.

## Run

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install pillow
python3 scripts/build_atlas.py
```

## Output

- `assets/sprites-v1/atlas/sprites_v1_atlas.png`
- `assets/sprites-v1/atlas/sprites_v1_atlas.json`

## Packing

- Each variant is **trimmed to its opaque bbox** before packing.
- Frames are shelf-packed (height descending, then key) with 1px padding.
- JSON records native `w` / `h` plus `anchor_x` / `anchor_y`.
- Anchors map the **normalized canvas foot** (bottom-center of the
  `normalize_sprites.py` canvas, currently 96×96) into trimmed-frame space.
  They may sit outside the opaque bbox for asymmetric art.
- Frame keys stay `sprites/DefineSprite_*/*_v0N.png`.
- There is no `tile: 32` field; map cell size is owned by the renderer.

Normalize places feet on the bottom edge of its canvas before variants are
generated. Atlas packing must preserve that foot, not re-center on the
opaque bbox, or isometric grid grounding drifts.
