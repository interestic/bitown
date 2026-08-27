package render

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestE2E_BuildingsStayOffRoadCells(t *testing.T) {
	forceFallbackAtlas(t)

	city := &citycore.City{Slug: "e2e-roads", Pop: 500}
	_, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	grid := buildCityGridForCity(&citycore.City{Slug: city.Slug, Pop: 500})
	occupancy := lotOccupancy(city, grid)

	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if lot, ok := occupancy[[2]int{x, y}]; ok && lot.use == lotBuilding && grid[y][x] == cellRoad {
				t.Fatalf("occupancy placed a building on road cell (%d,%d)", x, y)
			}
		}
	}
}

func TestE2E_LotBuildingsAppearOffRoads(t *testing.T) {
	forceFallbackAtlas(t)

	city := &citycore.City{Slug: "e2e-lots", Pop: 500}
	data, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img := decodeMapPNG(t, data)

	grid := buildCityGridForCity(&citycore.City{Slug: city.Slug, Pop: 500})
	occupancy := lotOccupancy(city, grid)
	foundLot := false
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			lot, ok := occupancy[[2]int{x, y}]
			if !ok || lot.use != lotBuilding {
				continue
			}
			if grid[y][x] == cellRoad {
				t.Fatalf("building lot marked on road (%d,%d)", x, y)
			}
			foundLot = true
		}
	}
	if !foundLot {
		t.Fatal("expected at least one building lot off roads")
	}
	// Fitted PNG no longer shares isoCell coordinates; just require building paint.
	foundBuilding := false
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y && !foundBuilding; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			c := color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(bl >> 8), A: uint8(a >> 8)}
			if isFallbackBuildingColor(c) {
				foundBuilding = true
				break
			}
		}
	}
	if !foundBuilding {
		t.Fatal("expected at least one fallback building pixel")
	}
}

func isFallbackBuildingColor(c color.RGBA) bool {
	for _, bc := range buildingColor {
		if c == bc {
			return true
		}
	}
	return false
}

func TestE2E_IsoRoadDiamondsNotSquareCells(t *testing.T) {
	forceFallbackAtlas(t)

	city := &citycore.City{Slug: "e2e-iso-diamond", Pop: 120}
	grid := buildCityGridForCity(city)
	// Sample isoCell space on the working canvas (pre-fit). Fitted map.png
	// letterboxes and no longer shares absolute iso coordinates.
	img := newMapWorkingImage(plateIslandOrigin(city.Pop.Int()), plateIslandExtent(city.Pop.Int()))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: grassColor}, image.Point{}, draw.Src)
	drawRoadNetwork(img, grid)
	if img.Bounds().Dx() == 320 && img.Bounds().Dy() == 320 {
		t.Fatal("iso map must not be the legacy 320x320 square canvas")
	}

	foundRoad := false
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] != cellRoad {
				continue
			}
			topX, topY := isoCell(x, y)
			center := img.RGBAAt(topX, topY+isoTileH/2)
			if center != roadColor {
				t.Fatalf("road diamond center (%d,%d) want road, got %+v", x, y, center)
			}
			// Outside the diamond, laterally from the top vertex, should stay grass
			// (would be inside a 24x12 AABB / old square cell).
			outside := img.RGBAAt(topX+isoTileW/2-1, topY+1)
			if outside == roadColor {
				t.Fatalf("pixel beside diamond tip at (%d,%d) should not be road (square-cell leak)", x, y)
			}
			foundRoad = true
			break
		}
		if foundRoad {
			break
		}
	}
	if !foundRoad {
		t.Fatal("expected at least one road cell")
	}
}

func TestE2E_IsoPainterOrderCoversBackSprite(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	back := color.RGBA{R: 200, A: 255}
	front := color.RGBA{B: 200, A: 255}

	objs := []mapObject{
		{x: 0, y: 0, depth: 0, height: 40},
		{x: 1, y: 0, depth: 1, height: 20},
	}
	sortMapObjects(objs)

	drawStub := func(obj mapObject, c color.RGBA) {
		// Shared overlap column so painter order is observable.
		footX, footY := 32, 50
		h := obj.height
		for py := footY - h; py < footY; py++ {
			for px := footX - 4; px <= footX+4; px++ {
				img.SetRGBA(px, py, c)
			}
		}
	}
	for _, obj := range objs {
		if obj.depth == 0 {
			drawStub(obj, back)
		} else {
			drawStub(obj, front)
		}
	}
	got := img.RGBAAt(32, 35)
	if got != front {
		t.Fatalf("nearer/taller-last paint should win overlap: got %+v want %+v", got, front)
	}
}
