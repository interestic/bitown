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
	// ~300×300 (~7k×4k PNG). bitown crops density to displaySide squares (max 25)
	// and fits the live island into a square PNG (max(769, island width)).
	displaySide = 25
	squareSide  = 10 // Cs.SQUARE_SIDE

	mapCols = displaySide * squareSide
	mapRows = displaySide * squareSide

	isoTileW = 24 // Cs.WW
	isoTileH = 12 // Cs.HH
	// Pad for native-height towers (mcHouse3 raw ~484px) and wide overhangs.
	isoPad = 528

	// One mcDalle per square (Game.hx addBat size=5 at square origin).
	// genMiniSquare's 4-cell grid is building placement, not ground stamps.
	groundBlock = squareSide

	// mapMinSquare is the viewport for islands whose width is under a 3×3 plate
	// crop (769). Smaller towns sit centered in this square.
	mapMinSquare = 769
	mapCanvasPad = 24
)

var (
	isoOriginX = (mapRows-1)*(isoTileW/2) + isoPad
	isoOriginY = isoPad
	mapWidth   = isoOriginX + (mapCols-1)*(isoTileW/2) + isoPad + isoTileW/2
	mapHeight  = isoOriginY + (mapCols+mapRows-2)*(isoTileH/2) + isoTileH + isoPad
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
		if key == "" || !atlas.drawBuildingAtFoot(img, key, footX, footY, lotX, lotY, roads, plateGrass{}) {
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

func newMapWorkingImage(originCell, extentCells int) *image.RGBA {
	return image.NewRGBA(isoIslandWorkingBounds(originCell, extentCells))
}

func isoIslandWorkingBounds(originCell, extentCells int) image.Rectangle {
	if extentCells < 1 {
		return image.Rect(0, 0, mapWidth, mapHeight)
	}
	x0, y0 := originCell, originCell
	x1, y1 := originCell+extentCells-1, originCell+extentCells-1
	corners := [4][2]int{{x0, y0}, {x1, y0}, {x0, y1}, {x1, y1}}
	minX, minY := mapWidth, mapHeight
	maxX, maxY := 0, 0
	for _, c := range corners {
		tx, ty := isoCell(c[0], c[1])
		left, right := tx-isoTileW/2, tx+isoTileW/2
		top, bot := ty, ty+isoTileH
		if left < minX {
			minX = left
		}
		if top < minY {
			minY = top
		}
		if right > maxX {
			maxX = right
		}
		if bot > maxY {
			maxY = bot
		}
	}
	minX -= isoPad
	minY -= isoPad
	maxX += isoPad
	maxY += isoPad
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX > mapWidth {
		maxX = mapWidth
	}
	if maxY > mapHeight {
		maxY = mapHeight
	}
	if maxX <= minX || maxY <= minY {
		return image.Rect(0, 0, mapWidth, mapHeight)
	}
	return image.Rect(minX, minY, maxX, maxY)
}

func mapSquareSize(contentW int) int {
	if contentW < mapMinSquare {
		return mapMinSquare
	}
	return contentW
}

func fitMapPNGCanvas(src *image.RGBA, sky color.RGBA) *image.RGBA {
	cropped := cropMapCanvasToContent(src, sky, mapCanvasPad)
	return letterboxMapCanvasSquare(cropped, sky, mapSquareSize(cropped.Bounds().Dx()))
}

func cropMapCanvasToContent(src *image.RGBA, sky color.RGBA, pad int) *image.RGBA {
	b := src.Bounds()
	minX, minY := b.Max.X, b.Max.Y
	maxX, maxY := b.Min.X-1, b.Min.Y-1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			i := src.PixOffset(x, y)
			r, g, bl, a := src.Pix[i], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3]
			if a == 0 || (r == sky.R && g == sky.G && bl == sky.B) {
				continue
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if maxX < minX {
		return src
	}
	minX -= pad
	minY -= pad
	maxX += pad
	maxY += pad
	if minX < b.Min.X {
		minX = b.Min.X
	}
	if minY < b.Min.Y {
		minY = b.Min.Y
	}
	if maxX >= b.Max.X {
		maxX = b.Max.X - 1
	}
	if maxY >= b.Max.Y {
		maxY = b.Max.Y - 1
	}
	w, h := maxX-minX+1, maxY-minY+1
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(out, out.Bounds(), src, image.Pt(minX, minY), draw.Src)
	return out
}

func letterboxMapCanvasSquare(src *image.RGBA, sky color.RGBA, side int) *image.RGBA {
	if side < 1 {
		return src
	}
	b := src.Bounds()
	if b.Dx() == side && b.Dy() == side && b.Min.X == 0 && b.Min.Y == 0 {
		return src
	}
	out := image.NewRGBA(image.Rect(0, 0, side, side))
	draw.Draw(out, out.Bounds(), &image.Uniform{C: sky}, image.Point{}, draw.Src)
	dx, dy := b.Dx(), b.Dy()
	ox := (side - dx) / 2
	oy := (side - dy) / 2
	draw.Draw(out, image.Rect(ox, oy, ox+dx, oy+dy), src, b.Min, draw.Src)
	return out
}

func renderMapTiles(city *citycore.City, atlas *Atlas, paint tilePainter) ([]byte, error) {
	img, sky, err := renderMapTilesImage(city, atlas, paint)
	if err != nil {
		return nil, err
	}
	img = fitMapPNGCanvas(img, sky)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func renderMapTilesImage(city *citycore.City, atlas *Atlas, paint tilePainter) (*image.RGBA, color.RGBA, error) {
	img := newMapWorkingImage(plateIslandOrigin(city.Pop.Int()), plateIslandExtent(city.Pop.Int()))
	ctx := newMapRenderCtx(city, atlas)
	sky := mapCanvasColor(true)
	draw.Draw(img, img.Bounds(), &image.Uniform{C: sky}, image.Point{}, draw.Src)

	drawMapFloor(img, ctx)

	// Optional ground-only shade before sprites so buildings/trees stay unshaded.
	if groundShadeEnabled() {
		applyGroundShade(img)
	}

	objs := collectMapObjects(ctx)
	paintMapObjects(img, ctx, objs, paint)
	return img, sky, nil
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
