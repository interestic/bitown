# Step 9 — MagicaVoxel original tiles (Phase 3)

**Status:** Parked / scaffold only. Not a Phase 1 launch blocker.

**Runtime:** none — documentation scaffold only; no API, assets, or CI changes in this issue.

Phase 1 ships Miniville-derived `assets/sprites-v1/` under CC BY-NC-SA.
Phase 3 replaces them with original MagicaVoxel → isometric exports under a
permissive license (commercial path / separate private product repo).

## Goals

- [ ] Voxel kit for ground, road, residential / industrial / commercial / landmark
- [ ] Export pipeline to PNG frames compatible with atlas builder
- [ ] Tag + tier metadata in `buildings.json` (same contract as sprites-v1)
- [ ] Keep NC assets out of the commercial tree

## Non-goals (now)

- Shipping MagicaVoxel art in Phase 1 public builds
- Matching Flash pixels

## Suggested layout (later)

```
assets/sprites-v3/          # original, permissive
  raw/vox/
  normalized/
  variants/
  atlas/
docs/art-pipeline/09_magicavoxel.md
```

## Related

- Public lighting issue remains Phase 3 (ground shade opt-in already exists)
- Ground shade: `BITOWN_MAP_GROUND_SHADE=1`
