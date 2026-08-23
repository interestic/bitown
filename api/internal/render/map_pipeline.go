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
	platform     roadMaskData
	grass        peonGrass
	peon         bool
}

func newMapRenderCtx(city *citycore.City, atlas *Atlas) mapRenderCtx {
	slug := city.Slug.String()
	grid := buildCityGridForCity(city)
	occupancy := lotOccupancy(city, grid)
	ctx := mapRenderCtx{
		city:      city,
		atlas:     atlas,
		slug:      slug,
		grid:      grid,
		occupancy: occupancy,
		order:     mapDrawOrder(),
		roads:     buildRoadMaskData(grid),
		platform:  buildPlatformMask(),
		peon:      !arterialsEnabled(city),
	}
	if atlas != nil {
		ctx.buildingKeys = assignBuildingKeys(atlas, city, occupancy)
	}
	if ctx.peon {
		ctx.grass = buildPeonGrass()
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
		if ctx.peon && !inPeonIsland(cell.x, cell.y) {
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
		drawGroundBlocks(img, ctx.atlas, ctx.slug, ctx.peon)
		if !ctx.peon {
			drawRoadUnderlay(img, ctx.grid)
			softenRoadGrassBoundary(img)
		}
	} else if !ctx.peon {
		drawRoadNetwork(img, ctx.grid)
	}
}

func collectMapObjects(ctx mapRenderCtx) []mapObject {
	objs := make([]mapObject, 0, mapCols*mapRows)
	for _, cell := range ctx.order {
		cellKind := ctx.grid[cell.y][cell.x]
		cellSeed := hashCell(ctx.slug, cell.x, cell.y)
		depth := cell.x + cell.y

		if cellKind == cellRoad {
			key := ""
			height := isoTileH
			if ctx.atlas != nil {
				n := cell.y > 0 && ctx.grid[cell.y-1][cell.x] == cellRoad
				e := cell.x+1 < mapCols && ctx.grid[cell.y][cell.x+1] == cellRoad
				s := cell.y+1 < mapRows && ctx.grid[cell.y+1][cell.x] == cellRoad
				w := cell.x > 0 && ctx.grid[cell.y][cell.x-1] == cellRoad
				key = ctx.atlas.PickRoadKey(n, e, s, w, cell.x, cell.y)
				if h := ctx.atlas.frameHeight(key); h > 0 {
					height = h
				}
			}
			objs = append(objs, mapObject{
				x: cell.x, y: cell.y, depth: depth, height: height,
				seed: cellSeed, key: key, kind: objectRoad,
			})
			continue
		}

		lot, ok := ctx.occupancy[[2]int{cell.x, cell.y}]
		if !ok {
			continue
		}
		if ctx.peon && !inPeonIsland(cell.x, cell.y) {
			continue
		}
		switch lot.use {
		case lotPark:
			key := ""
			height := isoTileH
			if ctx.atlas != nil {
				key = ctx.atlas.PickKeyForTag(TagTree, cellSeed)
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
		case objectRoad:
			if ctx.atlas != nil && obj.key != "" {
				_ = ctx.atlas.drawRoadAtFoot(img, obj.key, footX, footY, ctx.platform)
			}
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
