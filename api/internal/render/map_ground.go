package render

import "image"

// groundSpriteBase is the Miniville dalle clip (raised grass tile with soil sides).
const groundSpriteBase = "sprites/DefineSprite_707_mcDalle"

// peonExtentCells is the centered dalle field size for no-arterial (peon) maps.
// Matches Game.hx displaySide growth, capped to our PNG crop (displaySide×SQUARE_SIDE).
func peonExtentCells(pop int) int {
	side := activeSquareSide(pop)
	return side * squareSide
}

func peonIslandOriginFor(pop int) int {
	e := peonExtentCells(pop)
	return (mapCols - e) / 2
}

func peonIslandExtentFor(pop int) int {
	return peonExtentCells(pop)
}

func inPeonIslandFor(pop, x, y int) bool {
	o := peonIslandOriginFor(pop)
	e := peonIslandExtentFor(pop)
	return x >= o && x < o+e && y >= o && y < o+e
}

func peonPlateOfFor(pop, x, y int) (plateX, plateY int) {
	o := peonIslandOriginFor(pop)
	return (x - o) / groundBlock, (y - o) / groundBlock
}

// dalleGrassLift is mcDalle's soil-side height in pixels (~1.6×isoTileH).
// Catalog dalleLip and peon champs use this full lift.
const dalleGrassLift = 20

// roadGrassLift raises arterial stamps on the dalle SE foot onto the grass.
// Full dalleLip (20) sits one iso diamond too far north on the city PNG;
// clip stays on the lifted plate so 702 art cannot hang past the north tip.
const roadGrassLift = dalleGrassLift - isoTileH

// peonGrassRimInset is how many mini-cells inward from the dalle field edge
// count as green diamond tops. mcDalle soil sides are ~20px (~1.6×isoTileH),
// so one cell of inset still lets feet sit on the brown ledge. Applies to both
// peon islands and arterial maps (full crop is the same 60×60 field).
const peonGrassRimInset = 2

// peonGrassTopCell reports whether a mini-cell foot sits on the green dalle
// top rather than the soil rim of the stamped ground field.
func peonGrassTopCell(pop, x, y int) bool {
	if !inPeonIslandFor(pop, x, y) {
		return false
	}
	o := peonIslandOriginFor(pop)
	last := o + peonIslandExtentFor(pop) - 1
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
	return d >= peonGrassRimInset
}

func peonPlateAnchorCellFor(pop, plateX, plateY int) (int, int) {
	o := peonIslandOriginFor(pop)
	last := o + peonIslandExtentFor(pop) - 1
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

// drawGroundBlocks stamps mcDalle plates. Peon cities get a centered field
// sized by Game.hx displaySide (capped to the PNG crop); larger cities tile
// the full map.
func drawGroundBlocks(img *image.RGBA, atlas *Atlas, slug string, peon bool, pop int) {
	if atlas == nil {
		return
	}
	x0, y0 := 0, 0
	x1, y1 := mapCols, mapRows
	if peon {
		o := peonIslandOriginFor(pop)
		e := peonIslandExtentFor(pop)
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
