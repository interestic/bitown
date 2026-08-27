package render

import "image"

// groundSpriteBase is the plate clip (raised grass tile with soil sides).
const groundSpriteBase = "sprites/DefineSprite_707_mcDalle"

// activeSquareOrigin is the top-left live square inside bitown's displaySide
// crop, aligned to Game.hx displayMargin on Cs.SIDE (not the geometric center
// of an odd crop — displaySide=25 would otherwise drift by one square).
func activeSquareOrigin(pop int) int {
	active := activeSquareSide(pop)
	gameOrigin := (csSide - active) / 2
	cropOrigin := (csSide - displaySide) / 2
	local := gameOrigin - cropOrigin
	if local < 0 {
		local = 0
	}
	if local > displaySide-active {
		local = displaySide - active
	}
	return local
}

// plateIslandExtentCells is the live plate island size (Game.hx displaySide
// growth, capped to our PNG field displaySide×SQUARE_SIDE).
func plateIslandExtentCells(pop int) int {
	return activeSquareSide(pop) * squareSide
}

func plateIslandOrigin(pop int) int {
	return activeSquareOrigin(pop) * squareSide
}

func plateIslandExtent(pop int) int {
	return plateIslandExtentCells(pop)
}

func plateIslandCenter(pop int) (int, int) {
	o := plateIslandOrigin(pop)
	e := plateIslandExtent(pop)
	return o + e/2, o + e/2
}

func inPlateIsland(pop, x, y int) bool {
	o := plateIslandOrigin(pop)
	e := plateIslandExtent(pop)
	return x >= o && x < o+e && y >= o && y < o+e
}

// plateGrassLift is the plate soil-side height in pixels (~1.6×isoTileH).
// Catalog plateLip, roads, farms, trees, and buildings use this full lift so
// stamps sit on grass (Townzzy / Storybook). Arterial hang past the city
// diamond is clipped (square road mask / plate grass), not by shortening lift.
const plateGrassLift = 20

// farmGrassLift is the farm/tree/building overlay lift (full soil lip).
const farmGrassLift = plateGrassLift

// roadGrassLift raises arterial road/cross stamps onto plate grass.
// Matches catalog buildGenSquareRoadLayers (y: -lip) and Townzzy crosses.
const roadGrassLift = plateGrassLift

// overlayLift raises trees and buildings onto plate grass.
// Same full soil lip for roadless and arterial — short arterial lift left
// houses and roads south of Townzzy's dalle-top plane.
func overlayLift(roadless bool) int {
	_ = roadless
	return farmGrassLift
}

func overlayFoot(x, y, lift int) (footX, footY int) {
	topX, topY := isoCell(x, y)
	return topX, topY + isoTileH - lift
}

// grassRimInset is how many cells inward from the plate island edge
// count as grass tops. Plate soil sides are ~20px (~1.6×isoTileH),
// so one cell of inset still lets feet sit on the brown ledge. Applies to both
// roadless and arterial plate islands (live Game.hx viewport, not the full field).
const grassRimInset = 2

// grassTopCell reports whether a cell foot sits on the green plate top
// rather than the soil rim of the stamped ground field.
func grassTopCell(pop, x, y int) bool {
	return grassMaskRimDist(pop, x, y) >= grassRimInset
}

// grassMaskRimDist is chebyshev distance from the plate island edge (cells).
// -1 when outside the island.
func grassMaskRimDist(pop, x, y int) int {
	if !inPlateIsland(pop, x, y) {
		return -1
	}
	o := plateIslandOrigin(pop)
	last := o + plateIslandExtent(pop) - 1
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
	return d
}

// drawGroundBlocks stamps plates for the live island only (Game.hx viewport).
// Arterial and roadless share the same island extent; the full displaySide
// field is not tiled.
func drawGroundBlocks(img *image.RGBA, atlas *Atlas, slug string, roadless bool, pop int) {
	if atlas == nil {
		return
	}
	_ = roadless
	o := plateIslandOrigin(pop)
	e := plateIslandExtent(pop)
	x0, y0 := o, o
	x1, y1 := o+e, o+e
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
