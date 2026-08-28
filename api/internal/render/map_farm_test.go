package render

import (
	"image"
	"image/color"
	"image/draw"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestFarmsEnabledThreshold(t *testing.T) {
	if farmsEnabled(2) {
		t.Fatal("pop=2 should not enable farms")
	}
	if !farmsEnabled(3) {
		t.Fatal("pop=3 should enable farms (Townzzy)")
	}
}

func TestMiniSquareOriginMatchesDensityLayout(t *testing.T) {
	got := make([][2]int, 4)
	for i := 0; i < 4; i++ {
		ox, oy := miniSquareOrigin(10, 20, i)
		got[i] = [2]int{ox, oy}
	}
	want := [][2]int{{10, 20}, {15, 20}, {10, 25}, {15, 25}}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("mini %d = %v, want %v", i, got[i], want[i])
		}
	}
	fx, fy := miniSquareFoot(10, 20)
	if fx != 13 || fy != 23 {
		t.Fatalf("foot = (%d,%d), want (13,23) (SE of 4×4 mini)", fx, fy)
	}
	sx, sy := squareFarmFoot(10, 20)
	if sx != 19 || sy != 29 {
		t.Fatalf("square foot = (%d,%d), want (19,29)", sx, sy)
	}
}

func TestPickFarmKeyFromMiniPool(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	key := atlas.PickFarmKey(0)
	if key == "" {
		t.Fatal("expected farm key")
	}
	if _, ok := atlas.Frames[key]; !ok {
		t.Fatalf("farm key %q missing from atlas", key)
	}
	found := false
	for _, k := range farmMiniKeys {
		if k == key {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("PickFarmKey = %q, not in farmMiniKeys", key)
	}
	if a, b := atlas.PickFarmKey(7), atlas.PickFarmKey(7); a != b {
		t.Fatalf("PickFarmKey not deterministic: %q vs %q", a, b)
	}
}

func TestPickBigFarmKeyFromBigPool(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	key := atlas.PickBigFarmKey(0)
	if key == "" {
		t.Fatal("expected big farm key")
	}
	if _, ok := atlas.Frames[key]; !ok {
		t.Fatalf("big farm key %q missing from atlas", key)
	}
	if !inFarmKeyPool(key, farmBigKeys) {
		t.Fatalf("PickBigFarmKey = %q, not in farmBigKeys", key)
	}
	if a, b := atlas.PickBigFarmKey(7), atlas.PickBigFarmKey(7); a != b {
		t.Fatalf("PickBigFarmKey not deterministic: %q vs %q", a, b)
	}
}

func inFarmKeyPool(key string, pool []string) bool {
	for _, k := range pool {
		if k == key {
			return true
		}
	}
	return false
}

func TestFarmStampsAppearAtPop3NotPop1(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	slug := citycore.Slug("testcity")
	low := collectFarmStamps(newMapRenderCtx(&citycore.City{Slug: slug, Pop: 1, Ind: 1, Com: 1, Env: 1, Sec: 1}, atlas))
	mid := collectFarmStamps(newMapRenderCtx(&citycore.City{Slug: slug, Pop: 3, Ind: 1, Com: 1, Env: 1, Sec: 1}, atlas))
	if len(low) != 0 {
		t.Fatalf("pop=1 should have 0 farm stamps, got %d", len(low))
	}
	if len(mid) == 0 {
		t.Fatal("pop=3 roadless should stamp fringe or empty-mini farms")
	}
	if len(mid) >= 36 {
		t.Fatalf("pop=3 must not carpet all plates, got %d", len(mid))
	}
	again := collectFarmStamps(newMapRenderCtx(&citycore.City{Slug: slug, Pop: 3, Ind: 1, Com: 1, Env: 1, Sec: 1}, atlas))
	if len(again) != len(mid) {
		t.Fatalf("stamp count drift: %d vs %d", len(again), len(mid))
	}
	for i := range mid {
		if mid[i] != again[i] {
			t.Fatalf("stamp %d drifted: %+v vs %+v", i, mid[i], again[i])
		}
	}
}

func TestRoadlessFarmsSkipBuildingMinis(t *testing.T) {
	requireAtlasFiles(t)
	city := &citycore.City{Slug: "farm-clear-lot", Pop: 8}
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	ctx := newMapRenderCtx(city, atlas)
	if !ctx.roadless {
		t.Fatal("pop=8 should be roadless")
	}
	stamps := collectFarmStamps(ctx)
	if len(stamps) == 0 {
		farmCells := 0
		for _, lot := range ctx.occupancy {
			if lot.use == lotFarm {
				farmCells++
			}
		}
		if farmCells == 0 {
			t.Fatal("pop=8 roadless should still mark some lotFarm cover")
		}
	}
	for _, s := range stamps {
		if cellBlocksFarm(ctx, s.fx, s.fy) {
			t.Fatalf("farm stamp on building/park/road cell (%d,%d)", s.fx, s.fy)
		}
		if s.cells != 4 {
			continue
		}
		for dy := 0; dy < 4; dy++ {
			for dx := 0; dx < 4; dx++ {
				if cellBlocksFarm(ctx, s.ox+dx, s.oy+dy) {
					t.Fatalf("mini farm origin (%d,%d) covers blocked cell (%d,%d)", s.ox, s.oy, s.ox+dx, s.oy+dy)
				}
			}
		}
	}
}

// TestMiniFarmStampsStayInEmptyMinis locks farm-under-house: mini farm feet
// sit in the empty 4×4 and never on a sibling building mini. Empty minis may
// stamp even when another mini on the same square has a house (#118).
func TestMiniFarmStampsStayInEmptyMinis(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	city := &citycore.City{Slug: "testcity5", Pop: 40, Ind: 130, Com: 150, Env: 110, Sec: 100}
	ctx := newMapRenderCtx(city, atlas)
	stamps := collectFarmStamps(ctx)
	for _, s := range stamps {
		if s.cells != 4 {
			continue
		}
		if s.fx != s.ox+3 || s.fy != s.oy+3 {
			t.Fatalf("mini foot (%d,%d) want SE of origin (%d,%d)", s.fx, s.fy, s.ox, s.oy)
		}
		for dy := 0; dy < 4; dy++ {
			for dx := 0; dx < 4; dx++ {
				lot := ctx.occupancy[[2]int{s.ox + dx, s.oy + dy}]
				if lot.use == lotBuilding {
					t.Fatalf("mini farm origin (%d,%d) covers building at (%d,%d)", s.ox, s.oy, s.ox+dx, s.oy+dy)
				}
			}
		}
	}
	// Sibling empty minis should still stamp when the square has houses.
	hasMiniOnOccupied := false
	for _, s := range stamps {
		if s.cells != 4 {
			continue
		}
		sx, sy := squareOf(s.ox, s.oy)
		if squareHasBuilding(ctx.occupancy, sx, sy) {
			hasMiniOnOccupied = true
			break
		}
	}
	if !hasMiniOnOccupied {
		t.Fatal("expected at least one empty-mini farm on a square that also has houses")
	}
}

func TestArterialMiniFarmsStillSkipOccupiedSquares(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	city := &citycore.City{Slug: "testcity", Pop: 80, Ind: 130, Com: 150, Env: 110, Sec: 100}
	ctx := newMapRenderCtx(city, atlas)
	if ctx.roadless {
		t.Fatal("pop=80 should enable arterials")
	}
	for _, s := range collectFarmStamps(ctx) {
		if s.cells != 4 {
			continue
		}
		sx, sy := squareOf(s.ox, s.oy)
		if squareHasBuilding(ctx.occupancy, sx, sy) {
			t.Fatalf("arterial mini farm on occupied square (%d,%d)", sx, sy)
		}
	}
}

func TestRoadlessFarmsAreFringeNotCarpet(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	city := &citycore.City{Slug: "testcity", Pop: 40, Ind: 130, Com: 150, Env: 110, Sec: 100}
	ctx := newMapRenderCtx(city, atlas)
	if !ctx.roadless {
		t.Fatal("pop=40 should be roadless")
	}
	stamps := collectFarmStamps(ctx)
	plates := plateIslandExtent(ctx.pop) / groundBlock
	big, mini := 0, 0
	for _, s := range stamps {
		sx, sy := squareOf(s.fx, s.fy)
		switch {
		case inFarmKeyPool(s.key, farmBigKeys):
			big++
			if ctx.dens.at(sx, sy) != 0 {
				t.Fatalf("big farm on density %d square (%d,%d)", ctx.dens.at(sx, sy), sx, sy)
			}
			side := squareSidePop(ctx.dens, sx, sy)
			if side < farmBigSidePopMin || side >= farmBigSidePopMax {
				t.Fatalf("big farm sidePop=%d out of range at (%d,%d)", side, sx, sy)
			}
		case inFarmKeyPool(s.key, farmMiniKeys):
			mini++
			if ctx.dens.at(sx, sy) <= 0 {
				t.Fatalf("mini farm on density-0 square (%d,%d)", sx, sy)
			}
		default:
			t.Fatalf("unexpected farm key %q", s.key)
		}
	}
	// Carpet = every plate gets a full-square 401. Mini stamps can exceed plate count.
	if plates*plates >= 36 && big >= plates*plates {
		t.Fatalf("roadless must not big-farm every plate, got %d want < %d (mini=%d)", big, plates*plates, mini)
	}
	if big == 0 && mini == 0 {
		t.Fatal("pop=40 should place fringe big farms or empty-mini farms")
	}
	farEmpty := 0
	active := activeSquareSide(ctx.pop)
	origin := activeSquareOrigin(ctx.pop)
	for sy := origin; sy < origin+active; sy++ {
		for sx := origin; sx < origin+active; sx++ {
			if ctx.dens.at(sx, sy) != 0 {
				continue
			}
			side := squareSidePop(ctx.dens, sx, sy)
			if side >= farmBigSidePopMin {
				continue
			}
			farEmpty++
			for _, s := range stamps {
				ssx, ssy := squareOf(s.fx, s.fy)
				if ssx == sx && ssy == sy {
					t.Fatalf("farm on far-empty square (%d,%d) sidePop=%d", sx, sy, side)
				}
			}
		}
	}
	if farEmpty == 0 {
		t.Fatal("expected some far-empty squares with sidePop < 2")
	}
}

func TestBuildCityMapPNG_FarmsRender(t *testing.T) {
	requireAtlasFiles(t)
	data, err := BuildCityMapPNG(&citycore.City{Slug: "beach-farm-png", Pop: 4})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img := decodeMapPNG(t, data)
	if n := countUniqueColors(img); n < 30 {
		t.Fatalf("expected farm to enrich palette, got %d unique colors", n)
	}
}

func TestRoadlessFarmsStayOnDalleGrass(t *testing.T) {
	requireAtlasFiles(t)
	city := &citycore.City{Slug: "testcity", Pop: 40, Ind: 130, Com: 150, Env: 110, Sec: 100}
	img := mustBuildMapWorkingImage(t, city)
	grass := buildPlateGrass(40)
	sky := mapCanvasColor(true)
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if x >= 0 && x < mapWidth && grass.col[x] && y > grass.maxY[x]+plateGrassLift+2 {
				if isChampsFieldColor(c) {
					t.Fatalf("farm on plate soil/sky fringe at (%d,%d) %+v", x, y, c)
				}
			}
			if !grassTopPixelSupported(grass, x, y) && nearRGBA(c, sky, 8) {
				continue
			}
			if !grassTopPixelSupported(grass, x, y) && isChampsFieldColor(c) {
				// mcDalle olive fringe can sit outside the geometric mask.
				if grassNeighbor(grass, x, y, 6) {
					continue
				}
				t.Fatalf("farm spilled off island at (%d,%d) %+v", x, y, c)
			}
		}
	}
}

func grassNeighbor(g plateGrass, px, py, radius int) bool {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if grassTopPixelSupported(g, px+dx, py+dy) {
				return true
			}
		}
	}
	return false
}

func isChampsFieldColor(c color.RGBA) bool {
	if c.A < 200 {
		return false
	}
	// Yellow fill / pumpkin (401) — not plate grass or sky.
	if c.R > 180 && c.G > 140 && c.B < 120 && int(c.R) > int(c.B)+40 {
		return true
	}
	return false
}

func TestHighComMapDoesNotPlaceFarmClipsAsBuildings(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	city := &citycore.City{Slug: "testcity", Pop: 460, Ind: 130, Com: 150, Env: 110, Sec: 100}
	ctx := newMapRenderCtx(city, atlas)
	if !arterialsEnabled(city) {
		t.Fatal("repro city must have arterials")
	}
	if len(ctx.buildingKeys) == 0 {
		t.Fatal("expected commercial lots to pick buildings")
	}
	banned := map[string]bool{
		"sprites/DefineSprite_401": true,
		"sprites/DefineSprite_521": true,
	}
	for pos, key := range ctx.buildingKeys {
		if banned[spriteFolderBase(key)] {
			t.Fatalf("farm clip %q placed as building at %v", key, pos)
		}
	}
	stamps := collectFarmStamps(ctx)
	if len(stamps) == 0 {
		t.Fatal("high-com arterial map should still stamp empty-mini farms")
	}
	for _, s := range stamps {
		if ctx.grid[s.fy][s.fx] == cellRoad {
			t.Fatalf("farm stamp landed on road cell (%d,%d)", s.fx, s.fy)
		}
		if !inFarmKeyPool(s.key, farmMiniKeys) && !inFarmKeyPool(s.key, farmBigKeys) {
			t.Fatalf("arterial stamp %q not in farm pools", s.key)
		}
	}
}

func TestFarmStampKindsFollowDensity(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	roadless := collectFarmStamps(newMapRenderCtx(&citycore.City{Slug: "testcity", Pop: 40, Ind: 130, Com: 150, Env: 110, Sec: 100}, atlas))
	if len(roadless) == 0 {
		t.Fatal("pop=40 roadless should stamp some farms")
	}
	big := 0
	for _, s := range roadless {
		if inFarmKeyPool(s.key, farmBigKeys) {
			big++
		}
	}
	if big >= 36 {
		t.Fatalf("pop=40 roadless must not carpet big farms, got %d", big)
	}
	again := collectFarmStamps(newMapRenderCtx(&citycore.City{Slug: "testcity", Pop: 40, Ind: 130, Com: 150, Env: 110, Sec: 100}, atlas))
	if len(again) != len(roadless) {
		t.Fatalf("roadless stamp count drift: %d vs %d", len(again), len(roadless))
	}
	for i := range roadless {
		if roadless[i] != again[i] {
			t.Fatalf("roadless stamp %d drifted: %+v vs %+v", i, roadless[i], again[i])
		}
	}
	arterial := collectFarmStamps(newMapRenderCtx(&citycore.City{Slug: "testcity", Pop: 80, Ind: 130, Com: 150, Env: 110, Sec: 100}, atlas))
	if len(arterial) == 0 {
		t.Fatal("pop=80 arterial should stamp some empty-mini or fringe farms")
	}
}

func TestArterialFarmsKeepFullSoilLipLift(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	city := &citycore.City{Slug: "testcity5", Pop: 90, Ind: 330, Com: 150, Env: 110, Sec: 100}
	ctx := newMapRenderCtx(city, atlas)
	if ctx.roadless {
		t.Fatal("pop=90 must enable arterials")
	}
	stamps := collectFarmStamps(ctx)
	if len(stamps) == 0 {
		t.Fatal("expected arterial farms")
	}
	img := mustBuildMapWorkingImage(t, city)
	// North tip band of the full-lip diamond should still show yellow farm fill.
	hits, checked := 0, 0
	for _, s := range stamps {
		n := s.cells
		if n <= 0 {
			n = 4
		}
		topX, topY := isoCell(s.ox, s.oy)
		y := topY - farmGrassLift + 4
		if !pointInIsoBlockOffset(topX, y, s.ox, s.oy, n, -farmGrassLift, 0) {
			continue
		}
		checked++
		c := img.RGBAAt(topX, y)
		if isChampsFieldColor(c) || isYellowFarmPixel(c) {
			hits++
		}
	}
	if checked == 0 {
		t.Fatal("expected farm diamonds near the full-lip north tip")
	}
	if hits < 3 {
		t.Fatalf("arterial yellow farms missing farmGrassLift band: %d/%d", hits, checked)
	}
}

func TestBigFarmEligibilityGates(t *testing.T) {
	dens := uniformDensity(0)
	dens.cells[2][2] = 5 // at(sx=2,sy=2)
	dens.cells[2][4] = 4 // at(sx=4,sy=2) — east neighbor of (3,2)
	ctx := mapRenderCtx{
		dens:      dens,
		grid:      emptyRoadPlan().grid,
		occupancy: map[[2]int]lotCell{},
	}
	if side := squareSidePop(dens, 3, 2); side != 9 {
		t.Fatalf("sidePop=%d, want 9", side)
	}
	if !bigFarmEligible(ctx, 3, 2) {
		t.Fatal("empty square with sidePop=9 and no roads should allow big farm")
	}
	if bigFarmEligible(ctx, 2, 2) {
		t.Fatal("density>0 square must not get big farm")
	}
	ctx.grid[2*squareSide][3*squareSide] = cellRoad
	if bigFarmEligible(ctx, 3, 2) {
		t.Fatal("road on square must block big farm")
	}
	ctx.grid[2*squareSide][3*squareSide] = cellLot
}

func TestSquareMiniPopsIsDeterministic(t *testing.T) {
	a := squareMiniPops("testcity", 2, 3, 7)
	b := squareMiniPops("testcity", 2, 3, 7)
	if a != b {
		t.Fatalf("drift %v vs %v", a, b)
	}
	sum := 0
	for _, n := range a {
		sum += n
	}
	if sum != 7 {
		t.Fatalf("mini pops sum %d, want 7", sum)
	}
	if squareMiniPops("testcity", 2, 3, 0) != [4]int{} {
		t.Fatal("zero density should be empty minis")
	}
}

func TestPop3FarmStampSnapshot(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	city := &citycore.City{Slug: "testcity", Pop: 3, Ind: 1, Com: 1, Env: 1, Sec: 1}
	stamps := collectFarmStamps(newMapRenderCtx(city, atlas))
	got := make([][3]int, len(stamps))
	for i, s := range stamps {
		kind := 0
		switch {
		case inFarmKeyPool(s.key, farmMiniKeys):
			kind = 0
		case inFarmKeyPool(s.key, farmBigKeys):
			kind = 1
		default:
			t.Fatalf("unexpected key %q", s.key)
		}
		got[i] = [3]int{s.fx, s.fy, kind}
	}
	want := [][3]int{
		{138, 123, 0},
		{133, 128, 0},
		{138, 128, 0},
		{129, 139, 1},
		{133, 138, 0},
		{138, 138, 0},
		{149, 139, 1},
		{139, 149, 1},
	}
	if len(got) != len(want) {
		t.Fatalf("pop=3 stamp count %d, want %d (carpet=36): %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("stamp %d = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestTreesStayOffFarmCover(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	city := &citycore.City{Slug: "testcity", Pop: 40, Ind: 130, Com: 150, Env: 110, Sec: 100}
	ctx := newMapRenderCtx(city, atlas)
	stamps := collectFarmStamps(ctx)
	if len(stamps) == 0 {
		t.Fatal("expected farm stamps for pop=40")
	}
	parks := 0
	for _, lot := range ctx.occupancy {
		if lot.use == lotPark {
			parks++
		}
	}
	if parks == 0 {
		t.Fatal("env=110 should still place trees on vacant grass")
	}
	for _, s := range stamps {
		sx, sy := squareOf(s.fx, s.fy)
		switch {
		case inFarmKeyPool(s.key, farmBigKeys):
			x0, y0 := sx*squareSide, sy*squareSide
			for y := y0; y < y0+squareSide; y++ {
				for x := x0; x < x0+squareSide; x++ {
					lot, ok := ctx.occupancy[[2]int{x, y}]
					if !ok {
						continue
					}
					if lot.use == lotPark {
						t.Fatalf("tree on big-farm square (%d,%d) cell (%d,%d)", sx, sy, x, y)
					}
					if lot.use != lotEmpty && lot.use != lotFarm && lot.use != lotBuilding {
						t.Fatalf("unexpected use %v on farm square cell (%d,%d)", lot.use, x, y)
					}
				}
			}
		case inFarmKeyPool(s.key, farmMiniKeys):
			ox, oy := s.ox, s.oy
			if ox == 0 && oy == 0 {
				ox, oy = s.fx-3, s.fy-3
			}
			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 4; dx++ {
					assertNoParkOnFarmCell(t, ctx.occupancy, ox+dx, oy+dy)
				}
			}
		}
	}
}

func assertNoParkOnFarmCell(t *testing.T, occ map[[2]int]lotCell, x, y int) {
	t.Helper()
	lot, ok := occ[[2]int{x, y}]
	if !ok {
		return
	}
	if lot.use == lotPark {
		t.Fatalf("tree on mini-farm cover cell (%d,%d)", x, y)
	}
}

func TestFarmPickOmitsDecorFrames(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	decor := map[string]struct{}{
		"sprites/DefineSprite_521/5_v00.png": {},
		"sprites/DefineSprite_401/1_v00.png": {},
	}
	for seed := uint32(0); seed < 64; seed++ {
		mini := atlas.PickFarmKey(seed)
		big := atlas.PickBigFarmKey(seed)
		if _, bad := decor[mini]; bad {
			t.Fatalf("PickFarmKey(%d)=%q is decor frame", seed, mini)
		}
		if _, bad := decor[big]; bad {
			t.Fatalf("PickBigFarmKey(%d)=%q is decor frame", seed, big)
		}
		if mini == "" || big == "" {
			t.Fatalf("empty farm pick at seed %d mini=%q big=%q", seed, mini, big)
		}
	}
}

func TestParksStayOffFarmNeighbors(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	city := &citycore.City{Slug: "testcity7", Pop: 40, Ind: 130, Com: 150, Env: 110, Sec: 100}
	ctx := newMapRenderCtx(city, atlas)
	parks := 0
	for pos, lot := range ctx.occupancy {
		if lot.use != lotPark {
			continue
		}
		parks++
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				n, ok := ctx.occupancy[[2]int{pos[0] + dx, pos[1] + dy}]
				if ok && n.use == lotFarm {
					t.Fatalf("park@(%d,%d) touches farm@(%d,%d)", pos[0], pos[1], pos[0]+dx, pos[1]+dy)
				}
			}
		}
	}
	if parks == 0 {
		t.Fatal("env=110 should still place some trees away from farms")
	}
}

func TestMiniFarmPixelsStayOffBuildingFeet(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	// Inset hut feet (#118) keep house samples outside neighbor farm diamonds
	// without a punch-out around sprites.
	for _, slug := range []string{"testcity1", "testcity5"} {
		city := &citycore.City{Slug: citycore.Slug(slug), Pop: 40, Ind: 130, Com: 150, Env: 110, Sec: 100}
		ctx := newMapRenderCtx(city, atlas)
		stamps := collectFarmStamps(ctx)
		hits := 0
		for _, lot := range ctx.occupancy {
			if lot.use != lotBuilding {
				continue
			}
			topX, topY := isoCell(lot.x, lot.y)
			footX := topX
			footY := topY + isoTileH - farmGrassLift
			dy := -farmGrassLift
			for _, d := range [][2]int{{0, -3}, {0, -6}, {-4, -5}, {4, -5}, {0, -1}} {
				px, py := footX+d[0], footY+d[1]
				for _, s := range stamps {
					n := s.cells
					if n <= 0 {
						n = 4
					}
					if pointInIsoBlockOffset(px, py, s.ox, s.oy, n, dy, 0) {
						hits++
						t.Logf("%s building (%d,%d) sample (%d,%d) still inside farm %s", slug, lot.x, lot.y, px, py, s.key)
					}
				}
			}
		}
		if hits != 0 {
			t.Fatalf("%s: farm mask still covers house feet samples=%d", slug, hits)
		}
	}
}

func TestYellowFarmNotUnderBuildingSprites(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatal(err)
	}
	for _, slug := range []string{"testcity1", "testcity5"} {
		city := &citycore.City{Slug: citycore.Slug(slug), Pop: 40, Ind: 130, Com: 150, Env: 110, Sec: 100}
		ctx := newMapRenderCtx(city, atlas)
		floor := image.NewRGBA(image.Rect(0, 0, mapWidth, mapHeight))
		draw.Draw(floor, floor.Bounds(), &image.Uniform{C: mapCanvasColor(ctx.roadless)}, image.Point{}, draw.Src)
		drawMapFloor(floor, ctx)
		full := mustBuildMapWorkingImage(t, city)

		hits := 0
		// Opaque sprite pixels only: AABB padding beside a house can correctly
		// show neighbor farm now that we do not punch (#118).
		for _, lot := range ctx.occupancy {
			if lot.use != lotBuilding {
				continue
			}
			key := ctx.buildingKeys[[2]int{lot.x, lot.y}]
			rect, ok := ctx.atlas.Frames[key]
			if !ok {
				continue
			}
			stamp, _ := ctx.atlas.StampForKey(key)
			if stamp.Kind == StampKindMiniFoot {
				continue
			}
			footX, footY := buildingStampFoot(ctx.atlas, key, lot.x, lot.y, ctx.roadless)
			dstX, dstY := footX-rect.AnchorX, footY-rect.AnchorY
			for sy := 0; sy < rect.H; sy++ {
				py := dstY + sy
				if !inBuildingGroundBand(footY, py) {
					continue
				}
				for sx := 0; sx < rect.W; sx++ {
					px := dstX + sx
					if px < 0 || py < 0 || px >= mapWidth || py >= mapHeight {
						continue
					}
					if atlasFrameAlpha(ctx.atlas, key, sx, sy) == 0 {
						continue
					}
					if !isYellowFarmPixel(floor.RGBAAt(px, py)) {
						continue
					}
					if isYellowFarmPixel(full.RGBAAt(px, py)) {
						hits++
					}
				}
			}
		}
		if hits > 0 {
			t.Fatalf("%s: yellow farm visible under building sprites: %d px", slug, hits)
		}
	}
}

func mustBuildMapPNG(t *testing.T, city *citycore.City) []byte {
	t.Helper()
	data, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// mustBuildMapWorkingImage returns the pre-fit working canvas (isoCell space).
func mustBuildMapWorkingImage(t *testing.T, city *citycore.City) *image.RGBA {
	t.Helper()
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	img, _, err := renderMapTilesImage(city, atlas, func(dst *image.RGBA, cellSeed uint32, tag string, footX, footY, lotX, lotY int, roads roadMaskData) {
		key := atlas.PickBuildingKeyForLot(city, tag, lotX, lotY, cellSeed)
		if key == "" || !atlas.drawBuildingAtFoot(dst, key, footX, footY, lotX, lotY, roads, plateGrass{}) {
			drawFallbackBuildingClipped(dst, cellSeed, footX, footY, lotX, lotY, roads)
		}
	})
	if err != nil {
		t.Fatalf("renderMapTilesImage: %v", err)
	}
	return img
}

func isYellowFarmPixel(c color.RGBA) bool {
	return c.A > 200 && c.R > 220 && c.G > 160 && c.B < 80
}

func atlasFrameRGBA(a *Atlas, key string, sx, sy int) color.RGBA {
	if a == nil {
		return color.RGBA{}
	}
	rect, ok := a.Frames[key]
	if !ok {
		return color.RGBA{}
	}
	src, hasRGBA := a.Image.(interface {
		RGBAAt(x, y int) color.RGBA
	})
	if hasRGBA {
		return src.RGBAAt(rect.X+sx, rect.Y+sy)
	}
	r, g, b, alpha := a.Image.At(rect.X+sx, rect.Y+sy).RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(alpha >> 8)} //#nosec G115 -- RGBA 16-bit channel
}

func atlasFrameAlpha(a *Atlas, key string, sx, sy int) uint8 {
	return atlasFrameRGBA(a, key, sx, sy).A
}
