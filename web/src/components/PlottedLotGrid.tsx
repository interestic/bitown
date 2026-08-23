import type { CSSProperties } from "react";
import {
  buildPlotLayers,
  boundsForLayers,
  buildGenSquareRoadLayers,
  dalleBlockFootDelta,
  layerClipPath,
  type PlotLayer,
} from "../catalog/plotLayers";
import type { GridTileSpec } from "../catalog/roadDiamondGrid";
import { atlasImageUrl } from "../catalog/loadCatalog";
import type { FrameRect } from "../catalog/types";

type PlottedLotGridProps = {
  dalle: FrameRect;
  /** Orientation-specific BIG ROADS frames: [dir0, dir1] from 702. */
  roadFrames: readonly [FrameRect, FrameRect];
  /** Optional CROSS ROADS (705) stamp used when a tile has `cross: true`. */
  crossFrame?: FrameRect;
  /** tiles[row][col], row 0 north · col 0 west */
  tiles: GridTileSpec[][];
  scale?: number;
  showAnchor?: boolean;
  label: string;
};

type PlacedLayer = PlotLayer & { dalle: FrameRect };

function renderLayer(
  layer: PlacedLayer,
  i: number,
  minX: number,
  minY: number,
  originX: number,
  originY: number,
  scale: number,
  boxW: number,
  boxH: number,
  pad: number,
) {
  const left = (layer.x - minX) * scale + pad;
  const top = (layer.y - minY) * scale + pad;
  const clip = layerClipPath(layer, layer.dalle, originX, originY, scale);

  const viewportStyle: CSSProperties = {
    position: "absolute",
    left: 0,
    top: 0,
    width: boxW,
    height: boxH,
    clipPath: clip,
    zIndex: layer.z,
    pointerEvents: "none",
  };

  const spriteStyle: CSSProperties = {
    position: "absolute",
    left: layer.flipX ? left + layer.frame.w * scale : left,
    top,
    width: layer.frame.w,
    height: layer.frame.h,
    backgroundImage: `url(${atlasImageUrl()})`,
    backgroundRepeat: "no-repeat",
    backgroundPosition: `-${layer.frame.x}px -${layer.frame.y}px`,
    imageRendering: "pixelated",
    transform: layer.flipX ? `scale(${-scale}, ${scale})` : `scale(${scale})`,
    transformOrigin: "top left",
  };

  return (
    <div
      key={`${layer.z}-${i}-${layer.x}-${layer.y}-${layer.flipX ? "flip" : "norm"}`}
      style={viewportStyle}
      aria-hidden
    >
      <div style={spriteStyle} />
    </div>
  );
}

/** Adjacent dalle plates with genSquare-style per-tile BIG ROADS / CROSS. */
export function PlottedLotGrid({
  dalle,
  roadFrames,
  crossFrame,
  tiles,
  scale = 3,
  showAnchor = false,
  label,
}: PlottedLotGridProps) {
  const placed: PlacedLayer[] = [];
  const anchors: { x: number; y: number }[] = [];

  for (let row = 0; row < tiles.length; row++) {
    for (let col = 0; col < tiles[row].length; col++) {
      const { dx, dy } = dalleBlockFootDelta(col, row);
      anchors.push({ x: dx, y: dy });
      const tileLayers = buildPlotLayers({
        dalle,
        footX: dx,
        footY: dy,
      });
      for (const layer of tileLayers) {
        placed.push({ ...layer, dalle });
      }
    }
  }

  for (const layer of buildGenSquareRoadLayers(dalle, roadFrames, tiles, crossFrame)) {
    placed.push({ ...layer, dalle });
  }

  const dalles = placed.filter((l) => l.z === 0);
  const roads = placed.filter((l) => l.z !== 0);
  roads.sort((a, b) => a.z - b.z || a.y + a.frame.h - (b.y + b.frame.h));
  const ordered = [...dalles, ...roads];

  const { minX, minY, boxW, boxH, originX, originY, pad } = boundsForLayers(placed, scale);

  return (
    <div className="plot-thumb plot-thumb--grid" style={{ width: boxW, height: boxH }} title={label}>
      <div className="plot-thumb__stage">
        {ordered.map((layer, i) =>
          renderLayer(layer, i, minX, minY, originX, originY, scale, boxW, boxH, pad),
        )}
        {showAnchor
          ? anchors.map((a, i) => (
              <span
                key={`anchor-${i}`}
                className="sprite-thumb__anchor"
                style={{
                  left: (a.x - minX) * scale + pad,
                  top: (a.y - minY) * scale + pad,
                }}
                title={`foot ${i}`}
              />
            ))
          : null}
      </div>
    </div>
  );
}
