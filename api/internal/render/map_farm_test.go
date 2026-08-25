package render

import (
	"image"
	"image/color"
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
	if fx != 14 || fy != 24 {
		t.Fatalf("foot = (%d,%d), want (14,24)", fx, fy)
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

func TestFarmStampsAppearAtPop4NotPop1(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	slug := citycore.Slug("beach-farm")
	low := collectFarmStamps(newMapRenderCtx(&citycore.City{Slug: slug, Pop: 1}, atlas))
	high := collectFarmStamps(newMapRenderCtx(&citycore.City{Slug: slug, Pop: 4}, atlas))
	if len(low) != 0 {
		t.Fatalf("pop=1 should have 0 farm stamps, got %d", len(low))
	}
	if len(high) < 8 {
		t.Fatalf("pop=4 peon should stamp many empty plates, got %d", len(high))
	}
	for _, s := range high {
		if !inFarmKeyPool(s.key, farmBigKeys) {
			t.Fatalf("pop=4 peon stamp %q not in farmBigKeys", s.key)
		}
	}
	// Deterministic keys for the same feet.
	again := collectFarmStamps(newMapRenderCtx(&citycore.City{Slug: slug, Pop: 4}, atlas))
	if len(again) != len(high) {
		t.Fatalf("stamp count drift: %d vs %d", len(again), len(high))
	}
	for i := range high {
		if high[i] != again[i] {
			t.Fatalf("stamp %d drifted: %+v vs %+v", i, high[i], again[i])
		}
	}
}

func TestPeonFarmFloorCoversBuildingPlates(t *testing.T) {
	requireAtlasFiles(t)
	city := &citycore.City{Slug: "farm-clear-lot", Pop: 8}
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	ctx := newMapRenderCtx(city, atlas)
	if !ctx.peon {
		t.Fatal("pop=8 should be peon")
	}
	stamps := collectFarmStamps(ctx)
	stampPlate := map[[2]int]farmStamp{}
	for _, s := range stamps {
		px, py := peonPlateOfFor(ctx.pop, s.fx, s.fy)
		stampPlate[[2]int{px, py}] = s
	}
	buildings := 0
	for pos, lot := range ctx.occupancy {
		if lot.use != lotBuilding {
			continue
		}
		buildings++
		px, py := peonPlateOfFor(ctx.pop, pos[0], pos[1])
		if _, ok := stampPlate[[2]int{px, py}]; !ok {
			t.Fatalf("occupied plate (%d,%d) should still get farm floor under building at %v", px, py, pos)
		}
	}
	if buildings == 0 {
		t.Fatal("pop=8 peon should place some buildings")
	}
	if len(stamps) == 0 {
		t.Fatal("pop=8 peon should stamp farm floor on plates")
	}
}

func TestPeonFarmsFillInteriorPlates(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	city := &citycore.City{Slug: "testcity", Pop: 40, Ind: 130, Com: 150, Env: 110, Sec: 100}
	ctx := newMapRenderCtx(city, atlas)
	if !ctx.peon {
		t.Fatal("pop=40 should be peon")
	}
	stamps := collectFarmStamps(ctx)
	side := peonIslandExtentFor(ctx.pop) / groundBlock
	if side < 3 {
		t.Fatalf("expected at least 3×3 plates, got %d", side)
	}
	want := side * side
	if len(stamps) != want {
		t.Fatalf("peon should stamp every plate, got %d want %d", len(stamps), want)
	}
	inner := 0
	for _, s := range stamps {
		px, py := peonPlateOfFor(ctx.pop, s.fx, s.fy)
		if px > 0 && py > 0 && px < side-1 && py < side-1 {
			inner++
		}
	}
	innerWant := (side - 2) * (side - 2)
	if inner != innerWant {
		t.Fatalf("interior plates should get big champs, got %d want %d (total %d)", inner, innerWant, len(stamps))
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
		t.Fatalf("expected champs to enrich palette, got %d unique colors", n)
	}
}

func TestPeonFarmsStayOnDalleGrass(t *testing.T) {
	requireAtlasFiles(t)
	data, err := BuildCityMapPNG(&citycore.City{Slug: "testcity", Pop: 40, Ind: 130, Com: 150, Env: 110, Sec: 100})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img, ok := decodeMapPNG(t, data).(*image.RGBA)
	if !ok {
		t.Fatal("expected RGBA map")
	}
	grass := buildPeonGrass(40)
	sky := mapCanvasColor(true)
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if x >= 0 && x < mapWidth && grass.col[x] && y > grass.maxY[x]+2 {
				if isChampsFieldColor(c) {
					t.Fatalf("champs on dalle soil/sky fringe at (%d,%d) %+v", x, y, c)
				}
			}
			if !peonPixelSupported(grass, x, y) && nearRGBA(c, sky, 8) {
				continue
			}
			if !peonPixelSupported(grass, x, y) && isChampsFieldColor(c) {
				t.Fatalf("champs spilled off island at (%d,%d) %+v", x, y, c)
			}
		}
	}
}

func isChampsFieldColor(c color.RGBA) bool {
	if c.A < 200 {
		return false
	}
	// Yellow fill / pumpkin (401) — not dalle grass or sky.
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
		if !inFarmKeyPool(s.key, farmMiniKeys) {
			t.Fatalf("arterial stamp %q not in farmMiniKeys", s.key)
		}
	}
}

func TestPeonUsesBigFarmArterialUsesMini(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	peon := collectFarmStamps(newMapRenderCtx(&citycore.City{Slug: "testcity", Pop: 40, Ind: 130, Com: 150, Env: 110, Sec: 100}, atlas))
	if len(peon) == 0 {
		t.Fatal("pop=40 peon should stamp empty plates with big champs")
	}
	for _, s := range peon {
		if !inFarmKeyPool(s.key, farmBigKeys) {
			t.Fatalf("peon stamp %q not in farmBigKeys", s.key)
		}
	}
	again := collectFarmStamps(newMapRenderCtx(&citycore.City{Slug: "testcity", Pop: 40, Ind: 130, Com: 150, Env: 110, Sec: 100}, atlas))
	if len(again) != len(peon) {
		t.Fatalf("peon stamp count drift: %d vs %d", len(again), len(peon))
	}
	for i := range peon {
		if peon[i] != again[i] {
			t.Fatalf("peon stamp %d drifted: %+v vs %+v", i, peon[i], again[i])
		}
	}
	arterial := collectFarmStamps(newMapRenderCtx(&citycore.City{Slug: "testcity", Pop: 80, Ind: 130, Com: 150, Env: 110, Sec: 100}, atlas))
	if len(arterial) == 0 {
		t.Fatal("pop=80 arterial should stamp empty minis")
	}
	for _, s := range arterial {
		if !inFarmKeyPool(s.key, farmMiniKeys) {
			t.Fatalf("arterial stamp %q not in farmMiniKeys", s.key)
		}
	}
}
