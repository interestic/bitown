import { useEffect, useMemo, useState } from "react";
import {
  loadPlacementCatalog,
  objectKey,
  objectLabel,
  type PlacementCatalog,
} from "../catalog/loadCatalog";
import {
  DALLE_BASE,
  DALLE_FRAME_ID,
  LOT_PATTERNS,
  TREE_PLOT_CELLS,
  lotPatternById,
  type LotPattern,
  type LotPatternId,
} from "../catalog/lotPatterns";
import {
  BIG_ROAD_SPRITE_BASE,
  CROSS_SPRITE_BASE,
  ROAD_STYLE_LABELS,
  bigRoadFrameIdForDir,
  crossFrameIdFromVariant,
  roadStyleFrameFromVariant,
} from "../catalog/roadDiamondLayouts";
import { ROAD_GRID_SCENARIOS, type RoadGridScenario } from "../catalog/roadDiamondGrid";
import type { FrameRect, SpriteObject } from "../catalog/types";
import { frameForColorVariant } from "./SpriteObjectCard";
import { PlottedLotThumb } from "./PlottedLotThumb";
import { PlottedLotGrid } from "./PlottedLotGrid";

export type LotPatternViewProps = {
  patternId?: LotPatternId;
  scale?: number;
  showAnchor?: boolean;
  variantIndex?: number;
  includeTreeObjects?: boolean;
};

function lookupClip(
  catalog: PlacementCatalog,
  base: string,
  frameId: string,
): SpriteObject | undefined {
  return catalog.objects.find(
    (obj) => obj.base === base && obj.frameId === frameId,
  );
}

function PatternClipCard({
  obj,
  dalle,
  catalog,
  scale,
  showAnchor,
  variantIndex,
  note,
  scatter,
  clipOverlayToDalle,
}: {
  obj: SpriteObject;
  dalle: SpriteObject;
  catalog: PlacementCatalog;
  scale: number;
  showAnchor: boolean;
  variantIndex: number;
  note?: string;
  scatter?: boolean;
  clipOverlayToDalle?: boolean;
}) {
  const dalleFrame = frameForColorVariant(dalle, catalog.atlas, 0);
  const overlayFrame = frameForColorVariant(obj, catalog.atlas, variantIndex);
  const isDalle = obj.base === DALLE_BASE;
  const label = objectLabel(obj);

  return (
    <article className="object-card">
      <PlottedLotThumb
        dalle={dalleFrame}
        overlay={isDalle || scatter ? undefined : overlayFrame}
        clipOverlayToDalle={clipOverlayToDalle}
        stamps={
          scatter
            ? TREE_PLOT_CELLS.map((cell) => ({ frame: overlayFrame, cell }))
            : undefined
        }
        scale={scale}
        showAnchor={showAnchor}
        label={label}
      />
      <div className="object-card__meta">
        <code className="object-card__name" title={`${obj.base} frame ${obj.frameId}`}>
          {label}
        </code>
        {note ? <p className="object-card__note">{note}</p> : null}
        <div className="object-card__badges">
          <span className="badge badge--pattern">on dalle</span>
          <span className={`badge badge--tag badge--${obj.tag}`}>{obj.tag}</span>
          <span className="badge badge--muted">frame {obj.frameId}</span>
          <span className="badge badge--muted">
            {overlayFrame.w}×{overlayFrame.h}
          </span>
        </div>
      </div>
    </article>
  );
}

function RoadGridScenarioCard({
  scenario,
  roadFrames,
  cross,
  dalle,
  catalog,
  scale,
  showAnchor,
  variantIndex,
}: {
  scenario: RoadGridScenario;
  roadFrames: readonly [FrameRect, FrameRect];
  cross?: SpriteObject;
  dalle: SpriteObject;
  catalog: PlacementCatalog;
  scale: number;
  showAnchor: boolean;
  variantIndex: number;
}) {
  const dalleFrame = frameForColorVariant(dalle, catalog.atlas, 0);
  const crossFrame = cross
    ? frameForColorVariant(cross, catalog.atlas, 0)
    : undefined;
  const styleId = roadStyleFrameFromVariant(variantIndex);
  const dir0Id = bigRoadFrameIdForDir(0, variantIndex);
  const dir1Id = bigRoadFrameIdForDir(1, variantIndex);
  const hasCross = scenario.tiles.some((row) => row.some((t) => t.cross));

  return (
    <article className="object-card object-card--grid">
      <PlottedLotGrid
        dalle={dalleFrame}
        roadFrames={roadFrames}
        crossFrame={crossFrame}
        tiles={scenario.tiles}
        scale={scale}
        showAnchor={showAnchor}
        label={scenario.title}
      />
      <div className="object-card__meta">
        <code className="object-card__name" title={scenario.id}>
          {scenario.title}
        </code>
        <p className="object-card__note">{scenario.note}</p>
        <div className="object-card__badges">
          <span className="badge badge--pattern">
            702/{dir0Id}+{dir1Id}
          </span>
          {hasCross ? <span className="badge badge--muted">705</span> : null}
          <span className="badge badge--muted">{ROAD_STYLE_LABELS[styleId]}</span>
        </div>
      </div>
    </article>
  );
}

function RoadDiamondSection({
  pattern,
  catalog,
  dalle,
  scale,
  showAnchor,
  variantIndex,
}: {
  pattern: LotPattern;
  catalog: PlacementCatalog;
  dalle: SpriteObject;
  scale: number;
  showAnchor: boolean;
  variantIndex: number;
}) {
  const dir0 = lookupClip(
    catalog,
    BIG_ROAD_SPRITE_BASE,
    bigRoadFrameIdForDir(0, variantIndex),
  );
  const dir1 = lookupClip(
    catalog,
    BIG_ROAD_SPRITE_BASE,
    bigRoadFrameIdForDir(1, variantIndex),
  );
  const roadFrames =
    dir0 && dir1
      ? ([
          frameForColorVariant(dir0, catalog.atlas, 0),
          frameForColorVariant(dir1, catalog.atlas, 0),
        ] as const)
      : null;
  const crossId = crossFrameIdFromVariant(variantIndex);
  const cross = lookupClip(catalog, CROSS_SPRITE_BASE, crossId);
  const crosses = pattern.clips.map((clip) => ({
    clip,
    obj: lookupClip(catalog, clip.base, clip.frameId),
  }));
  const dir0Id = bigRoadFrameIdForDir(0, variantIndex);
  const dir1Id = bigRoadFrameIdForDir(1, variantIndex);

  return (
    <section className="tag-section">
      <header className="tag-section__header">
        <h2>{pattern.title}</h2>
        <span className="tag-section__count">{pattern.titleEn}</span>
      </header>
      <p className="pattern-blurb">{pattern.description}</p>
      <p className="pattern-original">
        <span className="badge badge--pattern">Game.hx</span>
        {pattern.original}
      </p>

      {!roadFrames ? (
        <p className="hint">
          未検出: {BIG_ROAD_SPRITE_BASE}/{dir0Id} または /{dir1Id}
        </p>
      ) : (
        <div className="object-grid object-grid--plots object-grid--road-grid">
          {ROAD_GRID_SCENARIOS.map((scenario) => (
            <RoadGridScenarioCard
              key={scenario.id}
              scenario={scenario}
              roadFrames={roadFrames}
              cross={cross}
              dalle={dalle}
              catalog={catalog}
              scale={scale}
              showAnchor={showAnchor}
              variantIndex={variantIndex}
            />
          ))}
        </div>
      )}

      <h3 className="pattern-extras">X 交差クリップ単体（DefineSprite_705 · CROSS ROADS）</h3>
      <p className="pattern-blurb">
        genSquare CROSS ROADS（size=4）。上のグリッドではマスの cross フラグで合成済み。ここは単体見本。
      </p>
      <div className="object-grid object-grid--plots">
        {crosses.map(({ clip, obj }) =>
          obj ? (
            <PatternClipCard
              key={objectKey(obj)}
              obj={obj}
              dalle={dalle}
              catalog={catalog}
              scale={scale}
              showAnchor={showAnchor}
              variantIndex={0}
              note={clip.note}
              clipOverlayToDalle
            />
          ) : (
            <article
              key={`${clip.base}#${clip.frameId}`}
              className="object-card object-card--missing"
            >
              <p className="hint">
                未検出: {clip.base.split("/").pop()}/{clip.frameId}
              </p>
            </article>
          ),
        )}
      </div>
    </section>
  );
}

function PatternSection({
  pattern,
  catalog,
  dalle,
  scale,
  showAnchor,
  variantIndex,
  includeTreeObjects,
}: {
  pattern: LotPattern;
  catalog: PlacementCatalog;
  dalle: SpriteObject;
  scale: number;
  showAnchor: boolean;
  variantIndex: number;
  includeTreeObjects: boolean;
}) {
  if (pattern.id === "road-diamond") {
    return (
      <RoadDiamondSection
        pattern={pattern}
        catalog={catalog}
        dalle={dalle}
        scale={scale}
        showAnchor={showAnchor}
        variantIndex={variantIndex}
      />
    );
  }

  const resolved = pattern.clips.map((clip) => ({
    clip,
    obj: lookupClip(catalog, clip.base, clip.frameId),
  }));
  const extras =
    includeTreeObjects && pattern.extraTag
      ? (catalog.byTag[pattern.extraTag] ?? [])
      : [];

  return (
    <section className="tag-section">
      <header className="tag-section__header">
        <h2>{pattern.title}</h2>
        <span className="tag-section__count">{pattern.titleEn}</span>
      </header>
      <p className="pattern-blurb">{pattern.description}</p>
      <p className="pattern-original">
        <span className="badge badge--pattern">Game.hx</span>
        {pattern.original}
      </p>
      <div className="object-grid object-grid--plots">
        {resolved.map(({ clip, obj }) =>
          obj ? (
            <PatternClipCard
              key={objectKey(obj)}
              obj={obj}
              dalle={dalle}
              catalog={catalog}
              scale={scale}
              showAnchor={showAnchor}
              variantIndex={variantIndex}
              note={clip.note}
            />
          ) : (
            <article
              key={`${clip.base}#${clip.frameId}`}
              className="object-card object-card--missing"
            >
              <p className="hint">
                未検出: {clip.base.split("/").pop()}/{clip.frameId}
              </p>
            </article>
          ),
        )}
      </div>
      {extras.length > 0 ? (
        <>
          <h3 className="pattern-extras">type 14 の木を台座にプロット</h3>
          <div className="object-grid object-grid--plots">
            {extras.map((obj) => (
              <PatternClipCard
                key={objectKey(obj)}
                obj={obj}
                dalle={dalle}
                catalog={catalog}
                scale={scale}
                showAnchor={showAnchor}
                variantIndex={variantIndex}
                scatter
              />
            ))}
          </div>
        </>
      ) : null}
    </section>
  );
}

export function LotPatternView({
  patternId,
  scale = 3,
  showAnchor = false,
  variantIndex = 0,
  includeTreeObjects = true,
}: LotPatternViewProps) {
  const [catalog, setCatalog] = useState<PlacementCatalog | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    loadPlacementCatalog()
      .then((data) => {
        if (!cancelled) setCatalog(data);
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : String(err));
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const patterns = useMemo(() => {
    const one = lotPatternById(patternId);
    return one ? [one] : LOT_PATTERNS;
  }, [patternId]);

  if (error) {
    return (
      <div className="catalog-shell catalog-shell--error">
        <h1>ロットパターン</h1>
        <p>カタログの読み込みに失敗しました: {error}</p>
      </div>
    );
  }

  if (!catalog) {
    return (
      <div className="catalog-shell">
        <p className="hint">カタログを読み込み中…</p>
      </div>
    );
  }

  const dalle = lookupClip(catalog, DALLE_BASE, DALLE_FRAME_ID);
  if (!dalle) {
    return (
      <div className="catalog-shell catalog-shell--error">
        <h1>ロットパターン</h1>
        <p>台座クリップ {DALLE_BASE} が見つかりません。</p>
      </div>
    );
  }

  const heading = patterns.length === 1 ? patterns[0].title : "ロットパターン早見";
  const styleId = roadStyleFrameFromVariant(variantIndex);

  return (
    <div className="catalog-shell">
      <header className="catalog-header">
        <div>
          <h1>{heading}</h1>
          <p className="catalog-header__sub">
            各パターンを mcDalle 台座の天面にプロットした合成。道路は 702_mcRoad
            （dir0/dir1・{ROAD_STYLE_LABELS[styleId]}）を variantIndex で切替。
          </p>
        </div>
        <dl className="catalog-stats">
          {LOT_PATTERNS.map((p) => (
            <div key={p.id}>
              <dt>{p.id}</dt>
              <dd>
                {p.id === "road-diamond"
                  ? ROAD_GRID_SCENARIOS.length + p.clips.length
                  : p.clips.length}
              </dd>
            </div>
          ))}
        </dl>
      </header>

      {patterns.map((pattern) => (
        <PatternSection
          key={pattern.id}
          pattern={pattern}
          catalog={catalog}
          dalle={dalle}
          scale={scale}
          showAnchor={showAnchor}
          variantIndex={variantIndex}
          includeTreeObjects={includeTreeObjects}
        />
      ))}
    </div>
  );
}
