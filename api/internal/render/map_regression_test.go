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

func isFoliageColor(c color.RGBA) bool {
	if c.A != 255 {
		return false
	}
	if isTerrainColor(c) {
		return false
	}
	// mcDalle tops can be yellower than grassColor (low blue) and are not trees.
	if c.G >= 150 && c.G <= 230 && c.R >= 100 && c.R <= 200 && c.B <= 50 {
		return false
	}
	return int(c.G) > int(c.R)+25 && int(c.G) > int(c.B)+25
}

func isTerrainColor(c color.RGBA) bool {
	if c == roadColor {
		return true
	}
	// Roadless sky canvas (Townzzy-like).
	if c == (color.RGBA{R: 186, G: 220, B: 235, A: 255}) {
		return true
	}
	// Grass can vary by a few shades in atlas mode / dalle tops.
	if c.A == 255 && c.G >= 160 && c.G <= 230 && c.R >= 100 && c.R <= 200 && c.B >= 40 && c.B <= 120 {
		return true
	}
	// Dalle soil edges.
	if c.A == 255 && c.R >= 80 && c.R <= 160 && c.G >= 50 && c.G <= 120 && c.B <= 80 {
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
	requireAtlasFiles(t)
	slug := citycore.Slug("density-check")
	lowCity := &citycore.City{Slug: slug, Pop: 100}
	highCity := &citycore.City{Slug: slug, Pop: 500}
	lowOcc := countOccupiedBuildings(lowCity)
	highOcc := countOccupiedBuildings(highCity)
	if highOcc <= lowOcc {
		t.Fatalf("expected more building lots at pop=500 (%d) than pop=100 (%d)", highOcc, lowOcc)
	}

	// Pixel counts are a poor proxy once residential mix shifts to smaller
	// houses; lot fill is the growth contract. Fallback rectangles are covered
	// by TestBuildCityMapPNG_FallbackDensityIncreasesWithPop.
	if _, err := BuildCityMapPNG(lowCity); err != nil {
		t.Fatalf("low pop render: %v", err)
	}
	if _, err := BuildCityMapPNG(highCity); err != nil {
		t.Fatalf("high pop render: %v", err)
	}
}

func countOccupiedBuildings(city *citycore.City) int {
	occ := lotOccupancy(city, buildCityGridForCity(city))
	n := 0
	for _, lot := range occ {
		if lot.use == lotBuilding {
			n++
		}
	}
	return n
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
	if b.Dx() != b.Dy() {
		t.Fatalf("unexpected dimensions: got %dx%d, want square", b.Dx(), b.Dy())
	}
	if b.Dx() < mapMinSquare {
		t.Fatalf("png %dx%d, want at least %d square", b.Dx(), b.Dy(), mapMinSquare)
	}
	colors := countUniqueColors(img)
	if colors > 10 {
		t.Fatalf("expected rectangle fallback palette (<=10 colors), got %d", colors)
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

	slug := citycore.Slug("fallback-density")
	low, err := BuildCityMapPNG(&citycore.City{Slug: slug, Pop: 100})
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
		t.Fatalf("expected more building pixels at pop=500 (%d) than pop=100 (%d)", highCount, lowCount)
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

func TestBuildCityMapPNG_TreesStayOffDalleSoil(t *testing.T) {
	requireAtlasFiles(t)

	cases := []struct {
		pop int
		env int
	}{
		{pop: 2, env: 7},
		{pop: 70, env: 79},
		{pop: 80, env: 79},
		{pop: 120, env: 79},
	}
	for _, tc := range cases {
		city := &citycore.City{Slug: "testcity", Pop: citycore.SectorValue(tc.pop), Env: citycore.SectorValue(tc.env), Ind: 1, Com: 1, Sec: 1}
		// Pre-fit working canvas shares isoCell / grass-mask coordinates.
		img := mustBuildMapWorkingImage(t, city)
		grass := buildPlateGrass(tc.pop)
		b := img.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if x < 0 || x >= mapWidth || !grass.col[x] {
					continue
				}
				// Geometric grass top sits above the soil lip; skip the lip and a
				// little mcDalle fringe so side texels are not treated as trees.
				if y <= grass.maxY[x]+plateGrassLift+8 {
					continue
				}
				c := img.RGBAAt(x, y)
				if !isFoliageColor(c) {
					continue
				}
				t.Fatalf("pop=%d foliage on plate soil at (%d,%d) %+v", tc.pop, x, y, c)
			}
		}
	}
}
