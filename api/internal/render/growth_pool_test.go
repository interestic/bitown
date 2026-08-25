package render

import (
	"strings"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestMaxTierForPop(t *testing.T) {
	cases := []struct {
		pop  int
		want int
	}{
		{0, 1},
		{39, 1},
		{40, 2},
		{119, 2},
		{120, 3},
		{500, 3},
	}
	for _, tc := range cases {
		if got := maxTierForPop(tc.pop); got != tc.want {
			t.Fatalf("maxTierForPop(%d)=%d want %d", tc.pop, got, tc.want)
		}
	}
}

// maxTierForLotTag applies the generic distance cap, then a wider house belt
// on residential lots (test helper mirroring the periphery rules in
// maxTierForLotWithLocal without local density).
func maxTierForLotTag(pop, x, y int, tag string) int {
	max := maxTierForPop(pop)
	cx, cy := mapCols/2, mapRows/2
	dx, dy := x-cx, y-cy
	dist2 := dx*dx + dy*dy
	outer := outerLotDist2()
	if tag == TagResidential && pop < popTierHuge {
		outer = outer / 2
		if outer < 1 {
			outer = 1
		}
	}
	if dist2 > outer && max > 0 {
		max--
	}
	if tag != TagIndustrial && dist2 >= outer*2 && max > 1 {
		max = 1
	}
	return max
}

func TestMaxTierForLot(t *testing.T) {
	cx, cy := mapCols/2, mapRows/2
	outer := outerLotDist2()
	// Near-outer: just past outer threshold.
	near := 0
	for d := 1; d < mapCols; d++ {
		if d*d > outer {
			near = d
			break
		}
	}
	cases := []struct {
		name string
		pop  int
		x, y int
		want int
	}{
		{"center huge", 500, cx, cy, 3},
		{"near-outer huge drops one", 500, cx, cy - near, 2},
		{"far huge caps at 1", 500, 0, 0, 1},
		{"corner peon empty pool tier", 20, 0, 0, 0}, // peon 1 → outer 0
		{"center peon", 20, cx, cy, 1},
	}
	for _, tc := range cases {
		if got := maxTierForLotTag(tc.pop, tc.x, tc.y, ""); got != tc.want {
			t.Fatalf("%s: maxTierForLotTag(%d,%d,%d,\"\")=%d want %d (outer=%d)", tc.name, tc.pop, tc.x, tc.y, got, tc.want, outer)
		}
	}
}

func TestTierPickWeightBigDoesNotDominateSkyscrapers(t *testing.T) {
	if tierPickWeight(3, 120) > tierPickWeight(2, 120) {
		t.Fatalf("big pop must not weight tier 3 above tier 2: t3=%d t2=%d", tierPickWeight(3, 120), tierPickWeight(2, 120))
	}
	if tierPickWeight(3, 349) > tierPickWeight(2, 349) {
		t.Fatalf("big pop upper bound must not weight tier 3 above tier 2")
	}
	if tierPickWeight(3, 350) <= tierPickWeight(2, 350) {
		t.Fatalf("huge pop should prefer tier 3 over tier 2: t3=%d t2=%d", tierPickWeight(3, 350), tierPickWeight(2, 350))
	}
}

func TestTallResidentialIsTier3(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	// DefineSprite_633 is a child module (not pool_eligible). Pool landmarks carry tier 3.
	if got := atlas.folderTier("sprites/DefineSprite_692"); got != 3 {
		t.Fatalf("DefineSprite_692 (pool landmark) tier=%d want 3", got)
	}
}

func TestOuterLotsAtBigThresholdOmitTier3(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "testcity", Pop: 120, Ind: 5, Com: 3, Env: 80}
	cx, cy := mapCols/2, mapRows/2
	var outer, outerTier3 int
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= outerLotDist2() {
				continue
			}
			key := atlas.PickBuildingKeyForLot(city, zoneTag(city, x, y), x, y, hashCell(city.Slug.String(), x, y))
			if key == "" {
				continue
			}
			outer++
			parts := strings.Split(key, "/")
			folder := parts[0] + "/" + parts[1]
			if atlas.folderTier(folder) >= 3 {
				outerTier3++
			}
		}
	}
	if outer == 0 {
		t.Fatal("expected outer-lot picks")
	}
	if outerTier3 != 0 {
		t.Fatalf("pop=120 outer lots must not place tier 3, got %d/%d", outerTier3, outer)
	}
}

func TestCenterLotsAtBigThresholdCanPlaceTier3(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "testcity", Pop: 400, Ind: 50, Com: 50, Env: 80}
	cx, cy := mapCols/2, mapRows/2
	var center, centerTier3 int
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy > outerLotDist2() {
				continue
			}
			key := atlas.PickBuildingKeyForLot(city, zoneTag(city, x, y), x, y, hashCell(city.Slug.String(), x, y))
			if key == "" {
				continue
			}
			center++
			parts := strings.Split(key, "/")
			folder := parts[0] + "/" + parts[1]
			if atlas.folderTier(folder) >= 3 {
				centerTier3++
			}
		}
	}
	if center == 0 {
		t.Fatal("expected center-lot picks")
	}
	if centerTier3 == 0 {
		t.Fatalf("pop=400 center must place some tier 3, got 0/%d", center)
	}
}

func TestBigPopTierFirstDoesNotDrownMidRise(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "tier-mix", Pop: 120, Com: 50}
	// Commercial has many tall folders; tier-first must keep mid-rise competitive.
	var mid, tall int
	for seed := uint32(0); seed < 256; seed++ {
		key := atlas.pickBuildingFrameForTagAvoiding(city, TagCommercial, 3, 120, csPopHuge, seed, nil)
		if key == "" {
			continue
		}
		parts := strings.Split(key, "/")
		folder := parts[0] + "/" + parts[1]
		switch atlas.folderTier(folder) {
		case 2:
			mid++
		case 3:
			tall++
		}
	}
	if mid == 0 {
		t.Fatal("expected some commercial tier 2 at pop=120 maxTier=3")
	}
	if tall > mid*2 {
		t.Fatalf("tier-first should keep mid-rise competitive vs tall: mid=%d tall=%d", mid, tall)
	}
}

func TestLandmarkMixPermille(t *testing.T) {
	if landmarkMixPermille(20, 100, 100) != 0 {
		t.Fatal("low pop must not mix landmarks")
	}
	if landmarkMixPermille(200, 0, 0) <= 0 {
		t.Fatal("big pop should allow some landmark mix")
	}
	low := landmarkMixPermille(400, 0, 0)
	high := landmarkMixPermille(400, 100, 100)
	if high <= low {
		t.Fatalf("sec/com should boost landmark chance: low=%d high=%d", low, high)
	}
}

func TestResidentialPoolIncludesClassicHouses(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	folders := map[string]bool{}
	for _, base := range atlas.BasesForTag(TagResidential) {
		folders[spriteFolderBase(base)] = true
	}
	for _, want := range []string{
		"sprites/DefineSprite_200",
		"sprites/DefineSprite_269",
		"sprites/DefineSprite_374",
	} {
		if !folders[want] {
			t.Fatalf("expected classic house %s in residential pool (folders=%v)", want, folders)
		}
	}
}

func TestPickBuildingKeyForLotDoesNotFloodFromMultiFrameFolder(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "variety", Pop: 20}
	counts := map[string]int{}
	for seed := uint32(0); seed < 64; seed++ {
		key := atlas.PickBuildingKeyForLot(city, TagResidential, 10, 10, seed)
		if key == "" {
			continue
		}
		parts := strings.Split(key, "/")
		folder := parts[0] + "/" + parts[1]
		counts[folder]++
	}
	if len(counts) < 3 {
		t.Fatalf("expected variety across folders at low pop, got %v", counts)
	}
	for folder, n := range counts {
		if n > 40 {
			t.Fatalf("folder %s dominated picks (%d/64): %v", folder, n, counts)
		}
	}
}

func TestPickBuildingKeyForLotPrefersLowTierAtLowPop(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	if len(atlas.TierByFolder) == 0 {
		t.Fatal("expected TierByFolder from buildings.json")
	}

	city := &citycore.City{Slug: "growth-low", Pop: 20}
	var lowTier, samples int
	for seed := uint32(0); seed < 64; seed++ {
		key := atlas.PickBuildingKeyForLot(city, TagResidential, 10, 10, seed)
		if key == "" {
			continue
		}
		samples++
		folder := spriteFolderBase(strings.TrimSuffix(key, "_v00.png"))
		// strip frame index path: sprites/Foo/1 → sprites/Foo
		if i := strings.LastIndex(folder, "/"); i > 0 && !strings.Contains(folder[i+1:], "Define") {
			// key is sprites/Foo/1_v00.png → folder base sprites/Foo
			parts := strings.Split(key, "/")
			if len(parts) >= 2 {
				folder = parts[0] + "/" + parts[1]
			}
		}
		tier := atlas.folderTier(folder)
		if tier <= 1 {
			lowTier++
		}
		if tier >= 3 {
			t.Fatalf("low pop must not pick landmark-tier building, got %s tier=%d", key, tier)
		}
	}
	if samples < 16 {
		t.Fatalf("expected residential picks, got %d", samples)
	}
	if lowTier*2 < samples {
		t.Fatalf("low pop should mostly pick tier<=1, got %d/%d", lowTier, samples)
	}
}

func TestPickBuildingKeyForLotHighPopCanUseHighTiers(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}

	city := &citycore.City{Slug: "growth-high", Pop: 500, Ind: 50, Com: 50, Sec: 300, Env: 400}
	cx, cy := mapCols/2, mapRows/2
	var highTier, landmarks, samples int
	for seed := uint32(0); seed < 256; seed++ {
		key := atlas.PickBuildingKeyForLot(city, TagResidential, cx, cy, seed)
		if key == "" {
			continue
		}
		samples++
		parts := strings.Split(key, "/")
		folder := parts[0] + "/" + parts[1]
		tier := atlas.folderTier(folder)
		if tier >= 2 {
			highTier++
		}
		if frameBelongsToTag(key, atlas.BasesForTag(TagLandmark)) {
			landmarks++
		}
	}
	if samples < 32 {
		t.Fatalf("expected picks, got %d", samples)
	}
	if highTier == 0 {
		t.Fatal("high pop should sometimes pick tier>=2 buildings")
	}
	if landmarks == 0 {
		t.Fatal("high pop+sectors should sometimes mix landmark frames")
	}
}

func TestPickBuildingKeyForLotEmptyWithinTier(t *testing.T) {
	// Synthetic atlas: only tier-2 residential — peon outer lots (max tier 0) must
	// return empty so callers draw a rectangle, not bypass via PickBuildingKeyForTag.
	atlas := &Atlas{
		BasesByTag: map[string][]string{
			TagResidential: {"sprites/House_tall/1"},
		},
		TierByFolder: map[string]int{
			"sprites/House_tall": 2,
		},
		Frames: map[string]frameRect{
			"sprites/House_tall/1_v00.png": {W: 16, H: 24},
		},
	}
	city := &citycore.City{Slug: "peon-empty", Pop: 20}
	// Corner lot: maxTierForLotTag drops peon cap 1 → 0.
	key := atlas.PickBuildingKeyForLot(city, TagResidential, 0, 0, 1)
	if key != "" {
		t.Fatalf("expected empty key when no frames within tier cap, got %q", key)
	}
	// Uniform path still has the tall house — proves we did not fall through.
	if got := atlas.PickBuildingKeyForTag(TagResidential, 1); got == "" {
		t.Fatal("control: PickBuildingKeyForTag should still see the tall house")
	}
}

func TestPickBuildingKeyForLotDeterministic(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "growth-det", Pop: 200, Com: 3}
	a := atlas.PickBuildingKeyForLot(city, TagCommercial, 10, 10, 42)
	b := atlas.PickBuildingKeyForLot(city, TagCommercial, 10, 10, 42)
	if a == "" || a != b {
		t.Fatalf("deterministic pick mismatch: %q vs %q", a, b)
	}
}

func TestPickBuildingKeyForLotKeepsZoneTags(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "growth-zone", Pop: 300, Ind: 50, Com: 10}

	comKey := atlas.PickBuildingKeyForLot(city, TagCommercial, mapCols/2, mapRows/2, 7)
	indKey := atlas.PickBuildingKeyForLot(city, TagIndustrial, 0, 5, 7)
	if comKey == "" || indKey == "" {
		t.Fatalf("expected zone picks, com=%q ind=%q", comKey, indKey)
	}
	if len(atlas.BasesForTag(TagCommercial)) > 0 && !frameBelongsToTag(comKey, atlas.BasesForTag(TagCommercial)) &&
		!frameBelongsToTag(comKey, atlas.BasesForTag(TagLandmark)) {
		t.Fatalf("commercial lot key %q not in commercial/landmark pool", comKey)
	}
	if len(atlas.BasesForTag(TagIndustrial)) > 0 && !frameBelongsToTag(indKey, atlas.BasesForTag(TagIndustrial)) &&
		!frameBelongsToTag(indKey, atlas.BasesForTag(TagLandmark)) {
		t.Fatalf("industrial lot key %q not in industrial/landmark pool", indKey)
	}
}

func TestTierPickWeightResidentialPrefersHousesAtBigPop(t *testing.T) {
	if tierPickWeightForTag(TagResidential, 0, 300) <= tierPickWeightForTag(TagResidential, 3, 300) {
		t.Fatal("residential big pop must weight houses above the lone tier-3 tower")
	}
	if tierPickWeightForTag(TagResidential, 1, 300) <= tierPickWeightForTag(TagResidential, 3, 300) {
		t.Fatal("residential big pop must weight ordinary houses above tier 3")
	}
	if tierPickWeightForTag(TagCommercial, 3, 300) != tierPickWeight(3, 300) {
		t.Fatal("commercial must keep the generic big-pop curve")
	}
}

func TestMaxTierForLotTagResidentialWiderHouseBelt(t *testing.T) {
	cx, cy := mapCols/2, mapRows/2
	outer := outerLotDist2()
	mid := 0
	for d := 1; d < mapCols; d++ {
		if d*d > outer/2 && d*d <= outer {
			mid = d
			break
		}
	}
	if mid == 0 {
		t.Fatal("expected a mid-ring distance")
	}
	if got := maxTierForLotTag(300, cx, cy-mid, ""); got != 3 {
		t.Fatalf("generic mid-ring want 3, got %d", got)
	}
	if got := maxTierForLotTag(300, cx, cy-mid, TagResidential); got != 2 {
		t.Fatalf("residential mid-ring want 2, got %d", got)
	}
	if got := maxTierForLotTag(300, 0, 0, TagResidential); got != 1 {
		t.Fatalf("residential corner want 1, got %d", got)
	}
	if got := maxTierForLotTag(500, 0, cy, TagIndustrial); got < 2 {
		t.Fatalf("industrial rim should keep warehouses, got %d", got)
	}
}
