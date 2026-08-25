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
	grass        peonGrass
	peon         bool
	pop          int
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
	peon := !arterialsEnabled(city)
	roadLift := 0
	if atlas != nil && !peon {
		// SE foot (same as mcDalle) + roadGrassLift onto the plate top.
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
		peon:       peon,
		pop:        pop,
		densityMax: dens.max,
		roadLift:   roadLift,
		roadStamps: roads.stamps,
		roadCross:  roads.cross,
	}
	if atlas != nil {
		ctx.buildingKeys = assignBuildingKeys(atlas, city, occupancy, dens.max)
	}
	if ctx.peon {
		ctx.grass = buildPeonGrass(pop)
	}
	return ctx
}

func mapCanvasColor(peon bool) color.RGBA {
	if peon {
		// Soft sky field like Townzzy peon pages (not terrain grass).
		return color.RGBA{R: 186, G: 220, B: 235, A: 255}
	}
	return grassColor
}

func drawMapFloor(img *image.RGBA, ctx mapRenderCtx) {
	for _, cell := range ctx.order {
		if ctx.peon && !inPeonIslandFor(ctx.pop, cell.x, cell.y) {
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
		drawGroundBlocks(img, ctx.atlas, ctx.slug, ctx.peon, ctx.pop)
		// Townzzy: empty mini-squares get champs after dalle, before roads/buildings.
		drawFarmBlocks(img, ctx)
		if !ctx.peon {
			drawAtlasRoadSprites(img, ctx)
		}
	} else if !ctx.peon {
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
		_ = ctx.atlas.drawRoadOnSquare(img, key, footX+ox, footY+oy, st.sx, st.sy, -ctx.roadLift, 0.22)
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
			footX, footY := squareRoadFoot(sx, sy, -ctx.roadLift)
			_ = ctx.atlas.drawRoadOnSquare(img, key, footX, footY, sx, sy, -ctx.roadLift, 0.12)
		}
	}
}

// squareRoadFoot is the SE mini-cell of the square — same foot as
// drawGroundBlocks / catalog dalleBlockFootDelta, lifted onto mcDalle grass.
func squareRoadFoot(sx, sy, dy int) (footX, footY int) {
	x := sx*squareSide + squareSide - 1
	y := sy*squareSide + squareSide - 1
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
		if ctx.peon && !inPeonIslandFor(ctx.pop, cell.x, cell.y) {
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
		switch obj.kind {
		case objectPark:
			if ctx.atlas != nil && obj.key != "" {
				if ctx.peon {
					_ = ctx.atlas.drawFrameOnPeonGrass(img, obj.key, footX, footY, ctx.grass)
				} else {
					_ = ctx.atlas.drawFrameAtFoot(img, obj.key, footX, footY)
				}
			}
		case objectBuilding:
			if ctx.atlas != nil && obj.key != "" {
				if ctx.peon {
					if ctx.atlas.drawFrameOnPeonGrass(img, obj.key, footX, footY, ctx.grass) {
						continue
					}
				} else if ctx.atlas.drawBuildingAtFoot(img, obj.key, footX, footY, obj.x, obj.y, ctx.roads) {
					continue
				}
			}
			if ctx.peon {
				drawFallbackBuildingOnPeonGrass(img, obj.seed, footX, footY, ctx.grass)
				continue
			}
			paint(img, obj.seed, obj.tag, footX, footY, obj.x, obj.y, ctx.roads)
		}
	}
}
