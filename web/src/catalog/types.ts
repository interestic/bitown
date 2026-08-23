export type PlacementTag =
  | "residential"
  | "industrial"
  | "commercial"
  | "landmark"
  | "road"
  | "tree"
  | "water"
  | "ground"
  | "park"
  | "exclude";

export type FrameRect = {
  x: number;
  y: number;
  w: number;
  h: number;
  anchor_x: number;
  anchor_y: number;
};

export type CatalogEntry = {
  base: string;
  group: string;
  tag: PlacementTag | string;
  /** Growth rank 0=hut/low … 3=landmark (building tags only). */
  tier?: number;
  max_opaque_pixels?: number;
  max_bbox_height?: number;
  max_bbox_width?: number;
};

export type BuildingsManifest = {
  version: number;
  rules?: Record<string, string>;
  counts?: {
    by_tag?: Record<string, number>;
    by_tier?: Record<string, number>;
  };
  building_bases: string[];
  bases_by_tag: Record<string, string[]>;
  entries: CatalogEntry[];
};

export type AtlasMeta = {
  image: string;
  count: number;
  frames: Record<string, FrameRect>;
};

export type SpriteObject = {
  base: string;
  /** Frame index inside the Flash clip folder (1, 2, …). */
  frameId: string;
  tag: string;
  group: string;
  /** Growth rank when in the building pool; undefined for non-buildings. */
  tier?: number;
  /** Color-variant atlas keys for this frame (`*_v00` …), sorted. */
  colorVariants: string[];
  /** Preferred preview key (v00 when present). */
  previewKey: string;
  frame: FrameRect;
  inBuildingPool: boolean;
};

/** Tags used for map placement (exclude is catalogued but not placed). */
export const PLACEMENT_TAGS: PlacementTag[] = [
  "residential",
  "industrial",
  "commercial",
  "landmark",
  "road",
  "tree",
  "water",
  "ground",
  "park",
];

export const TAG_LABELS: Record<string, string> = {
  residential: "住宅 (residential)",
  industrial: "工業 (industrial)",
  commercial: "商業 (commercial)",
  landmark: "ランドマーク (landmark)",
  road: "道路 (road)",
  tree: "木 / 公園 (tree)",
  water: "水 (water)",
  ground: "床 / 地面 (ground)",
  park: "公園タグ (park)",
  exclude: "除外 (exclude)",
};
