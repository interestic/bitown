package render

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestDrawFrameBlitsNativeSize(t *testing.T) {
	assets := t.TempDir()
	writeMinimalSpritesV1(t, assets, true)
	sprites := filepath.Join(assets, "sprites-v1")
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	img.Set(2, 4, color.RGBA{R: 255, A: 255})
	pngFile, err := os.Create(filepath.Join(sprites, "atlas", "sprites_v1_atlas.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(pngFile, img); err != nil {
		_ = pngFile.Close()
		t.Fatal(err)
	}
	if err := pngFile.Close(); err != nil {
		t.Fatal(err)
	}
	meta := `{
  "image": "sprites_v1_atlas.png",
  "count": 1,
  "frames": {
    "sprites/House_a/1_v00.png": {"x": 0, "y": 0, "w": 8, "h": 8, "anchor_x": 4, "anchor_y": 8}
  }
}`
	if err := os.WriteFile(filepath.Join(sprites, "atlas", "sprites_v1_atlas.json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BITOWN_ASSETS_DIR", assets)
	ResetAtlasCacheForTest()
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	dst := image.NewRGBA(image.Rect(0, 0, 32, 32))
	if !atlas.drawFrameAtFoot(dst, "sprites/House_a/1_v00.png", 8, 16) {
		t.Fatal("drawFrameAtFoot returned false")
	}
	got := dst.RGBAAt(6, 12)
	if got.R != 255 || got.A != 255 {
		t.Fatalf("expected unscaled red pixel at foot-relative dest, got %+v", got)
	}
	scaled := dst.RGBAAt(15, 15)
	if scaled.R == 255 && scaled.A == 255 {
		t.Fatal("native blit should not fill the 16x16 cell")
	}
}

func TestDrawFrameAtFootHonorsZeroAnchor(t *testing.T) {
	assets := t.TempDir()
	writeMinimalSpritesV1(t, assets, true)
	sprites := filepath.Join(assets, "sprites-v1")
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(0, 0, color.RGBA{B: 255, A: 255})
	pngFile, err := os.Create(filepath.Join(sprites, "atlas", "sprites_v1_atlas.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(pngFile, img); err != nil {
		_ = pngFile.Close()
		t.Fatal(err)
	}
	if err := pngFile.Close(); err != nil {
		t.Fatal(err)
	}
	meta := `{
  "image": "sprites_v1_atlas.png",
  "count": 1,
  "frames": {
    "sprites/House_a/1_v00.png": {"x": 0, "y": 0, "w": 4, "h": 4, "anchor_x": 0, "anchor_y": 0}
  }
}`
	if err := os.WriteFile(filepath.Join(sprites, "atlas", "sprites_v1_atlas.json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BITOWN_ASSETS_DIR", assets)
	ResetAtlasCacheForTest()
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	dst := image.NewRGBA(image.Rect(0, 0, 32, 32))
	if !atlas.drawFrameAtFoot(dst, "sprites/House_a/1_v00.png", 10, 10) {
		t.Fatal("drawFrameAtFoot returned false")
	}
	got := dst.RGBAAt(10, 10)
	if got.B != 255 || got.A != 255 {
		t.Fatalf("zero anchor should place src(0,0) at foot, got %+v", got)
	}
	// Old sentinel remapped (0,0) to (w/2,h)=(2,4), which would land the pixel at (8,6).
	if dst.RGBAAt(8, 6).B == 255 {
		t.Fatal("zero anchor must not be rewritten to w/2,h")
	}
}

func TestAtlasFrameKeysKeepDefineSpritePaths(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	if len(atlas.Frames) == 0 {
		t.Fatal("expected atlas frames")
	}
	for key := range atlas.Frames {
		if !strings.HasPrefix(key, "sprites/") {
			t.Fatalf("frame key %q missing sprites/ prefix", key)
		}
		if !strings.HasSuffix(key, ".png") {
			t.Fatalf("frame key %q missing .png suffix", key)
		}
		parts := strings.Split(key, "/")
		if len(parts) < 3 {
			t.Fatalf("frame key %q should look like sprites/<folder>/<frame>.png", key)
		}
	}
}

func TestE2E_AtlasNativeBuildingTallerThanLegacyCell(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	var tallKey string
	tallH := 0
	for _, base := range atlas.BuildingBases {
		key := base + "_v00.png"
		if h := atlas.frameHeight(key); h > tallH {
			tallH = h
			tallKey = key
		}
	}
	if tallH <= 16 {
		t.Fatalf("expected native building taller than legacy 16px cell, max=%d", tallH)
	}
	dst := image.NewRGBA(image.Rect(0, 0, 256, 256))
	footX, footY := 128, 200
	if !atlas.drawFrameAtFoot(dst, tallKey, footX, footY) {
		t.Fatalf("drawFrameAtFoot(%s) failed", tallKey)
	}
	paintedRows := 0
	for y := footY - tallH; y < footY; y++ {
		for x := 0; x < 256; x++ {
			if dst.RGBAAt(x, y).A != 0 {
				paintedRows++
				break
			}
		}
	}
	if paintedRows <= 16 {
		t.Fatalf("native blit of %s should span more than 16 rows, got %d", tallKey, paintedRows)
	}
}

func TestLoadAtlasWhenPresent(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	if len(atlas.BuildingBases) == 0 {
		t.Fatal("expected building bases")
	}
	if len(atlas.BuildingBases) > 200 {
		t.Fatalf("expected tight building pool, got %d", len(atlas.BuildingBases))
	}
	varying := 0
	for _, rect := range atlas.Frames {
		if rect.W != 32 || rect.H != 32 {
			varying++
		}
		if rect.W <= 0 || rect.H <= 0 {
			t.Fatalf("invalid frame size %dx%d", rect.W, rect.H)
		}
	}
	if varying == 0 {
		t.Fatal("expected trimmed native frame sizes, all frames are 32x32")
	}
}

func TestBuildingPoolExcludesNonBuildings(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	for _, base := range atlas.BuildingBases {
		if deniedBuildingFolder(spriteFolderBase(base)) {
			t.Fatalf("building pool contains denied sprite %s", base)
		}
	}
	if len(atlas.BasesForTag(TagRoad)) == 0 {
		t.Fatal("expected road-tagged bases for later layout issues")
	}
	if road := atlas.PickKeyForTag(TagRoad, 7); road == "" {
		t.Fatal("expected road frame key")
	}
	if _, ok := atlas.Frames[atlas.PickKeyForTag(TagRoad, 7)]; !ok {
		t.Fatal("road pick is not an atlas frame")
	}
	if len(atlas.BasesForTag(TagIndustrial)) == 0 {
		t.Fatal("expected industrial-tagged bases for sector zoning")
	}
	if len(atlas.BasesForTag(TagCommercial)) == 0 {
		t.Fatal("expected commercial-tagged bases for sector zoning")
	}
	if ind := atlas.PickKeyForTag(TagIndustrial, 1); ind == "" {
		t.Fatal("expected industrial frame key")
	}
	if com := atlas.PickKeyForTag(TagCommercial, 1); com == "" {
		t.Fatal("expected commercial frame key")
	}
	if len(atlas.BasesForTag(TagTree)) == 0 {
		t.Fatal("expected tree-tagged bases for park lots")
	}
}

func TestBuildingAllowlistSnapshotMatchesCatalog(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	var allowlistPath string
	for _, dir := range spritesV1Candidates() {
		candidate := filepath.Join(dir, "building_bases.allowlist")
		if _, err := os.Stat(candidate); err == nil {
			allowlistPath = candidate
			break
		}
	}
	if allowlistPath == "" {
		t.Fatal("building_bases.allowlist not found next to sprites-v1 catalog")
	}
	raw, err := os.ReadFile(allowlistPath)
	if err != nil {
		t.Fatalf("read allowlist: %v", err)
	}
	var snapshot []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		snapshot = append(snapshot, line)
	}
	seen := make(map[string]struct{}, len(atlas.BuildingBases))
	var folders []string
	for _, base := range atlas.BuildingBases {
		folder := spriteFolderBase(base)
		if _, ok := seen[folder]; ok {
			continue
		}
		seen[folder] = struct{}{}
		folders = append(folders, folder)
	}
	sort.Strings(folders)
	if !reflect.DeepEqual(folders, snapshot) {
		t.Fatalf("building pool folders != allowlist snapshot\ngot  %v\nwant %v", folders, snapshot)
	}
}

func TestE2E_UISpritesNeverEnterBuildingPool(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	buildingSet := make(map[string]struct{}, len(atlas.BuildingBases))
	for _, base := range atlas.BuildingBases {
		buildingSet[base] = struct{}{}
	}
	for _, base := range atlas.BasesForTag(TagExclude) {
		if _, ok := buildingSet[base]; ok {
			t.Fatalf("exclude-tagged base %s leaked into building pool", base)
		}
		folder := spriteFolderBase(base)
		if strings.Contains(strings.ToLower(folder), "mcstats") ||
			strings.Contains(strings.ToLower(folder), "mcstat") ||
			strings.Contains(strings.ToLower(folder), "statusbar") {
			// known UI tokens must stay exclude-only
			continue
		}
	}
	knownUI := []string{"mcStats", "mcStat", "StatusBar", "mcLoading", "mcBg"}
	for _, token := range knownUI {
		for _, base := range atlas.BuildingBases {
			if strings.Contains(spriteFolderBase(base), token) {
				t.Fatalf("building pool contains UI token %q in %s", token, base)
			}
		}
	}
	if len(atlas.BasesForTag(TagExclude)) == 0 {
		t.Fatal("expected exclude-tagged UI/road/empty clips in catalog")
	}
}

func TestPickRoadKeyUsesNeighborTopology(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	cases := []struct {
		name       string
		n, e, s, w bool
		x, y       int
		want       string
	}{
		{"ew straight", false, true, false, true, 2, 0, roadSpriteBase + "/3_v00.png"},
		{"ns straight", true, false, true, false, 0, 2, roadSpriteBase + "/6_v00.png"},
		{"cross even", true, true, true, true, 0, 0, roadSpriteBase + "/3_v00.png"},
		{"cross odd", true, true, true, true, 1, 0, roadSpriteBase + "/6_v00.png"},
		{"t east stub", true, true, true, false, 0, 0, roadSpriteBase + "/6_v00.png"},
		{"corner ne", true, true, false, false, 0, 0, roadSpriteBase + "/1_v00.png"},
		{"corner es", false, true, true, false, 0, 0, roadSpriteBase + "/2_v00.png"},
		{"corner sw", false, false, true, true, 0, 0, roadSpriteBase + "/4_v00.png"},
		{"corner wn", true, false, false, true, 0, 0, roadSpriteBase + "/5_v00.png"},
		{"dead end e", false, true, false, false, 0, 0, roadSpriteBase + "/3_v00.png"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := atlas.PickRoadKey(tc.n, tc.e, tc.s, tc.w, tc.x, tc.y)
			if got != tc.want {
				t.Fatalf("PickRoadKey() = %q, want %q", got, tc.want)
			}
			if _, ok := atlas.Frames[got]; !ok {
				t.Fatalf("road key %q is not an atlas frame", got)
			}
		})
	}
}

func TestLoadBuildingsCatalogDropsDeniedTokens(t *testing.T) {
	assets := t.TempDir()
	writeMinimalSpritesV1(t, assets, true)

	sprites := filepath.Join(assets, "sprites-v1")
	atlasJSON := filepath.Join(sprites, "atlas", "sprites_v1_atlas.json")
	meta := `{
  "image": "sprites_v1_atlas.png",
  "count": 2,
  "frames": {
    "sprites/House_a/1_v00.png": {"x": 0, "y": 0, "w": 32, "h": 32, "anchor_x": 16, "anchor_y": 32},
    "sprites/DefineSprite_702_mcRoad/1_v00.png": {"x": 0, "y": 0, "w": 32, "h": 32, "anchor_x": 16, "anchor_y": 32}
  }
}`
	if err := os.WriteFile(atlasJSON, []byte(meta), 0o644); err != nil {
		t.Fatalf("write atlas json: %v", err)
	}
	manifest := `{
  "version": 2,
  "building_bases": ["sprites/House_a", "sprites/DefineSprite_702_mcRoad"],
  "bases_by_tag": {
    "residential": ["sprites/House_a"],
    "industrial": [],
    "commercial": [],
    "landmark": [],
    "road": ["sprites/DefineSprite_702_mcRoad"],
    "tree": [],
    "water": [],
    "park": [],
    "exclude": []
  },
  "entries": [
    {"base": "sprites/House_a", "group": "building", "tag": "residential"},
    {"base": "sprites/DefineSprite_702_mcRoad", "group": "other", "tag": "road"}
  ]
}`
	if err := os.WriteFile(filepath.Join(sprites, "buildings.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write buildings json: %v", err)
	}

	t.Setenv("BITOWN_ASSETS_DIR", assets)
	ResetAtlasCacheForTest()
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	if len(atlas.BuildingBases) != 1 || spriteFolderBase(atlas.BuildingBases[0]) != "sprites/House_a" {
		t.Fatalf("expected only House_a in building pool, got %v", atlas.BuildingBases)
	}
	if len(atlas.BasesForTag(TagRoad)) != 1 {
		t.Fatalf("expected road tag to keep mcRoad, got %v", atlas.BasesForTag(TagRoad))
	}
	roadKey := atlas.PickKeyForTag(TagRoad, 0)
	if _, ok := atlas.Frames[roadKey]; !ok {
		t.Fatalf("road pick %q is not an atlas frame", roadKey)
	}
}

func TestCatalogDropsExcludeWithoutDenyName(t *testing.T) {
	assets := t.TempDir()
	writeMinimalSpritesV1(t, assets, true)
	sprites := filepath.Join(assets, "sprites-v1")
	meta := `{
  "image": "sprites_v1_atlas.png",
  "count": 2,
  "frames": {
    "sprites/House_a/1_v00.png": {"x": 0, "y": 0, "w": 32, "h": 32, "anchor_x": 16, "anchor_y": 32},
    "sprites/Triangle_a/1_v00.png": {"x": 0, "y": 0, "w": 32, "h": 32, "anchor_x": 16, "anchor_y": 32}
  }
}`
	if err := os.WriteFile(filepath.Join(sprites, "atlas", "sprites_v1_atlas.json"), []byte(meta), 0o644); err != nil {
		t.Fatalf("write atlas json: %v", err)
	}
	manifest := `{
  "version": 2,
  "building_bases": ["sprites/House_a", "sprites/Triangle_a"],
  "bases_by_tag": {
    "residential": ["sprites/House_a"],
    "industrial": [],
    "commercial": [],
    "landmark": [],
    "road": [],
    "tree": [],
    "water": [],
    "park": [],
    "exclude": ["sprites/Triangle_a"]
  },
  "entries": [
    {"base": "sprites/House_a", "group": "building", "tag": "residential"},
    {"base": "sprites/Triangle_a", "group": "other", "tag": "exclude"}
  ]
}`
	if err := os.WriteFile(filepath.Join(sprites, "buildings.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write buildings json: %v", err)
	}
	t.Setenv("BITOWN_ASSETS_DIR", assets)
	ResetAtlasCacheForTest()
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	if len(atlas.BuildingBases) != 1 || spriteFolderBase(atlas.BuildingBases[0]) != "sprites/House_a" {
		t.Fatalf("exclude clip leaked into building pool: %v", atlas.BuildingBases)
	}
}

func TestRejectsV1BuildingsManifest(t *testing.T) {
	assets := t.TempDir()
	writeMinimalSpritesV1(t, assets, true)
	manifest := `{"building_bases":["sprites/House_a"]}`
	if err := os.WriteFile(filepath.Join(assets, "sprites-v1", "buildings.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write buildings json: %v", err)
	}
	t.Setenv("BITOWN_ASSETS_DIR", assets)
	ResetAtlasCacheForTest()
	if _, err := loadAtlas(); err == nil {
		t.Fatal("expected v1 manifest to fail")
	}
}

func TestPickBuildingKeyDeterministic(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}

	a := atlas.pickBuildingKey(42)
	b := atlas.pickBuildingKey(42)
	if a != b {
		t.Fatalf("expected deterministic key, got %q vs %q", a, b)
	}
	if _, ok := atlas.Frames[a]; !ok {
		t.Fatalf("picked unknown frame key: %s", a)
	}
}
