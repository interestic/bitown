# Sprites v1 Step8: Ops Runbook

This runbook ties Step1-7 into an executable sequence for contributors.

## End-to-end order

1. `docs/art-pipeline/01_inventory.md`
2. `scripts/extract_sprites_ffdec.sh`
3. `scripts/normalize_sprites.py`
4. `scripts/generate_variants.py`
5. `scripts/build_atlas.py`
6. `scripts/generate_buildings_manifest.py`
7. `make assets-check`
8. `GET /api/cities/{slug}/map.png` (atlas renderer; rectangle fallback in dev)

## Quick commands

```bash
# step2
scripts/extract_sprites_ffdec.sh

# step3
python3 scripts/normalize_sprites.py

# step4
python3 scripts/generate_variants.py

# step5
python3 scripts/build_atlas.py

# step6
python3 scripts/generate_buildings_manifest.py

# step7
make assets-check
```

## Ownership / review checkpoints

- **Art source compliance**: verify assets remain under `assets/sprites-v1/` and keep NC license notice.
- **Determinism**: same slug should render same PoC map output.
- **Pipeline health**: run `make assets-check` before opening PR.

## Future TODOs

- Replace fixed-grid atlas with tighter packer.
- Add perceptual-diff snapshot tests for generated PNGs.
