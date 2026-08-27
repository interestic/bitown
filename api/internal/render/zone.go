package render

import (
	"sort"

	"github.com/interestic/bitown/internal/citycore"
)

type lotUse int

const (
	lotEmpty lotUse = iota
	lotFarm         // Game.hx farm cover; trees stay off (type 15 / mini type 2)
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
	roadless := !arterialsEnabled(city)

	lots := make([]lotCell, 0, mapCols*mapRows)
	pop := city.Pop.Int()
	cx, cy := plateIslandCenter(pop)
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] != cellLot {
				continue
			}
			if !inPlateIsland(pop, x, y) {
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

	// Buildings first, then farm cover (Game.hx farm XOR forest), then parks
	// and env scatter on leftover vacant grass — never on farm cells.
	fillLotsFromDensity(inner, city, dens, roadless)

	out := make(map[[2]int]lotCell, len(lots))
	for _, lot := range inner {
		out[[2]int{lot.x, lot.y}] = lot
	}
	for _, lot := range curb {
		lot.use = lotEmpty
		out[[2]int{lot.x, lot.y}] = lot
	}
	markFarmLots(out, city.Slug.String(), pop, dens, grid, roadless)
	expandFarmMargins(out)
	placeDedicatedParks(out, inner, parkN, pop)
	for pos, lot := range out {
		scatterTreeOnEmpty(&lot, city.Env.Int(), lotTouchesRoad(grid, lot.x, lot.y), pop)
		out[pos] = lot
	}
	clearParksOffGrass(out, pop)
	// One-hop farm margins still leave empties cardinal-adjacent to expanded
	// farm rings; parks there paint tree canopies onto farm (#113).
	clearParksNearFarms(out)
	return out
}

// clearParksNearFarms drops parks that touch lotFarm (chebyshev ≤ 1) so tree
// sprites are not drawn on top of farm after farm stamps.
func clearParksNearFarms(occ map[[2]int]lotCell) {
	for pos, lot := range occ {
		if lot.use != lotPark {
			continue
		}
		touch := false
		for dy := -1; dy <= 1 && !touch; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				n, ok := occ[[2]int{pos[0] + dx, pos[1] + dy}]
				if ok && n.use == lotFarm {
					touch = true
					break
				}
			}
		}
		if !touch {
			continue
		}
		lot.use = lotEmpty
		lot.tag = ""
		occ[pos] = lot
	}
}

func scatterTreeOnEmpty(lot *lotCell, env int, curb bool, pop int) {
	if lot.use != lotEmpty || env <= 0 {
		return
	}
	if !grassTopCell(pop, lot.x, lot.y) {
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

// clearParksOffGrass is a safety net for parks that still land on the plate
// soil rim after placement filters (density TagTree / parkN).
func clearParksOffGrass(occ map[[2]int]lotCell, pop int) {
	for pos, lot := range occ {
		if lot.use != lotPark {
			continue
		}
		if grassTopCell(pop, lot.x, lot.y) {
			continue
		}
		lot.use = lotEmpty
		lot.tag = ""
		occ[pos] = lot
	}
}

func zoneTag(city *citycore.City, x, y int) string {
	pop := 0
	if city != nil {
		pop = city.Pop.Int()
	}
	o := plateIslandOrigin(pop)
	e := plateIslandExtent(pop)
	if e < 1 {
		e = squareSide
	}
	cx, cy := o+e/2, o+e/2
	dx, dy := x-cx, y-cy
	dist := dx*dx + dy*dy
	comR2 := (e * e) / 33
	if dist <= comR2 && city != nil && city.Com > 0 {
		return TagCommercial
	}
	// Outer third of the live island (not the sparse e/10 fringe): with a full
	// Game.hx viewport the true edge is density=0, so warehouses must sit on the
	// populated outer belt.
	band := e / 3
	if band < 2 {
		band = 2
	}
	if (x < o+band || x >= o+e-band || y < o+band || y >= o+e-band) && city != nil && city.Ind > 0 {
		return TagIndustrial
	}
	return TagResidential
}
