/**
 * Road grid scenarios matching Game.hx genSquare BIG ROADS / CROSS ROADS.
 *
 * Each square stamps up to two axis roads (n in 0...2) plus optional 705 cross.
 * Continuous arterials emerge when neighbouring squares share a direction —
 * not from clipping a rim stamp onto a 2×2 dalle outer edge.
 */

/** Game.hx `for (n in 0...2)` direction index. */
export type RoadDir = 0 | 1;

/** One dalle square in the grid. Row 0 = north, col 0 = west. */
export type GridTileSpec = {
  /** Axes that get a size=3 (702_mcRoad) stamp. */
  dirs: RoadDir[];
  /** size=4 CROSS ROADS (705) on this square. */
  cross?: boolean;
};

/**
 * Dense mesh — every square both axes + cross (Ventura-like intersections).
 *
 *     NW [0,1]+X   NE [0,1]+X
 *     SW [0,1]+X   SE [0,1]+X
 */
export const ROAD_GRID_MESH_2X2: GridTileSpec[][] = [
  [
    { dirs: [0, 1], cross: true },
    { dirs: [0, 1], cross: true },
  ],
  [
    { dirs: [0, 1], cross: true },
    { dirs: [0, 1], cross: true },
  ],
];

/**
 * Shared-axis corridors — neighbours with the same dir meet flush.
 *
 *     NW [0]   NE [0]     ← dir0 continuous across the north row
 *     SW [1]   SE [1]     ← dir1 continuous across the south row
 */
export const ROAD_GRID_AXIS_2X2: GridTileSpec[][] = [
  [{ dirs: [0] }, { dirs: [0] }],
  [{ dirs: [1] }, { dirs: [1] }],
];

/**
 * Mixed arterials — some squares one axis, some both, one cross.
 */
export const ROAD_GRID_ARTERIAL_2X2: GridTileSpec[][] = [
  [{ dirs: [0, 1] }, { dirs: [0] }],
  [{ dirs: [1] }, { dirs: [0, 1], cross: true }],
];

export type RoadGridScenario = {
  id: string;
  title: string;
  note: string;
  tiles: GridTileSpec[][];
};

export const ROAD_GRID_SCENARIOS: RoadGridScenario[] = [
  {
    id: "mesh-2x2",
    title: "2×2 — 両軸 + CROSS",
    note:
      "各マスに 702 dir0/dir1 と 705 CROSS。向き別スタンプ＋隣マスへのオーバーラップで幹線を繋ぐ。",
    tiles: ROAD_GRID_MESH_2X2,
  },
  {
    id: "axis-2x2",
    title: "2×2 — 軸ごと連続",
    note:
      "北列は dir0 のみ、南列は dir1 のみ。同じ向きの隣マスが辺で合流する見え方。",
    tiles: ROAD_GRID_AXIS_2X2,
  },
  {
    id: "arterial-2x2",
    title: "2×2 — 混在幹線",
    note:
      "一部マスは片軸、SE だけ両軸+CROSS。Townzzy の疎密のある幹線網に近い配置。",
    tiles: ROAD_GRID_ARTERIAL_2X2,
  },
];
