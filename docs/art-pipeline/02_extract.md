# Sprites v1 Step2: Extract

This step standardizes the extraction workflow from `gfx.swf` using FFDec.

## Setup

```bash
chmod +x scripts/extract_sprites_ffdec.sh
scripts/extract_sprites_ffdec.sh
```

## Expected layout

- Input SWF: `assets/sprites-v1/source/gfx.swf`
- Output folders:
  - `assets/sprites-v1/raw/terrain`
  - `assets/sprites-v1/raw/residential`
  - `assets/sprites-v1/raw/landmarks`

## Notes

- Extraction itself is manual in FFDec for now.
- The script provides deterministic folder prep and validation.
