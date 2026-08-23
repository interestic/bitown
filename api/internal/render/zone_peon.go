package render

import (
	"math"
	"sort"
)

// fillPeonIslandLots places at most one hut per mcDalle plate. The foot sits
// on the green diamond top (inset from the island rim) so sprites do not hang
// into the sky or stack on the same visual tile.
func fillPeonIslandLots(inner []lotCell, pop, parkN int) {
	idxOf := make(map[[2]int]int, len(inner))
	for i, lot := range inner {
		idxOf[[2]int{lot.x, lot.y}] = i
	}
	islandIdx := make([]int, 0, peonPlateCount())
	for py := 0; py < peonDalleGrid; py++ {
		for px := 0; px < peonDalleGrid; px++ {
			ax, ay := peonPlateAnchorCell(px, py)
			i, ok := idxOf[[2]int{ax, ay}] //#nosec G602 -- map key is [2]int
			if !ok {
				continue
			}
			islandIdx = append(islandIdx, i)
		}
	}
	islandParkN := parkN
	if islandParkN > len(islandIdx) {
		islandParkN = len(islandIdx)
	}
	parkSet := make(map[int]struct{}, islandParkN)
	if islandParkN > 0 {
		parkOrder := append([]int(nil), islandIdx...)
		sort.Slice(parkOrder, func(i, j int) bool {
			a, b := inner[parkOrder[i]], inner[parkOrder[j]]
			if a.dist != b.dist {
				return a.dist > b.dist
			}
			return a.jitter > b.jitter
		})
		for i := 0; i < islandParkN; i++ {
			idx := parkOrder[i]
			parkSet[idx] = struct{}{}
			inner[idx].use = lotPark
		}
	}
	buildIdx := make([]int, 0, len(islandIdx)-islandParkN)
	for _, i := range islandIdx {
		if _, ok := parkSet[i]; ok {
			continue
		}
		buildIdx = append(buildIdx, i)
	}
	fillN := pop
	if fillN > len(buildIdx) {
		fillN = len(buildIdx)
	}
	if fillN < 0 {
		fillN = 0
	}
	placeSpacedBuildings(inner, buildIdx, fillN)
	offIslandParkN := parkN - islandParkN
	if offIslandParkN > 0 {
		for i := len(inner) - 1; i >= 0 && offIslandParkN > 0; i-- {
			if _, onIsland := parkSet[i]; onIsland {
				continue
			}
			if inner[i].use == lotBuilding {
				continue
			}
			inner[i].use = lotPark
			offIslandParkN--
		}
	}
}

func chebyshev(ax, ay, bx, by int) int {
	dx := ax - bx
	if dx < 0 {
		dx = -dx
	}
	dy := ay - by
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

func peonBuildingSpacing(islandLots, fillN int) int {
	if fillN <= 1 || islandLots <= 0 {
		return 1
	}
	s := int(math.Sqrt(float64(islandLots) / float64(fillN)))
	if s < 2 {
		return 2
	}
	return s
}

// placeSpacedBuildings marks fillN candidates as buildings. Candidates are
// visited in jitter order so the spread is deterministic per slug. Spacing
// relaxes to 2 then 1 if fillN cannot be met.
func placeSpacedBuildings(lots []lotCell, islandIdx []int, fillN int) {
	if fillN <= 0 || len(islandIdx) == 0 {
		return
	}
	if fillN >= len(islandIdx) {
		for _, i := range islandIdx {
			lots[i].use = lotBuilding
		}
		return
	}
	order := append([]int(nil), islandIdx...)
	sort.Slice(order, func(i, j int) bool {
		a, b := lots[order[i]], lots[order[j]]
		if a.jitter != b.jitter {
			return a.jitter < b.jitter
		}
		if a.y != b.y {
			return a.y < b.y
		}
		return a.x < b.x
	})
	try := func(spacing int) int {
		n := 0
		for _, i := range islandIdx {
			if lots[i].use == lotBuilding {
				n++
			}
		}
		if n >= fillN {
			return n
		}
		for _, i := range order {
			if lots[i].use == lotBuilding {
				continue
			}
			if !peonLotFarEnough(lots, islandIdx, i, spacing) {
				continue
			}
			lots[i].use = lotBuilding
			n++
			if n >= fillN {
				return n
			}
		}
		return n
	}
	spacing := peonBuildingSpacing(len(islandIdx), fillN)
	if try(spacing) < fillN && spacing > 2 {
		try(2)
	}
	if countIslandBuildings(lots, islandIdx) < fillN {
		try(1)
	}
}

func countIslandBuildings(lots []lotCell, islandIdx []int) int {
	n := 0
	for _, i := range islandIdx {
		if lots[i].use == lotBuilding {
			n++
		}
	}
	return n
}

func peonLotFarEnough(lots []lotCell, islandIdx []int, cand, spacing int) bool {
	cx, cy := peonPlateOf(lots[cand].x, lots[cand].y)
	for _, i := range islandIdx {
		if i == cand || lots[i].use != lotBuilding {
			continue
		}
		px, py := peonPlateOf(lots[i].x, lots[i].y)
		if chebyshev(cx, cy, px, py) < spacing {
			return false
		}
	}
	return true
}
