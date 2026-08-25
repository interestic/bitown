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
	// hold sparse peon huts, but the center band should be denser per area.
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

func TestLotOccupancyPeonSpreadsBuildings(t *testing.T) {
	city := &citycore.City{Slug: "testcity", Pop: 8}
	grid := buildCityGridForCity(city)
	occ := lotOccupancy(city, grid)

	var buildings []lotCell
	plates := make(map[[2]int]int)
	for _, lot := range occ {
		if lot.use != lotBuilding {
			continue
		}
		pop := city.Pop.Int()
		if !inPeonIslandFor(pop, lot.x, lot.y) {
			t.Fatalf("peon building outside island at (%d,%d)", lot.x, lot.y)
		}
		px, py := peonPlateOfFor(pop, lot.x, lot.y)
		ax, ay := peonPlateAnchorCellFor(pop, px, py)
		if lot.x != ax || lot.y != ay {
			t.Fatalf("peon building at (%d,%d) want plate anchor (%d,%d)", lot.x, lot.y, ax, ay)
		}
		plates[[2]int{px, py}]++
		buildings = append(buildings, lot)
	}
	if len(buildings) == 0 {
		t.Fatal("pop=8 peon should place some buildings on the island")
	}
	for plate, n := range plates {
		if n != 1 {
			t.Fatalf("plate %v has %d buildings; want 1 per plate", plate, n)
		}
	}

	again := lotOccupancy(city, grid)
	for pos, lot := range occ {
		got := again[pos]
		if got.use != lot.use || got.x != lot.x || got.y != lot.y {
			t.Fatalf("peon occupancy must be deterministic at %v: %+v vs %+v", pos, lot, got)
		}
	}
}

func TestPeonBuildingsCapAtDallePlates(t *testing.T) {
	city := &citycore.City{Slug: "peon-dense", Pop: 39}
	occ := lotOccupancy(city, buildCityGridForCity(city))
	n := 0
	plates := make(map[[2]int]struct{})
	pop := city.Pop.Int()
	for _, lot := range occ {
		if lot.use != lotBuilding {
			continue
		}
		if !inPeonIslandFor(pop, lot.x, lot.y) {
			t.Fatalf("peon building outside island at (%d,%d)", lot.x, lot.y)
		}
		px, py := peonPlateOfFor(pop, lot.x, lot.y)
		plates[[2]int{px, py}] = struct{}{}
		n++
	}
	if n == 0 {
		t.Fatal("peon pop=39 should place buildings")
	}
	if n > peonDalleGridFor(pop)*peonDalleGridFor(pop) {
		t.Fatalf("peon buildings exceed plate count: %d > %d", n, peonDalleGridFor(pop)*peonDalleGridFor(pop))
	}
	if len(plates) != n {
		t.Fatalf("expected 1 building per plate, got %d plates for %d buildings", len(plates), n)
	}
	// Continuous with pop=40 (no fillRate cliff).
	occ40 := lotOccupancy(&citycore.City{Slug: "peon-dense", Pop: 40}, buildCityGridForCity(city))
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

func TestPeonGrassTopCellInsetsFromRim(t *testing.T) {
	pop := 2
	o := peonIslandOriginFor(pop)
	last := o + peonIslandExtentFor(pop) - 1
	if peonGrassTopCell(pop, o, o) {
		t.Fatal("island corner must not count as grass top")
	}
	if peonGrassTopCell(pop, last, last) {
		t.Fatal("south tip must not count as grass top")
	}
	mid := (o + last) / 2
	if !peonGrassTopCell(pop, mid, mid) {
		t.Fatal("island center must count as grass top")
	}
	if peonGrassTopCell(pop, o+1, mid) {
		t.Fatalf("rim inset 1 at x=%d should still be soil ledge", o+1)
	}
	if !peonGrassTopCell(pop, o+peonGrassRimInset, mid) {
		t.Fatalf("rim inset %d should be grass top", peonGrassRimInset)
	}
}

func TestPeonParksStayOnGrassTops(t *testing.T) {
	city := &citycore.City{Slug: "testcity", Pop: 2, Env: 7, Ind: 1, Com: 1, Sec: 1}
	occ := lotOccupancy(city, buildCityGridForCity(city))
	pop := city.Pop.Int()
	parks := 0
	for _, lot := range occ {
		if lot.use != lotPark {
			continue
		}
		parks++
		if !peonGrassTopCell(pop, lot.x, lot.y) {
			t.Fatalf("peon park at (%d,%d) sits on soil rim", lot.x, lot.y)
		}
	}
	if parks == 0 {
		t.Fatal("env=7 peon should still place some trees on grass tops")
	}
}

func TestArterialParksStayOnGrassTops(t *testing.T) {
	// pop>=80 enables roads; trees must still stay off the dalle soil rim.
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
		if !peonGrassTopCell(pop, lot.x, lot.y) {
			t.Fatalf("arterial park at (%d,%d) sits on soil rim", lot.x, lot.y)
		}
	}
	if parks == 0 {
		t.Fatal("env=79 arterial should still place some trees on grass tops")
	}

	peon := &citycore.City{Slug: "testcity", Pop: 70, Env: 79, Ind: 1, Com: 1, Sec: 1}
	if arterialsEnabled(peon) {
		t.Fatal("pop=70 must stay peon (no arterials)")
	}
	peonOcc := lotOccupancy(peon, buildCityGridForCity(peon))
	for _, lot := range peonOcc {
		if lot.use != lotPark {
			continue
		}
		if !peonGrassTopCell(peon.Pop.Int(), lot.x, lot.y) {
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
	com := zoneTag(&citycore.City{Com: 10}, mapCols/2, mapRows/2)
	if com != TagCommercial {
		t.Fatalf("center with com should be commercial, got %s", com)
	}
	ind := zoneTag(&citycore.City{Ind: 10}, 0, 5)
	if ind != TagIndustrial {
		t.Fatalf("edge with ind should be industrial, got %s", ind)
	}
	// Outer ring width scales with map (rim=mapCols/10); on 60×60 rim=6 so x=2
	// is industrial and the first interior lot past the rim stays residential.
	ring2 := zoneTag(&citycore.City{Ind: 10}, 2, mapRows/2)
	if ring2 != TagIndustrial {
		t.Fatalf("x=2 with ind should be industrial, got %s", ring2)
	}
	rim := mapCols / 10
	if rim < 2 {
		rim = 2
	}
	midOuter := zoneTag(&citycore.City{Ind: 10}, rim, mapRows/2)
	if midOuter != TagResidential {
		t.Fatalf("x=%d with ind should stay residential, got %s", rim, midOuter)
	}
	res := zoneTag(&citycore.City{}, mapCols/2, mapRows/2)
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
	comTag := zoneTag(city, mapCols/2, mapRows/2)
	indTag := zoneTag(city, 0, 5)
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
