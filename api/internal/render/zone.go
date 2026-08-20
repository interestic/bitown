package render

import (
	"math"
	"sort"

	"github.com/interestic/bitown/internal/citycore"
)

type lotUse int

const (
	lotEmpty lotUse = iota
	lotPark
	lotBuilding
)

type lotCell struct {
	x, y   int
	dist   int
	jitter uint32
	use    lotUse
	tag    string
}

func lotOccupancy(city *citycore.City, grid cityGrid) map[[2]int]lotCell {
	lots := make([]lotCell, 0, mapCols*mapRows)
	cx, cy := mapCols/2, mapRows/2
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] != cellLot {
				continue
			}
			dx, dy := x-cx, y-cy
			lots = append(lots, lotCell{
				x:      x,
				y:      y,
				dist:   dx*dx + dy*dy,
				jitter: hashCell(city.Slug, x, y),
				tag:    zoneTag(city, x, y),
			})
		}
	}
	sort.Slice(lots, func(i, j int) bool {
		if lots[i].dist != lots[j].dist {
			return lots[i].dist < lots[j].dist
		}
		return lots[i].jitter < lots[j].jitter
	})

	parkN := city.Env / 80
	if parkN > len(lots)/6 {
		parkN = len(lots) / 6
	}
	baseRate := math.Min(float64(city.Pop)/500.0, 1.0)
	// Keep roads readable at high population while preserving center-first growth.
	fillRate := 0.68 * math.Sqrt(baseRate)
	fillN := int(math.Round(float64(len(lots)-parkN) * fillRate))
	if fillN < 0 {
		fillN = 0
	}

	for i := range lots {
		switch {
		case i >= len(lots)-parkN:
			lots[i].use = lotPark
		case i < fillN:
			lots[i].use = lotBuilding
		default:
			lots[i].use = lotEmpty
		}
	}

	out := make(map[[2]int]lotCell, len(lots))
	for _, lot := range lots {
		out[[2]int{lot.x, lot.y}] = lot
	}
	return out
}

func zoneTag(city *citycore.City, x, y int) string {
	cx, cy := mapCols/2, mapRows/2
	dx, dy := x-cx, y-cy
	dist := dx*dx + dy*dy
	if dist <= 12 && city.Com > 0 {
		return TagCommercial
	}
	if (x <= 2 || x >= mapCols-3 || y <= 2 || y >= mapRows-3) && city.Ind > 0 {
		return TagIndustrial
	}
	return TagResidential
}
