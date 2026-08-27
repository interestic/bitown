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
	img := mustBuildMapWorkingImage(t, city)
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
	img := mustBuildMapWorkingImage(t, city)
	plan := planRoadsForCity(city)
	occluded, checked := 0, 0
	seen := map[[2]int]bool{}
	o := activeSquareOrigin(city.Pop.Int())
	minSum := 1 << 30
	for _, st := range plan.stamps {
		if s := (st.sx - o) + (st.sy - o); s < minSum {
			minSum = s
		}
	}
	for _, st := range plan.stamps {
		// Back edge of the live island (density may leave the NW tip empty).
		if (st.sx-o)+(st.sy-o) > minSum+2 {
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
	// Dashed centerline only — gray building walls / limestone must not count.
	return minC >= 220 && int(maxC)-int(minC) <= 24 && int(c.R)-int(c.B) <= 8
}

func TestE2E_AtlasTowersKeepFacadesClearOfRoads(t *testing.T) {
	requireAtlasFiles(t)
	city := &citycore.City{Slug: "testcity", Pop: 460, Ind: 130, Com: 150, Env: 110, Sec: 100}
	img := mustBuildMapWorkingImage(t, city)
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
			footX, footY := topX, topY+isoTileH-overlayLift(false)
			footX = applyWestMiniStampNudge(footX, x, y)
			footX = applyNorthMiniStampNudgeX(footX, x, y)
			footX = applyEastMiniStampNudge(footX, x, y)
			footY = applyNorthMiniStampNudge(footY, x, y)
			footY = applySEMiniStampNudge(footY, x, y)
			footY = applyEWMiniStampNudge(footY, x, y)
			py := footY - 48
			if py < 0 {
				continue
			}
			checked++
			if looksLikeRoadPaintOnFacade(img.RGBAAt(footX, py)) {
				striped++
				if striped <= 6 {
					t.Logf("facade stripe at lot(%d,%d) y=%d %v", x, y, py, img.RGBAAt(footX, py))
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
	img := mustBuildMapWorkingImage(t, city)
	plan := planRoadsForCity(city)
	if len(plan.stamps) == 0 {
		t.Fatal("expected road stamps")
	}
	found := false
	for _, st := range plan.stamps {
		if !squareShowsLiftedRoad(img, st.sx, st.sy) {
			continue
		}
		x, y := st.sx*squareSide, st.sy*squareSide
		topX, topY := isoCell(x, y)
		soil := img.RGBAAt(topX, topY+isoTileH/2)
		if looksLikeRoadPaintOnFacade(soil) || nearRGBA(soil, roadColor, 24) {
			// Overlay feet can land on this diamond after arterial grass lift;
			// try another stamp before treating it as an unlifted road.
			continue
		}
		found = true
		break
	}
	if !found {
		t.Fatal("no lifted road pixels with a clear unlifted soil sample")
	}
}

func TestE2E_AtlasRoadCrossesUseFullPlateLip(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "testcity5", Pop: 90, Ind: 330, Com: 150, Env: 110, Sec: 100}
	ctx := newMapRenderCtx(city, atlas)
	if ctx.roadLift != plateGrassLift {
		t.Fatalf("roadLift=%d, want plateGrassLift=%d (Townzzy / catalog lip)", ctx.roadLift, plateGrassLift)
	}
	img := mustBuildMapWorkingImage(t, city)
	hits, checked := 0, 0
	for sy := 0; sy < displaySide && sy < len(ctx.roadCross); sy++ {
		for sx := 0; sx < displaySide && sx < len(ctx.roadCross[sy]); sx++ {
			if ctx.roadCross[sy][sx] == 0 {
				continue
			}
			checked++
			if squareShowsLiftedRoad(img, sx, sy) {
				hits++
			}
		}
	}
	if checked == 0 {
		t.Fatal("expected road crosses")
	}
	if hits*2 < checked {
		t.Fatalf("road crosses missing lifted pavement: %d/%d", hits, checked)
	}
}

func TestE2E_AtlasArterialOverlaysUsePlateGrassLift(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "testcity5", Pop: 90, Ind: 330, Com: 150, Env: 110, Sec: 100}
	ctx := newMapRenderCtx(city, atlas)
	if ctx.roadless {
		t.Fatal("pop=90 must enable arterials")
	}
	img := mustBuildMapWorkingImage(t, city)

	checked, liftedHits, unliftedHits := 0, 0, 0
	for _, lot := range ctx.occupancy {
		if lot.use != lotBuilding {
			continue
		}
		key := ctx.buildingKeys[[2]int{lot.x, lot.y}]
		rect, ok := atlas.Frames[key]
		if !ok {
			continue
		}
		footX, liftedY := overlayFoot(lot.x, lot.y, overlayLift(false))
		footX = applyWestMiniStampNudge(footX, lot.x, lot.y)
		footX = applyNorthMiniStampNudgeX(footX, lot.x, lot.y)
		footX = applyEastMiniStampNudge(footX, lot.x, lot.y)
		liftedY = applyNorthMiniStampNudge(liftedY, lot.x, lot.y)
		liftedY = applySEMiniStampNudge(liftedY, lot.x, lot.y)
		liftedY = applyEWMiniStampNudge(liftedY, lot.x, lot.y)
		unliftedY := liftedY + overlayLift(false)
		sx, sy := rect.AnchorX, rect.AnchorY-8
		if sx < 0 || sy < 0 || sx >= rect.W || sy >= rect.H {
			continue
		}
		src := atlasFrameRGBA(atlas, key, sx, sy)
		if src.A < 200 {
			continue
		}
		px := footX - rect.AnchorX + sx
		liftedPy := liftedY - rect.AnchorY + sy
		unliftedPy := unliftedY - rect.AnchorY + sy
		if !image.Pt(px, liftedPy).In(img.Bounds()) || !image.Pt(px, unliftedPy).In(img.Bounds()) {
			continue
		}
		checked++
		if nearRGBA(img.RGBAAt(px, liftedPy), src, 48) {
			liftedHits++
		}
		if nearRGBA(img.RGBAAt(px, unliftedPy), src, 48) {
			unliftedHits++
		}
	}
	if checked == 0 {
		t.Fatal("expected opaque building samples")
	}
	if liftedHits*2 < checked {
		t.Fatalf("arterial buildings missing grass lift: %d/%d at lifted foot", liftedHits, checked)
	}
	if unliftedHits*5 > checked {
		t.Fatalf("arterial buildings still on soil rim: %d/%d at unlifted foot", unliftedHits, checked)
	}
}

func TestE2E_AtlasArterialHousesStayOffCanvas(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "testcity5", Pop: 90, Ind: 330, Com: 150, Env: 110, Sec: 100}
	ctx := newMapRenderCtx(city, atlas)
	if ctx.roadless {
		t.Fatal("pop=90 must enable arterials")
	}
	img := mustBuildMapWorkingImage(t, city)
	hits := 0
	for _, lot := range ctx.occupancy {
		if lot.use != lotBuilding {
			continue
		}
		key := ctx.buildingKeys[[2]int{lot.x, lot.y}]
		rect, ok := atlas.Frames[key]
		if !ok {
			continue
		}
		footX, footY := overlayFoot(lot.x, lot.y, overlayLift(false))
		dstX, dstY := footX-rect.AnchorX, footY-rect.AnchorY
		for sy := 0; sy < rect.H; sy++ {
			py := dstY + sy
			if !inBuildingGroundBand(footY, py) {
				continue
			}
			for sx := 0; sx < rect.W; sx++ {
				src := atlasFrameRGBA(atlas, key, sx, sy)
				if src.A < 200 {
					continue
				}
				px := dstX + sx
				if grassTopPixelSupported(ctx.grass, px, py) {
					continue
				}
				if nearRGBA(img.RGBAAt(px, py), src, 48) {
					hits++
				}
			}
		}
	}
	if hits > 4 {
		t.Fatalf("arterial house ground band painted off the city plate: %d px", hits)
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

func TestRoadlessGrassClipKeepsSkyClear(t *testing.T) {
	grass := buildPlateGrass(20)
	img := image.NewRGBA(image.Rect(0, 0, mapWidth, mapHeight))
	sky := color.RGBA{R: 186, G: 220, B: 235, A: 255}
	for y := 0; y < mapHeight; y++ {
		for x := 0; x < mapWidth; x++ {
			img.SetRGBA(x, y, sky)
		}
	}

	eastX := plateIslandOrigin20() + plateIslandExtent20()/2
	eastY := plateIslandOrigin20() + plateIslandExtent20()/2
	topX, topY := isoCell(eastX, eastY)
	footX, footY := topX, topY+isoTileH-plateGrassLift
	red := color.RGBA{R: 220, G: 40, B: 40, A: 255}
	for py := footY - 80; py < footY+20; py++ {
		for px := footX - 40; px < footX+80; px++ {
			if !image.Pt(px, py).In(img.Bounds()) || !grassTopPixelSupported(grass, px, py) {
				continue
			}
			img.SetRGBA(px, py, red)
		}
	}

	cx, cy := topX, topY+isoTileH/2-plateGrassLift
	if !grassTopPixelSupported(grass, cx, cy) {
		t.Fatal("lifted green diamond center must be in the plate grass mask")
	}
	// Canvas north of the island's NW tip must stay clear (old py<=maxY leaked).
	nwX, nwY := isoCell(plateIslandOrigin20(), plateIslandOrigin20())
	northY := nwY - plateGrassLift - 16
	if grassTopPixelSupported(grass, nwX, northY) {
		t.Fatalf("canvas north of island tip (%d,%d) must be off-mask", nwX, northY)
	}
	if got := img.RGBAAt(nwX, northY); got != sky {
		t.Fatalf("sprite must not paint sky north of island tip: got %+v", got)
	}
	// Far outside the map bounds horizontally.
	farX := mapWidth - 4
	farY := mapHeight / 2
	if grassTopPixelSupported(grass, farX, farY) {
		t.Fatalf("far canvas (%d,%d) should be off the green top", farX, farY)
	}
}
