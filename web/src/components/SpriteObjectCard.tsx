import type { PlacementCatalog } from "../catalog/loadCatalog";
import { objectLabel } from "../catalog/loadCatalog";
import type { SpriteObject } from "../catalog/types";
import { SpriteThumb } from "./SpriteThumb";

export function frameForColorVariant(
  obj: SpriteObject,
  atlas: PlacementCatalog["atlas"],
  variantIndex: number,
) {
  const key =
    obj.colorVariants[
      Math.min(Math.max(variantIndex, 0), obj.colorVariants.length - 1)
    ] ?? obj.previewKey;
  return atlas.frames[key] ?? obj.frame;
}

export function SpriteObjectCard({
  obj,
  atlas,
  scale,
  showAnchor,
  variantIndex,
  note,
}: {
  obj: SpriteObject;
  atlas: PlacementCatalog["atlas"];
  scale: number;
  showAnchor: boolean;
  variantIndex: number;
  note?: string;
}) {
  const frame = frameForColorVariant(obj, atlas, variantIndex);
  const label = objectLabel(obj);

  return (
    <article className="object-card">
      <SpriteThumb
        frame={frame}
        label={`${obj.base}/${obj.frameId}`}
        scale={scale}
        showAnchor={showAnchor}
      />
      <div className="object-card__meta">
        <code className="object-card__name" title={`${obj.base} frame ${obj.frameId}`}>
          {label}
        </code>
        {note ? <p className="object-card__note">{note}</p> : null}
        <div className="object-card__badges">
          <span className={`badge badge--tag badge--${obj.tag}`}>{obj.tag}</span>
          {typeof obj.tier === "number" ? (
            <span className="badge badge--muted">tier {obj.tier}</span>
          ) : null}
          {obj.inBuildingPool ? (
            <span className="badge badge--pool">building pool</span>
          ) : null}
          <span className="badge badge--muted">frame {obj.frameId}</span>
          <span className="badge badge--muted">
            {frame.w}×{frame.h}
          </span>
          <span className="badge badge--muted">
            {obj.colorVariants.length} colors
          </span>
        </div>
      </div>
    </article>
  );
}
