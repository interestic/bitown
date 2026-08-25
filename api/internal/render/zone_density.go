package render

import (
	"github.com/interestic/bitown/internal/citycore"
)

// fillLotsFromDensity places buildings using Game.hx genSquare / genMiniSquare
// rules on the displaySide×displaySide density field.
//
// peonClip: when true (no arterials), only cells inside the peon dalle field
// (Game.hx displaySide–sized, capped to the PNG crop) are eligible.
func fillLotsFromDensity(inner []lotCell, city *citycore.City, dens popDensity, peonClip bool, parkN int) {
	idxOf := make(map[[2]int]int, len(inner))
	for i, lot := range inner {
		idxOf[[2]int{lot.x, lot.y}] = i
		inner[i].use = lotEmpty
	}

	pop := city.Pop.Int()
	active := activeSquareSide(pop)
	origin := (displaySide - active) / 2
	slug := city.Slug.String()

	for sy := origin; sy < origin+active; sy++ {
		for sx := origin; sx < origin+active; sx++ {
			sqPop := dens.at(sx, sy)
			baseX := sx * squareSide
			baseY := sy * squareSide
			sqRNG := newMapRNGKeyed(slug, uint32(sx*131+sy*17+1)) //#nosec G115

			if sqPop <= 0 {
				continue
			}
			if sqPop >= csPopHuge {
				placeDensityLot(inner, idxOf, baseX, baseY, city, peonClip, pop, sqRNG, true)
				continue
			}

			rep := [4]int{}
			for i := 0; i < sqPop; i++ {
				rep[sqRNG.Intn(4)]++
			}
			for i := 0; i < 4; i++ {
				px := (i % 2) * 4
				py := (i / 2) * 4
				if px > 1 {
					px++
				}
				if py > 1 {
					py++
				}
				genMiniSquareLots(inner, idxOf, baseX+px, baseY+py, rep[i], city, peonClip, pop, slug, sx, sy, i)
			}
		}
	}

	if peonClip {
		snapPeonOnePerPlate(inner, idxOf, pop)
	}

	if parkN <= 0 {
		return
	}
	empties := make([]int, 0, len(inner))
	for i := range inner {
		if inner[i].use != lotEmpty {
			continue
		}
		if !peonGrassTopCell(pop, inner[i].x, inner[i].y) {
			continue
		}
		empties = append(empties, i)
	}
	if parkN > len(empties) {
		parkN = len(empties)
	}
	for k := 0; k < parkN; k++ {
		i := empties[len(empties)-1-k]
		inner[i].use = lotPark
		inner[i].tag = TagTree
	}
}

// snapPeonOnePerPlate keeps Caerphilly spacing: at most one building per mcDalle
// plate, foot on the plate anchor (density still chose which plates are occupied).
func snapPeonOnePerPlate(inner []lotCell, idxOf map[[2]int]int, pop int) {
	type plateKey struct{ px, py int }
	best := map[plateKey]int{} // plate → inner index of kept building
	for i := range inner {
		if inner[i].use != lotBuilding {
			continue
		}
		if !inPeonIslandFor(pop, inner[i].x, inner[i].y) {
			inner[i].use = lotEmpty
			continue
		}
		px, py := peonPlateOfFor(pop, inner[i].x, inner[i].y)
		key := plateKey{px, py}
		prev, ok := best[key]
		if !ok {
			best[key] = i
			continue
		}
		// Prefer the candidate closer to the plate anchor; then lower jitter.
		ax, ay := peonPlateAnchorCellFor(pop, px, py)
		dNew := (inner[i].x-ax)*(inner[i].x-ax) + (inner[i].y-ay)*(inner[i].y-ay)
		dOld := (inner[prev].x-ax)*(inner[prev].x-ax) + (inner[prev].y-ay)*(inner[prev].y-ay)
		if dNew < dOld || (dNew == dOld && inner[i].jitter < inner[prev].jitter) {
			inner[prev].use = lotEmpty
			best[key] = i
		} else {
			inner[i].use = lotEmpty
		}
	}
	// Move survivors onto plate anchors so feet sit on green diamond tops.
	for key, i := range best {
		ax, ay := peonPlateAnchorCellFor(pop, key.px, key.py)
		if inner[i].x == ax && inner[i].y == ay {
			continue
		}
		dst, ok := idxOf[[2]int{ax, ay}]
		if !ok {
			continue
		}
		if dst == i {
			continue
		}
		if inner[dst].use == lotBuilding {
			continue
		}
		tag := inner[i].tag
		dens := inner[i].density
		inner[i].use = lotEmpty
		inner[dst].use = lotBuilding
		inner[dst].tag = tag
		inner[dst].density = dens
	}
}

func genMiniSquareLots(
	inner []lotCell,
	idxOf map[[2]int]int,
	bx, by, sqPop int,
	city *citycore.City,
	peonClip bool,
	pop int,
	slug string,
	sx, sy, n int,
) {
	rval := uint32(sx*1009 + sy*917 + n*131 + 1) //#nosec G115
	rng := newMapRNGKeyed(slug, rval)

	if sqPop <= 0 {
		return
	}
	if sqPop >= csPopBig {
		placeDensityLot(inner, idxOf, bx, by, city, peonClip, pop, rng, true)
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
		nx := bx + (i%2)*2
		ny := by + (i/2)*2
		forceHut := sqPop < csPopPeon
		rich := rep[i] > csPopNormal
		placeDensityLotAt(inner, idxOf, nx, ny, city, peonClip, pop, sub, forceHut, rich)
	}
}

func placeDensityLot(
	inner []lotCell,
	idxOf map[[2]int]int,
	x, y int,
	city *citycore.City,
	peonClip bool,
	pop int,
	rng *mapRNG,
	rich bool,
) {
	placeDensityLotAt(inner, idxOf, x, y, city, peonClip, pop, rng, false, rich)
}

func placeDensityLotAt(
	inner []lotCell,
	idxOf map[[2]int]int,
	x, y int,
	city *citycore.City,
	peonClip bool,
	pop int,
	rng *mapRNG,
	forceHut bool,
	rich bool,
) {
	if peonClip && !inPeonIslandFor(pop, x, y) {
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
			if peonGrassTopCell(pop, x, y) {
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
