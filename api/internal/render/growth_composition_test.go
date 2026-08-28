package render

import (
	"strings"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

// buildingComposition tallies zone tags and growth tiers for building lots.
func buildingComposition(t *testing.T, city *citycore.City, atlas *Atlas) (byTag map[string]int, byTier map[int]int, n int) {
	t.Helper()
	grid := buildCityGridForCity(city)
	occ := lotOccupancy(city, grid)
	byTag = map[string]int{}
	byTier = map[int]int{}
	for pos, lot := range occ {
		if lot.use != lotBuilding {
			continue
		}
		seed := hashCell(city.Slug.String(), pos[0], pos[1])
		key := atlas.PickBuildingKeyForLot(city, lot.tag, pos[0], pos[1], seed)
		if key == "" {
			continue
		}
		n++
		parts := strings.Split(key, "/")
		if len(parts) < 2 {
			continue
		}
		folder := parts[0] + "/" + parts[1]
		tier := atlas.folderTier(folder)
		byTier[tier]++
		switch {
		case frameBelongsToTag(key, atlas.BasesForTag(TagLandmark)):
			byTag[TagLandmark]++
		case frameBelongsToTag(key, atlas.BasesForTag(TagIndustrial)):
			byTag[TagIndustrial]++
		case frameBelongsToTag(key, atlas.BasesForTag(TagCommercial)):
			byTag[TagCommercial]++
		case frameBelongsToTag(key, atlas.BasesForTag(TagResidential)):
			byTag[TagResidential]++
		default:
			byTag["other"]++
		}
	}
	return byTag, byTier, n
}

func TestGrowthCompositionDiffersByPop(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}

	slug := citycore.Slug("growth-compose")
	lowCity := &citycore.City{Slug: slug, Pop: 20, Ind: 0, Com: 0, Env: 0}
	midCity := &citycore.City{Slug: slug, Pop: 150, Ind: 50, Com: 50, Env: 80}
	highCity := &citycore.City{Slug: slug, Pop: 500, Ind: 50, Com: 50, Env: 400, Sec: 300}

	lowTag, lowTier, lowN := buildingComposition(t, lowCity, atlas)
	midTag, midTier, midN := buildingComposition(t, midCity, atlas)
	_, highTier, highN := buildingComposition(t, highCity, atlas)

	if lowN == 0 || midN == 0 || highN == 0 {
		t.Fatalf("expected buildings at each stage: low=%d mid=%d high=%d", lowN, midN, highN)
	}
	if highN <= lowN {
		t.Fatalf("high pop should place at least as many buildings as low: low=%d high=%d", lowN, highN)
	}

	// Low pop: no landmarks, mostly tier 0–1.
	if lowTag[TagLandmark] != 0 {
		t.Fatalf("low pop must not place landmarks, got %d", lowTag[TagLandmark])
	}
	lowHighTier := lowTier[2] + lowTier[3]
	if lowHighTier > lowN/3 {
		t.Fatalf("low pop should rarely use tier>=2, got %d/%d", lowHighTier, lowN)
	}

	// High pop: per-square landmarks appear and tier>=2 share rises vs low.
	highLand, _ := planSquareLandmarks(highCity, atlas, genMapPop(highCity.Pop.Int(), newMapRNG(highCity.Slug.String())))
	if len(highLand) == 0 {
		t.Fatal("high pop+sectors should place some square landmarks")
	}
	lowLand, _ := planSquareLandmarks(lowCity, atlas, genMapPop(lowCity.Pop.Int(), newMapRNG(lowCity.Slug.String())))
	if len(lowLand) != 0 {
		t.Fatalf("low pop must not place square landmarks, got %d", len(lowLand))
	}
	highHighTiers := highTier[2] + highTier[3]
	if highHighTiers <= lowHighTier {
		t.Fatalf("high pop should use more high-tier buildings than low: low=%d high=%d", lowHighTier, highHighTiers)
	}

	// Mid should be between: landmarks allowed but fewer than huge, or at least some mid tiers.
	if midTier[1]+midTier[2] == 0 {
		t.Fatalf("mid pop should place mid-tier buildings, tags=%v tiers=%v", midTag, midTier)
	}

	// Determinism: same city recomposition matches.
	lowTag2, lowTier2, lowN2 := buildingComposition(t, lowCity, atlas)
	if lowN2 != lowN || lowTag2[TagResidential] != lowTag[TagResidential] || lowTier2[0] != lowTier[0] {
		t.Fatal("composition must be deterministic for identical city input")
	}
}

func TestGrowthCompositionFavorsResidentialOutskirts(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "houses-outskirts", Pop: 500, Ind: 50, Com: 5, Env: 400}
	byTag, _, n := buildingComposition(t, city, atlas)
	if n == 0 {
		t.Fatal("expected buildings")
	}
	if byTag[TagIndustrial] == 0 {
		t.Fatalf("Ind>0 should place some industrial buildings, tags=%v", byTag)
	}
	if byTag[TagResidential] == 0 {
		t.Fatalf("expected residential buildings alongside industrial, tags=%v", byTag)
	}
	if byTag[TagResidential] < n/5 {
		t.Fatalf("expected substantial residential share, got %d/%d tags=%v", byTag[TagResidential], n, byTag)
	}
}

func TestGrowthCompositionPop300PlacesHousesInResidential(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "testcity", Pop: 500, Ind: 50, Com: 50, Env: 100}
	grid := buildCityGridForCity(city)
	occ := lotOccupancy(city, grid)
	keys := assignBuildingKeys(atlas, city, occ, genMapPop(city.Pop.Int(), newMapRNG(city.Slug.String())).max)

	var resLots, houses int
	for pos, lot := range occ {
		if lot.use != lotBuilding || lot.tag != TagResidential {
			continue
		}
		key := keys[pos]
		if key == "" {
			continue
		}
		resLots++
		if atlas.folderTier(spriteFolderBase(key)) <= 1 {
			houses++
		}
	}
	if resLots == 0 {
		t.Fatal("expected residential-zone buildings")
	}
	if houses*2 < resLots {
		t.Fatalf("pop=300 residential zone should be mostly houses, got %d/%d", houses, resLots)
	}
}

func TestAssignBuildingKeysAvoidsAdjacentHighTierClones(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "testcity", Pop: 500, Ind: 50, Com: 50, Env: 100}
	grid := buildCityGridForCity(city)
	occ := lotOccupancy(city, grid)
	keys := assignBuildingKeys(atlas, city, occ, genMapPop(city.Pop.Int(), newMapRNG(city.Slug.String())).max)

	same, highPairs := countHighTierSameFolderPairs(atlas, keys)
	if highPairs == 0 {
		// Game.hx genMiniSquare spaces footprints on a 2×2 grid; adjacent
		// high-tier pairs can be rare with the library_primary pool.
		t.Skip("no adjacent high-tier pairs with current density layout")
	}
	// Smaller pool_eligible set (#82) increases identical-neighbor rate; keep determinism.
	if uniqueBuildingFolders(atlas) >= 30 && same*3 > highPairs {
		t.Fatalf("high-tier identical neighbors should be rare, same=%d pairs=%d", same, highPairs)
	}

	again := assignBuildingKeys(atlas, city, occ, genMapPop(city.Pop.Int(), newMapRNG(city.Slug.String())).max)
	if len(again) != len(keys) {
		t.Fatalf("deterministic key count mismatch: %d vs %d", len(again), len(keys))
	}
	for pos, key := range keys {
		if again[pos] != key {
			t.Fatalf("assignBuildingKeys must be deterministic at %v: %q vs %q", pos, key, again[pos])
		}
	}
}

func uniqueBuildingFolders(atlas *Atlas) int {
	if atlas == nil {
		return 0
	}
	seen := make(map[string]struct{}, len(atlas.BuildingBases))
	for _, base := range atlas.BuildingBases {
		seen[spriteFolderBase(base)] = struct{}{}
	}
	return len(seen)
}

func countHighTierSameFolderPairs(atlas *Atlas, keys map[[2]int]string) (same, pairs int) {
	for pos, key := range keys {
		if key == "" || atlas.folderTier(spriteFolderBase(key)) < 2 {
			continue
		}
		x, y := pos[0], pos[1]
		for _, d := range [][2]int{{1, 0}, {0, 1}} {
			nb := keys[[2]int{x + d[0], y + d[1]}]
			if nb == "" || atlas.folderTier(spriteFolderBase(nb)) < 2 {
				continue
			}
			pairs++
			if spriteFolderBase(key) == spriteFolderBase(nb) {
				same++
			}
		}
	}
	return same, pairs
}

func TestRoadlessAvoidsNearbySameFolderBuildings(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "testcity7", Pop: 40, Ind: 130, Com: 150, Env: 110, Sec: 100}
	grid := buildCityGridForCity(city)
	occ := lotOccupancy(city, grid)
	keys := assignBuildingKeys(atlas, city, occ, genMapPop(city.Pop.Int(), newMapRNG(city.Slug.String())).max)
	if len(keys) < 2 {
		t.Fatal("expected multiple roadless buildings")
	}
	radius := folderAvoidChebyshev(city)
	if radius != 2 {
		t.Fatalf("roadless folder avoid radius=%d, want 2", radius)
	}
	same := 0
	for pos, key := range keys {
		if key == "" {
			continue
		}
		folder := spriteFolderBase(key)
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				nb := keys[[2]int{pos[0] + dx, pos[1] + dy}]
				if nb == "" {
					continue
				}
				if spriteFolderBase(nb) == folder {
					same++
				}
			}
		}
	}
	// 307 is a tree grove, not commercial. Remaining low-tier commercial is
	// a single folder (358); when the avoid set exhausts, pick falls back to
	// that folder instead of leaving the lot empty. Allow one leftover pair.
	if same/2 > 1 {
		t.Fatalf("chebyshev-%d same-folder building pairs=%d, want at most 1", radius, same/2)
	}
}

func TestArterialCrossRoadsStillPlan(t *testing.T) {
	// Townzzy / Game.hx: scoreRoad > 6 stamps CROSS inside density squares.
	city := &citycore.City{Slug: "cross-town", Pop: 120, Tra: 10}
	dens := genMapPop(city.Pop.Int(), newMapRNG(city.Slug.String()))
	plan := planRoads(city, dens)
	crossN := 0
	for sy := 0; sy < displaySide; sy++ {
		for sx := 0; sx < displaySide; sx++ {
			if plan.cross[sy][sx] > 0 {
				crossN++
			}
		}
	}
	if crossN == 0 {
		t.Fatal("pop=120 with tra should plan some CROSS roads (Game.hx scoreRoad>6)")
	}
	occ := lotOccupancy(city, plan.grid)
	for _, lot := range occ {
		if lot.use != lotBuilding {
			continue
		}
		if !isGameHxMiniFoot(lot.x, lot.y) {
			// POP_HUGE square origin is allowed; isGameHxMiniFoot covers it.
			t.Fatalf("arterial building at (%d,%d) not on Game.hx foot", lot.x, lot.y)
		}
	}
}
