package render

import (
	"image"
	"image/color"
	"image/draw"
)

type pixelMask func(px, py int) bool

func (a *Atlas) drawFrameAtFoot(dst *image.RGBA, key string, footX, footY int) bool {
	rect, ok := a.Frames[key]
	if !ok || rect.W == 0 || rect.H == 0 {
		return false
	}

	// Anchors are required metadata from the packer (including legitimate 0,0
	// for empty frames). Do not reinterpret (0,0) as "missing".
	anchorX, anchorY := rect.AnchorX, rect.AnchorY
	dstX := footX - anchorX
	dstY := footY - anchorY
	srcPt := image.Pt(rect.X, rect.Y)
	dstRect := image.Rect(dstX, dstY, dstX+rect.W, dstY+rect.H)
	draw.Draw(dst, dstRect, a.Image, srcPt, draw.Over)
	return true
}

// drawFrameOnGrassTop paints a sprite but only where the pixel is supported by
// the green diamond tops (column hits grass, and is not below the island).
func (a *Atlas) drawFrameOnGrassTop(dst *image.RGBA, key string, footX, footY int, grass plateGrass) bool {
	return a.drawFrameMasked(dst, key, footX, footY, func(px, py int) bool {
		return grassTopPixelSupported(grass, px, py)
	})
}

// drawRoadOnSquare paints one edge stamp clipped to that square's iso dalle
// (catalog expandedDalleTopWorld) and to plate grass so arterial stamps cannot
// hang past the city diamond onto the flat canvas.
func (a *Atlas) drawRoadOnSquare(dst *image.RGBA, key string, footX, footY, sx, sy, dy int, expand float64, grass plateGrass) bool {
	return a.drawFrameMasked(dst, key, footX, footY, func(px, py int) bool {
		if !grassTopPixelSupported(grass, px, py) {
			return false
		}
		return pointInIsoBlockOffset(px, py, sx*squareSide, sy*squareSide, squareSide, dy, expand)
	})
}

func (a *Atlas) drawFrameMasked(dst *image.RGBA, key string, footX, footY int, mask pixelMask) bool {
	rect, ok := a.Frames[key]
	if !ok || rect.W == 0 || rect.H == 0 {
		return false
	}
	dstX := footX - rect.AnchorX
	dstY := footY - rect.AnchorY
	src, hasRGBA := a.Image.(interface {
		RGBAAt(x, y int) color.RGBA
	})
	bounds := dst.Bounds()
	for sy := 0; sy < rect.H; sy++ {
		py := dstY + sy
		if py < bounds.Min.Y || py >= bounds.Max.Y {
			continue
		}
		for sx := 0; sx < rect.W; sx++ {
			px := dstX + sx
			if px < bounds.Min.X || px >= bounds.Max.X {
				continue
			}
			if !mask(px, py) {
				continue
			}
			var c color.RGBA
			if hasRGBA {
				c = src.RGBAAt(rect.X+sx, rect.Y+sy)
			} else {
				converted, ok := color.RGBAModel.Convert(a.Image.At(rect.X+sx, rect.Y+sy)).(color.RGBA)
				if !ok {
					continue
				}
				c = converted
			}
			if c.A == 0 {
				continue
			}
			if c.A == 255 {
				dst.SetRGBA(px, py, c)
				continue
			}
			draw.Draw(dst, image.Rect(px, py, px+1, py+1), a.Image, image.Pt(rect.X+sx, rect.Y+sy), draw.Over)
		}
	}
	return true
}
