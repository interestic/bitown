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

	var cross [][]uint8
	var roadMask roadMaskData
	if !roadless {
		plan := planRoads(city, dens)
		cross = plan.cross
		roadMask = buildRoadMaskDataWithCross(grid, cross, -roadGrassLift)
	}

	inner := make([]lotCell, 0, len(lots))
	curb := make([]lotCell, 0, len(lots)/4)
	for _, lot := range lots {
		if lotTouchesRoad(grid, lot.x, lot.y) {
			curb = append(curb, lot)
			continue
		}
		if lotReservedForCross(lot.x, lot.y, cross) {
			curb = append(curb, lot)
			continue
		}
		inner = append(inner, lot)
	}

	parkN := city.Env.Int() / 40
	if parkN > len(inner)/3 {
		parkN = len(inner) / 3
	}

	fillLotsFromDensity(inner, city, dens, roadless, cross)

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
		if lot.use == lotEmpty && !lotReservedForCross(lot.x, lot.y, cross) && !lotTouchesRoad(grid, lot.x, lot.y) {
			scatterTreeOnEmpty(&lot, city.Env.Int(), pop)
		}
		out[pos] = lot
	}
	clearParksOffGrass(out, pop)
	// One-hop farm margins still leave empties cardinal-adjacent to expanded
	// farm rings; parks there paint tree canopies onto farm (#113).
	clearParksNearFarms(out)
	clearParksNearBuildings(out)
	clearParksOnRoads(out, grid, cross, roadMask, roadless)
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

// clearParksNearBuildings drops parks that touch lotBuilding (chebyshev ≤ 1) so
// tall tree sprites are not drawn in front of adjacent building facades.
func clearParksNearBuildings(occ map[[2]int]lotCell) {
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
				if ok && n.use == lotBuilding {
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

// clearParksOnRoads drops parks on road-adjacent lots or painted asphalt (CROSS
// foot / arterial diamonds) so tree sprites do not sit on street tiles.
func clearParksOnRoads(occ map[[2]int]lotCell, grid cityGrid, cross [][]uint8, roads roadMaskData, roadless bool) {
	for pos, lot := range occ {
		if lot.use != lotPark {
			continue
		}
		x, y := pos[0], pos[1] //#nosec G602 -- map key is [2]int
		if lotTouchesRoad(grid, x, y) || lotReservedForCross(x, y, cross) {
			lot.use = lotEmpty
			lot.tag = ""
			occ[pos] = lot
			continue
		}
		if !roadless && lotOverlapsRoadMask(roads, x, y) {
			lot.use = lotEmpty
			lot.tag = ""
			occ[pos] = lot
		}
	}
}

func scatterTreeOnEmpty(lot *lotCell, env int, pop int) {
	if lot.use != lotEmpty || env <= 0 {
		return
	}
	if !grassTopCell(pop, lot.x, lot.y) {
		return
	}
	chance := env / 2
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
