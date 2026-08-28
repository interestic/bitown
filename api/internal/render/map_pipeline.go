package render

import (
	"image"
	"image/color"

	"github.com/interestic/bitown/internal/citycore"
)

// mapRenderCtx holds shared state for the floor → objects → paint pipeline.
type mapRenderCtx struct {
	city         *citycore.City
	atlas        *Atlas
	slug         string
	grid         cityGrid
	occupancy    map[[2]int]lotCell
	order        []mapCoord
	buildingKeys map[[2]int]string
	roads        roadMaskData
	grass        plateGrass
	roadless     bool
	pop          int
	dens         popDensity
	densityMax   int
	roadLift     int
	roadStamps   []roadStamp
	roadCross    [][]uint8
	landmarks    []landmarkStamp
	landmarkSq   map[[2]int]struct{}
}

func newMapRenderCtx(city *citycore.City, atlas *Atlas) mapRenderCtx {
	slug := city.Slug.String()
	pop := city.Pop.Int()
	dens := genMapPop(pop, newMapRNG(slug))
	roads := planRoads(city, dens)
	occupancy := lotOccupancyWithDensity(city, roads.grid, dens)
	roadless := !arterialsEnabled(city)
	roadLift := 0
	if atlas != nil && !roadless {
		// Center cell foot + plate lip onto the grass top (Townzzy cross).
		roadLift = roadGrassLift
	}
	ctx := mapRenderCtx{
		city:       city,
		atlas:      atlas,
		slug:       slug,
		grid:       roads.grid,
		occupancy:  occupancy,
		order:      mapDrawOrder(),
		roads:      buildRoadMaskDataWithCross(roads.grid, roads.cross, -roadLift),
		roadless:   roadless,
		pop:        pop,
		dens:       dens,
		densityMax: dens.max,
		roadLift:   roadLift,
		roadStamps: roads.stamps,
		roadCross:  roads.cross,
	}
	if atlas != nil {
		ctx.buildingKeys = assignBuildingKeys(atlas, city, occupancy, dens.max)
		ctx.landmarks, ctx.landmarkSq = planSquareLandmarks(city, atlas, dens)
	}
	ctx.grass = buildPlateGrass(pop)
	return ctx
}

func mapCanvasColor(roadless bool) color.RGBA {
	if roadless {
		// Soft sky field like Townzzy roadless pages (not terrain grass).
		return color.RGBA{R: 186, G: 220, B: 235, A: 255}
	}
	return grassColor
}

func drawMapFloor(img *image.RGBA, ctx mapRenderCtx) {
	for _, cell := range ctx.order {
		if !inPlateIsland(ctx.pop, cell.x, cell.y) {
			continue
		}
		topX, topY := isoCell(cell.x, cell.y)
		ground := grassColor
		if ctx.atlas != nil {
			ground = grassTileColor(hashCell(ctx.slug, cell.x, cell.y))
		}
		drawIsoDiamond(img, topX, topY, ground, 0)
	}
	if ctx.atlas != nil {
		drawGroundBlocks(img, ctx.atlas, ctx.slug, ctx.roadless, ctx.pop)
		// Townzzy: empty mini-squares get farm after plates, before roads/buildings.
		drawFarmBlocks(img, ctx)
		// Storybook ground deco (tennis, pebbles) on vacant grass — not park objects.
		drawGroundDecoBlocks(img, ctx)
		if !ctx.roadless {
			drawAtlasRoadSprites(img, ctx)
		}
	} else if !ctx.roadless {
		drawRoadNetwork(img, ctx.grid)
	}
}

// drawAtlasArterialRoadSprites stamps mcRoad axes on the floor pass (before buildings).
func drawAtlasArterialRoadSprites(img *image.RGBA, ctx mapRenderCtx) {
	if ctx.atlas == nil {
		return
	}
	for _, st := range ctx.roadStamps {
		dir0 := st.dir == 0
		key := ctx.atlas.PickRoadKey(dir0, !dir0, st.style)
		if key == "" {
			continue
		}
		footX, footY := squareRoadFoot(st.sx, st.sy, -ctx.roadLift)
		ox, oy := axisOverlapNudge(ctx.roadStamps, st)
		// Clip at the same lift as the foot (catalog: dalle-top diamond).
		_ = ctx.atlas.drawRoadOnSquare(img, key, footX+ox, footY+oy, st.sx, st.sy, -ctx.roadLift, 0.22, ctx.grass)
	}
}

// drawAtlasCrossRoadSprites stamps DefineSprite_705 after buildings so junction
// asphalt stays visible when adjacent yards overlap the intersection in screen space.
func drawAtlasCrossRoadSprites(img *image.RGBA, ctx mapRenderCtx) {
	if ctx.atlas == nil || len(ctx.roadCross) == 0 {
		return
	}
	for sy := 0; sy < displaySide && sy < len(ctx.roadCross); sy++ {
		for sx := 0; sx < displaySide && sx < len(ctx.roadCross[sy]); sx++ {
			kind := ctx.roadCross[sy][sx]
			if kind == 0 {
				continue
			}
			key := ctx.atlas.PickRoadCrossKey(kind >= 2)
			if key == "" {
				continue
			}
			footX, footY := squareCrossFoot(sx, sy, -ctx.roadLift)
			_ = ctx.atlas.drawRoadOnSquare(img, key, footX, footY, sx, sy, -ctx.roadLift, 0.12, ctx.grass)
		}
	}
}

// drawAtlasRoadSprites stamps arterials then CROSS (legacy single-pass helper).
func drawAtlasRoadSprites(img *image.RGBA, ctx mapRenderCtx) {
	drawAtlasArterialRoadSprites(img, ctx)
	drawAtlasCrossRoadSprites(img, ctx)
}

// squareRoadFoot is the SE mini-cell of the square, lifted onto plate grass.
// Matches catalog dalleBlockFootDelta / Game.hx genSquare stamp origin.
func squareRoadFoot(sx, sy, dy int) (footX, footY int) {
	return cellRoadFoot(sx*squareSide+squareSide-1, sy*squareSide+squareSide-1, dy)
}

// crossStampFootLocal is the square-local cell for DefineSprite_705.
// Game.hx / older pipeline used SE (9). Sandbox map-base settled on 7 so the
// X sits less 下象限-biased while 705's SE-anchored art stays on-plate.
const crossStampFootLocal = 7

// crossStampNudgeY shifts CROSS up in screen space (negative = up).
// Arterials (702) stay on SE foot — confirmed with sandbox map-base #11.
const crossStampNudgeY = -13

// squareCrossFoot is the CROSS stamp foot for a square (local 7 + screen nudge).
func squareCrossFoot(sx, sy, dy int) (footX, footY int) {
	footX, footY = cellRoadFoot(sx*squareSide+crossStampFootLocal, sy*squareSide+crossStampFootLocal, dy)
	return footX, footY + crossStampNudgeY
}

// plateCrossFootCell is the map cell for the plate-center CROSS stamp (local 7,7).
// Building feet must avoid this cell whenever arterials are drawn, even if the
// CROSS sprite is not painted (stage 21 left/back plates).
func plateCrossFootCell(bx, by int) (x, y int) {
	return bx + crossStampFootLocal, by + crossStampFootLocal
}

func lotOnPlateCrossFoot(x, y, bx, by int) bool {
	cx, cy := plateCrossFootCell(bx, by)
	return x == cx && y == cy
}

// lotOnCrossFoot reports whether (x,y) is the CROSS stamp foot (local 7,7).
func lotOnCrossFoot(x, y int, cross [][]uint8) bool {
	return crossFootChebyshev(x, y, cross) == 0
}

// lotReservedForCross reports whether (x,y) is the CROSS asphalt foot cell only.
// Yards ring around it via arterial_yard foot 3 on the other minis (sandbox contract).
func lotReservedForCross(x, y int, cross [][]uint8) bool {
	return lotOnCrossFoot(x, y, cross)
}

// crossFootChebyshev returns chebyshev distance from (x,y) to the CROSS foot of
// its square, or -1 when the square has no planned CROSS.
func crossFootChebyshev(x, y int, cross [][]uint8) int {
	if len(cross) == 0 {
		return -1
	}
	sx := x / squareSide
	sy := y / squareSide
	if sx < 0 || sy < 0 || sy >= len(cross) || sx >= len(cross[sy]) {
		return -1
	}
	if cross[sy][sx] == 0 {
		return -1
	}
	cx := sx*squareSide + crossStampFootLocal
	cy := sy*squareSide + crossStampFootLocal
	dx := x - cx
	if dx < 0 {
		dx = -dx
	}
	dy := y - cy
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

func cellRoadFoot(x, y, dy int) (footX, footY int) {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x >= mapCols {
		x = mapCols - 1
	}
	if y >= mapRows {
		y = mapRows - 1
	}
	topX, topY := isoCell(x, y)
	return topX, topY + isoTileH + dy
}

// axisOverlapNudge pulls a stamp toward neighbouring squares that share the
// same axis so square-cut ends meet (catalog axisOverlapNudge).
func axisOverlapNudge(stamps []roadStamp, st roadStamp) (ox, oy int) {
	const pull = 0.1
	hereX, hereY := squareRoadFoot(st.sx, st.sy, 0)
	for _, other := range stamps {
		if other.dir != st.dir {
			continue
		}
		dx, dy := other.sx-st.sx, other.sy-st.sy
		if st.dir == 0 {
			if dy != 0 || (dx != -1 && dx != 1) {
				continue
			}
		} else if dx != 0 || (dy != -1 && dy != 1) {
			continue
		}
		thereX, thereY := squareRoadFoot(other.sx, other.sy, 0)
		ox += int(float64(thereX-hereX) * pull)
		oy += int(float64(thereY-hereY) * pull)
	}
	return ox, oy
}

func collectMapObjects(ctx mapRenderCtx) []mapObject {
	objs := make([]mapObject, 0, mapCols*mapRows)
	for _, cell := range ctx.order {
		cellKind := ctx.grid[cell.y][cell.x]
		cellSeed := hashCell(ctx.slug, cell.x, cell.y)
		depth := cell.x + cell.y

		if cellKind == cellRoad {
			continue
		}

		lot, ok := ctx.occupancy[[2]int{cell.x, cell.y}]
		if !ok {
			continue
		}
		if !inPlateIsland(ctx.pop, cell.x, cell.y) {
			continue
		}
		switch lot.use {
		case lotPark:
			if squareHasLandmark(ctx.landmarkSq, cell.x, cell.y) {
				continue
			}
			key := ""
			height := isoTileH
			if ctx.atlas != nil {
				key = ctx.atlas.PickKeyForTagUnlocked(ctx.city, TagTree, cellSeed)
				if h := ctx.atlas.frameHeight(key); h > 0 {
					height = h
				}
			}
			objs = append(objs, mapObject{
				x: cell.x, y: cell.y, depth: depth, height: height,
				seed: cellSeed, key: key, kind: objectPark,
			})
		case lotBuilding:
			if squareHasLandmark(ctx.landmarkSq, cell.x, cell.y) {
				continue
			}
			key := ""
			height := 14 // fallback building rect height
			if ctx.atlas != nil {
				key = ctx.buildingKeys[[2]int{cell.x, cell.y}]
				if h := ctx.atlas.frameHeight(key); h > 0 {
					height = h
				}
			}
			objs = append(objs, mapObject{
				x: cell.x, y: cell.y, depth: depth, height: height,
				seed: cellSeed, tag: lot.tag, key: key, kind: objectBuilding,
			})
		}
	}
	for _, lm := range ctx.landmarks {
		height := isoTileH
		if ctx.atlas != nil {
			if h := ctx.atlas.frameHeight(lm.key); h > 0 {
				height = h
			}
		}
		objs = append(objs, mapObject{
			x: lm.x, y: lm.y, depth: lm.x + lm.y, height: height,
			key: lm.key, tag: TagLandmark, kind: objectLandmark,
		})
	}
	sortMapObjects(objs)
	return objs
}

func paintMapObjects(img *image.RGBA, ctx mapRenderCtx, objs []mapObject, paint tilePainter) {
	for _, obj := range objs {
		topX, topY := isoCell(obj.x, obj.y)
		footX, footY := topX, topY+isoTileH
		if ctx.atlas != nil || ctx.roadless {
			footX, footY = overlayFoot(obj.x, obj.y, overlayLift(ctx.roadless))
		}
		switch obj.kind {
		case objectPark:
			if ctx.atlas != nil && obj.key != "" {
				_ = ctx.atlas.drawFrameOnGrassTop(img, obj.key, footX, footY, ctx.grass)
			}
		case objectBuilding:
			if ctx.atlas != nil && obj.key != "" {
				footX, footY = buildingStampFoot(ctx.atlas, obj.key, obj.x, obj.y, ctx.roadless)
			} else if !ctx.roadless {
				footX = applyWestMiniStampNudge(footX, obj.x, obj.y)
				footX = applyNorthMiniStampNudgeX(footX, obj.x, obj.y)
				footX = applyEastMiniStampNudge(footX, obj.x, obj.y)
				footY = applyNorthMiniStampNudge(footY, obj.x, obj.y)
				footY = applySEMiniStampNudge(footY, obj.x, obj.y)
				footY = applyEWMiniStampNudge(footY, obj.x, obj.y)
				footX, footY = applyArterialYardStampNudge(footX, footY, obj.x, obj.y)
			}
			if ctx.atlas != nil && obj.key != "" {
				if ctx.roadless {
					if ctx.atlas.drawFrameOnGrassTop(img, obj.key, footX, footY, ctx.grass) {
						continue
					}
				} else if ctx.atlas.drawBuildingAtFoot(img, obj.key, footX, footY, obj.x, obj.y, ctx.roads, ctx.grass) {
					continue
				}
			}
			if ctx.roadless {
				drawFallbackBuildingOnGrassTop(img, obj.seed, footX, footY, ctx.grass)
				continue
			}
			paint(img, obj.seed, obj.tag, footX, footY, obj.x, obj.y, ctx.roads)
		case objectLandmark:
			if ctx.atlas == nil || obj.key == "" {
				continue
			}
			footX, footY = landmarkStampFoot(obj.x, obj.y)
			_ = ctx.atlas.drawFrameAtFoot(img, obj.key, footX, footY)
		}
	}
}
