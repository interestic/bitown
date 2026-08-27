package render

import (
	"strings"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestBuildCityBadgeSVG_Content(t *testing.T) {
	svg, err := BuildCityBadgeSVG(&citycore.City{Name: "Tokyo", Pop: 42})
	if err != nil {
		t.Fatalf("BuildCityBadgeSVG: %v", err)
	}
	got := string(svg)
	if !strings.Contains(got, "<svg") {
		t.Fatalf("expected svg root, got %q", got)
	}
	if !strings.Contains(got, "Tokyo") {
		t.Fatalf("expected city name, got %q", got)
	}
	if !strings.Contains(got, "pop 42") {
		t.Fatalf("expected pop label, got %q", got)
	}
}

func TestBuildCityBadgeSVG_EscapesName(t *testing.T) {
	svg, err := BuildCityBadgeSVG(&citycore.City{Name: `<img src=x onerror=alert(1)>`, Pop: 1})
	if err != nil {
		t.Fatalf("BuildCityBadgeSVG: %v", err)
	}
	got := string(svg)
	if strings.Contains(got, "<img") {
		t.Fatalf("name was not escaped: %q", got)
	}
	if !strings.Contains(got, "&lt;img") {
		t.Fatalf("expected HTML-escaped name, got %q", got)
	}
}

func TestBuildCityBadgeSVG_NilCity(t *testing.T) {
	if _, err := BuildCityBadgeSVG(nil); err == nil {
		t.Fatal("expected error for nil city")
	}
}
