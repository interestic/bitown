package render

import (
	"strings"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestFolderUnlockSatisfied(t *testing.T) {
	u := FolderUnlock{MinPop: 120, MinInd: 50}
	city := &citycore.City{Pop: 150, Ind: 60}
	if !u.satisfied(city) {
		t.Fatal("expected unlock satisfied")
	}
	city.Pop = 100
	if u.satisfied(city) {
		t.Fatal("expected pop gate to fail")
	}
	city.Pop = 150
	city.Ind = 10
	if u.satisfied(city) {
		t.Fatal("expected ind gate to fail")
	}
}

func TestFilterBasesByUnlock(t *testing.T) {
	atlas := &Atlas{
		UnlockByFolder: map[string]FolderUnlock{
			"sprites/low":  {},
			"sprites/high": {MinPop: 350},
		},
	}
	bases := []string{
		"sprites/low/frame",
		"sprites/high/frame",
	}
	lowPop := &citycore.City{Pop: 100}
	filtered := filterBasesByUnlock(atlas, bases, lowPop)
	if len(filtered) != 1 || !strings.Contains(filtered[0], "/low/") {
		t.Fatalf("low pop should only unlock low base, got %v", filtered)
	}
	highPop := &citycore.City{Pop: 500}
	filtered = filterBasesByUnlock(atlas, bases, highPop)
	if len(filtered) != 2 {
		t.Fatalf("high pop should unlock both bases, got %v", filtered)
	}
}

func TestPickBuildingKeyRespectsUnlockAtLowPop(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	if len(atlas.UnlockByFolder) == 0 {
		t.Fatal("expected UnlockByFolder from buildings.json")
	}
	city := &citycore.City{Slug: "unlock-low", Pop: 20, Ind: 0, Com: 0, Env: 0}
	cx, cy := mapCols/2, mapRows/2
	key := atlas.PickBuildingKeyForLot(city, TagResidential, cx, cy, hashCell(city.Slug.String(), cx, cy))
	if key == "" {
		t.Fatal("expected a building at center peon pop")
	}
	if atlas.folderTier(spriteFolderBase(key)) >= 2 {
		t.Fatalf("pop=20 must not place tier>=2, got tier %d key %s", atlas.folderTier(spriteFolderBase(key)), key)
	}
}

func TestTreeUnlockVariesByEnv(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	low := &citycore.City{Pop: 500, Env: 0}
	high := &citycore.City{Pop: 500, Env: 400}
	seenLow := map[string]struct{}{}
	seenHigh := map[string]struct{}{}
	for i := 0; i < 32; i++ {
		seed := uint32(i * 991) //#nosec G115
		if k := atlas.PickKeyForTagUnlocked(low, TagTree, seed); k != "" {
			seenLow[spriteFolderBase(k)] = struct{}{}
		}
		if k := atlas.PickKeyForTagUnlocked(high, TagTree, seed); k != "" {
			seenHigh[spriteFolderBase(k)] = struct{}{}
		}
	}
	if len(seenHigh) <= len(seenLow) {
		t.Fatalf("env=400 should unlock more tree folders than env=0: low=%d high=%d", len(seenLow), len(seenHigh))
	}
}

func TestIndustrialUnlockNeedsInd(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	lowInd := &citycore.City{Slug: "rim", Pop: 500, Ind: 1, Com: 0}
	withInd := &citycore.City{Slug: "rim", Pop: 500, Ind: 50, Com: 0}
	rimX := 1
	if zoneTag(lowInd, rimX, mapRows/2) != TagIndustrial {
		t.Fatal("expected industrial rim with ind>0")
	}
	var tier2LowInd, tier2WithInd int
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if zoneTag(withInd, x, y) != TagIndustrial {
				continue
			}
			seed := hashCell(withInd.Slug.String(), x, y)
			k0 := atlas.PickBuildingKeyForLot(lowInd, TagIndustrial, x, y, seed)
			k1 := atlas.PickBuildingKeyForLot(withInd, TagIndustrial, x, y, seed)
			if k0 != "" && atlas.folderTier(spriteFolderBase(k0)) >= 2 {
				tier2LowInd++
			}
			if k1 != "" && atlas.folderTier(spriteFolderBase(k1)) >= 2 {
				tier2WithInd++
			}
		}
	}
	if tier2WithInd == 0 {
		t.Fatal("ind>=50 should place tier>=2 industrial on rim")
	}
	if tier2LowInd > 0 {
		t.Fatalf("ind=1 should not place tier>=2 industrial, got %d lots", tier2LowInd)
	}
}
