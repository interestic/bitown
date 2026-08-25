package render

import "testing"

func TestMapDrawOrderDepthSorted(t *testing.T) {
	order := mapDrawOrder()
	if len(order) != mapCols*mapRows {
		t.Fatalf("expected %d cells, got %d", mapCols*mapRows, len(order))
	}
	for i := 1; i < len(order); i++ {
		a, b := order[i-1], order[i]
		da, db := a.x+a.y, b.x+b.y
		if da > db {
			t.Fatalf("depth regressed at %d: (%d,%d) after (%d,%d)", i, b.x, b.y, a.x, a.y)
		}
		if da == db && a.y > b.y {
			t.Fatalf("tie depth y-order at %d: (%d,%d) after (%d,%d)", i, b.x, b.y, a.x, a.y)
		}
		if da == db && a.y == b.y && a.x > b.x {
			t.Fatalf("tie depth x-order at %d: (%d,%d) after (%d,%d)", i, b.x, b.y, a.x, a.y)
		}
	}
}

func TestSortMapObjectsByDepthThenHeight(t *testing.T) {
	objs := []mapObject{
		{x: 2, y: 0, depth: 2, height: 40},
		{x: 1, y: 1, depth: 2, height: 10},
		{x: 0, y: 0, depth: 0, height: 50},
		{x: 0, y: 2, depth: 2, height: 10},
	}
	sortMapObjects(objs)
	if objs[0].depth != 0 {
		t.Fatalf("expected farthest depth first, got depth=%d", objs[0].depth)
	}
	if objs[1].height != 10 || objs[2].height != 10 || objs[3].height != 40 {
		t.Fatalf("expected shorter sprites before taller on same depth: %+v", objs)
	}
	if objs[3].height != 40 {
		t.Fatalf("taller same-depth sprite should paint last, got %+v", objs[3])
	}
}

func TestBuildingPoolExcludesWaterAndTrees(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	for _, base := range atlas.BuildingBases {
		folder := spriteFolderBase(base)
		for _, tag := range []string{TagWater, TagTree} {
			for _, tagged := range atlas.BasesForTag(tag) {
				if spriteFolderBase(tagged) == folder {
					t.Fatalf("building pool contains %s tagged %s", base, tag)
				}
			}
		}
	}
	if len(atlas.BasesForTag(TagTree)) == 0 {
		t.Fatal("expected tree-tagged bases for parks")
	}
}
