package render

import (
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestIsoTileMatchesMinivilleCs(t *testing.T) {
	if isoTileW != 24 || isoTileH != 12 {
		t.Fatalf("iso tile = %dx%d, want Cs.WW×Cs.HH = 24×12", isoTileW, isoTileH)
	}
	if groundBlock != 4 {
		t.Fatalf("groundBlock = %d, want 4 (genMiniSquare scale)", groundBlock)
	}
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
