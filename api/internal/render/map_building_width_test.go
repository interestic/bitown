package render

import (
	"testing"
)

func TestBuildingPoolExcludesOversizedSingleLotFrames(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	if len(atlas.BuildingBases) == 0 {
		t.Fatal("expected non-empty narrow building pool")
	}
	if len(atlas.BasesForTag(TagResidential)) == 0 {
		t.Fatal("expected some residential frames within maxSingleLotBuildingW")
	}
	for _, base := range atlas.BuildingBases {
		if frameBaseWiderThan(base, atlas.Frames, maxSingleLotBuildingW) {
			t.Fatalf("oversized frame in building pool: %s", base)
		}
	}
	for _, tag := range []string{TagResidential, TagIndustrial, TagCommercial, TagLandmark} {
		for _, base := range atlas.BasesForTag(tag) {
			if frameBaseWiderThan(base, atlas.Frames, maxSingleLotBuildingW) {
				t.Fatalf("oversized frame in %s pool: %s", tag, base)
			}
		}
	}
}
