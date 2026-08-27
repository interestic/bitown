package render

import (
	"math"

	"github.com/interestic/bitown/internal/citycore"
)

const (
	cellLot = iota
	cellRoad
)

// Road pavement styles from Game.hx `fr = 3*n + style` / catalog ROAD_STYLE_LABELS.
const (
	roadStyleThin = 0 // 細暗
	roadStyleDirt = 1 // 土
	roadStylePave = 2 // アスファルト
)

const (
	roadDir0 uint8 = 1 << 0 // Game.hx n=0 / DIR[1,0] — mcRoad frames 1–3
	roadDir1 uint8 = 1 << 1 // Game.hx n=1 / DIR[0,1] — mcRoad frames 4–6
)

type cityGrid [][]int

type roadStamp struct {
	sx, sy int
	dir    int
	style  int
}

type roadPlan struct {
	grid   cityGrid
	pave   [][]uint8
	dirs   [][]uint8
	cross  [][]uint8
	stamps []roadStamp
}

func emptyRoadPlan() roadPlan {
	grid := make(cityGrid, mapRows)
	pave := make([][]uint8, mapRows)
	dirs := make([][]uint8, mapRows)
	for y := 0; y < mapRows; y++ {
		grid[y] = make([]int, mapCols)
		pave[y] = make([]uint8, mapCols)
		dirs[y] = make([]uint8, mapCols)
		for x := 0; x < mapCols; x++ {
			grid[y][x] = cellLot
		}
	}
	cross := make([][]uint8, displaySide)
	for sy := 0; sy < displaySide; sy++ {
		cross[sy] = make([]uint8, displaySide)
	}
	return roadPlan{grid: grid, pave: pave, dirs: dirs, cross: cross}
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

// cityRoadCoef is Core.hx getRoadCoef: min((tra*5+100)/pop, 1).
func cityRoadCoef(city *citycore.City) float64 {
	if city == nil {
		return 1
	}
	pop := float64(city.Pop.Int())
	if pop <= 0 {
		return 1
	}
	return math.Min((float64(city.Tra.Int())*5+100)/pop, 1)
}

func squareRoadC(dens popDensity, sx, sy, dir int) float64 {
	pop := dens.at(sx, sy)
	dx, dy := 1, 0
	if dir == 1 {
		dx, dy = 0, 1
	}
	c := 0.0
	bis := 0
	for _, sens := range []int{-1, 1} {
		pp := float64(dens.at(sx+dx*sens, sy+dy*sens)) * 0.25
		if pp > 0 {
			bis++
		}
		c += pp
	}
	if pop > 1 {
		c += float64(pop) * 0.5
	}
	if pop == 0 && bis < 2 {
		c = 0
	}
	return c
}

func roadStyleFromScore(score float64) int {
	if score < 1 {
		return -1
	}
	style := roadStyleThin
	if score >= 1.5 {
		style = roadStyleDirt
	}
	if score >= 5 {
		style = roadStylePave
	}
	return style
}

// planRoads follows Game.hx genSquare BIG ROADS / CROSS ROADS: each live
// square may get 0–2 axis stamps (not a packed x%10||y%10 mesh). Style is
// 細暗 / 土 / アスファルト from local `c` (Game.hx thresholds 1 / 1.5 / 5).
func planRoads(city *citycore.City, dens popDensity) roadPlan {
	plan := emptyRoadPlan()
	if city == nil || !arterialsEnabled(city) {
		return plan
	}
	coef := cityRoadCoef(city)
	active := activeSquareSide(city.Pop.Int())
	origin := activeSquareOrigin(city.Pop.Int())
	for sy := origin; sy < origin+active; sy++ {
		for sx := origin; sx < origin+active; sx++ {
			for dir := 0; dir < 2; dir++ {
				c := squareRoadC(dens, sx, sy, dir)
				// Existence follows Game.hx `c >= 1` on the visible crop.
				// Style uses local `c` (not c*roadCoef): the PNG is the center
				// crop of SIDE=30, so city-wide roadCoef would keep even downtown
				// as dirt when Tra is 0. Local density paves the core and leaves
				// the crop rim as 土, like Townzzy suburbs.
				if c < 1 {
					continue
				}
				markSquareAxis(&plan, sx, sy, dir, roadStyleFromScore(c))
			}
			pop := dens.at(sx, sy)
			if pop <= 0 || pop >= csPopHuge {
				continue
			}
			score := float64(pop) * (0.1 + coef*0.9)
			if score > 6 {
				plan.cross[sy][sx] = 1
				if score > 9 {
					plan.cross[sy][sx] = 2
				}
			}
		}
	}
	return plan
}

func markSquareAxis(plan *roadPlan, sx, sy, dir, style int) {
	plan.stamps = append(plan.stamps, roadStamp{sx: sx, sy: sy, dir: dir, style: style})
	x0 := sx * squareSide
	y0 := sy * squareSide
	st := uint8(style) //#nosec G115 -- style is 0..2
	if dir == 0 {
		if y0 < 0 || y0 >= mapRows {
			return
		}
		for x := x0; x < x0+squareSide && x < mapCols; x++ {
			if x < 0 {
				continue
			}
			plan.grid[y0][x] = cellRoad
			if st > plan.pave[y0][x] {
				plan.pave[y0][x] = st
			}
			plan.dirs[y0][x] |= roadDir0
		}
		return
	}
	if x0 < 0 || x0 >= mapCols {
		return
	}
	for y := y0; y < y0+squareSide && y < mapRows; y++ {
		if y < 0 {
			continue
		}
		plan.grid[y][x0] = cellRoad
		if st > plan.pave[y][x0] {
			plan.pave[y][x0] = st
		}
		plan.dirs[y][x0] |= roadDir1
	}
}
