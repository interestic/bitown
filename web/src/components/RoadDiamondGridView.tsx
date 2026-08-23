import { useEffect, useState } from "react";
import {
  DALLE_BASE,
  DALLE_FRAME_ID,
} from "../catalog/lotPatterns";
import {
  loadPlacementCatalog,
  type PlacementCatalog,
} from "../catalog/loadCatalog";
import {
  BIG_ROAD_SPRITE_BASE,
  CROSS_SPRITE_BASE,
  ROAD_STYLE_LABELS,
  bigRoadFrameIdForDir,
  crossFrameIdFromVariant,
  roadStyleFrameFromVariant,
} from "../catalog/roadDiamondLayouts";
import { ROAD_GRID_SCENARIOS } from "../catalog/roadDiamondGrid";
import type { FrameRect } from "../catalog/types";
import type { SpriteObject } from "../catalog/types";
import { frameForColorVariant } from "./SpriteObjectCard";
import { PlottedLotGrid } from "./PlottedLotGrid";

export type RoadDiamondGridViewProps = {
  scale?: number;
  showAnchor?: boolean;
  variantIndex?: number;
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

function resolveDirFrames(
  catalog: PlacementCatalog,
  variantIndex: number,
): readonly [FrameRect, FrameRect] | null {
  const a = lookupClip(catalog, BIG_ROAD_SPRITE_BASE, bigRoadFrameIdForDir(0, variantIndex));
  const b = lookupClip(catalog, BIG_ROAD_SPRITE_BASE, bigRoadFrameIdForDir(1, variantIndex));
  if (!a || !b) return null;
  return [
    frameForColorVariant(a, catalog.atlas, 0),
    frameForColorVariant(b, catalog.atlas, 0),
  ] as const;
}

export function RoadDiamondGridView({
  scale = 3,
  showAnchor = false,
  variantIndex = 2,
}: RoadDiamondGridViewProps) {
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

  if (error) {
    return (
      <div className="catalog-shell catalog-shell--error">
        <h1>道路 — genSquare BIG ROADS</h1>
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
  const styleId = roadStyleFrameFromVariant(variantIndex);
  const roadFrames = resolveDirFrames(catalog, variantIndex);
  const crossId = crossFrameIdFromVariant(variantIndex);
  const cross = lookupClip(catalog, CROSS_SPRITE_BASE, crossId);
  const dir0Id = bigRoadFrameIdForDir(0, variantIndex);
  const dir1Id = bigRoadFrameIdForDir(1, variantIndex);

  if (!dalle) {
    return (
      <div className="catalog-shell catalog-shell--error">
        <h1>道路 — genSquare BIG ROADS</h1>
        <p>台座クリップ {DALLE_BASE} が見つかりません。</p>
      </div>
    );
  }

  const dalleFrame = frameForColorVariant(dalle, catalog.atlas, 0);

  return (
    <div className="catalog-shell">
      <header className="catalog-header">
        <div>
          <h1>道路 — genSquare BIG ROADS</h1>
          <p className="catalog-header__sub">
            702 dir0={dir0Id} / dir1={dir1Id}（{ROAD_STYLE_LABELS[styleId]}）を向き別スタンプ。
            クリップを隣マスへ少し広げて合流。CROSS は 705/{crossId}。
          </p>
        </div>
      </header>

      {!roadFrames ? (
        <p className="hint">
          未検出: {BIG_ROAD_SPRITE_BASE}/{dir0Id} または /{dir1Id}
        </p>
      ) : (
        <div className="object-grid object-grid--plots object-grid--road-grid">
          {ROAD_GRID_SCENARIOS.map((s) => {
            const crossFrame = cross
              ? frameForColorVariant(cross, catalog.atlas, 0)
              : undefined;
            return (
              <article key={s.id} className="object-card object-card--grid">
                <PlottedLotGrid
                  dalle={dalleFrame}
                  roadFrames={roadFrames}
                  crossFrame={crossFrame}
                  tiles={s.tiles}
                  scale={scale}
                  showAnchor={showAnchor}
                  label={s.title}
                />
                <div className="object-card__meta">
                  <code className="object-card__name" title={s.id}>
                    {s.title}
                  </code>
                  <p className="object-card__note">{s.note}</p>
                  <div className="object-card__badges">
                    <span className="badge badge--pattern">
                      702/{dir0Id}+{dir1Id}
                    </span>
                    {s.tiles.some((row) => row.some((t) => t.cross)) ? (
                      <span className="badge badge--muted">705/{crossId}</span>
                    ) : null}
                    <span className="badge badge--muted">{ROAD_STYLE_LABELS[styleId]}</span>
                  </div>
                </div>
              </article>
            );
          })}
        </div>
      )}
    </div>
  );
}
