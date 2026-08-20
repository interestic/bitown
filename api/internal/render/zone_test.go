package render

import (
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestLotOccupancyFillsFromCenter(t *testing.T) {
	grid := buildCityGrid("zone-city")
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
		if lot.use == lotEmpty && d < fullMinEmptyDist {
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

func TestLotOccupancyAddsParksWithEnv(t *testing.T) {
	grid := buildCityGrid("park-city")
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
	comKey := atlas.PickKeyForTag(zoneTag(city, mapCols/2, mapRows/2), 1)
	indKey := atlas.PickKeyForTag(zoneTag(city, 0, 5), 1)
	if comKey == "" || indKey == "" {
		t.Fatalf("expected tagged picks, com=%q ind=%q", comKey, indKey)
	}
	if comKey == indKey {
		t.Fatalf("commercial and industrial picks should differ, both %q", comKey)
	}
	if !frameBelongsToTag(comKey, atlas.BasesForTag(TagCommercial)) {
		t.Fatalf("commercial pick %q not in commercial bases", comKey)
	}
	if !frameBelongsToTag(indKey, atlas.BasesForTag(TagIndustrial)) {
		t.Fatalf("industrial pick %q not in industrial bases", indKey)
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
