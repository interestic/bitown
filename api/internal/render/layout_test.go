package render

import (
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestBuildCityGridConnectsRoads(t *testing.T) {
	grid := buildCityGridForCity(&citycore.City{Slug: "testcity", Pop: 500})

	startX, startY := -1, -1
	roadCount := 0
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] != cellRoad {
				continue
			}
			roadCount++
			if startX < 0 {
				startX, startY = x, y
			}
		}
	}
	if roadCount == 0 {
		t.Fatal("expected roads")
	}

	seen := floodRoads(grid, startX, startY)
	if seen != roadCount {
		t.Fatalf("roads are disconnected: reached %d of %d", seen, roadCount)
	}

	again := buildCityGridForCity(&citycore.City{Slug: "testcity", Pop: 500})
	if again[3][3] != grid[3][3] {
		t.Fatal("expected deterministic grid for same slug")
	}
}

func TestLotsAreNotRoads(t *testing.T) {
	grid := buildCityGridForCity(&citycore.City{Slug: "lot-check", Pop: 500})
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			wantRoad := isRoadCell(x, y)
			gotRoad := grid[y][x] == cellRoad
			if wantRoad != gotRoad {
				t.Fatalf("cell %d,%d road=%v want %v", x, y, gotRoad, wantRoad)
			}
		}
	}
}

func TestCityGridUsesSquareSideNesting(t *testing.T) {
	if mapCols != displaySide*squareSide || mapRows != displaySide*squareSide {
		t.Fatalf("map %dx%d, want displaySide(%d)*squareSide(%d)", mapCols, mapRows, displaySide, squareSide)
	}
	if !isRoadCell(0, 5) || !isRoadCell(10, 7) || !isRoadCell(5, 0) {
		t.Fatal("expected square-boundary arterial roads")
	}
	if isRoadCell(1, 1) || isRoadCell(4, 4) || isRoadCell(9, 3) {
		t.Fatal("expected continuous lots inside squares")
	}
}

func TestPeonCityHasNoArterials(t *testing.T) {
	grid := buildCityGridForCity(&citycore.City{Slug: "peon", Pop: 1})
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] == cellRoad {
				t.Fatalf("peon map should have no roads, found at (%d,%d)", x, y)
			}
		}
	}
}

func TestTransportUnlocksArterials(t *testing.T) {
	grid := buildCityGridForCity(&citycore.City{Slug: "tra", Pop: 1, Tra: 10})
	found := false
	for y := 0; y < mapRows && !found; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] == cellRoad {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("Tra>0 should unlock arterial roads even at low pop")
	}
}

func floodRoads(grid cityGrid, sx, sy int) int {
	type pt struct{ x, y int }
	q := []pt{{sx, sy}}
	seen := map[pt]struct{}{{sx, sy}: {}}
	for len(q) > 0 {
		cur := q[0]
		q = q[1:]
		for _, d := range []pt{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
			nx, ny := cur.x+d.x, cur.y+d.y
			if nx < 0 || ny < 0 || nx >= mapCols || ny >= mapRows {
				continue
			}
			if grid[ny][nx] != cellRoad {
				continue
			}
			np := pt{nx, ny}
			if _, ok := seen[np]; ok {
				continue
			}
			seen[np] = struct{}{}
			q = append(q, np)
		}
	}
	return len(seen)
}
