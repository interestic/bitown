# Sprites v1 Step4: Color Variants

Generate multiple color variants from normalized sprites.

## Run

```bash
pip install pillow
python3 scripts/generate_variants.py
```

## Behavior

- Input: `assets/sprites-v1/normalized/**/*.png`
- Output: `assets/sprites-v1/variants/**/*.png`
- Current preset: 4 hue shifts (`0, +18, +36, -24`)

## Naming

`<original>_v00.png`, `<original>_v01.png`, ...
