package render

import (
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestMapEntityTag_StableForSameCity(t *testing.T) {
	requireAtlasFiles(t)
	city := &citycore.City{Slug: "etag-city", Pop: 42}

	a, err := MapEntityTag(city)
	if err != nil {
		t.Fatalf("first tag: %v", err)
	}
	b, err := MapEntityTag(city)
	if err != nil {
		t.Fatalf("second tag: %v", err)
	}
	if a != b {
		t.Fatalf("expected stable etag, got %q vs %q", a, b)
	}
	if a == "" || a == `""` {
		t.Fatalf("expected non-empty quoted etag, got %q", a)
	}
}

func TestMapEntityTag_ChangesWithPop(t *testing.T) {
	requireAtlasFiles(t)
	low, err := MapEntityTag(&citycore.City{Slug: "etag-pop", Pop: 0})
	if err != nil {
		t.Fatalf("low pop tag: %v", err)
	}
	high, err := MapEntityTag(&citycore.City{Slug: "etag-pop", Pop: 500})
	if err != nil {
		t.Fatalf("high pop tag: %v", err)
	}
	if low == high {
		t.Fatalf("expected different etags for different pop, both %q", low)
	}
}

func TestBuildCityMapPNG_AtlasRequired(t *testing.T) {
	ResetAtlasCacheForTest()
	t.Setenv("BITOWN_ATLAS_REQUIRED", "true")
	t.Setenv("BITOWN_ASSETS_DIR", t.TempDir())

	_, err := BuildCityMapPNG(&citycore.City{Slug: "required-atlas", Pop: 1})
	if err == nil {
		t.Fatal("expected error when atlas is required but unavailable")
	}
}

func TestMapEntityTag_AtlasRequired(t *testing.T) {
	ResetAtlasCacheForTest()
	t.Setenv("BITOWN_ATLAS_REQUIRED", "true")
	t.Setenv("BITOWN_ASSETS_DIR", t.TempDir())

	_, err := MapEntityTag(&citycore.City{Slug: "required-atlas", Pop: 1})
	if err == nil {
		t.Fatal("expected etag error when atlas is required but unavailable")
	}
}

func TestAtlasFallbackReasonSentinelErrors(t *testing.T) {
	cases := []struct {
		err    error
		reason string
	}{
		{ErrAtlasNotFound, "atlas_not_found"},
		{ErrBuildingsManifestMissing, "buildings_manifest_missing"},
		{ErrBuildingsManifestEmpty, "buildings_manifest_empty"},
	}
	for _, tc := range cases {
		if got := atlasFallbackReason(tc.err); got != tc.reason {
			t.Fatalf("atlasFallbackReason(%v) = %q, want %q", tc.err, got, tc.reason)
		}
	}
}

func TestWrapAtlasError_MissingBuildingsJSONMessage(t *testing.T) {
	err := wrapAtlasError(fmt.Errorf("read buildings manifest: %w", fs.ErrNotExist))
	if !errors.Is(err, ErrBuildingsManifestMissing) {
		t.Fatalf("wrapAtlasError() = %v, want ErrBuildingsManifestMissing", err)
	}
	if got := atlasFallbackReason(err); got != "buildings_manifest_missing" {
		t.Fatalf("atlasFallbackReason = %q, want buildings_manifest_missing", got)
	}
}

func TestLoadAtlas_MissingBuildingsJSON(t *testing.T) {
	dir := t.TempDir()
	writeMinimalSpritesV1(t, dir, false)
	t.Setenv("BITOWN_ASSETS_DIR", dir)
	t.Setenv("BITOWN_ATLAS_REQUIRED", "")
	ResetAtlasCacheForTest()

	_, err := loadAtlas()
	if !errors.Is(err, ErrBuildingsManifestMissing) {
		t.Fatalf("loadAtlas() err = %v, want ErrBuildingsManifestMissing", err)
	}
}

func TestMatchIfNoneMatch(t *testing.T) {
	etag := `"abc123"`
	cases := []struct {
		header string
		want   bool
	}{
		{"", false},
		{etag, true},
		{`W/` + etag, true},
		{`w/` + etag, true},
		{` "other" , ` + etag, true},
		{`*`, true},
		{`"nope"`, false},
	}
	for _, tc := range cases {
		if got := MatchIfNoneMatch(tc.header, etag); got != tc.want {
			t.Fatalf("MatchIfNoneMatch(%q, %q) = %v, want %v", tc.header, etag, got, tc.want)
		}
	}
}

func TestAtlasReloadsWhenRevisionChanges(t *testing.T) {
	requireAtlasFiles(t)

	first, err := loadAtlas()
	if err != nil {
		t.Fatalf("load atlas: %v", err)
	}
	firstRevision := first.revision

	ResetAtlasCacheForTest()
	second, err := loadAtlas()
	if err != nil {
		t.Fatalf("reload atlas: %v", err)
	}
	if second.revision != firstRevision {
		t.Fatalf("expected same revision after reload, got %q vs %q", second.revision, firstRevision)
	}
	if second.sourceDir != first.sourceDir {
		t.Fatalf("unexpected source dir change: %q vs %q", second.sourceDir, first.sourceDir)
	}
}
