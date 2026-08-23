import type { CSSProperties } from "react";
import {
  buildPlotLayers,
  boundsForLayers,
  layerClipPath,
  type PlotStamp,
} from "../catalog/plotLayers";
import type { RoadEdge } from "../catalog/roadDiamondLayouts";
import { atlasImageUrl } from "../catalog/loadCatalog";
import type { FrameRect } from "../catalog/types";

export type { PlotStamp } from "../catalog/plotLayers";

type PlottedLotThumbProps = {
  dalle: FrameRect;
  overlay?: FrameRect;
  clipOverlayToDalle?: boolean;
  roadEdges?: RoadEdge[];
  roadFrame?: FrameRect;
  stamps?: PlotStamp[];
  scale?: number;
  showAnchor?: boolean;
  label: string;
};

export function PlottedLotThumb({
  dalle,
  overlay,
  clipOverlayToDalle = false,
  roadEdges,
  roadFrame,
  stamps = [],
  scale = 3,
  showAnchor = false,
  label,
}: PlottedLotThumbProps) {
  const layers = buildPlotLayers({
    dalle,
    overlay,
    clipOverlayToDalle,
    roadEdges,
    roadFrame,
    stamps,
  });

  layers.sort((a, b) => a.y + a.frame.h - (b.y + b.frame.h) || a.z - b.z);

  const { minX, minY, boxW, boxH, originX, originY, pad } = boundsForLayers(layers, scale);
  const footLeft = (0 - minX) * scale + pad;
  const footTop = (0 - minY) * scale + pad;

  return (
    <div className="plot-thumb" style={{ width: boxW, height: boxH }} title={label}>
      <div className="plot-thumb__stage">
        {layers.map((layer, i) => {
          const left = (layer.x - minX) * scale + pad;
          const top = (layer.y - minY) * scale + pad;
          const clip = layerClipPath(layer, dalle, originX, originY, scale);

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
              key={`${layer.z}-${i}-${layer.flipX ? "flip" : "norm"}`}
              style={viewportStyle}
              aria-hidden
            >
              <div style={spriteStyle} />
            </div>
          );
        })}
        {showAnchor ? (
          <span
            className="sprite-thumb__anchor"
            style={{ left: footLeft, top: footTop }}
            title="dalle foot"
          />
        ) : null}
      </div>
    </div>
  );
}
