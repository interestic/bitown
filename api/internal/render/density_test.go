package render

import (
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestGenMapPopContinuousAcrossRoadlessThreshold(t *testing.T) {
	slug := "dens-cliff"
	n39 := countBuildings(t, slug, 39)
	n40 := countBuildings(t, slug, 40)
	if n39 == 0 || n40 == 0 {
		t.Fatalf("expected buildings on both sides of roadless threshold: 39=%d 40=%d", n39, n40)
	}
	// Old fillRate path jumped ~16 → ~256 island lots; Game.hx density must not.
	if n40 > n39*3 {
		t.Fatalf("pop 39→40 density cliff: 39=%d 40=%d", n39, n40)
	}
	if n40+3 < n39 {
		t.Fatalf("pop 40 should not shrink vs 39: 39=%d 40=%d", n39, n40)
	}
}

func TestGenMapPopGrowsWithPop(t *testing.T) {
	slug := "dens-grow"
	low := countBuildings(t, slug, 20)
	high := countBuildings(t, slug, 300)
	if high <= low {
		t.Fatalf("higher pop should place more buildings: low=%d high=%d", low, high)
	}
}

func TestGetRayMaxMonotonic(t *testing.T) {
	prev := getRayMax(0)
	for _, n := range []float64{1, 10, 40, 200, 500, 2000} {
		got := getRayMax(n)
		if got < prev {
			t.Fatalf("getRayMax not monotonic at %v: %v < %v", n, got, prev)
		}
		prev = got
	}
}

func TestFlashDisplaySideGrowsWithPop(t *testing.T) {
	low := flashDisplaySide(1)
	mid := flashDisplaySide(200)
	high := flashDisplaySide(2000)
	if low < 1 || low > mid || mid > high {
		t.Fatalf("displaySide should grow with pop: %d %d %d", low, mid, high)
	}
	if activeSquareSide(1) > displaySide {
		t.Fatal("activeSquareSide must cap at displaySide")
	}
}

func TestFlashDisplaySideMatchesGameHxTruncation(t *testing.T) {
	// Game.hx: displayMargin = Std.int(max(0,(SIDE-ray*2)*0.5)); side = SIDE-2*margin.
	if got := flashDisplaySide(0); got != 4 {
		t.Fatalf("pop=0 displaySide=%d, want 4", got)
	}
	if got := flashDisplaySide(1); got != 6 {
		t.Fatalf("pop=1 displaySide=%d, want 6 (Townzzy / Game.hx Std.int)", got)
	}
	if plateGridFor(1) != 6 {
		t.Fatalf("pop=1 dalle grid=%d, want 6×6 plates", plateGridFor(1))
	}
	if displaySide != 25 {
		t.Fatalf("PNG field displaySide=%d, want 25 (Game.hx grow cap)", displaySide)
	}
}

func TestUpdateLibHouseGates(t *testing.T) {
	if !updateLibHouseUnlocked(libHouse1, 0, 0) {
		t.Fatal("mcHouse1 always unlocked")
	}
	if updateLibHouseUnlocked(libHouse2, csPopBig-1, 0) {
		t.Fatal("mcHouse2 locked when densityMax < POP_BIG and city pop low")
	}
	if !updateLibHouseUnlocked(libHouse2, csPopBig, 0) {
		t.Fatal("mcHouse2 unlocked at POP_BIG")
	}
	if !updateLibHouseUnlocked(libHouse2, 0, houseBandPop) {
		t.Fatal("mcHouse2 unlocked by city pop fallback")
	}
	if updateLibHouseUnlocked(libHouse3, csPopHuge-1, 0) {
		t.Fatal("mcHouse3 locked when densityMax < POP_HUGE and city pop low")
	}
	if !updateLibHouseUnlocked(libHouse3, csPopHuge, 0) {
		t.Fatal("mcHouse3 unlocked at POP_HUGE")
	}
	if !updateLibHouseUnlocked(libHouse3, 0, cityHugePop) {
		t.Fatal("mcHouse3 unlocked by city pop fallback")
	}
}

func TestGenMapPopKeepsDepositsOnceFarmsUnlock(t *testing.T) {
	low := genMapPop(1, newMapRNG("testcity"))
	if low.max != 0 {
		t.Fatalf("pop=1 densityMax=%d, want 0 (empty initial town)", low.max)
	}
	mid := genMapPop(3, newMapRNG("testcity"))
	if mid.max == 0 {
		t.Fatal("pop=3 should keep genMapPop deposits so neighbor farms can spawn")
	}
	high := genMapPop(40, newMapRNG("testcity"))
	if high.max == 0 {
		t.Fatal("pop=40 should have blurred density")
	}
}

func TestBlurSoftensDensityPeaks(t *testing.T) {
	src := make([][]int, 5)
	for y := 0; y < 5; y++ {
		src[y] = make([]int, 5)
	}
	src[2][2] = 100
	out := blurDensity(src)
	if out[2][2] >= 100 {
		t.Fatalf("blur should lower isolated peak, got %d", out[2][2])
	}
	if out[2][1] == 0 && out[1][2] == 0 {
		t.Fatal("blur should spread mass to neighbors")
	}
}

func countBuildings(t *testing.T, slug string, pop int) int {
	t.Helper()
	city := &citycore.City{Slug: citycore.Slug(slug), Pop: citycore.SectorValue(pop)}
	occ := lotOccupancy(city, buildCityGridForCity(city))
	n := 0
	for _, lot := range occ {
		if lot.use == lotBuilding {
			n++
		}
	}
	return n
}
