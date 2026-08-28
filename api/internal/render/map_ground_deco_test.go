package render

import (
	"strings"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestGroundDecoStampsStayOffRoadsAndRim(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "testcity1", Pop: 370, Ind: 230, Com: 150, Env: 210, Sec: 330}
	ctx := newMapRenderCtx(city, atlas)
	stamps := collectGroundDecoStamps(ctx)
	if len(stamps) == 0 {
		t.Fatal("env=210 should place some ground deco stamps")
	}
	mini := make(map[string]struct{}, len(groundDecoMiniKeys))
	for _, k := range groundDecoMiniKeys {
		mini[k] = struct{}{}
	}
	scat := make(map[string]struct{}, len(groundDecoScatterKeys))
	for _, k := range groundDecoScatterKeys {
		scat[k] = struct{}{}
	}
	sawMini, sawScat := false, false
	for _, s := range stamps {
		if ctx.grid[s.fy][s.fx] == cellRoad {
			t.Fatalf("ground deco on road cell (%d,%d) key=%q", s.fx, s.fy, s.key)
		}
		if !grassTopCell(ctx.pop, s.fx, s.fy) {
			t.Fatalf("ground deco on plate rim (%d,%d) key=%q", s.fx, s.fy, s.key)
		}
		lot, ok := ctx.occupancy[[2]int{s.fx, s.fy}]
		if !ok {
			t.Fatalf("ground deco foot missing occupancy (%d,%d)", s.fx, s.fy)
		}
		if lot.use == lotBuilding || lot.use == lotFarm {
			t.Fatalf("ground deco foot on %v (%d,%d) key=%q", lot.use, s.fx, s.fy, s.key)
		}
		if _, isMini := mini[s.key]; isMini && lot.use != lotEmpty {
			t.Fatalf("mini stamp foot not lotEmpty (%d,%d) use=%v", s.fx, s.fy, lot.use)
		}
		n := s.cells
		if n <= 0 {
			n = 1
		}
		for dy := 0; dy < n; dy++ {
			for dx := 0; dx < n; dx++ {
				x, y := s.ox+dx, s.oy+dy
				if ctx.grid[y][x] == cellRoad {
					t.Fatalf("ground deco covers road (%d,%d) key=%q", x, y, s.key)
				}
				cell, ok := ctx.occupancy[[2]int{x, y}]
				if ok && (cell.use == lotBuilding || cell.use == lotFarm) {
					t.Fatalf("ground deco covers %v at (%d,%d) key=%q", cell.use, x, y, s.key)
				}
				if _, isMini := mini[s.key]; isMini && ok && cell.use != lotEmpty {
					t.Fatalf("mini covers non-empty %v at (%d,%d)", cell.use, x, y)
				}
			}
		}
		if _, ok := mini[s.key]; ok {
			sawMini = true
			if s.cells != 4 {
				t.Fatalf("mini stamp cells=%d, want 4 key=%q", s.cells, s.key)
			}
		}
		if _, ok := scat[s.key]; ok {
			sawScat = true
			if s.cells != 1 {
				t.Fatalf("scatter stamp cells=%d, want 1 key=%q", s.cells, s.key)
			}
		}
	}
	if !sawMini {
		t.Fatal("expected some mini stamps (516 fence / 616 tennis / …)")
	}
	if !sawScat {
		t.Fatal("expected some 514 pebble scatter stamps")
	}
}

func TestGroundDecoFencePlotsUseMiniFootprint(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "testcity6", Pop: 150, Ind: 30, Com: 25, Env: 120, Sec: 40}
	ctx := newMapRenderCtx(city, atlas)
	stamps := collectGroundDecoStamps(ctx)
	fence := 0
	for _, s := range stamps {
		switch {
		case strings.Contains(s.key, "DefineSprite_516"),
			strings.Contains(s.key, "DefineSprite_518"),
			strings.Contains(s.key, "DefineSprite_520"),
			strings.Contains(s.key, "DefineSprite_521/6"),
			strings.Contains(s.key, "DefineSprite_521/7"),
			strings.Contains(s.key, "DefineSprite_521/8"):
			if s.cells != 4 {
				t.Fatalf("fence plot %q must use 4-cell clip, got cells=%d", s.key, s.cells)
			}
			fence++
		}
	}
	if fence < 2 {
		t.Fatalf("expected several fenced plot stamps (516/518/520/521), got %d of %d", fence, len(stamps))
	}
}

func TestGroundDecoVisibleAtModestEnv(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "testcity6", Pop: 50, Ind: 30, Com: 25, Env: 20, Sec: 40}
	ctx := newMapRenderCtx(city, atlas)
	stamps := collectGroundDecoStamps(ctx)
	mini, scat := 0, 0
	miniKeys := make(map[string]struct{}, len(groundDecoMiniKeys))
	for _, k := range groundDecoMiniKeys {
		miniKeys[k] = struct{}{}
	}
	for _, s := range stamps {
		if _, ok := miniKeys[s.key]; ok {
			mini++
		} else {
			scat++
		}
	}
	if mini < 3 {
		t.Fatalf("env=20 should place several mini stamps, got %d (total %d)", mini, len(stamps))
	}
	if scat > mini*2+10 {
		t.Fatalf("514 scatter must not dominate minis: scatter=%d mini=%d", scat, mini)
	}
}

func TestGroundDecoMiniClipKeepsNECorner(t *testing.T) {
	// Tight n=4 diamond shaves 521/5 / fence art; expand must keep NE pixels.
	ox, oy := 10, 10
	n, dy := 4, -farmGrassLift
	topX, topY := isoCell(ox, oy)
	topY += dy
	h := n * isoTileH
	w := n * isoTileW
	// A point near the NE tip, slightly outside the unexpanded diamond.
	px := topX + w/2 + 6
	py := topY + h/4
	if pointInIsoBlockOffset(px, py, ox, oy, n, dy, 0) {
		t.Fatal("expected NE probe outside tight diamond")
	}
	if !pointInIsoBlockOffset(px, py, ox, oy, n, dy, 0.55) {
		t.Fatalf("expand=0.55 should keep NE probe (%d,%d) inside deco clip", px, py)
	}
}

func TestGroundDecoSkippedAtZeroEnv(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "testcity1", Pop: 370, Ind: 230, Com: 150, Env: 0, Sec: 330}
	ctx := newMapRenderCtx(city, atlas)
	if stamps := collectGroundDecoStamps(ctx); len(stamps) != 0 {
		t.Fatalf("env=0 must not stamp ground deco, got %d", len(stamps))
	}
}

func TestParkObjectsAreTreesOnly(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "testcity7", Pop: 120, Env: 200, Ind: 50, Com: 50}
	ctx := newMapRenderCtx(city, atlas)
	objs := collectMapObjects(ctx)
	banned := make(map[string]struct{}, len(groundDecoMiniKeys)+len(groundDecoScatterKeys))
	for _, k := range groundDecoMiniKeys {
		banned[k] = struct{}{}
	}
	for _, k := range groundDecoScatterKeys {
		banned[k] = struct{}{}
	}
	parks := 0
	for _, obj := range objs {
		if obj.kind != objectPark {
			continue
		}
		parks++
		if _, bad := banned[obj.key]; bad {
			t.Fatalf("park object used ground deco key %q (must be TagTree)", obj.key)
		}
	}
	if parks == 0 {
		t.Fatal("expected park lots at env=200")
	}
}
