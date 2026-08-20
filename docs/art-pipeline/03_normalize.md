# Sprites v1 Step3: Normalize

Normalize extracted PNGs into deterministic canvas size and folder layout.

## Run

```bash
pip install pillow
python3 scripts/normalize_sprites.py
```

## Behavior

- Reads all PNG files under `assets/sprites-v1/raw/`
- Writes normalized images to `assets/sprites-v1/normalized/`
- Current canvas size: `96x96`
- Keeps folder structure from `raw/`

## Why

- Unified anchors and dimensions make renderer integration stable.
