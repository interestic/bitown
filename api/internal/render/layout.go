package render

import "github.com/interestic/bitown/internal/citycore"

const (
	cellLot = iota
	cellRoad
)

type cityGrid [][]int

func buildCityGridForCity(city *citycore.City) cityGrid {
	grid := make(cityGrid, mapRows)
	roads := arterialsEnabled(city)
	for y := 0; y < mapRows; y++ {
		grid[y] = make([]int, mapCols)
		for x := 0; x < mapCols; x++ {
			if roads && isRoadCell(x, y) {
				grid[y][x] = cellRoad
			} else {
				grid[y][x] = cellLot
			}
		}
	}
	return grid
}

func arterialsEnabled(city *citycore.City) bool {
	if city == nil {
		return true
	}
	// Match original roadCoef gate roughly: tiny cities have no street network.
	if city.Tra.Int() > 0 {
		return true
	}
	return city.Pop.Int() >= 80
}

func isRoadCell(x, y int) bool {
	return x%squareSide == 0 || y%squareSide == 0
}
