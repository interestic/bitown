import {
  DALLE_CELLS,
  DALLE_TOP_H,
  CROSS_STAMP_FOOT_LOCAL,
  CROSS_STAMP_NUDGE_Y,
  CROSS_STAMP_SE_LOCAL,
  ISO_TILE_H,
  ISO_TILE_W,
} from "./lotPatterns";
import type { RoadEdge } from "./roadDiamondLayouts";
import type { FrameRect } from "./types";

export type PlotStamp = {
  frame: FrameRect;
  cell?: readonly [number, number];
  /** Clip stamp pixels to the dalle grass top (farm overlays). */
  clipToDalleTop?: boolean;
  /** Screen-space nudge after foot placement (API westMiniStampNudgeX). */
  offsetX?: number;
  offsetY?: number;
  /** Paint order; default 2 (roads/cross use 1 so they sit under huts). */
  z?: number;
};

export type EdgeClipMode = "rim" | "shared" | "exterior";

export type PlotLayer = {
  frame: FrameRect;
  x: number;
  y: number;
  z: number;
  clipToDalleTop?: boolean;
  clipEdge?: RoadEdge;
  clipEdgeMode?: EdgeClipMode;
  /** Foot used for clip polygons (grid tiles offset from origin). */
  clipFootX?: number;
  clipFootY?: number;
  /** Block-center hub in foot-local coords (exterior rim). */
  clipBlockCenter?: Pt;
  /** Precomputed world-space clip (grid junction caps). */
  clipWorldPolygon?: Pt[];
  flipX?: boolean;
};

type Pt = readonly [number, number];

export const OPPOSITE_EDGE: Record<RoadEdge, RoadEdge> = {
  ne: "sw",
  se: "nw",
  sw: "ne",
  nw: "se",
};

/** Grid step from a tile toward the neighbor across `edge`. */
export const EDGE_NEIGHBOR: Record<RoadEdge, readonly [number, number]> = {
  se: [1, 0],
  nw: [-1, 0],
  sw: [0, 1],
  ne: [0, -1],
};

function dalleLip(dalle: FrameRect): number {
  return Math.max(0, dalle.h - DALLE_TOP_H);
}

function cellFootDelta(cx: number, cy: number): { dx: number; dy: number } {
  const se = DALLE_CELLS - 1;
  return {
    dx: (cx - cy) * (ISO_TILE_W / 2),
    dy: (cx + cy - se * 2) * (ISO_TILE_H / 2),
  };
}

function dalleVerts(dalle: FrameRect): {
  N: Pt;
  E: Pt;
  S: Pt;
  W: Pt;
  C: Pt;
} {
  const halfW = dalle.w / 2;
  const topY = -dalle.h;
  const midY = topY + DALLE_TOP_H / 2;
  const botY = topY + DALLE_TOP_H;
  return {
    N: [0, topY],
    E: [halfW, midY],
    S: [0, botY],
    W: [-halfW, midY],
    C: [0, midY],
  };
}

function lerp(a: Pt, b: Pt, t: number): Pt {
  return [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t];
}

export function edgeBand(
  dalle: FrameRect,
  edge: RoadEdge,
  depth = 0.42,
  rimInset = 0.06,
  mode: EdgeClipMode = "rim",
  blockCenter?: Pt,
): Pt[] {
  const { N, E, S, W, C } = dalleVerts(dalle);
  const pair: Record<RoadEdge, [Pt, Pt]> = {
    ne: [N, E],
    se: [E, S],
    sw: [S, W],
    nw: [W, N],
  };
  const [a, b] = pair[edge];
  if (mode === "exterior") {
    // Inset toward block center; slight outward reach at rim vertices for corner tips.
    const hub: Pt = blockCenter ?? C;
    const bandDepth = 0.44;
    const extend = 0.1;
    const aOut = lerp(a, C, -extend);
    const bOut = lerp(b, C, -extend);
    return [aOut, bOut, lerp(bOut, hub, bandDepth), lerp(aOut, hub, bandDepth)];
  }
  if (mode === "shared") {
    // Full rim + slight outward reach so segments meet at tile borders and corners.
    const innerDepth = 0.58;
    const extend = 0.035;
    const aOut = lerp(a, C, -extend);
    const bOut = lerp(b, C, -extend);
    return [aOut, bOut, lerp(bOut, C, innerDepth), lerp(aOut, C, innerDepth)];
  }
  const aOut = lerp(a, C, rimInset);
  const bOut = lerp(b, C, rimInset);
  return [aOut, bOut, lerp(bOut, C, depth), lerp(aOut, C, depth)];
}

function edgeRoadOffset(
  dalle: FrameRect,
  edge: RoadEdge,
  mode: EdgeClipMode = "rim",
): { dx: number; dy: number; flipX: boolean } {
  const { N, E, S, W, C } = dalleVerts(dalle);
  const mid: Record<RoadEdge, Pt> = {
    ne: [(N[0] + E[0]) / 2, (N[1] + E[1]) / 2],
    se: [(E[0] + S[0]) / 2, (E[1] + S[1]) / 2],
    sw: [(S[0] + W[0]) / 2, (S[1] + W[1]) / 2],
    nw: [(W[0] + N[0]) / 2, (W[1] + N[1]) / 2],
  };
  const m = mid[edge];
  const pull =
    mode === "exterior" ? 0.9 : mode === "shared" ? 0.95 : 0.72;
  return {
    dx: (m[0] - C[0]) * pull,
    dy: (m[1] - C[1]) * pull,
    flipX: edge === "ne" || edge === "sw",
  };
}

function placeAtFoot(
  frame: FrameRect,
  footX: number,
  footY: number,
  z: number,
  opts: {
    clipToDalleTop?: boolean;
    clipEdge?: RoadEdge;
    clipEdgeMode?: EdgeClipMode;
    clipFootX?: number;
    clipFootY?: number;
    clipBlockCenter?: Pt;
    clipWorldPolygon?: Pt[];
    flipX?: boolean;
  } = {},
): PlotLayer {
  return {
    frame,
    x: footX - frame.anchor_x,
    y: footY - frame.anchor_y,
    z,
    clipToDalleTop: opts.clipToDalleTop,
    clipEdge: opts.clipEdge,
    clipEdgeMode: opts.clipEdgeMode,
    clipFootX: opts.clipFootX,
    clipFootY: opts.clipFootY,
    clipBlockCenter: opts.clipBlockCenter,
    clipWorldPolygon: opts.clipWorldPolygon,
    flipX: opts.flipX,
  };
}

function overlayFoot(dalle: FrameRect, overlay: FrameRect, lip: number): { x: number; y: number } {
  const fullPlate = overlay.w >= dalle.w * 0.75;
  if (fullPlate) {
    return { x: 0, y: -lip };
  }
  const mid = cellFootDelta(1, 1);
  return { x: mid.dx, y: mid.dy - lip };
}

export type BuildPlotLayersInput = {
  dalle: FrameRect;
  footX?: number;
  footY?: number;
  overlay?: FrameRect;
  clipOverlayToDalle?: boolean;
  roadEdges?: RoadEdge[];
  roadFrame?: FrameRect;
  /** Per-edge clip mode for grid connectivity. */
  roadEdgeModes?: Partial<Record<RoadEdge, EdgeClipMode>>;
  /** World foot of grid block center (exterior rim clips inset toward this). */
  blockCenter?: Pt;
  stamps?: PlotStamp[];
};

/** Layers for one dalle plate; foot defaults to (0,0). */
export function buildPlotLayers({
  dalle,
  footX = 0,
  footY = 0,
  overlay,
  clipOverlayToDalle = false,
  roadEdges,
  roadFrame,
  roadEdgeModes,
  blockCenter,
  stamps = [],
}: BuildPlotLayersInput): PlotLayer[] {
  const lip = dalleLip(dalle);
  const layers: PlotLayer[] = [placeAtFoot(dalle, footX, footY, 0)];

  if (overlay) {
    const foot = overlayFoot(dalle, overlay, lip);
    layers.push(
      placeAtFoot(overlay, footX + foot.x, footY + foot.y, 1, {
        clipToDalleTop: clipOverlayToDalle,
        clipFootX: footX,
        clipFootY: footY,
      }),
    );
  }

  if (roadFrame && roadEdges && roadEdges.length > 0) {
    const base = overlayFoot(dalle, roadFrame, lip);
    const hubLocal: Pt | undefined = blockCenter
      ? [blockCenter[0] - footX, blockCenter[1] - footY]
      : undefined;
    for (const edge of roadEdges) {
      const mode = roadEdgeModes?.[edge] ?? "rim";
      const { dx, dy, flipX } = edgeRoadOffset(dalle, edge, mode);
      layers.push(
        placeAtFoot(roadFrame, footX + base.x + dx, footY + base.y + dy, 1, {
          clipEdge: edge,
          clipEdgeMode: mode,
          clipFootX: footX,
          clipFootY: footY,
          clipBlockCenter: hubLocal,
          flipX,
        }),
      );
    }
  }

  for (const stamp of stamps) {
    const cell = stamp.cell ?? ([1, 1] as const);
    const delta = cellFootDelta(cell[0], cell[1]);
    layers.push(
      placeAtFoot(
        stamp.frame,
        footX + delta.dx + (stamp.offsetX ?? 0),
        footY + delta.dy - lip + (stamp.offsetY ?? 0),
        stamp.z ?? 2,
        {
          clipToDalleTop: stamp.clipToDalleTop,
          clipFootX: footX,
          clipFootY: footY,
        },
      ),
    );
  }

  return layers;
}

/** Iso foot of the SE corner of a 4×4 dalle block (relative to block 0,0 foot). */
export function dalleBlockFootDelta(col: number, row: number): { dx: number; dy: number } {
  const ax = col * DALLE_CELLS + (DALLE_CELLS - 1);
  const ay = row * DALLE_CELLS + (DALLE_CELLS - 1);
  const bx = DALLE_CELLS - 1;
  const by = DALLE_CELLS - 1;
  const topDx = (ax - ay - (bx - by)) * (ISO_TILE_W / 2);
  const topDy = (ax + ay - bx - by) * (ISO_TILE_H / 2);
  return { dx: topDx, dy: topDy };
}

export function polygonCss(pts: Pt[], originX: number, originY: number, scale: number): string {
  const local = pts
    .map(([px, py]) => `${(px - originX) * scale}px ${(py - originY) * scale}px`)
    .join(", ");
  return `polygon(${local})`;
}

export function layerClipPath(
  layer: PlotLayer,
  dalle: FrameRect,
  boxOriginX: number,
  boxOriginY: number,
  scale: number,
): string | undefined {
  const fx = layer.clipFootX ?? 0;
  const fy = layer.clipFootY ?? 0;
  const toWorld = (pts: Pt[]): Pt[] => pts.map(([px, py]) => [px + fx, py + fy]);

  if (layer.clipWorldPolygon) {
    return polygonCss(layer.clipWorldPolygon, boxOriginX, boxOriginY, scale);
  }

  if (layer.clipEdge) {
    const mode = layer.clipEdgeMode ?? "rim";
    return polygonCss(
      toWorld(
        edgeBand(dalle, layer.clipEdge, 0.42, 0.06, mode, layer.clipBlockCenter),
      ),
      boxOriginX,
      boxOriginY,
      scale,
    );
  }
  if (!layer.clipToDalleTop) return undefined;
  const { N, E, S, W } = dalleVerts(dalle);
  return polygonCss(toWorld([E, S, W, N]), boxOriginX, boxOriginY, scale);
}

export function boundsForLayers(layers: PlotLayer[], scale: number, pad = 12) {
  const minX = Math.min(...layers.map((l) => l.x));
  const minY = Math.min(...layers.map((l) => l.y));
  const maxX = Math.max(...layers.map((l) => l.x + l.frame.w));
  const maxY = Math.max(...layers.map((l) => l.y + l.frame.h));
  const boxW = (maxX - minX) * scale + pad * 2;
  const boxH = (maxY - minY) * scale + pad * 2;
  const originX = minX - pad / scale;
  const originY = minY - pad / scale;
  return { minX, minY, maxX, maxY, boxW, boxH, originX, originY, pad };
}

/** True when `edge` faces another tile inside a rows×cols grid. */
export function edgeFacesInternalNeighbor(
  col: number,
  row: number,
  edge: RoadEdge,
  cols: number,
  rows: number,
): boolean {
  const [dc, dr] = EDGE_NEIGHBOR[edge];
  const nc = col + dc;
  const nr = row + dr;
  return nc >= 0 && nr >= 0 && nc < cols && nr < rows;
}

/** Iso foot of block center for a rows×cols dalle grid. */
export function blockCenterFoot(
  dalle: FrameRect,
  cols: number,
  rows: number,
): Pt {
  const { C } = dalleVerts(dalle);
  let sx = 0;
  let sy = 0;
  let n = 0;
  for (let row = 0; row < rows; row++) {
    for (let col = 0; col < cols; col++) {
      const { dx, dy } = dalleBlockFootDelta(col, row);
      sx += dx + C[0];
      sy += dy + C[1];
      n++;
    }
  }
  return [sx / n, sy / n];
}


/**
 * Game.hx genSquare BIG ROADS / CROSS ROADS on a dalle grid.
 *
 * Uses orientation-specific 702 frames (dir0 vs dir1 — not a CSS flip).
 * Clip is expanded past the tile so stamps bleed slightly into neighbours.
 */
export function buildGenSquareRoadLayers(
  dalle: FrameRect,
  roadFrames: readonly [FrameRect, FrameRect],
  tiles: { dirs: (0 | 1)[]; cross?: boolean }[][],
  crossFrame?: FrameRect,
): PlotLayer[] {
  const lip = dalleLip(dalle);
  const crossBase = crossFrame ? { x: 0, y: -lip } : null;
  const layers: PlotLayer[] = [];

  for (let row = 0; row < tiles.length; row++) {
    for (let col = 0; col < tiles[row].length; col++) {
      const tile = tiles[row][col];
      const { dx, dy } = dalleBlockFootDelta(col, row);

      for (const dir of tile.dirs) {
        const roadFrame = roadFrames[dir];
        // Pin to square foot (genSquare stamps at square origin — not mid-cell).
        const base = { x: 0, y: -lip };
        const nudge = axisOverlapNudge(dir, col, row, tiles);
        layers.push(
          placeAtFoot(roadFrame, dx + base.x + nudge.ox, dy + base.y + nudge.oy, 1, {
            clipWorldPolygon: expandedDalleTopWorld(dalle, dx, dy, 0.22),
          }),
        );
      }

      if (tile.cross && crossFrame && crossBase) {
        // API squareCrossFoot: local 7 vs SE 9 → Δcell (−2,−2) + CROSS_STAMP_NUDGE_Y.
        const dCell = CROSS_STAMP_FOOT_LOCAL - CROSS_STAMP_SE_LOCAL;
        const crossOx = (dCell - dCell) * (ISO_TILE_W / 2);
        const crossOy = (dCell + dCell) * (ISO_TILE_H / 2) + CROSS_STAMP_NUDGE_Y;
        layers.push(
          placeAtFoot(
            crossFrame,
            dx + crossBase.x + crossOx,
            dy + crossBase.y + crossOy,
            2,
            {
              clipWorldPolygon: expandedDalleTopWorld(dalle, dx, dy, 0.12),
            },
          ),
        );
      }
    }
  }

  return layers;
}

/** Dalle top diamond expanded past the rim so roads can overlap neighbours. */
function expandedDalleTopWorld(
  dalle: FrameRect,
  footX: number,
  footY: number,
  expand: number,
): Pt[] {
  const { N, E, S, W, C } = dalleVerts(dalle);
  const push = (p: Pt): Pt => [
    footX + C[0] + (p[0] - C[0]) * (1 + expand),
    footY + C[1] + (p[1] - C[1]) * (1 + expand),
  ];
  return [push(N), push(E), push(S), push(W)];
}

/**
 * Nudge a stamp toward neighbouring tiles that share the same axis,
 * so square-cut ends overlap at the shared edge.
 */
function axisOverlapNudge(
  dir: 0 | 1,
  col: number,
  row: number,
  tiles: { dirs: (0 | 1)[] }[][],
): { ox: number; oy: number } {
  const pull = 0.1;
  let ox = 0;
  let oy = 0;
  const here = dalleBlockFootDelta(col, row);
  const candidates =
    dir === 0
      ? [
          [col - 1, row],
          [col + 1, row],
        ]
      : [
          [col, row - 1],
          [col, row + 1],
        ];
  for (const [nc, nr] of candidates) {
    if (nr < 0 || nr >= tiles.length || nc < 0 || nc >= tiles[nr].length) continue;
    if (!tiles[nr][nc].dirs.includes(dir)) continue;
    const there = dalleBlockFootDelta(nc, nr);
    ox += (there.dx - here.dx) * pull;
    oy += (there.dy - here.dy) * pull;
  }
  return { ox, oy };
}
