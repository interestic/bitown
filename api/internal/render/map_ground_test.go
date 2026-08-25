package render

import (
	"os"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestIsoTileMatchesMinivilleCs(t *testing.T) {
	if isoTileW != 24 || isoTileH != 12 {
		t.Fatalf("iso tile = %dx%d, want Cs.WW×Cs.HH = 24×12", isoTileW, isoTileH)
	}
	if groundBlock != squareSide {
		t.Fatalf("groundBlock = %d, want squareSide=%d (one mcDalle per square)", groundBlock, squareSide)
	}
}

// peonDalleGridFor is the number of mcDalle plates along one axis of the peon field.
func peonDalleGridFor(pop int) int {
	e := peonExtentCells(pop)
	g := e / groundBlock
	if g < 1 {
		g = 1
	}
	return g
}

// Legacy helpers used by tests that assume a typical peon pop (pop=20 →
// flashDisplaySide=6 → full 60×60 crop of 6×6 dales).
func peonIslandOrigin() int {
	return peonIslandOriginFor(20)
}

func peonIslandExtent() int {
	return peonIslandExtentFor(20)
}

func TestAtlasMapUsesGroundAndRoadSprites(t *testing.T) {
	requireAtlasFiles(t)
	city := &citycore.City{Slug: "ground-road-check", Pop: 80}
	data, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img := decodeMapPNG(t, data)
	// Textured floor + dashed roads should produce more unique colors than flat fill.
	if n := countUniqueColors(img); n < 40 {
		t.Fatalf("expected rich palette from dalle/road sprites, got %d unique colors", n)
	}
}

func TestPickGroundKeyPrefersMcDalle(t *testing.T) {
	requireAtlasFiles(t)
	atlas, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	key := atlas.PickGroundKey(0)
	if key == "" {
		t.Fatal("expected ground key")
	}
	if _, ok := atlas.Frames[key]; !ok {
		t.Fatalf("ground key %q missing from atlas", key)
	}
	if len(key) < len(groundSpriteBase) || key[:len(groundSpriteBase)] != groundSpriteBase {
		t.Fatalf("PickGroundKey = %q, want mcDalle prefix %q", key, groundSpriteBase)
	}
}

func TestPeonPop1RendersSixBySixCrop(t *testing.T) {
	requireAtlasFiles(t)
	data, err := BuildCityMapPNG(&citycore.City{Slug: "duisburg", Pop: 1})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img := decodeMapPNG(t, data)
	b := img.Bounds()
	if b.Dx() != mapWidth || b.Dy() != mapHeight {
		t.Fatalf("png %dx%d, want %dx%d", b.Dx(), b.Dy(), mapWidth, mapHeight)
	}
	if displaySide != 6 || peonDalleGridFor(1) != 6 {
		t.Fatalf("pop=1 field %d squares / %d dales, want 6×6", displaySide, peonDalleGridFor(1))
	}
	if path := os.Getenv("BITOWN_DUMP_MAP"); path != "" {
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDalleGrassLiftMatchesFarmAndCatalogLip(t *testing.T) {
	if dalleGrassLift != 20 {
		t.Fatalf("dalleGrassLift=%d, want 20 (mcDalle soil sides)", dalleGrassLift)
	}
	if roadGrassLift != dalleGrassLift-isoTileH {
		t.Fatalf("roadGrassLift=%d, want dalleGrassLift-isoTileH (%d)", roadGrassLift, dalleGrassLift-isoTileH)
	}
	if farmGrassLift != dalleGrassLift {
		t.Fatalf("farmGrassLift=%d, want dalleGrassLift=%d", farmGrassLift, dalleGrassLift)
	}
}
