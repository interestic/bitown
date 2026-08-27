#!/usr/bin/env python3
"""
Step7 quality gate for sprites pipeline artifacts.

Requires pipeline directories (raw/normalized/variants) to exist. When atlas
artifacts are committed, also validates the atlas PNG, JSON frames, and
buildings.json catalog tags.
"""

from pathlib import Path
import json
import struct
import sys

ROOT = Path(__file__).resolve().parent.parent

required_dirs = [
    ROOT / "assets" / "sprites-v1" / "raw",
    ROOT / "assets" / "sprites-v1" / "normalized",
    ROOT / "assets" / "sprites-v1" / "variants",
]

atlas_json = ROOT / "assets" / "sprites-v1" / "atlas" / "sprites_v1_atlas.json"
buildings_json = ROOT / "assets" / "sprites-v1" / "buildings.json"
building_allowlist = ROOT / "assets" / "sprites-v1" / "building_bases.allowlist"

MAP_TAGS = {
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
}
BUILDING_TAGS = {"residential", "industrial", "commercial", "landmark"}
NON_BUILDING_TAGS = MAP_TAGS - BUILDING_TAGS
UNLOCK_KEYS = frozenset({"min_pop", "min_ind", "min_com", "min_env", "min_sec", "min_tra"})

# Substrings that must never appear in the map building pool.
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


def fail(msg: str) -> None:
    print(f"[FAIL] {msg}")
    sys.exit(1)


def png_size(path: Path) -> tuple[int, int]:
    with path.open("rb") as f:
        if f.read(8) != b"\x89PNG\r\n\x1a\n":
            fail(f"not a PNG: {path}")
        length, chunk = struct.unpack(">I4s", f.read(8))
        if chunk != b"IHDR" or length < 8:
            fail(f"PNG missing IHDR: {path}")
        width, height = struct.unpack(">II", f.read(8))
        return width, height


def validate_atlas_frames(data: dict, png_path: Path) -> None:
    frames = data["frames"]
    png_w, png_h = png_size(png_path)
    square32 = 0
    for key, frame in frames.items():
        if not isinstance(frame, dict):
            fail(f"frame {key} is not an object")
        for field in ("x", "y", "w", "h", "anchor_x", "anchor_y"):
            value = frame.get(field)
            if not isinstance(value, int):
                fail(f"frame {key} missing int {field}")
        x, y, w, h = frame["x"], frame["y"], frame["w"], frame["h"]
        ax, ay = frame["anchor_x"], frame["anchor_y"]
        if w <= 0 or h <= 0:
            fail(f"frame {key} has non-positive size")
        if x < 0 or y < 0 or x + w > png_w or y + h > png_h:
            fail(f"frame {key} exceeds atlas png {png_w}x{png_h}")
        # Foot anchors map the normalized canvas foot into trimmed-frame space
        # and may sit outside the opaque bbox (asymmetric sprites). Native-height
        # towers use canvases up to ~528px, so slack matches isoPad.
        slack = 528
        if ax < -slack or ay < -slack or ax > w + slack or ay > h + slack:
            fail(f"frame {key} anchor implausibly far from frame")
        if w == 32 and h == 32:
            square32 += 1
    if len(frames) > 0 and square32 == len(frames):
        fail("atlas frames are still all 32x32; expected trimmed native sizes")
    print(f"[OK] atlas bounds {png_w}x{png_h}; native frames {len(frames) - square32}/{len(frames)}")


def validate_buildings_manifest(data: dict) -> None:
    version = data.get("version")
    if not isinstance(version, int) or version < 2:
        fail("buildings.json version must be integer >= 2")

    bases = data.get("building_bases") or []
    if not isinstance(bases, list) or not bases:
        fail("buildings.json has no building_bases")
    if bases != sorted(set(bases)):
        fail("building_bases must be unique and sorted")

    by_tag = data.get("bases_by_tag") or {}
    if not isinstance(by_tag, dict):
        fail("buildings.json missing bases_by_tag")
    missing_tags = MAP_TAGS - set(by_tag)
    if missing_tags:
        fail(f"bases_by_tag missing keys: {sorted(missing_tags)}")

    counts = data.get("counts") or {}
    tag_counts = counts.get("by_tag") or {}
    for tag in sorted(MAP_TAGS):
        tagged = by_tag.get(tag) or []
        if tagged != sorted(set(tagged)):
            fail(f"bases_by_tag[{tag}] must be unique and sorted")
        if tag_counts.get(tag) != len(tagged):
            fail(f"counts.by_tag[{tag}] != len(bases_by_tag[{tag}])")

    expected_buildings = []
    for tag in ("residential", "industrial", "commercial", "landmark"):
        expected_buildings.extend(by_tag.get(tag) or [])
    expected_buildings = sorted(expected_buildings)
    if bases != expected_buildings:
        fail("building_bases must equal union of residential/industrial/commercial/landmark")
    if counts.get("building") != len(bases):
        fail("counts.building != len(building_bases)")

    entries = data.get("entries") or []
    if not isinstance(entries, list) or not entries:
        fail("buildings.json has no entries")
    seen = set()
    for entry in entries:
        base = entry.get("base")
        tag = entry.get("tag")
        group = entry.get("group")
        if not base or tag not in MAP_TAGS:
            fail(f"invalid catalog entry: {entry}")
        if group not in {"building", "character", "ui", "other"}:
            fail(f"invalid group for {base}: {group}")
        if base in seen:
            fail(f"duplicate catalog entry {base}")
        seen.add(base)
        tagged = by_tag.get(tag) or []
        if base not in tagged:
            fail(f"{base} tag {tag} missing from bases_by_tag[{tag}]")
        if tag in BUILDING_TAGS and group != "building":
            fail(f"{base} has building tag {tag} but group {group}")
        if tag in BUILDING_TAGS and base not in bases:
            if entry.get("pool_eligible") is True:
                fail(f"{base} pool_eligible but missing from building_bases")
            # Former building modules keep building_class while tagged exclude.
        if entry.get("pool_eligible") is True and base not in bases:
            fail(f"{base} pool_eligible=true but not in building_bases")
        if entry.get("pool_eligible") is True and tag not in BUILDING_TAGS:
            fail(f"{base} pool_eligible=true but tag is {tag}")
        if base in bases and entry.get("pool_eligible") is not True:
            fail(f"{base} in building_bases but pool_eligible is not true")
        if tag in NON_BUILDING_TAGS and base in bases:
            fail(f"{base} tagged {tag} must not be in building_bases")
        if tag in BUILDING_TAGS:
            tier = entry.get("tier")
            if not isinstance(tier, int) or tier < 0 or tier > 3:
                fail(f"{base} building entry needs int tier 0..3, got {tier!r}")
            unlock = entry.get("unlock")
            if unlock is not None:
                if not isinstance(unlock, dict):
                    fail(f"{base} unlock must be an object")
                for key, value in unlock.items():
                    if key not in UNLOCK_KEYS:
                        fail(f"{base} unlock has unknown key {key!r}")
                    if not isinstance(value, int) or value < 0:
                        fail(f"{base} unlock.{key} must be int >= 0")
        if tag == "tree":
            unlock = entry.get("unlock")
            if unlock is not None:
                if not isinstance(unlock, dict):
                    fail(f"{base} unlock must be an object")
                for key, value in unlock.items():
                    if key not in UNLOCK_KEYS:
                        fail(f"{base} unlock has unknown key {key!r}")
                    if not isinstance(value, int) or value < 0:
                        fail(f"{base} unlock.{key} must be int >= 0")
        for field in ("needed_characters", "needed_direct", "dependent_characters", "dependent_direct"):
            val = entry.get(field)
            if val is None:
                continue
            if not isinstance(val, list) or any(not isinstance(x, int) or x < 0 for x in val):
                fail(f"{base} {field} must be a list of non-negative ints")
        if "character_id" in entry and not isinstance(entry.get("character_id"), int):
            fail(f"{base} character_id must be int")
        if "pool_eligible" in entry and not isinstance(entry.get("pool_eligible"), bool):
            fail(f"{base} pool_eligible must be bool")
        role = entry.get("role")
        if role is not None and not isinstance(role, str):
            fail(f"{base} role must be string")
        lib_ref = entry.get("library_ref")
        if lib_ref is not None:
            if not isinstance(lib_ref, dict):
                fail(f"{base} library_ref must be object")
            for key in ("library_id", "library_name", "frame"):
                if key not in lib_ref:
                    fail(f"{base} library_ref missing {key}")

    by_tier = counts.get("by_tier") or {}
    if not isinstance(by_tier, dict):
        fail("counts.by_tier missing")
    expected_tier = {0: 0, 1: 0, 2: 0, 3: 0}
    for entry in entries:
        if entry.get("tag") in BUILDING_TAGS:
            expected_tier[int(entry["tier"])] += 1
    for tier, n in expected_tier.items():
        key = str(tier)
        if by_tier.get(key) != n:
            fail(f"counts.by_tier[{key}] != building entries with tier {tier}")

    for base in bases:
        lowered = base
        for token in BUILDING_DENY_SUBSTR:
            if token.lower() in lowered.lower():
                fail(f"building_bases contains denied sprite {base} ({token})")

    for tag in NON_BUILDING_TAGS:
        for base in by_tag.get(tag) or []:
            if base in bases:
                fail(f"{base} tagged {tag} must not be in building_bases")

    if not building_allowlist.exists():
        fail(f"missing building allowlist snapshot: {building_allowlist}")
    snapshot = [
        line.strip()
        for line in building_allowlist.read_text(encoding="utf-8").splitlines()
        if line.strip() and not line.strip().startswith("#")
    ]
    if snapshot != bases:
        fail("building_bases.allowlist must match buildings.json building_bases exactly")

    print(f"[OK] building bases: {len(bases)}")
    print(f"[OK] building allowlist snapshot: {len(snapshot)}")
    print(f"[OK] catalog tags: {json.dumps(tag_counts, sort_keys=True)}")


def main() -> None:
    atlas_present = atlas_json.exists()
    for d in required_dirs:
        if not d.exists():
            if atlas_present:
                print(f"[WARN] optional pipeline dir missing: {d}")
            else:
                fail(f"missing directory: {d}")

    if atlas_present:
        data = json.loads(atlas_json.read_text(encoding="utf-8"))
        if "frames" not in data or not isinstance(data["frames"], dict):
            fail("atlas json has no valid frames object")
        print(f"[OK] atlas frames: {len(data['frames'])}")
        image_name = Path(str(data.get("image") or "sprites_v1_atlas.png")).name
        atlas_png = atlas_json.parent / image_name
        if not atlas_png.exists():
            fail(f"atlas png missing: {atlas_png}")
        print(f"[OK] atlas png: {atlas_png.name}")
        validate_atlas_frames(data, atlas_png)
        if not buildings_json.exists():
            fail("buildings.json missing (run: python3 scripts/generate_buildings_manifest.py)")
        building_data = json.loads(buildings_json.read_text(encoding="utf-8"))
        validate_buildings_manifest(building_data)
    else:
        print("[WARN] atlas json not found (run step5 when assets are ready)")

    print("[OK] asset pipeline quality gate passed")


if __name__ == "__main__":
    main()
