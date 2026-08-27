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
		roads:      buildRoadMaskDataOffset(roads.grid, -roadLift),
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
		if !ctx.roadless {
			drawAtlasRoadSprites(img, ctx)
		}
	} else if !ctx.roadless {
		drawRoadNetwork(img, ctx.grid)
	}
}

// drawAtlasRoadSprites stamps one mcRoad per live square axis (Game.hx size=3:
// 1 edge = 1 part). Floor pass is before buildings, so stamps are not clipped
// to mini-cells — the full ~133px strip can span the square edge.
func drawAtlasRoadSprites(img *image.RGBA, ctx mapRenderCtx) {
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
	if len(ctx.roadCross) == 0 {
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
			// CROSS foot: local 7 + screen nudgeY (sandbox map-base contract).
			// Arterials stay on SE foot so strip tips still meet square edges.
			footX, footY := squareCrossFoot(sx, sy, -ctx.roadLift)
			_ = ctx.atlas.drawRoadOnSquare(img, key, footX, footY, sx, sy, -ctx.roadLift, 0.12, ctx.grass)
		}
	}
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
			// overlayFoot already applied plate lift; add mini stamp nudges
			// from sandbox map-base (west/north/SE/EW/east).
			footX = applyWestMiniStampNudge(footX, obj.x, obj.y)
			footX = applyNorthMiniStampNudgeX(footX, obj.x, obj.y)
			footX = applyEastMiniStampNudge(footX, obj.x, obj.y)
			footY = applyNorthMiniStampNudge(footY, obj.x, obj.y)
			footY = applySEMiniStampNudge(footY, obj.x, obj.y)
			footY = applyEWMiniStampNudge(footY, obj.x, obj.y)
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
		}
	}
}
