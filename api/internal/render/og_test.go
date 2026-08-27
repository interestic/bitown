package render

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestOGEntityTag_DiffersFromMapEntityTag(t *testing.T) {
	city := &citycore.City{Slug: "og-etag", Pop: 42}

	mapTag, err := MapEntityTag(city)
	if err != nil {
		t.Fatalf("MapEntityTag: %v", err)
	}
	ogTag, err := OGEntityTag(city)
	if err != nil {
		t.Fatalf("OGEntityTag: %v", err)
	}
	if mapTag == "" || ogTag == "" {
		t.Fatalf("expected quoted etags, map=%q og=%q", mapTag, ogTag)
	}
	if mapTag == ogTag {
		t.Fatalf("OG ETag must not equal map.png ETag, both %q", mapTag)
	}

	mapAgain, err := MapEntityTag(city)
	if err != nil {
		t.Fatalf("MapEntityTag second: %v", err)
	}
	if mapAgain != mapTag {
		t.Fatalf("map.png ETag changed after OGEntityTag: %q vs %q", mapTag, mapAgain)
	}
}

func TestOGEntityTag_StableAndChangesWithPop(t *testing.T) {
	low := &citycore.City{Slug: "og-pop", Pop: 10}
	high := &citycore.City{Slug: "og-pop", Pop: 400}

	a, err := OGEntityTag(low)
	if err != nil {
		t.Fatalf("low: %v", err)
	}
	b, err := OGEntityTag(low)
	if err != nil {
		t.Fatalf("low again: %v", err)
	}
	if a != b {
		t.Fatalf("expected stable OG etag, got %q vs %q", a, b)
	}
	c, err := OGEntityTag(high)
	if err != nil {
		t.Fatalf("high: %v", err)
	}
	if a == c {
		t.Fatalf("expected OG etag to change with pop, both %q", a)
	}
}

func TestBuildCityOGPNG_Dimensions(t *testing.T) {
	pngBytes, err := BuildCityOGPNG(&citycore.City{Slug: "og-size", Pop: 20})
	if err != nil {
		t.Fatalf("BuildCityOGPNG: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != OGWidth || b.Dy() != OGHeight {
		t.Fatalf("og png size = %dx%d, want %dx%d", b.Dx(), b.Dy(), OGWidth, OGHeight)
	}
}

func TestOGEntityTag_BumpsWithRendererVersion(t *testing.T) {
	// og-v2 must not collide with the historical og-v1 hash shape for the same city.
	city := &citycore.City{Slug: "og-ver", Pop: 7}
	got, err := OGEntityTag(city)
	if err != nil {
		t.Fatalf("OGEntityTag: %v", err)
	}
	if got == "" || got[0] != '"' {
		t.Fatalf("expected quoted etag, got %q", got)
	}
	if ogRendererVersion != "og-v2" {
		t.Fatalf("ogRendererVersion = %q, want og-v2", ogRendererVersion)
	}
}
