package render

import "image"

// groundSpriteBase is the Miniville dalle clip (raised grass tile with soil sides).
const groundSpriteBase = "sprites/DefineSprite_707_mcDalle"

func peonIslandOrigin() int {
	return (mapCols - peonDalleGrid*groundBlock) / 2
}

func peonIslandExtent() int {
	return peonDalleGrid * groundBlock
}

func inPeonIsland(x, y int) bool {
	o := peonIslandOrigin()
	e := peonIslandExtent()
	return x >= o && x < o+e && y >= o && y < o+e
}

func peonPlateCount() int {
	return peonDalleGrid * peonDalleGrid
}

func peonPlateOf(x, y int) (plateX, plateY int) {
	o := peonIslandOrigin()
	return (x - o) / groundBlock, (y - o) / groundBlock
}

// peonPlateAnchorCell is the 4×4 dalle plate cell farthest from the island
// rim so the sprite foot sits on the green diamond top, not the soil edge.
func peonPlateAnchorCell(plateX, plateY int) (int, int) {
	o := peonIslandOrigin()
	last := o + peonIslandExtent() - 1
	bx := o + plateX*groundBlock
	by := o + plateY*groundBlock
	bestX, bestY := bx, by
	best := -1
	for y := by; y < by+groundBlock && y <= last; y++ {
		for x := bx; x < bx+groundBlock && x <= last; x++ {
			d := x - o
			if y-o < d {
				d = y - o
			}
			if last-x < d {
				d = last - x
			}
			if last-y < d {
				d = last - y
			}
			if d > best {
				best = d
				bestX, bestY = x, y
			}
		}
	}
	return bestX, bestY
}

// drawGroundBlocks stamps mcDalle plates. Peon cities get a centered 4×4 field
// (Townzzy Caerphilly); larger cities tile the full map.
func drawGroundBlocks(img *image.RGBA, atlas *Atlas, slug string, peon bool) {
	if atlas == nil {
		return
	}
	x0, y0 := 0, 0
	x1, y1 := mapCols, mapRows
	if peon {
		o := peonIslandOrigin()
		e := peonIslandExtent()
		x0, y0 = o, o
		x1, y1 = o+e, o+e
	}
	for by := y0; by < y1; by += groundBlock {
		for bx := x0; bx < x1; bx += groundBlock {
			key := atlas.PickGroundKey(hashCell(slug, bx, by))
			if key == "" {
				continue
			}
			fx := bx + groundBlock - 1
			fy := by + groundBlock - 1
			if fx >= x1 {
				fx = x1 - 1
			}
			if fy >= y1 {
				fy = y1 - 1
			}
			topX, topY := isoCell(fx, fy)
			_ = atlas.drawFrameAtFoot(img, key, topX, topY+isoTileH)
		}
	}
}

// PickGroundKey prefers mcDalle frames; falls back to any TagGround base.
func (a *Atlas) PickGroundKey(seed uint32) string {
	if a == nil {
		return ""
	}
	variants := []string{"1_v00.png", "2_v00.png", "10_v00.png", "20_v00.png"}
	pick := variants[int(seed%uint32(len(variants)))] //#nosec G115 -- variant count is fixed at 4
	key := groundSpriteBase + "/" + pick
	if _, ok := a.Frames[key]; ok {
		return key
	}
	fallback := groundSpriteBase + "/1_v00.png"
	if _, ok := a.Frames[fallback]; ok {
		return fallback
	}
	return a.PickKeyForTag(TagGround, seed)
}
