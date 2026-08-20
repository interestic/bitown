package render

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestIsoCanvasIsDiamondGrid(t *testing.T) {
	if mapWidth == 20*16 && mapHeight == 20*16 {
		t.Fatal("iso canvas should not be the old 320x320 square grid")
	}
	x0, y0 := isoCell(0, 0)
	x1, y1 := isoCell(1, 0)
	if x1 <= x0 || y1 <= y0 {
		t.Fatalf("expected 2:1 step from (0,0)=(%d,%d) to (1,0)=(%d,%d)", x0, y0, x1, y1)
	}
	dx, dy := x1-x0, y1-y0
	if dx != isoTileW/2 || dy != isoTileH/2 || dx != 2*dy {
		t.Fatalf("expected 2:1 iso step (%d,%d), got (%d,%d)", isoTileW/2, isoTileH/2, dx, dy)
	}
	xDown, yDown := isoCell(0, 1)
	if xDown >= x0 || yDown <= y0 {
		t.Fatalf("expected +y to move left and down, got (%d,%d) -> (%d,%d)", x0, y0, xDown, yDown)
	}
}

func TestIsoCanvasFitsCornersAndTallOverhang(t *testing.T) {
	const tallH, wideW = 96, 73
	corners := [][2]int{{0, 0}, {mapCols - 1, 0}, {0, mapRows - 1}, {mapCols - 1, mapRows - 1}}
	for _, c := range corners {
		topX, topY := isoCell(c[0], c[1])
		footX, footY := topX, topY+isoTileH
		// Diamond vertices must stay on-canvas.
		left, right := topX-isoTileW/2, topX+isoTileW/2
		bottom := topY + isoTileH
		if left < 0 || right >= mapWidth || topY < 0 || bottom >= mapHeight {
			t.Fatalf("diamond for (%d,%d) escapes canvas: L=%d T=%d R=%d B=%d size=%dx%d",
				c[0], c[1], left, topY, right, bottom, mapWidth, mapHeight)
		}
		// Tall building bbox around the foot must stay on-canvas.
		bLeft, bTop := footX-wideW/2, footY-tallH
		bRight := footX + wideW/2
		if bLeft < 0 || bTop < 0 || bRight >= mapWidth || footY >= mapHeight {
			t.Fatalf("tall overhang for (%d,%d) escapes canvas: L=%d T=%d R=%d footY=%d size=%dx%d",
				c[0], c[1], bLeft, bTop, bRight, footY, mapWidth, mapHeight)
		}
	}
}

func TestBuildCityMapPNG(t *testing.T) {
	city := &citycore.City{
		Slug: "testcity",
		Pop:  120,
	}
	data, err := BuildCityMapPNG(city)
	if err != nil {
		t.Fatalf("BuildCityMapPNG error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected png bytes, got empty")
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("failed to decode png: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != mapWidth || b.Dy() != mapHeight {
		t.Fatalf("unexpected dimensions: got %dx%d, want %dx%d", b.Dx(), b.Dy(), mapWidth, mapHeight)
	}
}
