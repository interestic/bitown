package render

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log/slog"
	"os"

	"github.com/interestic/bitown/internal/citycore"
)

const (
	mapCols  = 20
	mapRows  = 20
	isoTileW = 32
	isoTileH = 16
	isoPad   = 96
)

var (
	isoOriginX = (mapRows-1)*(isoTileW/2) + isoPad
	isoOriginY = isoPad
	mapWidth   = isoOriginX + (mapCols-1)*(isoTileW/2) + isoPad + isoTileW/2
	mapHeight  = isoOriginY + (mapCols+mapRows-2)*(isoTileH/2) + isoTileH + isoPad + 96
)

var (
	grassColor    = color.RGBA{R: 152, G: 207, B: 84, A: 255}
	roadColor     = color.RGBA{R: 78, G: 78, B: 78, A: 255}
	buildingColor = []color.RGBA{
		{R: 182, G: 201, B: 230, A: 255},
		{R: 224, G: 188, B: 160, A: 255},
		{R: 197, G: 178, B: 218, A: 255},
		{R: 157, G: 205, B: 188, A: 255},
	}
)

const buildingColorCount uint32 = 4

type tilePainter func(img *image.RGBA, cellSeed uint32, tag string, footX, footY int)

// BuildCityMapPNG renders a deterministic city map PNG using the v1 sprite atlas when available.
func BuildCityMapPNG(city *citycore.City) ([]byte, error) {
	atlas, atlasErr := loadAtlas()
	if atlasErr != nil {
		if atlasRequired() {
			return nil, fmt.Errorf("atlas required: %w", atlasErr)
		}
		slog.Warn(
			"atlas unavailable, using fallback map renderer",
			"reason", atlasFallbackReason(atlasErr),
			"err", atlasErr,
		)
		return buildFallbackMapPNG(city)
	}
	return buildAtlasMapPNG(city, atlas)
}

func atlasRequired() bool {
	if os.Getenv("BITOWN_ATLAS_REQUIRED") == "true" {
		return true
	}
	return os.Getenv("ENV") == "production"
}

func buildAtlasMapPNG(city *citycore.City, atlas *Atlas) ([]byte, error) {
	return renderMapTiles(city, atlas, func(img *image.RGBA, cellSeed uint32, tag string, footX, footY int) {
		key := atlas.PickKeyForTag(tag, cellSeed)
		if key == "" {
			key = atlas.pickBuildingKey(cellSeed)
		}
		if !atlas.drawFrameAtFoot(img, key, footX, footY) {
			drawFallbackBuilding(img, cellSeed, footX, footY)
		}
	})
}

func buildFallbackMapPNG(city *citycore.City) ([]byte, error) {
	return renderMapTiles(city, nil, func(img *image.RGBA, cellSeed uint32, _ string, footX, footY int) {
		drawFallbackBuilding(img, cellSeed, footX, footY)
	})
}

func isoCell(x, y int) (topX, topY int) {
	topX = isoOriginX + (x-y)*(isoTileW/2)
	topY = isoOriginY + (x+y)*(isoTileH/2)
	return topX, topY
}

func renderMapTiles(city *citycore.City, atlas *Atlas, paint tilePainter) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, mapWidth, mapHeight))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: grassColor}, image.Point{}, draw.Src)

	grid := buildCityGrid(city.Slug)
	occupancy := lotOccupancy(city, grid)
	order := mapDrawOrder()

	// Floor pass: grass on lots, then road underlay. Skip textured grass under
	// roads so 1px iso seams do not reveal yellow-green speckle between tiles.
	for _, cell := range order {
		if grid[cell.y][cell.x] == cellRoad {
			continue
		}
		topX, topY := isoCell(cell.x, cell.y)
		ground := grassColor
		if atlas != nil {
			ground = grassTileColor(hashCell(city.Slug, cell.x, cell.y))
		}
		drawIsoDiamond(img, topX, topY, ground, 0)
	}
	if atlas != nil {
		drawRoadUnderlay(img, grid)
	} else {
		drawRoadNetwork(img, grid)
	}

	// Object pass: depth-sorted road overlays, trees, then buildings.
	// Same-depth cells are further ordered by sprite height (issue #7).
	objs := make([]mapObject, 0, mapCols*mapRows)
	for _, cell := range order {
		cellKind := grid[cell.y][cell.x]
		cellSeed := hashCell(city.Slug, cell.x, cell.y)
		depth := cell.x + cell.y

		if cellKind == cellRoad {
			key := ""
			height := isoTileH
			if atlas != nil {
				n, e, s, w := roadNeighbors(grid, cell.x, cell.y)
				key = atlas.PickRoadKey(n, e, s, w, cell.x, cell.y)
				if h := atlas.frameHeight(key); h > 0 {
					height = h
				}
			}
			objs = append(objs, mapObject{
				x: cell.x, y: cell.y, depth: depth, height: height,
				seed: cellSeed, key: key, kind: objectRoad,
			})
			continue
		}

		lot, ok := occupancy[[2]int{cell.x, cell.y}]
		if !ok {
			continue
		}
		switch lot.use {
		case lotPark:
			key := ""
			height := isoTileH
			if atlas != nil {
				key = atlas.PickKeyForTag(TagTree, cellSeed)
				if h := atlas.frameHeight(key); h > 0 {
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
			if atlas != nil {
				key = atlas.PickKeyForTag(lot.tag, cellSeed)
				if key == "" {
					key = atlas.pickBuildingKey(cellSeed)
				}
				if h := atlas.frameHeight(key); h > 0 {
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

	for _, obj := range objs {
		topX, topY := isoCell(obj.x, obj.y)
		footX, footY := topX, topY+isoTileH
		switch obj.kind {
		case objectRoad:
			if atlas != nil && obj.key != "" {
				_ = atlas.drawFrameAtFoot(img, obj.key, footX, footY)
			}
		case objectPark:
			if atlas != nil && obj.key != "" {
				_ = atlas.drawFrameAtFoot(img, obj.key, footX, footY)
			}
		case objectBuilding:
			if atlas != nil && obj.key != "" {
				if atlas.drawFrameAtFoot(img, obj.key, footX, footY) {
					continue
				}
			}
			paint(img, obj.seed, obj.tag, footX, footY)
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func drawIsoDiamond(img *image.RGBA, topX, topY int, c color.RGBA, edgeOverlap int) {
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
		for px := topX - half; px <= topX+half; px++ {
			if !image.Pt(px, py).In(img.Bounds()) {
				continue
			}
			img.SetRGBA(px, py, c)
		}
	}
}

func grassTileColor(seed uint32) color.RGBA {
	// Subtle deterministic variation gives a textured ground feel.
	switch seed % 5 {
	case 0:
		return color.RGBA{R: 146, G: 200, B: 80, A: 255}
	case 1:
		return color.RGBA{R: 158, G: 213, B: 88, A: 255}
	case 2:
		return color.RGBA{R: 150, G: 205, B: 82, A: 255}
	default:
		return grassColor
	}
}

func drawFallbackBuilding(img *image.RGBA, cellSeed uint32, footX, footY int) {
	c := buildingColor[cellSeed%buildingColorCount]
	w, h := 10, 14
	rect := image.Rect(footX-w/2, footY-h, footX+w/2, footY)
	draw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, draw.Src)
}

func hashCell(slug string, x, y int) uint32 {
	h := fnv.New32a()
	_, _ = fmt.Fprintf(h, "%s/%d/%d", slug, x, y)
	return h.Sum32()
}
