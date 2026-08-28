package render

import (
	"github.com/interestic/bitown/internal/citycore"
)

// squareMiniPops mirrors Game.hx genSquare: scatter square density across
// four mini-squares with the same keyed RNG as fillLotsFromDensity.
func squareMiniPops(slug string, sx, sy, sqPop int) [4]int {
	var rep [4]int
	if sqPop <= 0 {
		return rep
	}
	rng := newMapRNGKeyed(slug, uint32(sx*131+sy*17+1)) //#nosec G115 -- square index
	for i := 0; i < sqPop; i++ {
		rep[rng.Intn(4)]++
	}
	return rep
}

// fillLotsFromDensity places buildings using Game.hx genSquare / genMiniSquare
// rules on the displaySide×displaySide density field. Parks and farm cover are
// applied afterward so trees stay off farm (Game.hx farm XOR forest).
//
// Only cells inside the live plate island (Game.hx viewport, capped to the PNG
// field) are eligible — arterial and roadless share the same island.
func fillLotsFromDensity(inner []lotCell, city *citycore.City, dens popDensity, roadless bool, cross [][]uint8) {
	idxOf := make(map[[2]int]int, len(inner))
	for i, lot := range inner {
		idxOf[[2]int{lot.x, lot.y}] = i
		inner[i].use = lotEmpty
	}

	pop := city.Pop.Int()
	active := activeSquareSide(pop)
	origin := activeSquareOrigin(pop)
	slug := city.Slug.String()

	for sy := origin; sy < origin+active; sy++ {
		for sx := origin; sx < origin+active; sx++ {
			sqPop := dens.at(sx, sy)
			baseX := sx * squareSide
			baseY := sy * squareSide

			if sqPop <= 0 {
				continue
			}
			if sqPop >= csPopHuge {
				hasCross := !roadless && sy < len(cross) && sx < len(cross[sy]) && cross[sy][sx] > 0
				if hasCross {
					for i := 0; i < 4; i++ {
						px := (i % 2) * 4
						py := (i / 2) * 4
						if px > 1 {
							px++
						}
						if py > 1 {
							py++
						}
						genMiniSquareLots(inner, idxOf, baseX+px, baseY+py, 1, city, roadless, pop, slug, sx, sy, i, cross)
					}
					continue
				}
				sqRNG := newMapRNGKeyed(slug, uint32(sx*131+sy*17+1)) //#nosec G115 -- square index
				placeDensityLot(inner, idxOf, baseX, baseY, city, roadless, pop, sqRNG, true, cross)
				continue
			}

			rep := squareMiniPops(slug, sx, sy, sqPop)
			for i := 0; i < 4; i++ {
				px := (i % 2) * 4
				py := (i / 2) * 4
				if px > 1 {
					px++
				}
				if py > 1 {
					py++
				}
				genMiniSquareLots(inner, idxOf, baseX+px, baseY+py, rep[i], city, roadless, pop, slug, sx, sy, i, cross)
			}
		}
	}
	// roadless keeps quadrant 2×2 feet (not one center anchor per plate, #116),
	// inset into the mini diamond so they do not sit on seams (#118).
}

// placeDedicatedParks claims env/40 vacant grass lots (not farm cover) for trees.
// Order matches the former fillLotsFromDensity trail: farthest empties first.
func placeDedicatedParks(occ map[[2]int]lotCell, inner []lotCell, parkN, pop int) {
	if parkN <= 0 {
		return
	}
	empties := make([][2]int, 0, len(inner))
	for _, lot := range inner {
		cur, ok := occ[[2]int{lot.x, lot.y}]
		if !ok || cur.use != lotEmpty {
			continue
		}
		if !grassTopCell(pop, lot.x, lot.y) {
			continue
		}
		empties = append(empties, [2]int{lot.x, lot.y})
	}
	if parkN > len(empties) {
		parkN = len(empties)
	}
	for k := 0; k < parkN; k++ {
		pos := empties[len(empties)-1-k]
		cur := occ[pos]
		cur.use = lotPark
		cur.tag = TagTree
		occ[pos] = cur
	}
}

func genMiniSquareLots(
	inner []lotCell,
	idxOf map[[2]int]int,
	bx, by, sqPop int,
	city *citycore.City,
	roadless bool,
	pop int,
	slug string,
	sx, sy, n int,
	cross [][]uint8,
) {
	rval := uint32(sx*1009 + sy*917 + n*131 + 1) //#nosec G115
	rng := newMapRNGKeyed(slug, rval)

	if sqPop <= 0 {
		return
	}
	// CROSS squares: arterial_yard ring — one building per mini at foot 3 (toward
	// junction). miniSE foot 3 is the CROSS cell itself and stays vacant.
	hasCross := !roadless && sy < len(cross) && sx < len(cross[sy]) && cross[sy][sx] > 0
	if hasCross {
		nx, ny := miniHutFoot(bx, by, 3, false)
		if lotOnCrossFoot(nx, ny, cross) {
			return
		}
		forceHut := sqPop < densityHut
		rich := sqPop > csPopNormal
		placeDensityLotAt(inner, idxOf, nx, ny, city, roadless, pop, rng, forceHut, rich, cross)
		return
	}
	if sqPop >= csPopBig {
		placeDensityLot(inner, idxOf, bx, by, city, roadless, pop, rng, true, cross)
		return
	}

	rep := [4]int{}
	for i := 0; i < sqPop; i++ {
		rep[rng.Intn(4)]++
	}
	sub := newMapRNGKeyed(slug, rval^0x9e3779b9)
	for i := 0; i < 4; i++ {
		if rep[i] <= 0 {
			continue
		}
		nx, ny := miniHutFoot(bx, by, i, roadless)
		forceHut := sqPop < densityHut
		rich := rep[i] > csPopNormal
		placeDensityLotAt(inner, idxOf, nx, ny, city, roadless, pop, sub, forceHut, rich, cross)
	}
}

// miniHutInset shifts Game.hx 2×2 hut feet one cell SE on roadless maps so they
// sit inside the 4×4 mini diamond instead of on NW origin / plate seams (#118).
// Arterial maps keep the Game.hx origin so towers stay clear of road facades.
const miniHutInset = 1

func miniHutFoot(bx, by, i int, roadless bool) (int, int) {
	inset := 0
	if roadless {
		inset = miniHutInset
	}
	return bx + (i%2)*2 + inset, by + (i/2)*2 + inset
}

func placeDensityLot(
	inner []lotCell,
	idxOf map[[2]int]int,
	x, y int,
	city *citycore.City,
	roadless bool,
	pop int,
	rng *mapRNG,
	rich bool,
	cross [][]uint8,
) {
	placeDensityLotAt(inner, idxOf, x, y, city, roadless, pop, rng, false, rich, cross)
}

func placeDensityLotAt(
	inner []lotCell,
	idxOf map[[2]int]int,
	x, y int,
	city *citycore.City,
	roadless bool,
	pop int,
	rng *mapRNG,
	forceHut bool,
	rich bool,
	cross [][]uint8,
) {
	if !inPlateIsland(pop, x, y) {
		return
	}
	if !roadless && lotOnCrossFoot(x, y, cross) {
		return
	}
	// Feet stay on plate grass tops (not soil rim). Center snap used to force
	// this; without it, skip rim cells so sprites are not clipped off the ledge.
	if roadless && !grassTopCell(pop, x, y) {
		return
	}
	i, ok := idxOf[[2]int{x, y}]
	if !ok {
		return
	}
	if inner[i].use == lotBuilding || inner[i].use == lotPark {
		return
	}

	if forceHut {
		inner[i].use = lotBuilding
		inner[i].tag = TagResidential
		return
	}

	tag := zoneTag(city, x, y)
	if rich && tag == TagResidential {
		typ := getBatType(city, x, y, rng)
		alt := batTypeToTag(typ)
		switch alt {
		case TagTree:
			if grassTopCell(pop, x, y) {
				inner[i].use = lotPark
				inner[i].tag = TagTree
				return
			}
			// Rim cell: keep a building instead of skipping the density lot.
		case TagIndustrial:
			if city.Ind.Int() > 0 {
				tag = TagIndustrial
			}
		case TagCommercial:
			if city.Com.Int() > 0 {
				tag = TagCommercial
			}
		}
	}
	inner[i].use = lotBuilding
	inner[i].tag = tag
}
