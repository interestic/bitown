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
	// Miniville Cs.hx: SIDE squares of SQUARE_SIDE mini-cells. Full SIDE=30 is
	// ~300×300 (~7k×4k PNG). bitown crops to displaySide squares so map.png
	// stays API-sized; peon maps paint only a centered 4×4 dalle island.
	displaySide = 4  // cropped Cs.SIDE (4×10 = 40 mini-cells per axis)
	squareSide  = 10 // Cs.SQUARE_SIDE

	mapCols = displaySide * squareSide
	mapRows = displaySide * squareSide

	isoTileW = 24 // Cs.WW
	isoTileH = 12 // Cs.HH
	isoPad   = 96

	// groundBlock matches one mcDalle stamp (genMiniSquare).
	groundBlock = 4
	// peonDalleGrid is the Townzzy Caerphilly-style raised field (4×4 plates).
	peonDalleGrid = 4
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

type tilePainter func(img *image.RGBA, cellSeed uint32, tag string, footX, footY, lotX, lotY int, roads roadMaskData)

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
	return renderMapTiles(city, atlas, func(img *image.RGBA, cellSeed uint32, tag string, footX, footY, lotX, lotY int, roads roadMaskData) {
		key := atlas.PickBuildingKeyForLot(city, tag, lotX, lotY, cellSeed)
		if key == "" || !atlas.drawBuildingAtFoot(img, key, footX, footY, lotX, lotY, roads) {
			drawFallbackBuildingClipped(img, cellSeed, footX, footY, lotX, lotY, roads)
		}
	})
}

func buildFallbackMapPNG(city *citycore.City) ([]byte, error) {
	return renderMapTiles(city, nil, func(img *image.RGBA, cellSeed uint32, _ string, footX, footY, lotX, lotY int, roads roadMaskData) {
		drawFallbackBuildingClipped(img, cellSeed, footX, footY, lotX, lotY, roads)
	})
}

func isoCell(x, y int) (topX, topY int) {
	topX = isoOriginX + (x-y)*(isoTileW/2)
	topY = isoOriginY + (x+y)*(isoTileH/2)
	return topX, topY
}

func renderMapTiles(city *citycore.City, atlas *Atlas, paint tilePainter) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, mapWidth, mapHeight))
	ctx := newMapRenderCtx(city, atlas)
	draw.Draw(img, img.Bounds(), &image.Uniform{C: mapCanvasColor(ctx.peon)}, image.Point{}, draw.Src)

	drawMapFloor(img, ctx)

	// Optional ground-only shade before sprites so buildings/trees stay unshaded.
	if groundShadeEnabled() {
		applyGroundShade(img)
	}

	objs := collectMapObjects(ctx)
	paintMapObjects(img, ctx, objs, paint)

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

func hashCell(slug string, x, y int) uint32 {
	h := fnv.New32a()
	_, _ = fmt.Fprintf(h, "%s/%d/%d", slug, x, y)
	return h.Sum32()
}
