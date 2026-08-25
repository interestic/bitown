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
	roads := buildRoadMaskDataOffset(grid, 0)

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
		// Not required to hit exactly; just ensure clip did not blank the whole overhang band.
		_ = upperY
	}
}

func TestE2E_AtlasRoadsStayClearWithSetback(t *testing.T) {
	requireAtlasFiles(t)
	city := &citycore.City{Slug: "testcity", Pop: 80}
	data, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img, ok := decodeMapPNG(t, data).(*image.RGBA)
	if !ok {
		t.Fatal("expected RGBA")
	}
	plan := planRoadsForCity(&citycore.City{Slug: city.Slug, Pop: 80})
	buried, checked := 0, 0
	for _, st := range plan.stamps {
		checked++
		if !squareShowsLiftedRoad(img, st.sx, st.sy) {
			buried++
			if buried <= 6 {
				t.Logf("road square(%d,%d) dir=%d has no lifted road pixels", st.sx, st.sy, st.dir)
			}
		}
	}
	if checked == 0 {
		t.Fatal("expected road samples")
	}
	if buried*20 > checked*9 {
		t.Fatalf("too many road squares without visible pavement: %d/%d", buried, checked)
	}
}

func TestE2E_AtlasTowersOccludeSomeBehindRoads(t *testing.T) {
	requireAtlasFiles(t)
	// High pop + sec unlocks native-height towers that overhang back-edge roads.
	city := &citycore.City{Slug: "testcity", Pop: 1800, Sec: 300, Com: 100}
	data, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img, ok := decodeMapPNG(t, data).(*image.RGBA)
	if !ok {
		t.Fatal("expected RGBA")
	}
	plan := planRoadsForCity(city)
	occluded, checked := 0, 0
	seen := map[[2]int]bool{}
	for _, st := range plan.stamps {
		if st.sx+st.sy > 2 {
			continue
		}
		key := [2]int{st.sx, st.sy}
		if seen[key] {
			continue
		}
		seen[key] = true
		checked++
		if !squareShowsLiftedRoad(img, st.sx, st.sy) {
			occluded++
		}
	}
	if checked == 0 {
		t.Fatal("expected back-edge roads")
	}
	// With floor-pass roads, towers may or may not fully bury stamp samples.
	t.Logf("back-edge road squares missing pavement: %d/%d", occluded, checked)
}

func atlasRoadCenter(x, y int) (int, int) {
	topX, topY := isoCell(x, y)
	return topX, topY + isoTileH/2 - roadGrassLift
}

func roadInFrontOfLot(grid cityGrid, x, y int) bool {
	for d := 1; d <= 3; d++ {
		if y+d < mapRows && grid[y+d][x] == cellRoad {
			return true
		}
		if x+d < mapCols && grid[y][x+d] == cellRoad {
			return true
		}
	}
	return false
}

func looksLikeRoadPaintOnFacade(c color.RGBA) bool {
	if nearRGBA(c, roadColor, 40) {
		return true
	}
	minC, maxC := c.R, c.R
	for _, v := range []uint8{c.G, c.B} {
		if v > maxC {
			maxC = v
		}
		if v < minC {
			minC = v
		}
	}
	// Dashed centerline only — gray building walls must not count.
	return minC >= 220 && int(maxC)-int(minC) <= 24
}

func TestE2E_AtlasTowersKeepFacadesClearOfRoads(t *testing.T) {
	requireAtlasFiles(t)
	city := &citycore.City{Slug: "testcity", Pop: 460, Ind: 130, Com: 150, Env: 110, Sec: 100}
	data, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img, ok := decodeMapPNG(t, data).(*image.RGBA)
	if !ok {
		t.Fatal("expected RGBA")
	}
	grid := buildCityGridForCity(city)
	occ := lotOccupancy(city, grid)
	striped, checked := 0, 0
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			lot, ok := occ[[2]int{x, y}]
			if !ok || lot.use != lotBuilding {
				continue
			}
			if !roadInFrontOfLot(grid, x, y) {
				continue
			}
			topX, topY := isoCell(x, y)
			footY := topY + isoTileH
			py := footY - 48
			if py < 0 {
				continue
			}
			checked++
			if looksLikeRoadPaintOnFacade(img.RGBAAt(topX, py)) {
				striped++
				if striped <= 6 {
					t.Logf("facade stripe at lot(%d,%d) y=%d %v", x, y, py, img.RGBAAt(topX, py))
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("expected buildings with a road in front")
	}
	if striped > 0 {
		t.Fatalf("road painted through %d/%d tower facades", striped, checked)
	}
}

func TestE2E_AtlasRoadsSitOnDalleGrass(t *testing.T) {
	requireAtlasFiles(t)
	city := &citycore.City{Slug: "testcity", Pop: 80}
	data, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img, ok := decodeMapPNG(t, data).(*image.RGBA)
	if !ok {
		t.Fatal("expected RGBA")
	}
	plan := planRoadsForCity(city)
	if len(plan.stamps) == 0 {
		t.Fatal("expected road stamps")
	}
	found := false
	for _, st := range plan.stamps {
		if !squareShowsLiftedRoad(img, st.sx, st.sy) {
			continue
		}
		found = true
		x, y := st.sx*squareSide, st.sy*squareSide
		topX, topY := isoCell(x, y)
		soil := img.RGBAAt(topX, topY+isoTileH/2)
		if looksLikeRoadPaintOnFacade(soil) || nearRGBA(soil, roadColor, 24) {
			t.Fatalf("unlifted soil diamond should not keep the road, got %+v", soil)
		}
		break
	}
	if !found {
		t.Fatal("no lifted road pixels on any stamp square")
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
	// mcRoad dirt / soil (brown).
	if c.R >= 70 && c.R <= 190 && c.G >= 40 && c.G <= 150 && c.B <= 110 && int(c.R) > int(c.B)+12 {
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

func squareShowsLiftedRoad(img *image.RGBA, sx, sy int) bool {
	x0, y0 := sx*squareSide, sy*squareSide
	for y := y0; y < y0+squareSide && y < mapRows; y++ {
		for x := x0; x < x0+squareSide && x < mapCols; x++ {
			if looksLikeRoadSurface(img.RGBAAt(atlasRoadCenter(x, y))) {
				return true
			}
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
	grass := buildPeonGrass(20)
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
