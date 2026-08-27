package render

import (
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestLotOccupancyFillsFromCenter(t *testing.T) {
	grid := buildCityGridForCity(&citycore.City{Slug: "zone-city", Pop: 500})
	empty := lotOccupancy(&citycore.City{Slug: "zone-city", Pop: 0}, grid)
	full := lotOccupancy(&citycore.City{Slug: "zone-city", Pop: 500}, grid)

	emptyBuildings, fullBuildings := 0, 0
	cx, cy := mapCols/2, mapRows/2
	var centerBuildings, outerBuildings int
	band := mapCols / 4
	ring := band * band // dist² inside the inner quarter-radius
	for _, lot := range empty {
		if lot.use == lotBuilding {
			emptyBuildings++
		}
	}
	for pos, lot := range full {
		dx, dy := pos[0]-cx, pos[1]-cy
		d := dx*dx + dy*dy
		if lot.use != lotBuilding {
			continue
		}
		fullBuildings++
		if d <= ring {
			centerBuildings++
		} else {
			outerBuildings++
		}
	}
	if emptyBuildings != 0 {
		t.Fatalf("pop=0 should leave lots empty, got %d buildings", emptyBuildings)
	}
	if fullBuildings <= emptyBuildings {
		t.Fatalf("pop=500 should place buildings, got %d", fullBuildings)
	}
	// Game.hx genMapPop concentrates density at center; outer ring may still
	// hold sparse roadless huts, but the center band should be denser per area.
	if centerBuildings == 0 {
		t.Fatal("pop=500 should place buildings near the center")
	}
	centerArea := float64(ring)
	if centerArea < 1 {
		centerArea = 1
	}
	outerArea := float64(mapCols*mapRows - ring)
	if outerArea < 1 {
		outerArea = 1
	}
	if float64(centerBuildings)/centerArea <= float64(outerBuildings)/outerArea {
		t.Fatalf("center should be denser per area: center=%d/%.0f outer=%d/%.0f", centerBuildings, centerArea, outerBuildings, outerArea)
	}
}

func TestMiniHutFootInsetsFromGameHxOrigin(t *testing.T) {
	got := make([][2]int, 4)
	for i := 0; i < 4; i++ {
		x, y := miniHutFoot(10, 20, i, true)
		got[i] = [2]int{x, y}
	}
	want := [][2]int{{11, 21}, {13, 21}, {11, 23}, {13, 23}}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("roadless hut foot %d = %v, want %v (Game.hx 2×2 + inset)", i, got[i], want[i])
		}
	}
	ax, ay := miniHutFoot(10, 20, 0, false)
	if ax != 10 || ay != 20 {
		t.Fatalf("arterial hut foot 0 = (%d,%d), want Game.hx origin (10,20)", ax, ay)
	}
}

func TestLotOccupancyRoadlessSpreadsBuildings(t *testing.T) {
	city := &citycore.City{Slug: "testcity", Pop: 8}
	grid := buildCityGridForCity(city)
	occ := lotOccupancy(city, grid)

	var buildings []lotCell
	plates := make(map[[2]int]int)
	minis := make(map[[2]int]int) // plate*4+mini → count via (sx,sy,qi) packed
	atCenter := 0
	for _, lot := range occ {
		if lot.use != lotBuilding {
			continue
		}
		pop := city.Pop.Int()
		if !inPlateIsland(pop, lot.x, lot.y) {
			t.Fatalf("roadless building outside island at (%d,%d)", lot.x, lot.y)
		}
		if !isGameHxMiniFoot(lot.x, lot.y) {
			t.Fatalf("roadless building at (%d,%d) is not a genMiniSquare foot", lot.x, lot.y)
		}
		if !grassTopCell(pop, lot.x, lot.y) {
			t.Fatalf("roadless building at (%d,%d) is on soil rim, want grass top", lot.x, lot.y)
		}
		px, py := plateOfFor(pop, lot.x, lot.y)
		plates[[2]int{px, py}]++
		o := plateIslandOrigin(pop)
		lx := lot.x - o - px*groundBlock
		ly := lot.y - o - py*groundBlock
		if lx == 4 && ly == 4 {
			atCenter++
		}
		qi := miniQuadrantIndex(lx, ly)
		minis[[2]int{px*10 + py, qi}]++
		buildings = append(buildings, lot)
	}
	if len(buildings) == 0 {
		t.Fatal("pop=8 roadless should place some buildings on the island")
	}
	if atCenter != 0 {
		t.Fatalf("roadless buildings at plate center (4,4)=%d, want 0 (Townzzy quadrants)", atCenter)
	}
	for mini, n := range minis {
		if n > 4 {
			t.Fatalf("mini %v has %d buildings; Game.hx max is 4", mini, n)
		}
	}

	again := lotOccupancy(city, grid)
	for pos, lot := range occ {
		got := again[pos]
		if got.use != lot.use || got.x != lot.x || got.y != lot.y {
			t.Fatalf("roadless occupancy must be deterministic at %v: %+v vs %+v", pos, lot, got)
		}
	}
}

func TestRoadlessBuildingsCapAtDallePlates(t *testing.T) {
	city := &citycore.City{Slug: "roadless-dense", Pop: 39}
	occ := lotOccupancy(city, buildCityGridForCity(city))
	n := 0
	plates := make(map[[2]int]int)
	pop := city.Pop.Int()
	for _, lot := range occ {
		if lot.use != lotBuilding {
			continue
		}
		if !inPlateIsland(pop, lot.x, lot.y) {
			t.Fatalf("roadless building outside island at (%d,%d)", lot.x, lot.y)
		}
		if !isGameHxMiniFoot(lot.x, lot.y) {
			t.Fatalf("building at (%d,%d) not on genMiniSquare foot", lot.x, lot.y)
		}
		px, py := plateOfFor(pop, lot.x, lot.y)
		plates[[2]int{px, py}]++
		n++
	}
	if n == 0 {
		t.Fatal("roadless pop=39 should place buildings")
	}
	// Townzzy: up to 4 minis × 4 huts = 16 per dalle plate.
	maxPerPlate := 16
	for plate, c := range plates {
		if c > maxPerPlate {
			t.Fatalf("plate %v has %d buildings; max %d", plate, c, maxPerPlate)
		}
	}
	// Continuous with pop=40 (no fillRate cliff).
	occ40 := lotOccupancy(&citycore.City{Slug: "roadless-dense", Pop: 40}, buildCityGridForCity(city))
	n40 := 0
	for _, lot := range occ40 {
		if lot.use == lotBuilding {
			n40++
		}
	}
	if n40 > n*3 {
		t.Fatalf("pop 39→40 must not cliff: pop39=%d pop40=%d", n, n40)
	}
}

func plateOfFor(pop, x, y int) (plateX, plateY int) {
	o := plateIslandOrigin(pop)
	return (x - o) / groundBlock, (y - o) / groundBlock
}

// isGameHxMiniFoot reports whether (x,y) is a genMiniSquare / genSquare foot
// (Game.hx or roadless-inset 2×2, mini origin for POP_BIG, or square origin for HUGE).
func isGameHxMiniFoot(x, y int) bool {
	sx, sy := squareOf(x, y)
	lx := x - sx*squareSide
	ly := y - sy*squareSide
	if lx < 0 || ly < 0 || lx >= squareSide || ly >= squareSide {
		return false
	}
	if lx == 0 && ly == 0 {
		return true // POP_HUGE square bat / NW mini origin
	}
	// Gap seam between minis (lx/ly == 4) is never a house cell.
	if lx == 4 || ly == 4 {
		return false
	}
	ox, oy := lx, ly
	if lx >= 5 {
		ox = lx - 5
	}
	if ly >= 5 {
		oy = ly - 5
	}
	if ox > 3 || oy > 3 {
		return false
	}
	if ox == 0 && oy == 0 {
		return true // POP_BIG mini origin
	}
	for i := 0; i < 4; i++ {
		gx, gy := (i%2)*2, (i/2)*2
		if ox == gx && oy == gy {
			return true // Game.hx 2×2 (arterial)
		}
		hx, hy := miniHutFoot(0, 0, i, true)
		if ox == hx && oy == hy {
			return true // roadless inset (#118)
		}
	}
	return false
}

func isInsetMiniHutFoot(x, y int) bool {
	sx, sy := squareOf(x, y)
	lx := x - sx*squareSide
	ly := y - sy*squareSide
	if lx == 4 || ly == 4 {
		return false
	}
	ox, oy := lx, ly
	if lx >= 5 {
		ox = lx - 5
	}
	if ly >= 5 {
		oy = ly - 5
	}
	for i := 0; i < 4; i++ {
		hx, hy := miniHutFoot(0, 0, i, true)
		if ox == hx && oy == hy {
			return true
		}
	}
	return false
}

func miniQuadrantIndex(lx, ly int) int {
	qi := 0
	if lx >= 5 {
		qi++
	}
	if ly >= 5 {
		qi += 2
	}
	return qi
}

func TestRoadlessHousesStayInQuadrantsNotPlateCenter(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	for _, slug := range []string{"testcity2", "testcity3"} {
		city := &citycore.City{Slug: citycore.Slug(slug), Pop: 40, Ind: 130, Com: 150, Env: 110, Sec: 100}
		ctx := newMapRenderCtx(city, atlas)
		occ := ctx.occupancy
		pop := city.Pop.Int()
		n, atCenter, multiPlate, offGrass := 0, 0, 0, 0
		plates := map[[2]int]int{}
		for _, lot := range occ {
			if lot.use != lotBuilding {
				continue
			}
			n++
			if !isGameHxMiniFoot(lot.x, lot.y) {
				t.Fatalf("%s building (%d,%d) not a mini foot", slug, lot.x, lot.y)
			}
			if !grassTopCell(pop, lot.x, lot.y) {
				offGrass++
			}
			px, py := plateOfFor(pop, lot.x, lot.y)
			plates[[2]int{px, py}]++
			o := plateIslandOrigin(pop)
			lx := lot.x - o - px*groundBlock
			ly := lot.y - o - py*groundBlock
			if lx == 4 && ly == 4 {
				atCenter++
			}
		}
		for _, c := range plates {
			if c > 1 {
				multiPlate++
			}
		}
		if n == 0 {
			t.Fatalf("%s: expected buildings", slug)
		}
		if atCenter != 0 {
			t.Fatalf("%s: %d buildings at plate center, want 0", slug, atCenter)
		}
		if multiPlate == 0 {
			t.Fatalf("%s: expected at least one plate with multiple houses (Townzzy)", slug)
		}
		if offGrass != 0 {
			t.Fatalf("%s: %d buildings on soil rim, want 0 (grass tops only)", slug, offGrass)
		}
		for _, s := range collectFarmStamps(ctx) {
			lot := occ[[2]int{s.fx, s.fy}]
			if lot.use == lotBuilding {
				t.Fatalf("%s: farm stamp foot (%d,%d) is also a building", slug, s.fx, s.fy)
			}
		}
	}
}

func TestPeonHutFeetStayInsideMiniDiamond(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	for _, slug := range []string{"testcity1", "testcity5"} {
		city := &citycore.City{Slug: citycore.Slug(slug), Pop: 40, Ind: 130, Com: 150, Env: 110, Sec: 100}
		ctx := newMapRenderCtx(city, atlas)
		n, onSeam, onOrigin := 0, 0, 0
		for _, lot := range ctx.occupancy {
			if lot.use != lotBuilding {
				continue
			}
			n++
			sx, sy := squareOf(lot.x, lot.y)
			lx := lot.x - sx*squareSide
			ly := lot.y - sy*squareSide
			if lx == 0 || ly == 0 || lx == 4 || ly == 4 || lx == squareSide-1 || ly == squareSide-1 {
				onSeam++
			}
			ox, oy := lx, ly
			if lx >= 5 {
				ox = lx - 5
			}
			if ly >= 5 {
				oy = ly - 5
			}
			if ox == 0 && oy == 0 {
				onOrigin++
			}
			if !isInsetMiniHutFoot(lot.x, lot.y) {
				t.Fatalf("%s building (%d,%d) local (%d,%d) is not an inset mini hut foot", slug, lot.x, lot.y, ox, oy)
			}
		}
		if n == 0 {
			t.Fatalf("%s: expected peon buildings", slug)
		}
		if onSeam != 0 {
			t.Fatalf("%s: %d buildings on square/mini seams, want 0 (#118)", slug, onSeam)
		}
		if onOrigin != 0 {
			t.Fatalf("%s: %d buildings on mini origin, want inset feet", slug, onOrigin)
		}
	}
}

func TestLotOccupancyAddsParksWithEnv(t *testing.T) {
	grid := buildCityGridForCity(&citycore.City{Slug: "park-city", Pop: 500})
	plain := lotOccupancy(&citycore.City{Slug: "park-city", Pop: 500, Env: 0}, grid)
	green := lotOccupancy(&citycore.City{Slug: "park-city", Pop: 500, Env: 400}, grid)
	plainParks, greenParks := 0, 0
	for _, lot := range plain {
		if lot.use == lotPark {
			plainParks++
		}
	}
	for _, lot := range green {
		if lot.use == lotPark {
			greenParks++
		}
	}
	if greenParks <= plainParks {
		t.Fatalf("expected env to add parks, got %d vs %d", greenParks, plainParks)
	}
	// big_city greenery: env=400 should plant dozens of trees (dedicated + scatter).
	if greenParks < 40 {
		t.Fatalf("env=400 should place many trees, got %d", greenParks)
	}
	if plainParks != 0 {
		t.Fatalf("env=0 must not scatter trees, got %d parks", plainParks)
	}
}

func TestLotOccupancyScattersRoadsideTrees(t *testing.T) {
	grid := buildCityGridForCity(&citycore.City{Slug: "curb-trees", Pop: 500})
	occ := lotOccupancy(&citycore.City{Slug: "curb-trees", Pop: 500, Env: 400}, grid)
	curbParks := 0
	for pos, lot := range occ {
		if lot.use == lotPark && lotTouchesRoad(grid, pos[0], pos[1]) {
			curbParks++
		}
	}
	if curbParks < 20 {
		t.Fatalf("expected roadside tree scatter, got %d curb parks", curbParks)
	}
}

func TestRoadlessGrassTopCellInsetsFromRim(t *testing.T) {
	pop := 2
	o := plateIslandOrigin(pop)
	last := o + plateIslandExtent(pop) - 1
	if grassTopCell(pop, o, o) {
		t.Fatal("island corner must not count as grass top")
	}
	if grassTopCell(pop, last, last) {
		t.Fatal("south tip must not count as grass top")
	}
	mid := (o + last) / 2
	if !grassTopCell(pop, mid, mid) {
		t.Fatal("island center must count as grass top")
	}
	if grassTopCell(pop, o+1, mid) {
		t.Fatalf("rim inset 1 at x=%d should still be soil ledge", o+1)
	}
	if !grassTopCell(pop, o+grassRimInset, mid) {
		t.Fatalf("rim inset %d should be grass top", grassRimInset)
	}
}

func TestRoadlessParksStayOnGrassTops(t *testing.T) {
	city := &citycore.City{Slug: "testcity", Pop: 2, Env: 7, Ind: 1, Com: 1, Sec: 1}
	occ := lotOccupancy(city, buildCityGridForCity(city))
	pop := city.Pop.Int()
	parks := 0
	for _, lot := range occ {
		if lot.use != lotPark {
			continue
		}
		parks++
		if !grassTopCell(pop, lot.x, lot.y) {
			t.Fatalf("roadless park at (%d,%d) sits on soil rim", lot.x, lot.y)
		}
	}
	if parks == 0 {
		t.Fatal("env=7 roadless should still place some trees on grass tops")
	}
}

func TestArterialParksStayOnGrassTops(t *testing.T) {
	// pop>=80 enables roads; trees must still stay off the plate soil rim.
	city := &citycore.City{Slug: "testcity", Pop: 80, Env: 79, Ind: 1, Com: 1, Sec: 1}
	if !arterialsEnabled(city) {
		t.Fatal("pop=80 must enable arterials")
	}
	occ := lotOccupancy(city, buildCityGridForCity(city))
	pop := city.Pop.Int()
	parks := 0
	for _, lot := range occ {
		if lot.use != lotPark {
			continue
		}
		parks++
		if !grassTopCell(pop, lot.x, lot.y) {
			t.Fatalf("arterial park at (%d,%d) sits on soil rim", lot.x, lot.y)
		}
	}
	if parks == 0 {
		t.Fatal("env=79 arterial should still place some trees on grass tops")
	}

	roadless := &citycore.City{Slug: "testcity", Pop: 70, Env: 79, Ind: 1, Com: 1, Sec: 1}
	if arterialsEnabled(roadless) {
		t.Fatal("pop=70 must stay roadless (no arterials)")
	}
	roadlessOcc := lotOccupancy(roadless, buildCityGridForCity(roadless))
	for _, lot := range roadlessOcc {
		if lot.use != lotPark {
			continue
		}
		if !grassTopCell(roadless.Pop.Int(), lot.x, lot.y) {
			t.Fatalf("pop=70 park at (%d,%d) sits on soil rim", lot.x, lot.y)
		}
	}
}

func TestIndustrialZonePlacesBuildings(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "ind-visible", Pop: 500, Ind: 50}
	grid := buildCityGridForCity(&citycore.City{Slug: city.Slug, Pop: 500})
	occ := lotOccupancy(city, grid)
	indBuild := 0
	for pos, lot := range occ {
		if lot.use != lotBuilding || lot.tag != TagIndustrial {
			continue
		}
		if lotTouchesRoad(grid, pos[0], pos[1]) {
			t.Fatalf("industrial building on curb lot (%d,%d)", pos[0], pos[1])
		}
		key := atlas.PickBuildingKeyForLot(city, lot.tag, pos[0], pos[1], hashCell(city.Slug.String(), pos[0], pos[1]))
		if key == "" || !frameBelongsToTag(key, atlas.BasesForTag(TagIndustrial)) {
			t.Fatalf("expected industrial sprite at (%d,%d), got %q", pos[0], pos[1], key)
		}
		indBuild++
	}
	if indBuild == 0 {
		t.Fatal("Ind>0 must place at least one industrial building on a non-curb lot")
	}
}

func TestZoneTagUsesSectors(t *testing.T) {
	// pop=0 → 4×4 island; zone rings are relative to that island, not the field.
	cityCom := &citycore.City{Pop: 0, Com: 10}
	cx, cy := plateIslandCenter(0)
	com := zoneTag(cityCom, cx, cy)
	if com != TagCommercial {
		t.Fatalf("center with com should be commercial, got %s", com)
	}
	o := plateIslandOrigin(0)
	e := plateIslandExtent(0)
	ind := zoneTag(&citycore.City{Pop: 0, Ind: 10}, o, o+5)
	if ind != TagIndustrial {
		t.Fatalf("island edge with ind should be industrial, got %s", ind)
	}
	// Outer third of the island is industrial; first cell past that band is residential.
	band := e / 3
	if band < 2 {
		band = 2
	}
	ring2 := zoneTag(&citycore.City{Pop: 0, Ind: 10}, o+2, cy)
	if ring2 != TagIndustrial {
		t.Fatalf("x=%d with ind should be industrial, got %s", o+2, ring2)
	}
	midOuter := zoneTag(&citycore.City{Pop: 0, Ind: 10}, o+band, cy)
	if midOuter != TagResidential {
		t.Fatalf("x=%d with ind should stay residential, got %s", o+band, midOuter)
	}
	res := zoneTag(&citycore.City{Pop: 0}, cx, cy)
	if res != TagResidential {
		t.Fatalf("default should be residential, got %s", res)
	}
}

func TestSectorZonesPickTaggedSprites(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	city := &citycore.City{Slug: "zone-pick", Pop: 500, Com: 50, Ind: 50}
	cx, cy := plateIslandCenter(city.Pop.Int())
	o := plateIslandOrigin(city.Pop.Int())
	comTag := zoneTag(city, cx, cy)
	indTag := zoneTag(city, o, o+5)
	comKey := atlas.PickBuildingKeyForTag(comTag, 1)
	indKey := atlas.PickBuildingKeyForTag(indTag, 1)
	if comKey == "" || indKey == "" {
		t.Fatalf("expected tagged picks, com=%q ind=%q", comKey, indKey)
	}
	if comKey == indKey {
		t.Fatalf("commercial and industrial picks should differ, both %q", comKey)
	}
	if indTag == TagIndustrial && !frameBelongsToTag(indKey, atlas.BasesForTag(TagIndustrial)) {
		t.Fatalf("industrial pick %q not in industrial bases", indKey)
	}
	// Commercial multi-tile clips are filtered for 1-lot maps; fallback may be residential.
	if len(atlas.BasesForTag(TagCommercial)) > 0 {
		if !frameBelongsToTag(comKey, atlas.BasesForTag(TagCommercial)) {
			t.Fatalf("commercial pick %q not in commercial bases", comKey)
		}
	} else if !frameBelongsToTag(comKey, atlas.BasesForTag(TagResidential)) {
		t.Fatalf("commercial fallback %q not in residential bases", comKey)
	}
}

func frameBelongsToTag(frameKey string, bases []string) bool {
	folder := spriteFolderBase(frameKey)
	for _, base := range bases {
		if spriteFolderBase(base) == folder {
			return true
		}
	}
	return false
}

func TestBuildingFrameColorKey(t *testing.T) {
	atlas := &Atlas{Frames: map[string]frameRect{
		"sprites/House_a_v00.png": {W: 1, H: 1},
		"sprites/House_a_v02.png": {W: 1, H: 1},
	}}
	// seed high bits → color 2
	got := buildingFrameColorKey(atlas, "sprites/House_a", 2<<16)
	if got != "sprites/House_a_v02.png" {
		t.Fatalf("got %q, want v02", got)
	}
	// missing v01 → fallback v00
	got = buildingFrameColorKey(atlas, "sprites/House_a", 1<<16)
	if got != "sprites/House_a_v00.png" {
		t.Fatalf("missing variant fallback: got %q", got)
	}
	// no Frames → always v00
	got = buildingFrameColorKey(&Atlas{}, "sprites/House_a", 3<<16)
	if got != "sprites/House_a_v00.png" {
		t.Fatalf("empty atlas: got %q", got)
	}
}

func TestEmptyZoneTagFallsBackToResidential(t *testing.T) {
	atlas := &Atlas{
		BasesByTag: map[string][]string{
			TagResidential: {"sprites/House_a"},
			TagIndustrial:  {},
		},
	}
	if got := atlas.PickKeyForTag(TagIndustrial, 1); got != "" {
		t.Fatalf("expected empty industrial catalog to return \"\", got %q", got)
	}
	got := atlas.PickBuildingKeyForTag(TagIndustrial, 1)
	want := "sprites/House_a_v00.png"
	if got != want {
		t.Fatalf("empty industrial should fall back to residential, got %q want %q", got, want)
	}
}

func TestEmptyResidentialFallsBackToEmptyKey(t *testing.T) {
	atlas := &Atlas{
		BasesByTag: map[string][]string{
			TagResidential: {},
			TagIndustrial:  {},
			TagCommercial:  {"sprites/Shop_a"},
		},
	}
	if got := atlas.PickBuildingKeyForTag(TagIndustrial, 7); got != "" {
		t.Fatalf("empty industrial+residential should leave key empty for rectangle fallback, got %q", got)
	}
	if got := atlas.PickBuildingKeyForTag(TagResidential, 3); got != "" {
		t.Fatalf("empty residential should leave key empty, got %q", got)
	}
	// Commercial still resolves when its catalog is non-empty.
	if got := atlas.PickBuildingKeyForTag(TagCommercial, 2); got != "sprites/Shop_a_v00.png" {
		t.Fatalf("commercial with bases should pick its own frame, got %q", got)
	}
}

func TestMapEntityTagChangesWithEnv(t *testing.T) {
	requireAtlasFiles(t)
	a, err := MapEntityTag(&citycore.City{Slug: "etag-env", Pop: 100, Env: 0})
	if err != nil {
		t.Fatal(err)
	}
	b, err := MapEntityTag(&citycore.City{Slug: "etag-env", Pop: 100, Env: 400})
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("expected etag to change with env")
	}
}
