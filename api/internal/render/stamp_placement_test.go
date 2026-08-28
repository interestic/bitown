package render

import "testing"

func TestBuildingStampFootMiniFootSkipsArterialYard(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Skipf("atlas unavailable: %v", err)
	}
	key := "sprites/DefineSprite_269/1_v00.png"
	stamp, ok := atlas.StampForKey(key)
	if !ok || stamp.Kind != StampKindMiniFoot {
		t.Fatalf("269 stamp=%+v, want mini_foot", stamp)
	}
	bx, by := 20, 20
	ox, oy := miniSquareOrigin(bx, by, miniSE)
	lx, ly := miniHutFoot(ox, oy, 0, false)

	fx, fy := buildingStampFoot(atlas, key, lx, ly, false)
	ax, ay := overlayFoot(lx, ly, overlayLift(false))
	ax = applyWestMiniStampNudge(ax, lx, ly)
	ax = applyNorthMiniStampNudgeX(ax, lx, ly)
	ax = applyEastMiniStampNudge(ax, lx, ly)
	ay = applyNorthMiniStampNudge(ay, lx, ly)
	ay = applySEMiniStampNudge(ay, lx, ly)
	ay = applyEWMiniStampNudge(ay, lx, ly)
	ax, ay = applyArterialYardStampNudge(ax, ay, lx, ly)
	if fy >= ay {
		t.Fatalf("mini_foot fy=%d should be less than full arterial stack fy=%d", fy, ay)
	}
	_ = fx
	_ = ax
}

func TestBuildingStampFootArterialYardUsesLift(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Skipf("atlas unavailable: %v", err)
	}
	key := "sprites/DefineSprite_493/8_v00.png"
	stamp, ok := atlas.StampForKey(key)
	if !ok || stamp.Kind != StampKindArterialYard {
		t.Fatalf("493 stamp=%+v, want arterial_yard", stamp)
	}
	bx, by := 20, 20
	ox, oy := miniSquareOrigin(bx, by, miniSE)
	lx, ly := miniHutFoot(ox, oy, 0, false)
	_, baseY := overlayFoot(lx, ly, overlayLift(false))
	_, fy := buildingStampFoot(atlas, key, lx, ly, false)
	if fy <= baseY+arterialYardLiftY-2 {
		t.Fatalf("arterial_yard footY=%d, base=%d, want lift >= %d", fy, baseY, arterialYardLiftY)
	}
}
