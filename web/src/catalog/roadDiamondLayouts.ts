/**
 * Road sprites for genSquare BIG ROADS (Game.hx size=3).
 *
 * DefineSprite_702_mcRoad has 6 frames = 2 axes × 3 styles:
 *   dir0 → frames 1,2,3   (one iso diagonal)
 *   dir1 → frames 4,5,6   (the other diagonal — separate art, not a flip)
 *
 * DefineSprite_724 is a single-orientation dashed arterial (legacy rim demos).
 */

export type RoadEdge = "ne" | "se" | "sw" | "nw";

export type RoadDir = 0 | 1;

/** Parent mcRoad brush used by genSquare BIG ROADS (size=3 types 0..5). */
export const BIG_ROAD_SPRITE_BASE = "sprites/DefineSprite_702_mcRoad";

/** Legacy single-diagonal arterial (flip was used for the second axis). */
export const ROAD_SPRITE_BASE = "sprites/DefineSprite_724";

export const CROSS_SPRITE_BASE = "sprites/DefineSprite_705";

/** Style index within an axis: 0=細暗, 1=土, 2=アスファルト. */
export const ROAD_STYLE_FRAMES = ["1", "2", "3"] as const;
export type RoadStyleFrame = (typeof ROAD_STYLE_FRAMES)[number];

export const ROAD_STYLE_LABELS: Record<RoadStyleFrame, string> = {
  "1": "細暗",
  "2": "土",
  "3": "アスファルト",
};

function normalizeRoadVariantIndex(variantIndex: number): number {
  if (!Number.isFinite(variantIndex)) return 0;
  return Math.min(
    Math.max(Math.trunc(variantIndex), 0),
    ROAD_STYLE_FRAMES.length - 1,
  );
}

export function roadStyleFrameFromVariant(variantIndex: number): RoadStyleFrame {
  const idx = normalizeRoadVariantIndex(variantIndex);
  return ROAD_STYLE_FRAMES[idx];
}

/**
 * Game.hx `fr = 3*n + style` → mcHouse type frame id (1-based).
 * dir0 → 1..3, dir1 → 4..6.
 */
export function bigRoadFrameIdForDir(
  dir: RoadDir,
  variantIndex: number,
): string {
  const style = normalizeRoadVariantIndex(variantIndex);
  return String(dir * 3 + style + 1);
}

/** 705: 1=土系, 2=アスファルト. */
export function crossFrameIdFromVariant(variantIndex: number): "1" | "2" {
  return normalizeRoadVariantIndex(variantIndex) >= 2 ? "2" : "1";
}
