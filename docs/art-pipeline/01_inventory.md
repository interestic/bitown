# Sprites v1 Step1: Inventory

Phase 1 (public repo) uses extracted assets from original Flash archive under
CC BY-NC-SA. This file defines the minimum inventory needed to move forward.

## Source

- Archive: Motion Twin WebGamesArchives (Miniville)
- Input artifact: `gfx.swf`
- Tool: FFDec (manual export in Step2)

## Minimum asset groups (M1 target)

1. `terrain`
   - grass tile
   - road tile (horizontal/vertical patterns)
2. `residential`
   - low-rise house variants
   - mid-rise apartment variants
3. `landmarks` (optional for M1, required by M2)
   - special building placeholders

## Proposed normalized naming

- `terrain_grass_01.png`
- `terrain_road_01.png`
- `res_house_a_01.png`
- `res_house_b_01.png`
- `res_apartment_a_01.png`

## Output directory contract

All extracted raw files must be placed under:

`assets/sprites-v1/raw/`

Subfolders:

- `assets/sprites-v1/raw/terrain`
- `assets/sprites-v1/raw/residential`
- `assets/sprites-v1/raw/landmarks`

## Acceptance criteria for Step1

- Inventory list is agreed
- Naming convention fixed
- Directory contract fixed
