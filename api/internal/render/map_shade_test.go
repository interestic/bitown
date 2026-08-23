package render

import (
	"bytes"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestGroundShade_DefaultUnchanged(t *testing.T) {
	requireAtlasFiles(t)
	city := &citycore.City{Slug: "shade-default", Pop: 200}

	t.Setenv("BITOWN_MAP_GROUND_SHADE", "")
	offPNG, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("png off: %v", err)
	}
	offTag, err := MapEntityTag(city)
	if err != nil {
		t.Fatalf("etag off: %v", err)
	}

	// Explicit unset vs empty should both behave as default off.
	t.Setenv("BITOWN_MAP_GROUND_SHADE", "0")
	off2PNG, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("png off=0: %v", err)
	}
	off2Tag, err := MapEntityTag(city)
	if err != nil {
		t.Fatalf("etag off=0: %v", err)
	}
	if !bytes.Equal(offPNG, off2PNG) {
		t.Fatal("default PNG changed when BITOWN_MAP_GROUND_SHADE is not 1")
	}
	if offTag != off2Tag {
		t.Fatalf("default ETag changed: %q vs %q", offTag, off2Tag)
	}
}

func TestGroundShade_EnabledDiffersAndDeterministic(t *testing.T) {
	requireAtlasFiles(t)
	city := &citycore.City{Slug: "shade-on", Pop: 200}

	t.Setenv("BITOWN_MAP_GROUND_SHADE", "")
	offPNG, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("png off: %v", err)
	}
	offTag, err := MapEntityTag(city)
	if err != nil {
		t.Fatalf("etag off: %v", err)
	}

	t.Setenv("BITOWN_MAP_GROUND_SHADE", "1")
	onPNG, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("png on: %v", err)
	}
	onTag, err := MapEntityTag(city)
	if err != nil {
		t.Fatalf("etag on: %v", err)
	}
	onPNG2, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("png on second: %v", err)
	}

	if bytes.Equal(offPNG, onPNG) {
		t.Fatal("expected shaded PNG to differ from default")
	}
	if offTag == onTag {
		t.Fatalf("expected shaded ETag to differ, both %q", offTag)
	}
	if !bytes.Equal(onPNG, onPNG2) {
		t.Fatal("shaded PNG is not deterministic")
	}
	if groundShadeIdentity() != groundShadeVariant {
		t.Fatalf("identity = %q, want %q", groundShadeIdentity(), groundShadeVariant)
	}
}
