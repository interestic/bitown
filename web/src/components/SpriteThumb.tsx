import type { CSSProperties } from "react";
import type { FrameRect } from "../catalog/types";
import { atlasImageUrl } from "../catalog/loadCatalog";

type SpriteThumbProps = {
  frame: FrameRect;
  label: string;
  /** Visual scale; source sprites are small. */
  scale?: number;
  showAnchor?: boolean;
};

export function SpriteThumb({
  frame,
  label,
  scale = 2,
  showAnchor = true,
}: SpriteThumbProps) {
  const pad = 10;
  const boxW = Math.max(frame.w * scale + pad * 2, 56);
  const boxH = Math.max(frame.h * scale + pad * 2, 56);

  const cropStyle: CSSProperties = {
    width: frame.w,
    height: frame.h,
    backgroundImage: `url(${atlasImageUrl()})`,
    backgroundRepeat: "no-repeat",
    backgroundPosition: `-${frame.x}px -${frame.y}px`,
    imageRendering: "pixelated",
    transform: `scale(${scale})`,
    transformOrigin: "top left",
  };

  return (
    <div
      className="sprite-thumb"
      style={{ width: boxW, height: boxH }}
      title={label}
    >
      <div
        className="sprite-thumb__stage"
        style={{ width: frame.w * scale, height: frame.h * scale }}
      >
        <div style={cropStyle} aria-hidden />
        {showAnchor ? (
          <span
            className="sprite-thumb__anchor"
            style={{
              left: frame.anchor_x * scale,
              top: frame.anchor_y * scale,
            }}
            title={`anchor (${frame.anchor_x}, ${frame.anchor_y})`}
          />
        ) : null}
      </div>
    </div>
  );
}
