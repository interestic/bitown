# Sprites v1 Step7: Quality Gate

Add a lightweight quality gate before renderer integration advances.

## Run

```bash
make assets-check
```

## Checks

- required folders exist:
  - `raw/`
  - `normalized/`
  - `variants/`
- if atlas metadata exists, validate `frames` schema, the atlas PNG, and `buildings.json`:
  - each frame has `x,y,w,h,anchor_x,anchor_y` and stays inside the PNG
  - frames are not all 32×32
  - `version >= 2`
  - `bases_by_tag` / `counts.by_tag` for residential, industrial, commercial, landmark, road, tree, water, park, exclude
  - `building_bases` equals the union of the four building tags
  - known UI / road / empty clips are not in the building pool
  - `building_bases.allowlist` matches `buildings.json` `building_bases` exactly

## Purpose

- fail fast on missing pipeline outputs
- avoid silent renderer breakage due to absent artifacts
- keep map.png from drawing UI fragments as buildings

Re-generate the catalog after changing overrides:

```bash
python3 scripts/generate_buildings_manifest.py
make assets-check
```
