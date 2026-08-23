/**
 * Townzzy / Miniville lot patterns visible on empty squares.
 *
 * Original: Game.hx genSquare / genMiniSquare stamp mcHouse nested clips
 * (`brushHouse.gotoAndStop(size+1)` → smc type → smc.smc variant).
 * Townzzy names the same family `auth_champs_gfx` (mini) and
 * `auth_big_champs_gfx` (full square).
 *
 * bitown extracted the nested children as separate DefineSprites; mcDalle
 * frames 1–50 are identical (parent plate only).
 */

export type LotPatternId =
  | "trees-plot"
  | "grass-fill"
  | "yellow-fill"
  | "yellow-furrow-ns"
  | "yellow-furrow-ew"
  | "soil"
  | "pumpkin-5"
  | "quad-split"
  | "road-diamond"
  | "dalle";

export type LotPatternClip = {
  base: string;
  frameId: string;
  note?: string;
};

export type LotPattern = {
  id: LotPatternId;
  title: string;
  titleEn: string;
  description: string;
  original: string;
  clips: LotPatternClip[];
  /** Extra catalog tag to list under this pattern (tree objects for type 14). */
  extraTag?: "tree";
};

export const LOT_PATTERNS: LotPattern[] = [
  {
    id: "trees-plot",
    title: "木だけプロット",
    titleEn: "trees scatter",
    description:
      "ひし形の上に木や点在デコだけが載る空き地。外周の森は type 14 の木スプライトをばら撒き、ミニスクエア側は点在クリップ。",
    original:
      "genSquare 空きスクエア（sidePop が薄い）: addBat(tx,ty, size=0, type=14)。Townzzy auth_parc_gfx 相当。",
    clips: [
      { base: "sprites/DefineSprite_514", frameId: "1", note: "点在デコ" },
      { base: "sprites/DefineSprite_516", frameId: "1" },
      { base: "sprites/DefineSprite_518", frameId: "1" },
      { base: "sprites/DefineSprite_520", frameId: "1" },
      { base: "sprites/DefineSprite_521", frameId: "6", note: "521 点在" },
      { base: "sprites/DefineSprite_521", frameId: "7" },
      { base: "sprites/DefineSprite_521", frameId: "8" },
    ],
    extraTag: "tree",
  },
  {
    id: "grass-fill",
    title: "緑だけ敷き詰め",
    titleEn: "grass fill",
    description: "ミニスクエア全面が芝生。畑バリアントの無地緑。",
    original:
      "genMiniSquare pop==0: addBat(bx,by, size=1, type=2) の緑フレーム。Townzzy auth_champs_gfx の緑。",
    clips: [
      { base: "sprites/DefineSprite_401", frameId: "5" },
      { base: "sprites/DefineSprite_401", frameId: "6", note: "チェック寄りの緑" },
      { base: "sprites/DefineSprite_521", frameId: "2" },
    ],
  },
  {
    id: "yellow-fill",
    title: "黄色ベタ",
    titleEn: "yellow fill",
    description:
      "畑の無地黄色。507 は 2×2 セル相当の小サイズ、510 はやや横長。",
    original:
      "size=1 type=2 / 空き隣接スクエアの size=0 type=15。Townzzy auth_champs_gfx / auth_big_champs_gfx。",
    clips: [
      { base: "sprites/DefineSprite_399", frameId: "1" },
      { base: "sprites/DefineSprite_401", frameId: "4" },
      { base: "sprites/DefineSprite_521", frameId: "1" },
      { base: "sprites/DefineSprite_507", frameId: "1", note: "小（2×2 相当）" },
      { base: "sprites/DefineSprite_510", frameId: "1", note: "中サイズ" },
    ],
  },
  {
    id: "yellow-furrow-ns",
    title: "黄色ボーダー（縦）",
    titleEn: "yellow furrow NS",
    description:
      "黄色畑に、アイソメの一方の軸に沿った畝・ボーダーだけが入る。",
    original: "size=1 type=2 の畝バリアント（iso 軸 A）。",
    clips: [{ base: "sprites/DefineSprite_388", frameId: "1" }],
  },
  {
    id: "yellow-furrow-ew",
    title: "黄色ボーダー（横）",
    titleEn: "yellow furrow EW",
    description:
      "黄色畑に、もう一方のアイソメ軸に沿った畝・ボーダーだけが入る。",
    original: "size=1 type=2 の畝バリアント（iso 軸 B）。",
    clips: [{ base: "sprites/DefineSprite_503", frameId: "1" }],
  },
  {
    id: "soil",
    title: "茶色（土）",
    titleEn: "soil / brown",
    description:
      "耕した土のミニスクエア。格子・横畝。401/1 は角に小屋が載る。",
    original: "size=1 type=2 の土バリアント。",
    clips: [
      { base: "sprites/DefineSprite_401", frameId: "2", note: "格子" },
      { base: "sprites/DefineSprite_401", frameId: "1", note: "格子 + 小屋" },
      { base: "sprites/DefineSprite_521", frameId: "3", note: "横畝" },
    ],
  },
  {
    id: "pumpkin-5",
    title: "かぼちゃ 5つ",
    titleEn: "five pumpkins",
    description:
      "黄色畑にかぼちゃ状の塊が 5 つ（サイコロの 5＝クインクンクス）。",
    original: "size=1 type=2 の作物バリアント。",
    clips: [
      { base: "sprites/DefineSprite_396", frameId: "1" },
      { base: "sprites/DefineSprite_401", frameId: "3" },
    ],
  },
  {
    id: "quad-split",
    title: "4 分割",
    titleEn: "quad split",
    description:
      "1 枚のミニスクエアが 4 象限に分かれる焼き込み。加えて genMiniSquare は 10×10 スクエアを 4 つの 4×4 に分割する（中央 1 セルが道路ギャップ）。",
    original:
      "焼き込み: size=1 type=2 の 4 象限フレーム。構成: genSquare が pop<POP_HUGE のとき 4 回 genMiniSquare。",
    clips: [
      { base: "sprites/DefineSprite_521", frameId: "4", note: "畑ミックス" },
      { base: "sprites/DefineSprite_521", frameId: "5", note: "緑 + 小屋 + 黄" },
    ],
  },
  {
    id: "road-diamond",
    title: "道路 — genSquare BIG ROADS",
    titleEn: "square arterials + cross",
    description:
      "Game.hx genSquare どおり各マスに 702_mcRoad の向き別フレーム（dir0=1..3 / dir1=4..6）と任意で 705 CROSS。隣マスへ少しオーバーラップして幹線を繋ぐ。",
    original:
      "genSquare BIG ROADS size=3（fr=3*n+style → mcRoad 相当フレーム）/ CROSS ROADS size=4（705）。",
    clips: [
      { base: "sprites/DefineSprite_705", frameId: "1", note: "X 交差・土" },
      { base: "sprites/DefineSprite_705", frameId: "2", note: "X 交差・アスファルト" },
    ],
  },
  {
    id: "dalle",
    title: "台地プレート（mcDalle）",
    titleEn: "raised dalle",
    description:
      "毎スクエアの下地になる盛り土芝生。抽出では 50 フレームが同一（ネスト装飾は乗っていない）。マップはいま 1/2/10/20 からランダムに 1 枚スタンプ。",
    original:
      "Game.hx は addBat(x,y, size=5, type=0) が下地。mcDalle 直スタンプはコメントアウト済み。bitown は DefineSprite_707_mcDalle を 4×4 で敷く。",
    clips: [{ base: "sprites/DefineSprite_707_mcDalle", frameId: "1" }],
  },
];

/** Raised 4×4 plate used as the lot pedestal in Storybook composites. */
export const DALLE_BASE = "sprites/DefineSprite_707_mcDalle";
export const DALLE_FRAME_ID = "1";

export const ISO_TILE_W = 24;
export const ISO_TILE_H = 12;
export const DALLE_CELLS = 4;
/** Top-face diamond height of a 4×4 dalle (N × HH). Soil sides hang below this. */
export const DALLE_TOP_H = DALLE_CELLS * ISO_TILE_H;

/** Iso cells on the 4×4 plate (0 = NW … 3 = SE) used to scatter tree stamps. */
export const TREE_PLOT_CELLS: ReadonlyArray<readonly [number, number]> = [
  [1, 1],
  [3, 0],
  [0, 2],
  [2, 3],
];

export const LOT_PATTERN_IDS = LOT_PATTERNS.map((p) => p.id);

export function lotPatternById(id: LotPatternId | undefined): LotPattern | undefined {
  if (!id) return undefined;
  return LOT_PATTERNS.find((p) => p.id === id);
}
