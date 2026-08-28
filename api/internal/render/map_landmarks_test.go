package render

import (
	"strings"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestLandmarkStampFootMatchesSandbox(t *testing.T) {
	if landmarkStampLocal != 4 {
		t.Fatalf("landmarkStampLocal=%d, want 4", landmarkStampLocal)
	}
	if landmarkStampNudgeY != 62 {
		t.Fatalf("landmarkStampNudgeY=%d, want 62", landmarkStampNudgeY)
	}
	x, y := 10, 20
	gotX, gotY := landmarkStampFoot(x, y)
	wantX, wantY := overlayFoot(x, y, plateGrassLift)
	wantY += landmarkStampNudgeY
	if gotX != wantX || gotY != wantY {
		t.Fatalf("landmarkStampFoot=(%d,%d), want (%d,%d)", gotX, gotY, wantX, wantY)
	}
}

func TestPlanSquareLandmarksLowPopNone(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Skipf("atlas unavailable: %v", err)
	}
	city := &citycore.City{Slug: "lm-low", Pop: 20}
	dens := genMapPop(city.Pop.Int(), newMapRNG(city.Slug.String()))
	stamps, squares := planSquareLandmarks(city, atlas, dens)
	if len(stamps) != 0 || len(squares) != 0 {
		t.Fatalf("low pop landmarks=%d squares=%d, want none", len(stamps), len(squares))
	}
}

func TestPlanSquareLandmarksHighPopPlacesSome(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Skipf("atlas unavailable: %v", err)
	}
	city := &citycore.City{Slug: "lm-high", Pop: 500, Ind: 50, Com: 50, Sec: 300, Env: 400}
	dens := genMapPop(city.Pop.Int(), newMapRNG(city.Slug.String()))
	stamps, squares := planSquareLandmarks(city, atlas, dens)
	if len(stamps) == 0 {
		t.Fatal("high pop+sec should place at least one square landmark")
	}
	if len(squares) != len(stamps) {
		t.Fatalf("squares=%d stamps=%d", len(squares), len(stamps))
	}
	for _, s := range stamps {
		if !strings.Contains(s.key, "DefineSprite_") {
			t.Fatalf("unexpected landmark key %q", s.key)
		}
		if !inPlateIsland(city.Pop.Int(), s.x, s.y) {
			t.Fatalf("landmark (%d,%d) off island", s.x, s.y)
		}
		if s.x%squareSide != landmarkStampLocal || s.y%squareSide != landmarkStampLocal {
			t.Fatalf("landmark cell=(%d,%d) not square local %d", s.x, s.y, landmarkStampLocal)
		}
		if !squareHasLandmark(squares, s.x, s.y) {
			t.Fatalf("square map missing (%d,%d)", s.x, s.y)
		}
	}
	stamps2, _ := planSquareLandmarks(city, atlas, dens)
	if len(stamps2) != len(stamps) {
		t.Fatalf("planSquareLandmarks not deterministic: %d vs %d", len(stamps2), len(stamps))
	}
	for i := range stamps {
		if stamps2[i] != stamps[i] {
			t.Fatalf("planSquareLandmarks stamp %d changed", i)
		}
	}
}

func TestCollectMapObjectsSkipsLotsOnLandmarkSquares(t *testing.T) {
	grid := make(cityGrid, mapRows)
	for y := range grid {
		grid[y] = make([]int, mapCols)
	}
	lx, ly := 10+landmarkStampLocal, 20+landmarkStampLocal
	ctx := mapRenderCtx{
		pop:  500,
		grid: grid,
		occupancy: map[[2]int]lotCell{
			{lx, ly}:     {use: lotPark},
			{lx + 1, ly}: {use: lotBuilding, tag: TagResidential},
		},
		order:      []mapCoord{{lx, ly}, {lx + 1, ly}},
		landmarkSq: map[[2]int]struct{}{{1, 2}: {}},
		landmarks:  []landmarkStamp{{x: lx, y: ly, key: "sprites/DefineSprite_692/1_v00.png"}},
	}
	objs := collectMapObjects(ctx)
	if len(objs) != 1 || objs[0].kind != objectLandmark {
		t.Fatalf("objs=%+v, want one landmark", objs)
	}
}

func TestSquareHasLandmark(t *testing.T) {
	if squareHasLandmark(nil, 4, 4) {
		t.Fatal("empty map must not match")
	}
	m := map[[2]int]struct{}{{1, 2}: {}}
	if !squareHasLandmark(m, 10+landmarkStampLocal, 20+landmarkStampLocal) {
		t.Fatal("want hit on square 1,2")
	}
	if squareHasLandmark(m, 0, 0) {
		t.Fatal("square 0,0 should miss")
	}
}
