package render

import "testing"

func TestBuildCityGridConnectsRoads(t *testing.T) {
	grid := buildCityGrid("testcity")
	period := layoutPeriod("testcity")
	if period < 5 || period > 7 {
		t.Fatalf("period %d out of range", period)
	}

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

	again := buildCityGrid("testcity")
	if again[3][3] != grid[3][3] {
		t.Fatal("expected deterministic grid for same slug")
	}
}

func TestLotsAreNotRoads(t *testing.T) {
	grid := buildCityGrid("lot-check")
	period := layoutPeriod("lot-check")
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			wantRoad := x%period == 0 || y%period == 0
			gotRoad := grid[y][x] == cellRoad
			if wantRoad != gotRoad {
				t.Fatalf("cell %d,%d road=%v want %v", x, y, gotRoad, wantRoad)
			}
		}
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
