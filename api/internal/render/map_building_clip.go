package render

import (
	"image"
	"image/color"
	"image/draw"
)

// buildingGroundBand is the vertical footprint near the foot used to clear
// sideways spill into street diamonds without slicing mid/upper facades.
const buildingGroundBand = isoTileH + roadGrassLift // foot + grass lift onto streets

type roadMaskData struct {
	mask []bool
}

func buildRoadMaskDataOffset(grid cityGrid, dy int) roadMaskData {
	mask := make([]bool, mapWidth*mapHeight)
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] != cellRoad {
				continue
			}
			markIsoDiamondMaskOffset(mask, mapWidth, x, y, 1, dy)
		}
	}
	return roadMaskData{mask: mask}
}

// peonGrass is the green diamond tops of the peon dalle island. Sprite pixels
// whose column never hits this surface, or that fall below it, are off-island.
type peonGrass struct {
	mask []bool
	col  []bool
	maxY []int
}

func buildPeonGrass(pop int) peonGrass {
	g := peonGrass{
		mask: make([]bool, mapWidth*mapHeight),
		col:  make([]bool, mapWidth),
		maxY: make([]int, mapWidth),
	}
	for i := range g.maxY {
		g.maxY[i] = -1
	}
	o := peonIslandOriginFor(pop)
	e := peonIslandExtentFor(pop)
	for y := o; y < o+e; y++ {
		for x := o; x < o+e; x++ {
			markIsoDiamondMask(g.mask, mapWidth, x, y, 1)
		}
	}
	for py := 0; py < mapHeight; py++ {
		row := py * mapWidth
		for px := 0; px < mapWidth; px++ {
			if g.mask[row+px] {
				g.col[px] = true
				g.maxY[px] = py
			}
		}
	}
	return g
}

func peonPixelSupported(g peonGrass, px, py int) bool {
	if px < 0 || px >= mapWidth || py < 0 || py >= mapHeight {
		return false
	}
	if !g.col[px] {
		return false
	}
	return py <= g.maxY[px]
}

func markIsoDiamondMask(mask []bool, stride, cellX, cellY, edgeOverlap int) {
	markIsoDiamondMaskOffset(mask, stride, cellX, cellY, edgeOverlap, 0)
}

func markIsoDiamondMaskOffset(mask []bool, stride, cellX, cellY, edgeOverlap, dy int) {
	topX, topY := isoCell(cellX, cellY)
	topY += dy
	halfH := isoTileH / 2
	halfW := isoTileW / 2
	if halfH == 0 {
		return
	}
	for row := 0; row < isoTileH; row++ {
		var half int
		if row < halfH {
			half = row * halfW / halfH
		} else {
			half = (isoTileH - 1 - row) * halfW / halfH
		}
		half += edgeOverlap
		py := topY + row
		if py < 0 || py >= mapHeight {
			continue
		}
		for px := topX - half; px <= topX+half; px++ {
			if px < 0 || px >= mapWidth {
				continue
			}
			mask[py*stride+px] = true
		}
	}
}

func roadMaskedAt(mask []bool, px, py int) bool {
	if len(mask) == 0 || px < 0 || py < 0 || px >= mapWidth || py >= mapHeight {
		return false
	}
	return mask[py*mapWidth+px]
}

func inBuildingGroundBand(footY, py int) bool {
	dy := footY - py
	return dy >= 0 && dy <= buildingGroundBand
}

func pointInIsoDiamond(px, py, cellX, cellY int) bool {
	return pointInIsoDiamondOffset(px, py, cellX, cellY, 0)
}

func pointInIsoDiamondOffset(px, py, cellX, cellY, dy int) bool {
	topX, topY := isoCell(cellX, cellY)
	topY += dy
	halfH := isoTileH / 2
	halfW := isoTileW / 2
	row := py - topY
	if row < 0 || row >= isoTileH || halfH == 0 {
		return false
	}
	var half int
	if row < halfH {
		half = row * halfW / halfH
	} else {
		half = (isoTileH - 1 - row) * halfW / halfH
	}
	return px >= topX-half && px <= topX+half
}

// pointInIsoBlockOffset is the iso diamond covering an n×n mini-cell block
// (one Game.hx square / mcDalle plate), optionally expanded from its center.
func pointInIsoBlockOffset(px, py, x0, y0, n, dy int, expand float64) bool {
	if n <= 0 {
		return false
	}
	topX, topY := isoCell(x0, y0)
	topY += dy
	h := n * isoTileH
	w := n * isoTileW
	if expand > 0 {
		cx, cy := topX, topY+h/2
		px = cx + int(float64(px-cx)/(1+expand))
		py = cy + int(float64(py-cy)/(1+expand))
	}
	row := py - topY
	if row < 0 || row >= h {
		return false
	}
	halfH := h / 2
	halfW := w / 2
	if halfH == 0 {
		return false
	}
	var half int
	if row < halfH {
		half = row * halfW / halfH
	} else {
		half = (h - 1 - row) * halfW / halfH
	}
	return px >= topX-half && px <= topX+half
}

// skipBuildingPixelOnRoad clears ground-band spill onto lifted street diamonds.
// The band includes the dalle soil lip so curb stays clear after the 20px grass
// lift. Upper facade may overhang neighboring lots (Flash sprites are wider
// than one tile); forcing a lot-column clip sliced apartment walls in half.
func skipBuildingPixelOnRoad(roads roadMaskData, footY, px, py, lotX, lotY int) bool {
	if !inBuildingGroundBand(footY, py) || !roadMaskedAt(roads.mask, px, py) {
		return false
	}
	return !pointInIsoDiamond(px, py, lotX, lotY)
}

func (a *Atlas) drawBuildingAtFoot(dst *image.RGBA, key string, footX, footY, lotX, lotY int, roads roadMaskData) bool {
	rect, ok := a.Frames[key]
	if !ok || rect.W == 0 || rect.H == 0 {
		return false
	}
	dstX := footX - rect.AnchorX
	dstY := footY - rect.AnchorY

	src, ok := a.Image.(interface {
		RGBAAt(x, y int) color.RGBA
	})
	if !ok {
		return drawBuildingAtFootGeneric(dst, a.Image, rect, dstX, dstY, footY, lotX, lotY, roads)
	}

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
			if skipBuildingPixelOnRoad(roads, footY, px, py, lotX, lotY) {
				continue
			}
			c := src.RGBAAt(rect.X+sx, rect.Y+sy)
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

func drawBuildingAtFootGeneric(dst *image.RGBA, src image.Image, rect frameRect, dstX, dstY, footY, lotX, lotY int, roads roadMaskData) bool {
	tmp := image.NewRGBA(image.Rect(0, 0, rect.W, rect.H))
	draw.Draw(tmp, tmp.Bounds(), src, image.Pt(rect.X, rect.Y), draw.Src)
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
			if skipBuildingPixelOnRoad(roads, footY, px, py, lotX, lotY) {
				continue
			}
			c := tmp.RGBAAt(sx, sy)
			if c.A == 0 {
				continue
			}
			if c.A == 255 {
				dst.SetRGBA(px, py, c)
				continue
			}
			draw.Draw(dst, image.Rect(px, py, px+1, py+1), tmp, image.Pt(sx, sy), draw.Over)
		}
	}
	return true
}

func drawFallbackBuildingClipped(img *image.RGBA, cellSeed uint32, footX, footY, lotX, lotY int, roads roadMaskData) {
	c := buildingColor[cellSeed%buildingColorCount]
	w, h := 10, 14
	for py := footY - h; py < footY; py++ {
		for px := footX - w/2; px < footX+w/2; px++ {
			if !image.Pt(px, py).In(img.Bounds()) {
				continue
			}
			if skipBuildingPixelOnRoad(roads, footY, px, py, lotX, lotY) {
				continue
			}
			img.SetRGBA(px, py, c)
		}
	}
}

func drawFallbackBuildingOnPeonGrass(img *image.RGBA, cellSeed uint32, footX, footY int, grass peonGrass) {
	c := buildingColor[cellSeed%buildingColorCount]
	w, h := 10, 14
	for py := footY - h; py < footY; py++ {
		for px := footX - w/2; px < footX+w/2; px++ {
			if !image.Pt(px, py).In(img.Bounds()) {
				continue
			}
			if !peonPixelSupported(grass, px, py) {
				continue
			}
			img.SetRGBA(px, py, c)
		}
	}
}
