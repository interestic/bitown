package render

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestBuildCityMapPNG_Deterministic(t *testing.T) {
	city := &citycore.City{Slug: "testcity", Pop: 120}
	a, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("first render: %v", err)
	}
	b, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("second render: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("expected identical PNG bytes for same city input")
	}
}

func decodeMapPNG(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	return img
}

func isTerrainColor(c color.RGBA) bool {
	if c == roadColor {
		return true
	}
	// Grass can vary by a few shades in atlas mode.
	if c.A == 255 && c.G >= 190 && c.G <= 220 && c.R >= 135 && c.R <= 170 && c.B >= 70 && c.B <= 95 {
		return true
	}
	return false
}

func countBuildingPixels(img image.Image) int {
	b := img.Bounds()
	n := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			c := color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(bl >> 8),
				A: uint8(a >> 8),
			}
			if isTerrainColor(c) {
				continue
			}
			n++
		}
	}
	return n
}

func TestBuildCityMapPNG_DensityIncreasesWithPop(t *testing.T) {
	slug := "density-check"
	low, err := BuildCityMapPNG(&citycore.City{Slug: slug, Pop: 0})
	if err != nil {
		t.Fatalf("low pop render: %v", err)
	}
	high, err := BuildCityMapPNG(&citycore.City{Slug: slug, Pop: 500})
	if err != nil {
		t.Fatalf("high pop render: %v", err)
	}

	lowCount := countBuildingPixels(decodeMapPNG(t, low))
	highCount := countBuildingPixels(decodeMapPNG(t, high))
	if highCount <= lowCount {
		t.Fatalf("expected more building pixels at pop=500 (%d) than pop=0 (%d)", highCount, lowCount)
	}
}

func countUniqueColors(img image.Image) int {
	seen := make(map[color.RGBA]struct{})
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			c := color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(bl >> 8),
				A: uint8(a >> 8),
			}
			seen[c] = struct{}{}
		}
	}
	return len(seen)
}

func TestBuildCityMapPNG_UsesAtlasWhenPresent(t *testing.T) {
	requireAtlasFiles(t)

	data, err := BuildCityMapPNG(&citycore.City{Slug: "atlas-check", Pop: 500})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	colors := countUniqueColors(decodeMapPNG(t, data))
	if colors <= 6 {
		t.Fatalf("expected atlas rendering richness, got %d unique colors", colors)
	}
}

func TestBuildCityMapPNG_FallbackWhenAtlasMissing(t *testing.T) {
	forceFallbackAtlas(t)

	city := &citycore.City{Slug: "fallback-city", Pop: 120}
	data, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("fallback render: %v", err)
	}
	img := decodeMapPNG(t, data)
	b := img.Bounds()
	if b.Dx() != mapWidth || b.Dy() != mapHeight {
		t.Fatalf("unexpected dimensions: got %dx%d, want %dx%d", b.Dx(), b.Dy(), mapWidth, mapHeight)
	}
	colors := countUniqueColors(img)
	if colors > 6 {
		t.Fatalf("expected rectangle fallback palette (<=6 colors), got %d", colors)
	}

	again, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("second fallback render: %v", err)
	}
	if !bytes.Equal(data, again) {
		t.Fatal("expected deterministic fallback PNG bytes")
	}
}

func TestBuildCityMapPNG_FallbackDensityIncreasesWithPop(t *testing.T) {
	forceFallbackAtlas(t)

	slug := "fallback-density"
	low, err := BuildCityMapPNG(&citycore.City{Slug: slug, Pop: 0})
	if err != nil {
		t.Fatalf("low pop fallback: %v", err)
	}
	high, err := BuildCityMapPNG(&citycore.City{Slug: slug, Pop: 500})
	if err != nil {
		t.Fatalf("high pop fallback: %v", err)
	}
	lowCount := countBuildingPixels(decodeMapPNG(t, low))
	highCount := countBuildingPixels(decodeMapPNG(t, high))
	if highCount <= lowCount {
		t.Fatalf("expected more building pixels at pop=500 (%d) than pop=0 (%d)", highCount, lowCount)
	}
}

func TestMapEntityTag_FallbackWhenAtlasMissing(t *testing.T) {
	forceFallbackAtlas(t)

	tag, err := MapEntityTag(&citycore.City{Slug: "fallback-etag", Pop: 42})
	if err != nil {
		t.Fatalf("fallback etag: %v", err)
	}
	if tag == "" || tag == `""` {
		t.Fatalf("expected non-empty fallback etag, got %q", tag)
	}
}

