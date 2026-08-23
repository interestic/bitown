import { useEffect, useMemo, useState } from "react";
import {
  loadPlacementCatalog,
  objectKey,
  type PlacementCatalog,
} from "../catalog/loadCatalog";
import { PLACEMENT_TAGS, TAG_LABELS } from "../catalog/types";
import type { SpriteObject } from "../catalog/types";
import { SpriteObjectCard } from "./SpriteObjectCard";

export type PlacementCatalogViewProps = {
  /** Restrict to one tag; omit for overview of placement tags. */
  tag?: string;
  /** Include exclude-tagged sprites (noisy). */
  includeExclude?: boolean;
  scale?: number;
  showAnchor?: boolean;
  /** Color variant index 0..3 (v00–v03) within each frame. */
  variantIndex?: number;
};

function TagSection({
  tag,
  items,
  atlas,
  scale,
  showAnchor,
  variantIndex,
  baseCount,
}: {
  tag: string;
  items: SpriteObject[];
  atlas: PlacementCatalog["atlas"];
  scale: number;
  showAnchor: boolean;
  variantIndex: number;
  baseCount: number;
}) {
  return (
    <section className="tag-section">
      <header className="tag-section__header">
        <h2>{TAG_LABELS[tag] ?? tag}</h2>
        <span className="tag-section__count">
          {items.length} frames · {baseCount} bases
        </span>
      </header>
      {items.length === 0 ? (
        <p className="tag-section__empty">（このタグのオブジェクトはまだありません）</p>
      ) : (
        <div className="object-grid">
          {items.map((obj) => (
            <SpriteObjectCard
              key={objectKey(obj)}
              obj={obj}
              atlas={atlas}
              scale={scale}
              showAnchor={showAnchor}
              variantIndex={variantIndex}
            />
          ))}
        </div>
      )}
    </section>
  );
}

export function PlacementCatalogView({
  tag,
  includeExclude = false,
  scale = 2,
  showAnchor = true,
  variantIndex = 0,
}: PlacementCatalogViewProps) {
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

  const sections = useMemo(() => {
    if (!catalog) return [];
    if (tag) {
      return [{ tag, items: catalog.byTag[tag] ?? [] }];
    }
    const tags = includeExclude
      ? [...PLACEMENT_TAGS, "exclude"]
      : [...PLACEMENT_TAGS];
    return tags.map((t) => ({ tag: t, items: catalog.byTag[t] ?? [] }));
  }, [catalog, tag, includeExclude]);

  if (error) {
    return (
      <div className="catalog-shell catalog-shell--error">
        <h1>配置オブジェクト早見</h1>
        <p>カタログの読み込みに失敗しました: {error}</p>
        <p className="hint">
          Storybook の staticDirs が <code>assets/sprites-v1</code> を配信しているか確認してください。
        </p>
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

  const title = tag
    ? TAG_LABELS[tag] ?? tag
    : "配置オブジェクト早見";

  return (
    <div className="catalog-shell">
      <header className="catalog-header">
        <div>
          <h1>{title}</h1>
          <p className="catalog-header__sub">
            sprites-v1 · buildings.json v{catalog.manifest.version} · atlas{" "}
            {catalog.atlas.count} frames · catalog {catalog.objects.length}{" "}
            clip-frames
          </p>
        </div>
        <dl className="catalog-stats">
          {PLACEMENT_TAGS.map((t) => (
            <div key={t}>
              <dt>{t}</dt>
              <dd>
                {catalog.byTag[t]?.length ?? 0}
                <span className="catalog-stats__bases">
                  /{catalog.manifest.counts?.by_tag?.[t] ?? 0}
                </span>
              </dd>
            </div>
          ))}
          {["0", "1", "2", "3"].map((tier) => (
            <div key={`tier-${tier}`}>
              <dt>tier {tier}</dt>
              <dd>{catalog.manifest.counts?.by_tier?.[tier] ?? 0}</dd>
            </div>
          ))}
        </dl>
      </header>

      {sections.map(({ tag: sectionTag, items }) => (
        <TagSection
          key={sectionTag}
          tag={sectionTag}
          items={items}
          atlas={catalog.atlas}
          scale={scale}
          showAnchor={showAnchor}
          variantIndex={variantIndex}
          baseCount={catalog.manifest.counts?.by_tag?.[sectionTag] ?? 0}
        />
      ))}
    </div>
  );
}
