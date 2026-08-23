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
  - `bases_by_tag` / `counts.by_tag` for residential, industrial, commercial, landmark, road, tree, water, ground, park, exclude
  - `landmark` / `park` may be unused as zone tags in M1 (`park` lots use `tree`)
  - `ground` (`mcDalle`) is stamped as 4×4 raised tiles for the peon field look
  - `road` arterials unlock with pop≥80 or tra>0; peon maps stay roadless
  - `building_bases` equals the union of the four building tags
  - known UI / road / empty clips are not in the building pool
  - `building_bases.allowlist` matches `buildings.json` `building_bases` exactly

## Map zone contract

- zone lots pick from `residential` / `industrial` / `commercial` via `counts.by_tag`
- at high pop, zone lots may additionally draw from `landmark` (growth mix; see `docs/map-building-growth.md`)
- empty zone tag → fall back to `residential`, then a rectangle if residential is also empty
- do not fall back to the full `building_bases` pool on empty tags

## Purpose

- fail fast on missing pipeline outputs
- avoid silent renderer breakage due to absent artifacts
- keep map.png from drawing UI fragments as buildings

Re-generate the catalog after changing overrides:

```bash
python3 scripts/generate_buildings_manifest.py
make assets-check
```
