package render

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func requireAtlasFiles(t *testing.T) {
	t.Helper()
	ResetAtlasCacheForTest()
	_, err := loadAtlas()
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatalf("atlas required in CI: %v", err)
		}
		t.Skipf("atlas not available locally: %v", err)
	}
}

func forceFallbackAtlas(t *testing.T) {
	t.Helper()
	ResetAtlasCacheForTest()
	t.Setenv("BITOWN_ASSETS_DIR", t.TempDir())
	t.Setenv("BITOWN_ATLAS_REQUIRED", "")
	t.Setenv("ENV", "development")
}

func lotOccupancy(city *citycore.City, grid cityGrid) map[[2]int]lotCell {
	dens := genMapPop(city.Pop.Int(), newMapRNG(city.Slug.String()))
	return lotOccupancyWithDensity(city, grid, dens)
}

func writeMinimalSpritesV1(t *testing.T, assetsDir string, withBuildings bool) {
	t.Helper()
	sprites := filepath.Join(assetsDir, "sprites-v1")
	atlasDir := filepath.Join(sprites, "atlas")
	if err := os.MkdirAll(atlasDir, 0o755); err != nil {
		t.Fatalf("mkdir atlas: %v", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	pngFile, err := os.Create(filepath.Join(atlasDir, "sprites_v1_atlas.png"))
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	if err := png.Encode(pngFile, img); err != nil {
		_ = pngFile.Close()
		t.Fatalf("encode png: %v", err)
	}
	if err := pngFile.Close(); err != nil {
		t.Fatalf("close png: %v", err)
	}

	meta := `{
  "image": "sprites_v1_atlas.png",
  "count": 1,
  "frames": {
    "sprites/House_a/1_v00.png": {"x": 0, "y": 0, "w": 20, "h": 28, "anchor_x": 10, "anchor_y": 28}
  }
}
`
	if err := os.WriteFile(filepath.Join(atlasDir, "sprites_v1_atlas.json"), []byte(meta), 0o644); err != nil {
		t.Fatalf("write atlas json: %v", err)
	}
	if withBuildings {
		manifest := `{
  "version": 2,
  "building_bases": ["sprites/House_a"],
  "bases_by_tag": {
    "residential": ["sprites/House_a"],
    "industrial": [],
    "commercial": [],
    "landmark": [],
    "road": [],
    "tree": [],
    "water": [],
    "ground": [],
    "park": [],
    "exclude": []
  },
  "counts": {"building": 1, "by_tag": {"residential": 1, "industrial": 0, "commercial": 0, "landmark": 0, "road": 0, "tree": 0, "water": 0, "ground": 0, "park": 0, "exclude": 0}},
  "entries": [{"base": "sprites/House_a", "group": "building", "tag": "residential"}]
}`
		if err := os.WriteFile(filepath.Join(sprites, "buildings.json"), []byte(manifest), 0o644); err != nil {
			t.Fatalf("write buildings json: %v", err)
		}
	}
}
