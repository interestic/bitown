#!/usr/bin/env python3
"""Build a tagged sprite catalog for the map renderer.

Classification is heuristic plus a hand-maintained override file so reviewers
can move junk (UI fragments, empty clips) out of the building pool without
renaming Flash export folders.
"""

from __future__ import annotations

import json
import re
from collections import Counter
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
NORMALIZED = ROOT / "assets" / "sprites-v1" / "normalized" / "sprites"
OUT = ROOT / "assets" / "sprites-v1" / "buildings.json"
OVERRIDES = ROOT / "assets" / "sprites-v1" / "sprite_tag_overrides.json"

MAP_TAGS = (
    "residential",
    "industrial",
    "commercial",
    "landmark",
    "road",
    "tree",
    "water",
    "ground",
    "park",
    "exclude",
)
BUILDING_TAGS = frozenset({"residential", "industrial", "commercial", "landmark"})

# Growth placement tiers (docs/map-building-growth.md): 0=hut/low … 3=landmark.
DEFAULT_TIER = 1
MAX_TIER = 3
# bbox_h at or above this is "突出した高さ" → tier 3 even without the landmark tag.
TALL_TIER3_H = 80

UI_NAME_PATTERNS = (
    re.compile(r"mcLoading", re.I),
    re.compile(r"mcAntiPanel", re.I),
    re.compile(r"mcAntiLayer", re.I),
    re.compile(r"mcAnalog", re.I),
    re.compile(r"mcStatSlot", re.I),
    re.compile(r"mcStats", re.I),
    re.compile(r"mcCompt", re.I),
    re.compile(r"mcTest", re.I),
    re.compile(r"mcBg", re.I),
    re.compile(r"StatPanel", re.I),
    re.compile(r"StatusBar", re.I),
)

ROAD_NAME = re.compile(r"mcRoad", re.I)
HOUSE_NAME = re.compile(r"mcHouse", re.I)


def sprite_metrics(folder: Path) -> tuple[int, int, int, dict[str, float]]:
    max_opaque = 0
    max_bbox_h = 0
    max_bbox_w = 0
    blues = yellows = greens = blacks = semi = strong = cyan = grays = 0
    canvas_pixels = 0
    for png in folder.glob("*.png"):
        im = Image.open(png).convert("RGBA")
        pixels = im.load()
        w, h = im.size
        canvas_pixels = max(canvas_pixels, w * h)
        opaque = 0
        for y in range(h):
            for x in range(w):
                r, g, b, a = pixels[x, y]
                if a <= 16:
                    continue
                opaque += 1
                if b > r + 30 and b > g + 10:
                    blues += 1
                if b > r + 25 and g > r + 10 and b > 120 and g > 90:
                    cyan += 1
                if r > 180 and g > 150 and b < 120:
                    yellows += 1
                if g > r + 15 and g > b + 15:
                    greens += 1
                if r < 35 and g < 35 and b < 35:
                    blacks += 1
                if abs(r - g) < 15 and abs(g - b) < 15 and r < 160:
                    grays += 1
                if 16 < a < 180:
                    semi += 1
                if a >= 220:
                    strong += 1
        max_opaque = max(max_opaque, opaque)
        bbox = im.getbbox()
        if bbox:
            max_bbox_h = max(max_bbox_h, bbox[3] - bbox[1])
            max_bbox_w = max(max_bbox_w, bbox[2] - bbox[0])

    denom = max(max_opaque, 1)
    fill = max_opaque / max(canvas_pixels, 1)
    colors = {
        "blue_ratio": blues / denom,
        "cyan_ratio": cyan / denom,
        "yellow_ratio": yellows / denom,
        "green_ratio": greens / denom,
        "black_ratio": blacks / denom,
        "gray_ratio": grays / denom,
        "semi_ratio": semi / denom,
        "strong_ratio": strong / denom,
        "fill_ratio": fill,
    }
    return max_opaque, max_bbox_h, max_bbox_w, colors


def classify_building(bbox_h: int, bbox_w: int, colors: dict[str, float]) -> str:
    """Split building-like sprites into residential / industrial / commercial."""
    # Gray/black heavy mass → industrial plant / warehouse.
    if (colors["gray_ratio"] >= 0.2 or colors["black_ratio"] >= 0.08) and (
        bbox_h >= 60 or bbox_w >= 70
    ):
        return "industrial"
    # Wide mid-height mass → commercial block footprint.
    if bbox_w >= 70 and bbox_h <= 70:
        return "commercial"
    return "residential"


def is_ui_sprite(name: str) -> bool:
    return any(pattern.search(name) for pattern in UI_NAME_PATTERNS)


def classify(name: str, opaque: int, bbox_h: int, bbox_w: int, colors: dict[str, float]) -> tuple[str, str]:
    """Return (group, tag). group is the coarse bucket; tag is the map use."""
    if is_ui_sprite(name):
        return "ui", "exclude"
    if ROAD_NAME.search(name):
        return "other", "road"
    if HOUSE_NAME.search(name):
        if opaque == 0 or bbox_h == 0:
            return "other", "exclude"
        return "building", "residential"

    if colors["fill_ratio"] > 0.85 and opaque >= 900:
        return "other", "exclude"
    if colors["blue_ratio"] > 0.45:
        return "other", "water"
    if colors["yellow_ratio"] > 0.35:
        return "other", "exclude"
    if colors["green_ratio"] > 0.4 and bbox_h >= 14:
        return "other", "tree"
    # Semi-transparent plates/shadows look like ghost buildings on the map.
    if colors["semi_ratio"] > 0.62 and bbox_h >= 14:
        return "other", "exclude"
    if colors["strong_ratio"] < 0.05 and bbox_h >= 16:
        return "other", "exclude"
    if colors["cyan_ratio"] > 0.7 and bbox_h >= 16:
        return "other", "exclude"
    if colors["black_ratio"] > 0.35 and bbox_h >= 14:
        return "other", "exclude"
    if bbox_w <= 18 and bbox_h >= 22:
        return "other", "exclude"
    if bbox_h >= 14 and colors["fill_ratio"] < 0.17:
        return "other", "exclude"

    if opaque <= 180 and bbox_h <= 13:
        return "character", "exclude"
    if opaque >= 200 and bbox_h >= 18 and bbox_w >= 12:
        return "building", classify_building(bbox_h, bbox_w, colors)
    if opaque >= 200 and bbox_h >= 14 and bbox_w >= 12 and colors["fill_ratio"] >= 0.35:
        return "building", classify_building(bbox_h, bbox_w, colors)
    return "other", "exclude"


def load_overrides(path: Path) -> tuple[dict[str, str], dict[str, int]]:
    if not path.exists():
        return {}, {}
    data = json.loads(path.read_text(encoding="utf-8"))
    tags = data.get("tags") or {}
    out: dict[str, str] = {}
    for base, tag in tags.items():
        if tag not in MAP_TAGS:
            raise SystemExit(f"override {base} has unknown tag {tag!r}")
        out[str(base)] = str(tag)
    tiers_raw = data.get("tiers") or {}
    tiers: dict[str, int] = {}
    for base, tier in tiers_raw.items():
        if not isinstance(tier, int) or tier < 0 or tier > MAX_TIER:
            raise SystemExit(f"override tier {base} must be int 0..{MAX_TIER}, got {tier!r}")
        tiers[str(base)] = tier
    return out, tiers


def apply_override(group: str, tag: str, override: str | None) -> tuple[str, str]:
    if override is None:
        return group, tag
    if override in BUILDING_TAGS:
        return "building", override
    if override == "exclude":
        if group == "ui":
            return "ui", "exclude"
        if group == "character":
            return "character", "exclude"
        return "other", "exclude"
    return "other", override


def heuristic_tier(tag: str, bbox_h: int) -> int:
    """Assign growth tier from zone tag + bbox height (docs/map-building-growth.md)."""
    if tag == "landmark" or bbox_h >= TALL_TIER3_H:
        return 3
    if tag == "residential":
        if bbox_h < 35:
            return 0
        if bbox_h < 55:
            return 1
        return 2
    if tag == "industrial":
        if bbox_h < 40:
            return 1
        return 2
    if tag == "commercial":
        # Keep a mid-rise band (tier 2) under the tall cap so big pop does not
        # jump low→skyscraper after bbox_h>=80 became tier 3 (#47/#48 review).
        if bbox_h < 40:
            return 0
        if bbox_h < 52:
            return 1
        return 2
    return DEFAULT_TIER


def resolve_tier(tag: str, bbox_h: int, override: int | None) -> int | None:
    """Building tags get a tier; non-buildings stay unset (None)."""
    if tag not in BUILDING_TAGS:
        return None
    if override is not None:
        return override
    return heuristic_tier(tag, bbox_h)


def main() -> None:
    if not NORMALIZED.is_dir():
        raise SystemExit(f"missing normalized sprites dir: {NORMALIZED}")

    tag_overrides, tier_overrides = load_overrides(OVERRIDES)
    entries: list[dict[str, object]] = []
    bases_by_tag: dict[str, list[str]] = {tag: [] for tag in MAP_TAGS}

    for folder in sorted(NORMALIZED.iterdir()):
        if not folder.is_dir():
            continue
        opaque, bbox_h, bbox_w, colors = sprite_metrics(folder)
        group, tag = classify(folder.name, opaque, bbox_h, bbox_w, colors)
        base = f"sprites/{folder.name}"
        group, tag = apply_override(group, tag, tag_overrides.get(base))
        entry: dict[str, object] = {
            "base": base,
            "group": group,
            "tag": tag,
            "max_opaque_pixels": opaque,
            "max_bbox_height": bbox_h,
            "max_bbox_width": bbox_w,
        }
        tier = resolve_tier(tag, bbox_h, tier_overrides.get(base))
        if tier is not None:
            entry["tier"] = tier
        entries.append(entry)
        bases_by_tag[tag].append(base)

    for tag in MAP_TAGS:
        bases_by_tag[tag].sort()

    building_bases = []
    for tag in ("residential", "industrial", "commercial", "landmark"):
        building_bases.extend(bases_by_tag[tag])
    building_bases.sort()

    group_counts = Counter(str(e["group"]) for e in entries)
    tag_counts = {tag: len(bases_by_tag[tag]) for tag in MAP_TAGS}
    tier_counts = Counter(
        int(e["tier"]) for e in entries if e.get("tag") in BUILDING_TAGS and "tier" in e
    )
    by_tier = {str(i): tier_counts.get(i, 0) for i in range(MAX_TIER + 1)}

    payload = {
        "version": 2,
        "source": "scripts/generate_buildings_manifest.py",
        "overrides": str(OVERRIDES.relative_to(ROOT)),
        "rules": {
            "ui": "named Flash UI clips (mcStats, mcLoading, ...)",
            "road": "mcRoad in name",
            "water": "blue-dominant opaque fills (ponds, circles)",
            "tree": "green-dominant sprites with bbox_h>=14",
            "ground": "iso floor / grass tiles (override-curated; not auto-classified yet)",
            "residential": "building-like fills that are neither industrial nor commercial",
            "industrial": "gray/black-heavy tall or wide warehouse-like masses",
            "commercial": "wide mid-height commercial footprints (bbox_w>=70, bbox_h<=70)",
            "character": "opaque<=180 and bbox_h<=13",
            "exclude": "yellow fills, full-canvas junk, flat leftovers, overrides",
            "override_file": "sprite_tag_overrides.json wins after heuristics",
            "tier": "0=hut/low … 3=landmark; heuristic from tag+bbox_h (>=80 → 3; commercial 52..79 → 2); optional overrides.tiers",
        },
        "counts": {
            "building": len(building_bases),
            "character": group_counts.get("character", 0),
            "ui": group_counts.get("ui", 0),
            "other": group_counts.get("other", 0),
            "by_tag": tag_counts,
            "by_tier": by_tier,
        },
        "building_bases": building_bases,
        "bases_by_tag": bases_by_tag,
        "entries": entries,
    }

    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
    print(f"wrote {OUT} ({len(building_bases)} building bases)")
    print("by_tag:", json.dumps(tag_counts, sort_keys=True))


if __name__ == "__main__":
    main()
