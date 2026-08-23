package render

import (
	"image"
	"image/color"
	"image/draw"
)

// buildingGroundBand is the vertical footprint near the foot used to clear
// sideways spill into street diamonds without slicing mid/upper facades.
const buildingGroundBand = isoTileH // clear only near the foot onto streets

type roadMaskData struct {
	mask []bool
}

func buildRoadMaskData(grid cityGrid) roadMaskData {
	mask := make([]bool, mapWidth*mapHeight)
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] != cellRoad {
				continue
			}
			markIsoDiamondMask(mask, mapWidth, x, y, 1)
		}
	}
	return roadMaskData{mask: mask}
}

// buildPlatformMask marks every map cell diamond so road sprites can be clipped
// to the iso island (prevents mcRoad stubs hanging past the dalle rim).
func buildPlatformMask() roadMaskData {
	mask := make([]bool, mapWidth*mapHeight)
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			markIsoDiamondMask(mask, mapWidth, x, y, 1)
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

func buildPeonGrass() peonGrass {
	g := peonGrass{
		mask: make([]bool, mapWidth*mapHeight),
		col:  make([]bool, mapWidth),
		maxY: make([]int, mapWidth),
	}
	for i := range g.maxY {
		g.maxY[i] = -1
	}
	o := peonIslandOrigin()
	e := peonIslandExtent()
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
	topX, topY := isoCell(cellX, cellY)
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
	topX, topY := isoCell(cellX, cellY)
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

// skipBuildingPixelOnRoad clears only the ground-band spill onto street diamonds.
// Upper facade may overhang neighboring lots (Flash sprites are wider than one
// tile); forcing a lot-column clip sliced apartment walls in half.
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
