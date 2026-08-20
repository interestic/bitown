package render

const (
	cellLot = iota
	cellRoad
)

type cityGrid [][]int

func layoutPeriod(slug string) int {
	return 5 + int(hashCell(slug, 0, 0)%3)
}

func buildCityGrid(slug string) cityGrid {
	period := layoutPeriod(slug)
	grid := make(cityGrid, mapRows)
	for y := 0; y < mapRows; y++ {
		grid[y] = make([]int, mapCols)
		for x := 0; x < mapCols; x++ {
			if x%period == 0 || y%period == 0 {
				grid[y][x] = cellRoad
			} else {
				grid[y][x] = cellLot
			}
		}
	}
	return grid
}

func roadNeighbors(grid cityGrid, x, y int) (n, e, s, w bool) {
	if y > 0 && grid[y-1][x] == cellRoad {
		n = true
	}
	if x+1 < mapCols && grid[y][x+1] == cellRoad {
		e = true
	}
	if y+1 < mapRows && grid[y+1][x] == cellRoad {
		s = true
	}
	if x > 0 && grid[y][x-1] == cellRoad {
		w = true
	}
	return n, e, s, w
}
