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
	var fullMaxBuildingDist, fullMinEmptyDist int
	fullMinEmptyDist = 1 << 20
	for _, lot := range empty {
		if lot.use == lotBuilding {
			emptyBuildings++
		}
	}
	for pos, lot := range full {
		dx, dy := pos[0]-cx, pos[1]-cy
		d := dx*dx + dy*dy
		if lot.use == lotBuilding {
			fullBuildings++
			if d > fullMaxBuildingDist {
				fullMaxBuildingDist = d
			}
		}
		// Interior empties for center-out comparison (curb setback is separate).
		if lot.use == lotEmpty && !lotTouchesRoad(grid, pos[0], pos[1]) && d < fullMinEmptyDist {
			fullMinEmptyDist = d
		}
	}
	if emptyBuildings != 0 {
		t.Fatalf("pop=0 should leave lots empty, got %d buildings", emptyBuildings)
	}
	if fullBuildings <= emptyBuildings {
		t.Fatalf("pop=500 should fill lots, got %d buildings", fullBuildings)
	}
	if fullMinEmptyDist == 1<<20 {
		t.Fatal("pop=500 should leave some empty fringe lots for center-out comparison")
	}
	if fullMaxBuildingDist > fullMinEmptyDist {
		t.Fatalf("buildings should fill nearer the center than empties: maxBuilding=%d minEmpty=%d", fullMaxBuildingDist, fullMinEmptyDist)
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
		if !inPeonIsland(lot.x, lot.y) {
			t.Fatalf("peon building outside island at (%d,%d)", lot.x, lot.y)
		}
		px, py := peonPlateOf(lot.x, lot.y)
		ax, ay := peonPlateAnchorCell(px, py)
		if lot.x != ax || lot.y != ay {
			t.Fatalf("peon building at (%d,%d) want plate anchor (%d,%d)", lot.x, lot.y, ax, ay)
		}
		plates[[2]int{px, py}]++
		buildings = append(buildings, lot)
	}
	if len(buildings) != 8 {
		t.Fatalf("pop=8 peon should place 8 buildings, got %d", len(buildings))
	}
	if len(plates) != 8 {
		t.Fatalf("expected 1 building per dalle plate, got %d plates for %d buildings", len(plates), len(buildings))
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
	for _, lot := range occ {
		if lot.use != lotBuilding {
			continue
		}
		n++
		px, py := peonPlateOf(lot.x, lot.y)
		plates[[2]int{px, py}] = struct{}{}
	}
	if n != peonPlateCount() {
		t.Fatalf("peon pop=39 should cap at %d dalle plates, got %d", peonPlateCount(), n)
	}
	if len(plates) != n {
		t.Fatalf("expected 1 building per plate, got %d plates for %d buildings", len(plates), n)
	}
}

func TestPeonParksPreferIslandPlates(t *testing.T) {
	city := &citycore.City{Slug: "peon-park", Pop: 39, Env: 400}
	occ := lotOccupancy(city, buildCityGridForCity(city))
	islandParks := 0
	for pos, lot := range occ {
		if lot.use != lotPark {
			continue
		}
		px, py := peonPlateOf(pos[0], pos[1])
		ax, ay := peonPlateAnchorCell(px, py)
		if pos[0] == ax && pos[1] == ay {
			islandParks++
		}
	}
	if islandParks == 0 {
		t.Fatal("expected peon parks on island anchor plates")
	}
	if islandParks < 8 {
		t.Fatalf("expected most peon parks on island plates, got %d", islandParks)
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
	// Outer ring width scales with map (rim=mapCols/10); on 40×40 rim=4 so x=2
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
