package render

import (
	"image"
	"image/color"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestE2E_BuildingsStayOffRoadCells(t *testing.T) {
	forceFallbackAtlas(t)

	city := &citycore.City{Slug: "e2e-roads", Pop: 500}
	data, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img, ok := decodeMapPNG(t, data).(*image.RGBA)
	if !ok {
		t.Fatal("expected RGBA map")
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

	buildingOnRoad := 0
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] != cellRoad {
				continue
			}
			topX, topY := isoCell(x, y)
			// Sample near the diamond center; fallback buildings are solid fills.
			samples := []image.Point{
				{X: topX, Y: topY + isoTileH/2},
				{X: topX - 2, Y: topY + isoTileH/2},
				{X: topX + 2, Y: topY + isoTileH/2},
			}
			for _, pt := range samples {
				if !pt.In(img.Bounds()) {
					continue
				}
				c := img.RGBAAt(pt.X, pt.Y)
				if isFallbackBuildingColor(c) {
					buildingOnRoad++
				}
			}
		}
	}
	if buildingOnRoad > 0 {
		t.Fatalf("fallback map painted building colors on road cells (%d samples)", buildingOnRoad)
	}
}

func TestE2E_LotBuildingsAppearOffRoads(t *testing.T) {
	forceFallbackAtlas(t)

	city := &citycore.City{Slug: "e2e-lots", Pop: 500}
	data, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img, ok := decodeMapPNG(t, data).(*image.RGBA)
	if !ok {
		t.Fatal("expected RGBA map")
	}

	grid := buildCityGridForCity(&citycore.City{Slug: city.Slug, Pop: 500})
	occupancy := lotOccupancy(city, grid)
	foundBuilding := false
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			lot, ok := occupancy[[2]int{x, y}]
			if !ok || lot.use != lotBuilding {
				continue
			}
			if grid[y][x] == cellRoad {
				t.Fatalf("building lot marked on road (%d,%d)", x, y)
			}
			topX, topY := isoCell(x, y)
			footX, footY := topX, topY+isoTileH
			// Fallback buildings are drawn above the foot.
			c := img.RGBAAt(footX, footY-4)
			if isFallbackBuildingColor(c) {
				foundBuilding = true
			}
		}
	}
	if !foundBuilding {
		t.Fatal("expected at least one fallback building pixel on a lot cell")
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
	data, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img, ok := decodeMapPNG(t, data).(*image.RGBA)
	if !ok {
		t.Fatal("expected RGBA map")
	}
	if img.Bounds().Dx() == 320 && img.Bounds().Dy() == 320 {
		t.Fatal("iso map must not be the legacy 320x320 square canvas")
	}

	grid := buildCityGridForCity(city)
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
