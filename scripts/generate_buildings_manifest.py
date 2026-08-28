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
SWF_GRAPH = ROOT / "assets" / "sprites-v1" / "swf_character_graph.json"

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
BUILDING_DENY_SUBSTR = (
    "mcLoading",
    "mcAnti",
    "mcAnalog",
    "mcStat",
    "mcStats",
    "mcCompt",
    "mcObs",
    "mcTest",
    "mcBg",
    "mcDalle",
    "mcRoad",
    "brushWood",
    "StatPanel",
    "StatusBar",
)
POOL_ROLES = frozenset({"library_primary"})

# Growth placement tiers (docs/map-building-growth.md): 0=hut/low … 3=landmark.
DEFAULT_TIER = 1
MAX_TIER = 3
# bbox_h at or above this is "突出した高さ" → tier 3 even without the landmark tag.
TALL_TIER3_H = 80

# Unlock thresholds aligned with api/internal/render/growth_pool.go and citycore/sector.go.
POP_TIER_PEON = 40
POP_TIER_NORMAL = 120
POP_TIER_HUGE = 350
SECTOR_IND = 50
SECTOR_COM = 50
SECTOR_SEC = 300
SECTOR_TRA = 100
TREE_ENV_THRESHOLDS = (0, 80, 200, 400)
UNLOCK_KEYS = frozenset({"min_pop", "min_ind", "min_com", "min_env", "min_sec", "min_tra"})
STAMP_KINDS = frozenset({"mini_foot", "arterial_yard", "landmark_center", "packed_mini"})
NUDGE_PROFILES = frozenset(
    {"default", "landmark", "arterial_yard", "overlap_yard", "packed_no_cross", "packed_cross"}
)
SPRITE_ID_RE = re.compile(r"DefineSprite_(\d+)")

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
    total_opaque = 0
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
        total_opaque += opaque
        bbox = im.getbbox()
        if bbox:
            max_bbox_h = max(max_bbox_h, bbox[3] - bbox[1])
            max_bbox_w = max(max_bbox_w, bbox[2] - bbox[0])

    denom = max(total_opaque, 1)
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


def load_overrides(
    path: Path,
) -> tuple[
    dict[str, str],
    dict[str, int],
    dict[str, dict[str, int]],
    dict[str, str],
    dict[str, dict[str, object]],
]:
    if not path.exists():
        return {}, {}, {}, {}, {}
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
        if isinstance(tier, bool):
            raise SystemExit(f"override tier {base} must be int 0..{MAX_TIER}, got {tier!r}")
        if not isinstance(tier, int) or tier < 0 or tier > MAX_TIER:
            raise SystemExit(f"override tier {base} must be int 0..{MAX_TIER}, got {tier!r}")
        tiers[str(base)] = tier
    unlocks_raw = data.get("unlocks") or {}
    unlocks: dict[str, dict[str, int]] = {}
    for base, spec in unlocks_raw.items():
        if not isinstance(spec, dict):
            raise SystemExit(f"override unlock {base} must be an object")
        parsed: dict[str, int] = {}
        for key, value in spec.items():
            if key not in UNLOCK_KEYS:
                raise SystemExit(f"override unlock {base} has unknown key {key!r}")
            if not isinstance(value, int) or value < 0:
                raise SystemExit(f"override unlock {base}.{key} must be int >= 0, got {value!r}")
            parsed[key] = value
        unlocks[str(base)] = parsed
    frame_tags_raw = data.get("frame_tags") or {}
    frame_tags: dict[str, str] = {}
    frame_key_re = re.compile(r"^sprites/[^/]+/\d+$")
    for key, tag in frame_tags_raw.items():
        key_s = str(key)
        if not frame_key_re.match(key_s):
            raise SystemExit(
                f"override frame_tags key {key_s!r} must look like "
                f"sprites/DefineSprite_N/frameId"
            )
        if tag not in MAP_TAGS:
            raise SystemExit(f"override frame_tags {key_s} has unknown tag {tag!r}")
        frame_tags[key_s] = str(tag)
    stamps_raw = data.get("stamps") or {}
    stamps: dict[str, dict[str, object]] = {}
    for base, spec in stamps_raw.items():
        if not isinstance(spec, dict):
            raise SystemExit(f"override stamp {base} must be an object")
        kind = str(spec.get("kind") or "")
        if kind not in STAMP_KINDS:
            raise SystemExit(f"override stamp {base} has unknown kind {kind!r}")
        footprint = spec.get("footprint_minis")
        if not isinstance(footprint, int) or footprint < 1 or footprint > 16:
            raise SystemExit(
                f"override stamp {base}.footprint_minis must be int 1..16, got {footprint!r}"
            )
        cross_reserve = spec.get("cross_reserve")
        if not isinstance(cross_reserve, bool):
            raise SystemExit(f"override stamp {base}.cross_reserve must be bool")
        profile = str(spec.get("nudge_profile") or "default")
        if profile not in NUDGE_PROFILES:
            raise SystemExit(f"override stamp {base} has unknown nudge_profile {profile!r}")
        stamp: dict[str, object] = {
            "kind": kind,
            "footprint_minis": footprint,
            "cross_reserve": cross_reserve,
            "nudge_profile": profile,
        }
        for key in ("foot_extra_x", "foot_extra_y"):
            if key in spec:
                val = spec[key]
                if not isinstance(val, int):
                    raise SystemExit(f"override stamp {base}.{key} must be int, got {val!r}")
                stamp[key] = val
        stamps[str(base)] = stamp
    return out, tiers, unlocks, frame_tags, stamps


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


def heuristic_stamp(tag: str, bbox_h: int, tier: int | None) -> dict[str, object] | None:
    """Logical footprint + foot contract (docs/map-building-placement.md)."""
    if tag not in BUILDING_TAGS:
        return None
    if tag == "landmark" or (tier is not None and tier >= 3) or bbox_h >= TALL_TIER3_H:
        return {
            "kind": "landmark_center",
            "footprint_minis": 16,
            "cross_reserve": True,
            "nudge_profile": "landmark",
        }
    if tag == "residential" and bbox_h < 45:
        return {
            "kind": "mini_foot",
            "footprint_minis": 1,
            "cross_reserve": False,
            "nudge_profile": "default",
        }
    return {
        "kind": "arterial_yard",
        "footprint_minis": 1,
        "cross_reserve": True,
        "nudge_profile": "arterial_yard",
    }


def resolve_stamp(
    tag: str,
    bbox_h: int,
    tier: int | None,
    override: dict[str, object] | None,
) -> dict[str, object] | None:
    if override is not None:
        return override
    return heuristic_stamp(tag, bbox_h, tier)


def heuristic_unlock(tag: str, tier: int | None, tree_index: int = 0) -> dict[str, int]:
    """Sector minima for map placement (issue #79). Empty dict means no gates."""
    if tag == "tree":
        idx = min(tree_index, len(TREE_ENV_THRESHOLDS) - 1)
        env = TREE_ENV_THRESHOLDS[idx]
        return {"min_env": env} if env > 0 else {}

    if tag not in BUILDING_TAGS or tier is None:
        return {}

    unlock: dict[str, int] = {}
    if tier >= 3:
        unlock["min_pop"] = POP_TIER_HUGE
    elif tier >= 2:
        unlock["min_pop"] = POP_TIER_NORMAL
    elif tier >= 1:
        unlock["min_pop"] = POP_TIER_PEON

    if tag == "industrial":
        unlock["min_ind"] = SECTOR_IND if tier >= 2 else 1
    elif tag == "commercial":
        unlock["min_com"] = SECTOR_COM if tier >= 2 else 1
    elif tag == "landmark":
        unlock["min_pop"] = max(unlock.get("min_pop", 0), POP_TIER_NORMAL)
        unlock["min_sec"] = SECTOR_SEC

    return unlock


def load_swf_graph(path: Path) -> dict[int, dict[str, object]]:
    if not path.is_file():
        return {}
    data = json.loads(path.read_text(encoding="utf-8"))
    out: dict[int, dict[str, object]] = {}
    for sprite in data.get("sprites") or []:
        cid = sprite.get("character_id")
        if isinstance(cid, int):
            out[cid] = sprite
    return out


def character_id_from_base(base: str) -> int | None:
    m = SPRITE_ID_RE.search(base)
    if not m:
        return None
    return int(m.group(1))


def denied_building_base(base: str) -> bool:
    lower = base.lower()
    return any(token.lower() in lower for token in BUILDING_DENY_SUBSTR)


def merge_swf_graph(entry: dict[str, object], sprite: dict[str, object]) -> None:
    if sprite.get("exported_name"):
        entry["exported_name"] = sprite["exported_name"]
    for field in (
        "frame_count",
        "role",
        "library_ref",
        "parent_primaries",
        "child_repeat_counts",
        "needed_direct",
        "needed_characters",
        "dependent_direct",
        "dependent_characters",
    ):
        if field in sprite and sprite[field] is not None:
            entry[field] = sprite[field]
    entry["placeable_hint"] = bool(sprite.get("placeable_hint"))
    entry["place_count"] = int(sprite.get("place_count") or 0)
    entry["pool_eligible"] = bool(sprite.get("pool_eligible"))


def tag_for_library_primary(
    folder_name: str,
    opaque: int,
    bbox_h: int,
    bbox_w: int,
    colors: dict[str, float],
    tag_override: str | None,
) -> tuple[str, str]:
    """Library-frame primaries follow the same override contract as other clips.

    Building-tag overrides keep them in the spawn pool. Non-building overrides
    (ground, exclude, …) pull farm/floor clips out of that pool (#89).
    """
    group, tag = classify(folder_name, opaque, bbox_h, bbox_w, colors)
    return apply_override(group, tag, tag_override)


def finalize_pool_entry(
    entry: dict[str, object],
    base: str,
    *,
    catalog_override: str | None = None,
) -> None:
    role = str(entry.get("role") or "")
    tag = str(entry.get("tag") or "exclude")
    pool = (
        bool(entry.get("pool_eligible"))
        and role in POOL_ROLES
        and tag in BUILDING_TAGS
        and not denied_building_base(base)
    )
    entry["pool_eligible"] = pool
    if pool:
        entry["group"] = "building"
        return
    if tag in BUILDING_TAGS:
        entry["building_class"] = tag
        # Explicit override keeps the Storybook/catalog tag; spawn pool still
        # closed for modules (#82). Heuristic-only building tags stay exclude.
        if catalog_override is not None and catalog_override == tag:
            entry["group"] = "building"
            return
        entry["tag"] = "exclude"
        entry["group"] = "other"
        entry.pop("tier", None)
        entry.pop("unlock", None)


def main() -> None:
    if not NORMALIZED.is_dir():
        raise SystemExit(f"missing normalized sprites dir: {NORMALIZED}")

    tag_overrides, tier_overrides, unlock_overrides, frame_tags, stamp_overrides = load_overrides(OVERRIDES)
    swf_graph = load_swf_graph(SWF_GRAPH)
    entries: list[dict[str, object]] = []
    tree_index = 0

    for folder in sorted(NORMALIZED.iterdir()):
        if not folder.is_dir():
            continue
        opaque, bbox_h, bbox_w, colors = sprite_metrics(folder)
        base = f"sprites/{folder.name}"
        cid = character_id_from_base(base)
        sprite = swf_graph.get(cid) if cid is not None else None
        tag_override = tag_overrides.get(base)
        if sprite and sprite.get("role") == "library_primary":
            group, tag = tag_for_library_primary(
                folder.name, opaque, bbox_h, bbox_w, colors, tag_override
            )
        else:
            group, tag = classify(folder.name, opaque, bbox_h, bbox_w, colors)
            group, tag = apply_override(group, tag, tag_override)
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
        stamp = resolve_stamp(tag, bbox_h, tier, stamp_overrides.get(base))
        if stamp is not None:
            entry["stamp"] = stamp
        unlock = unlock_overrides.get(base)
        if unlock is None:
            unlock = heuristic_unlock(tag, tier, tree_index)
        if unlock:
            entry["unlock"] = unlock
        if cid is not None:
            entry["character_id"] = cid
            if sprite:
                merge_swf_graph(entry, sprite)
        finalize_pool_entry(entry, base, catalog_override=tag_override)
        if tag == "tree" and entry.get("tag") == "tree":
            tree_index += 1
        entries.append(entry)

    bases_by_tag: dict[str, list[str]] = {tag: [] for tag in MAP_TAGS}
    for entry in entries:
        bases_by_tag[str(entry["tag"])].append(str(entry["base"]))
    for tag in MAP_TAGS:
        bases_by_tag[tag].sort()

    building_bases = sorted(
        str(entry["base"])
        for entry in entries
        if entry.get("pool_eligible") and entry.get("tag") in BUILDING_TAGS
    )

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
        "swf_graph": str(SWF_GRAPH.relative_to(ROOT)) if SWF_GRAPH.is_file() else None,
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
            "unlock": "min_pop/ind/com/env/sec/tra gates for map placement (#79); optional overrides.unlocks",
            "stamp": "logical footprint + foot contract per folder; heuristic from tag+bbox; optional overrides.stamps",
            "needed_characters": "JPEXS Needed Characters (transitive PlaceObject graph from FFDec swf2xml)",
            "dependent_characters": "JPEXS Dependent Characters (sprites that instance this clip, transitive)",
            "placeable_hint": "false when other sprites depend on this clip (child module); mc* exports stay true",
            "role": "spawn_library | library_primary | building_module | deco_subpart | standalone | ui | other",
            "pool_eligible": "true only for mcHouse1/2/3 library-frame primaries with a building tag",
            "library_ref": "mcHouse library frame index when role=library_primary",
            "building_class": "building tag kept when a heuristic module is demoted to exclude; explicit overrides keep tag for Storybook",
            "frame_tags": "per-frame Storybook retags (sprites/Base/frameId); does not change map spawn pool",
        },
        "counts": {
            "building": len(building_bases),
            "pool_eligible": len(building_bases),
            "character": group_counts.get("character", 0),
            "ui": group_counts.get("ui", 0),
            "other": group_counts.get("other", 0),
            "by_tag": tag_counts,
            "by_tier": by_tier,
        },
        "building_bases": building_bases,
        "bases_by_tag": bases_by_tag,
        "frame_tags": dict(sorted(frame_tags.items())),
        "entries": entries,
    }

    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")
    print(f"wrote {OUT} ({len(building_bases)} building bases)")
    print("by_tag:", json.dumps(tag_counts, sort_keys=True))


if __name__ == "__main__":
    main()
