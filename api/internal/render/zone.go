package render

import (
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
	x, y    int
	dist    int
	jitter  uint32
	use     lotUse
	tag     string
	density int // Game.hx square density covering this cell
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

func lotOccupancyWithDensity(city *citycore.City, grid cityGrid, dens popDensity) map[[2]int]lotCell {
	slug := city.Slug.String()
	peonClip := !arterialsEnabled(city)

	lots := make([]lotCell, 0, mapCols*mapRows)
	cx, cy := mapCols/2, mapRows/2
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] != cellLot {
				continue
			}
			dx, dy := x-cx, y-cy
			lots = append(lots, lotCell{
				x:       x,
				y:       y,
				dist:    dx*dx + dy*dy,
				jitter:  hashCell(slug, x, y),
				tag:     zoneTag(city, x, y),
				density: localDensityAt(dens, x, y),
			})
		}
	}
	sort.Slice(lots, func(i, j int) bool {
		if lots[i].dist != lots[j].dist {
			return lots[i].dist < lots[j].dist
		}
		return lots[i].jitter < lots[j].jitter
	})

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

	fillLotsFromDensity(inner, city, dens, peonClip, parkN)

	pop := city.Pop.Int()
	out := make(map[[2]int]lotCell, len(lots))
	for _, lot := range inner {
		scatterTreeOnEmpty(&lot, city.Env.Int(), false, pop)
		out[[2]int{lot.x, lot.y}] = lot
	}
	for _, lot := range curb {
		lot.use = lotEmpty
		scatterTreeOnEmpty(&lot, city.Env.Int(), true, pop)
		out[[2]int{lot.x, lot.y}] = lot
	}
	clearParksOffGrass(out, pop)
	return out
}

func scatterTreeOnEmpty(lot *lotCell, env int, curb bool, pop int) {
	if lot.use != lotEmpty || env <= 0 {
		return
	}
	if !peonGrassTopCell(pop, lot.x, lot.y) {
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
		lot.tag = TagTree
	}
}

// clearParksOffGrass is a safety net for parks that still land on the dalle
// soil rim after placement filters (density TagTree / parkN).
func clearParksOffGrass(occ map[[2]int]lotCell, pop int) {
	for pos, lot := range occ {
		if lot.use != lotPark {
			continue
		}
		if peonGrassTopCell(pop, lot.x, lot.y) {
			continue
		}
		lot.use = lotEmpty
		lot.tag = ""
		occ[pos] = lot
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
