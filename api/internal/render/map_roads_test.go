package render

import (
	"image"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestRoadNetworkHasFewGrassSeams(t *testing.T) {
	city := &citycore.City{Slug: "road-seam-check", Pop: 500}
	data, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img := decodeMapPNG(t, data)
	seams := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if !isInteriorRoadSeam(img, x, y) {
				continue
			}
			seams++
		}
	}
	if seams > 120 {
		t.Fatalf("too many interior road seam pixels: %d", seams)
	}
}

func TestFallbackRoadSeamsDoNotEatLotGrass(t *testing.T) {
	assets := t.TempDir()
	t.Setenv("BITOWN_ASSETS_DIR", assets)
	ResetAtlasCacheForTest()
	t.Cleanup(ResetAtlasCacheForTest)

	city := &citycore.City{Slug: "seam-lot-guard", Pop: 0}
	data, err := buildFallbackMapPNG(city)
	if err != nil {
		t.Fatalf("fallback render: %v", err)
	}
	img, ok := decodeMapPNG(t, data).(*image.RGBA)
	if !ok {
		t.Fatal("expected RGBA map")
	}

	grid := buildCityGridForCity(&citycore.City{Slug: city.Slug, Pop: 500})
	eaten := 0
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] != cellLot {
				continue
			}
			topX, topY := isoCell(x, y)
			// Sample the diamond interior, away from shared edges with roads.
			cx, cy := topX, topY+isoTileH/2
			if img.RGBAAt(cx, cy) == roadColor {
				eaten++
			}
		}
	}
	if eaten > 0 {
		t.Fatalf("fallback seam fill painted road color into %d lot interiors", eaten)
	}

	blank := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			blank.SetRGBA(x, y, grassColor)
		}
	}
	blank.SetRGBA(3, 2, roadColor)
	blank.SetRGBA(3, 4, roadColor)
	if !shouldFillRoadSeam(blank, 3, 3) {
		t.Fatal("expected opposite road neighbors to mark a seam")
	}
	blank.SetRGBA(4, 3, roadColor)
	blank.SetRGBA(3, 4, grassColor)
	if shouldFillRoadSeam(blank, 3, 3) {
		t.Fatal("three-neighbor grass must not be treated as a road seam")
	}
}

func isInteriorRoadSeam(img image.Image, x, y int) bool {
	return shouldFillRoadSeam(img, x, y)
}
