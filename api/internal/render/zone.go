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

func lotTouchesRoad(grid cityGrid, x, y int) bool {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if nx < 0 || ny < 0 || nx >= mapCols || ny >= mapRows {
				continue
			}
			if grid[ny][nx] == cellRoad {
				return true
			}
		}
	}
	return false
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
				jitter: hashCell(city.Slug.String(), x, y),
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

	// Keep a 1-cell curb setback so Flash footprints do not bury arterial road
	// diamonds (#33). Arterial-only roads leave large interior blocks, so density
	// still reads Townzzy-like once interiors fill.
	inner := make([]lotCell, 0, len(lots))
	curb := make([]lotCell, 0, len(lots)/4)
	for _, lot := range lots {
		if lotTouchesRoad(grid, lot.x, lot.y) {
			curb = append(curb, lot)
			continue
		}
		inner = append(inner, lot)
	}

	parkN := city.Env.Int() / 40
	if parkN > len(inner)/3 {
		parkN = len(inner) / 3
	}
	pop := city.Pop.Int()
	if pop < popTierPeon {
		fillPeonIslandLots(inner, pop, parkN)
	} else {
		fillRate := math.Min(0.92, math.Sqrt(float64(pop)/220.0))
		fillN := int(math.Round(float64(len(inner)-parkN) * fillRate))
		if fillN < 0 {
			fillN = 0
		}
		for i := range inner {
			switch {
			case i >= len(inner)-parkN:
				inner[i].use = lotPark
			case i < fillN:
				inner[i].use = lotBuilding
			default:
				inner[i].use = lotEmpty
			}
		}
	}

	out := make(map[[2]int]lotCell, len(lots))
	for _, lot := range inner {
		scatterTreeOnEmpty(&lot, city.Env.Int(), false)
		out[[2]int{lot.x, lot.y}] = lot
	}
	for _, lot := range curb {
		lot.use = lotEmpty
		scatterTreeOnEmpty(&lot, city.Env.Int(), true)
		out[[2]int{lot.x, lot.y}] = lot
	}
	return out
}

func scatterTreeOnEmpty(lot *lotCell, env int, curb bool) {
	if lot.use != lotEmpty || env <= 0 {
		return
	}
	chance := env / 2
	if curb {
		chance = (env * 3) / 2
	}
	if chance > 750 {
		chance = 750
	}
	if int(lot.jitter%1000) < chance { //#nosec G115
		lot.use = lotPark
	}
}

func zoneTag(city *citycore.City, x, y int) string {
	cx, cy := mapCols/2, mapRows/2
	dx, dy := x-cx, y-cy
	dist := dx*dx + dy*dy
	comR2 := (mapCols * mapCols) / 33
	if dist <= comR2 && city.Com > 0 {
		return TagCommercial
	}
	rim := mapCols / 10
	if rim < 2 {
		rim = 2
	}
	if (x < rim || x >= mapCols-rim || y < rim || y >= mapRows-rim) && city.Ind > 0 {
		return TagIndustrial
	}
	return TagResidential
}
