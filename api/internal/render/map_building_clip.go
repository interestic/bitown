package render

import (
	"image"
	"image/color"
	"image/draw"
)

// buildingGroundBand is the vertical footprint near the foot used to clear
// sideways spill into street diamonds without slicing mid/upper facades.
// Depth matches the plate soil lip (same as the old isoTileH+shortRoadLift).
const buildingGroundBand = plateGrassLift

type roadMaskData struct {
	mask      []bool // cellRoad iso diamonds
	crossMask []bool // CROSS intersection — always clip building ground band
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
	return roadMaskData{
		mask:      mask,
		crossMask: make([]bool, mapWidth*mapHeight),
	}
}

// crossRoadMaskRadius is the chebyshev radius around CROSS foot used only for
// building ground-band clip (not occupancy — yards ring via foot 3).
const crossRoadMaskRadius = 1

// buildRoadMaskDataWithCross extends the arterial grid mask with CROSS foot
// diamonds and arm corridors so building placement matches painted asphalt.
func buildRoadMaskDataWithCross(grid cityGrid, cross [][]uint8, dy int) roadMaskData {
	data := buildRoadMaskDataOffset(grid, dy)
	markCrossMask(&data, cross, dy)
	return data
}

func markCrossMask(data *roadMaskData, cross [][]uint8, dy int) {
	if data == nil || len(cross) == 0 {
		return
	}
	for sy := 0; sy < len(cross); sy++ {
		for sx := 0; sx < len(cross[sy]); sx++ {
			if cross[sy][sx] == 0 {
				continue
			}
			cx, cy := squareCrossFootCell(sx, sy)
			for oy := -crossRoadMaskRadius; oy <= crossRoadMaskRadius; oy++ {
				for ox := -crossRoadMaskRadius; ox <= crossRoadMaskRadius; ox++ {
					markRoadMaskCell(data.crossMask, cx+ox, cy+oy, dy)
				}
			}
		}
	}
}

func squareCrossFootCell(sx, sy int) (x, y int) {
	return sx*squareSide + crossStampFootLocal, sy*squareSide + crossStampFootLocal
}

func markRoadMaskCell(mask []bool, x, y, dy int) {
	if x < 0 || y < 0 || x >= mapCols || y >= mapRows {
		return
	}
	markIsoDiamondMaskOffset(mask, mapWidth, x, y, 1, dy)
}

// lotOverlapsRoadMask reports whether a lot cell sits on painted road asphalt
// (arterial grid or CROSS arms), not just on the logical cellRoad grid.
func lotOverlapsRoadMask(roads roadMaskData, x, y int) bool {
	topX, topY := isoCell(x, y)
	samples := []struct{ px, py int }{
		{topX, topY + isoTileH/2 - roadGrassLift},
		{topX, topY + isoTileH - roadGrassLift},
	}
	for _, s := range samples {
		if roadMaskedAt(roads.mask, s.px, s.py) || roadMaskedAt(roads.crossMask, s.px, s.py) {
			return true
		}
	}
	return false
}

// plateGrass is the grass-top mask of the plate island (roadless or arterial).
// Built at -plateGrassLift so it matches mcDalle grass, not the flat cell floor
// or the soil lip. Sprite pixels off this surface are rejected.
type plateGrass struct {
	mask []bool
	col  []bool
	maxY []int
}

func buildPlateGrass(pop int) plateGrass {
	g := plateGrass{
		mask: make([]bool, mapWidth*mapHeight),
		col:  make([]bool, mapWidth),
		maxY: make([]int, mapWidth),
	}
	for i := range g.maxY {
		g.maxY[i] = -1
	}
	o := plateIslandOrigin(pop)
	e := plateIslandExtent(pop)
	for y := o; y < o+e; y++ {
		for x := o; x < o+e; x++ {
			// Align with overlayFoot / roadGrassLift (grass top, not soil rim).
			// edgeOverlap=2 covers geometric-vs-mcDalle raster fringe.
			markIsoDiamondMaskOffset(g.mask, mapWidth, x, y, 2, -plateGrassLift)
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

func grassTopPixelSupported(g plateGrass, px, py int) bool {
	if len(g.mask) == 0 {
		return true
	}
	if px < 0 || px >= mapWidth || py < 0 || py >= mapHeight {
		return false
	}
	// Require the pixel itself to sit on a grass-top diamond. The old
	// py<=maxY[col] test allowed north hang onto the flat canvas.
	return g.mask[py*mapWidth+px]
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
// The band includes the plate soil lip so curb stays clear after the 20px grass
// lift. Upper facade may overhang neighboring lots (Flash sprites are wider
// than one tile); forcing a lot-column clip sliced apartment walls in half.
func skipBuildingPixelOnRoad(roads roadMaskData, footY, px, py, lotX, lotY int) bool {
	if !inBuildingGroundBand(footY, py) {
		return false
	}
	// Iso lot diamonds overlap at CROSS — clip intersection asphalt unconditionally.
	if roadMaskedAt(roads.crossMask, px, py) {
		return true
	}
	if !roadMaskedAt(roads.mask, px, py) {
		return false
	}
	return !pointInIsoDiamond(px, py, lotX, lotY)
}

func (a *Atlas) drawBuildingAtFoot(dst *image.RGBA, key string, footX, footY, lotX, lotY int, roads roadMaskData, grass plateGrass) bool {
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
		return drawBuildingAtFootGeneric(dst, a.Image, rect, dstX, dstY, footY, lotX, lotY, roads, grass)
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
			if !grassTopPixelSupported(grass, px, py) {
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

func drawBuildingAtFootGeneric(dst *image.RGBA, src image.Image, rect frameRect, dstX, dstY, footY, lotX, lotY int, roads roadMaskData, grass plateGrass) bool {
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
			if !grassTopPixelSupported(grass, px, py) {
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

func drawFallbackBuildingOnGrassTop(img *image.RGBA, cellSeed uint32, footX, footY int, grass plateGrass) {
	c := buildingColor[cellSeed%buildingColorCount]
	w, h := 10, 14
	for py := footY - h; py < footY; py++ {
		for px := footX - w/2; px < footX+w/2; px++ {
			if !image.Pt(px, py).In(img.Bounds()) {
				continue
			}
			if !grassTopPixelSupported(grass, px, py) {
				continue
			}
			img.SetRGBA(px, py, c)
		}
	}
}
