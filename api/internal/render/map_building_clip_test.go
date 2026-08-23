package render

import (
	"image"
	"image/color"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestLotColumnClipKeepsPaintOffAdjacentRoad(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, mapWidth, mapHeight))
	grid := make(cityGrid, mapRows)
	for y := 0; y < mapRows; y++ {
		grid[y] = make([]int, mapCols)
		for x := 0; x < mapCols; x++ {
			grid[y][x] = cellLot
		}
	}
	grid[1][0] = cellRoad
	roads := buildRoadMaskData(grid)

	lotX, lotY := 1, 1
	topX, topY := isoCell(lotX, lotY)
	footX, footY := topX, topY+isoTileH
	roadTopX, roadTopY := isoCell(0, 1)
	roadCX, roadCY := roadTopX, roadTopY+isoTileH/2

	spill := color.RGBA{R: 200, G: 40, B: 40, A: 255}
	for py := footY - 80; py < footY; py++ {
		for px := footX - 40; px <= footX+40; px++ {
			if !image.Pt(px, py).In(img.Bounds()) {
				continue
			}
			if skipBuildingPixelOnRoad(roads, footY, px, py, lotX, lotY) {
				continue
			}
			img.SetRGBA(px, py, spill)
		}
	}
	if got := img.RGBAAt(roadCX, roadCY); got == spill {
		t.Fatalf("road center buried: %+v", got)
	}
	// Upper facade may extend past the lot diamond; only the ground band is clipped.
	upperY := footY - buildingGroundBand - 4
	if upperY >= 0 && img.RGBAAt(footX+isoTileW, upperY) != spill {
		t.Fatalf("upper facade at (%d,%d) was clipped; only the ground band must be clipped", footX+isoTileW, upperY)
	}
}

func TestE2E_AtlasRoadsStayClearWithSetback(t *testing.T) {
	requireAtlasFiles(t)
	city := &citycore.City{Slug: "testcity", Pop: 500}
	data, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img, ok := decodeMapPNG(t, data).(*image.RGBA)
	if !ok {
		t.Fatal("expected RGBA")
	}
	grid := buildCityGridForCity(&citycore.City{Slug: city.Slug, Pop: 500})
	buried, checked := 0, 0
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] != cellRoad {
				continue
			}
			topX, topY := isoCell(x, y)
			c := img.RGBAAt(topX, topY+isoTileH/2)
			checked++
			if !looksLikeRoadSurface(c) {
				buried++
				if buried <= 6 {
					t.Logf("road(%d,%d) %v", x, y, c)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("expected road samples")
	}
	// Setback + ground-band clip keep most street centers clear; a few may be
	// legitimately occluded by towers that share an iso screen column.
	// Width cap is 50px (#49); allow up to ~33% bury so the gate stays stable
	// under seed/composition jitter while still catching #33 regressions.
	if buried*3 > checked {
		t.Fatalf("too many road centers buried: %d/%d", buried, checked)
	}
}

func TestE2E_AtlasTowersOccludeSomeBehindRoads(t *testing.T) {
	requireAtlasFiles(t)
	city := &citycore.City{Slug: "testcity", Pop: 500}
	data, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img, ok := decodeMapPNG(t, data).(*image.RGBA)
	if !ok {
		t.Fatal("expected RGBA")
	}
	grid := buildCityGridForCity(&citycore.City{Slug: city.Slug, Pop: 500})
	occluded, checked := 0, 0
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] != cellRoad || x+y > 8 {
				continue
			}
			topX, topY := isoCell(x, y)
			c := img.RGBAAt(topX, topY+isoTileH/2)
			checked++
			if !looksLikeRoadSurface(c) {
				occluded++
			}
		}
	}
	if checked == 0 {
		t.Fatal("expected back-edge roads")
	}
	if occluded == 0 {
		t.Fatal("expected some behind-road centers occluded by towers")
	}
}

func TestE2E_AtlasBuildingsStayOffRoadCells(t *testing.T) {
	requireAtlasFiles(t)
	city := &citycore.City{Slug: "testcity", Pop: 500}
	grid := buildCityGridForCity(&citycore.City{Slug: city.Slug, Pop: 500})
	occ := lotOccupancy(city, grid)
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] != cellRoad {
				continue
			}
			if lot, ok := occ[[2]int{x, y}]; ok && lot.use == lotBuilding {
				t.Fatalf("road cell (%d,%d) has a building occupancy", x, y)
			}
		}
	}
}

func looksLikeRoadSurface(c color.RGBA) bool {
	if nearRGBA(c, roadColor, 48) {
		return true
	}
	maxC, minC := c.R, c.R
	for _, v := range []uint8{c.G, c.B} {
		if v > maxC {
			maxC = v
		}
		if v < minC {
			minC = v
		}
	}
	chroma := int(maxC) - int(minC)
	// Flat underlay / low-chroma asphalt.
	if maxC <= 160 && chroma <= 36 {
		return true
	}
	// Cool gray pavement.
	if minC >= 170 && chroma <= 40 && c.B+10 >= c.R {
		return true
	}
	// mcRoad dashed centerline (near-white).
	if minC >= 220 && chroma <= 24 {
		return true
	}
	// mcRoad warm asphalt / curb pixels (brown-gray, not vivid building paint).
	if maxC <= 200 && chroma <= 90 && int(c.R) >= int(c.B) && int(c.G) <= int(c.R)+20 {
		avg := (int(c.R) + int(c.G) + int(c.B)) / 3
		if avg <= 170 {
			return true
		}
	}
	return false
}

func nearRGBA(c, want color.RGBA, tol int) bool {
	dr := abs(int(c.R) - int(want.R))
	dg := abs(int(c.G) - int(want.G))
	db := abs(int(c.B) - int(want.B))
	return dr+dg+db <= tol
}

func TestPeonGrassClipKeepsSkyClear(t *testing.T) {
	grass := buildPeonGrass()
	img := image.NewRGBA(image.Rect(0, 0, mapWidth, mapHeight))
	sky := color.RGBA{R: 186, G: 220, B: 235, A: 255}
	for y := 0; y < mapHeight; y++ {
		for x := 0; x < mapWidth; x++ {
			img.SetRGBA(x, y, sky)
		}
	}

	eastX := peonIslandOrigin() + peonIslandExtent() - 1
	eastY := peonIslandOrigin()
	topX, topY := isoCell(eastX, eastY)
	footX, footY := topX, topY+isoTileH
	red := color.RGBA{R: 220, G: 40, B: 40, A: 255}
	for py := footY - 80; py < footY+20; py++ {
		for px := footX - 40; px < footX+80; px++ {
			if !image.Pt(px, py).In(img.Bounds()) || !peonPixelSupported(grass, px, py) {
				continue
			}
			img.SetRGBA(px, py, red)
		}
	}

	cx, cy := topX, topY+isoTileH/2
	if !peonPixelSupported(grass, cx, cy) {
		t.Fatal("green diamond center must be in the peon grass mask")
	}
	sampleX := footX + isoTileW + 24
	sampleY := footY - isoTileH/2
	if peonPixelSupported(grass, sampleX, sampleY) {
		t.Fatalf("sample (%d,%d) should be off the green top", sampleX, sampleY)
	}
	if got := img.RGBAAt(sampleX, sampleY); got != sky {
		t.Fatalf("sprite must not paint sky east of island: got %+v", got)
	}
}
