package render

import (
	"math"

	"github.com/interestic/bitown/internal/citycore"
)

// Cs.hx density thresholds (square density, not city population).
// densityHut mirrors Cs.POP_PEON.
const (
	csSide      = 30 // Cs.SIDE — full Flash square grid
	csPopHuge   = 200
	csPopBig    = 20
	csPopNormal = 2
	densityHut  = 3 // Cs.POP_PEON — hut-level square density

	// mcHouse export character ids (updateLib size frames).
	libHouse1 = 411
	libHouse2 = 522
	libHouse3 = 693
)

// popDensity is the displaySide×displaySide crop of Game.hx bmpPop
// (center of the virtual Cs.SIDE grid), after BlurFilter-style soften.
type popDensity struct {
	cells [][]int // [y][x], length displaySide
	max   int     // densityMax on the full SIDE grid (updateLib gate)
}

func getRayMax(n float64) float64 {
	if n < 0 {
		n = 0
	}
	return 1 + math.Pow(n, 0.6)*0.15
}

// flashDisplaySide mirrors Game.hx displaySide: getRayMax + displayMargin
// with Std.int truncation (not rounding). pop=1 → 6, matching Townzzy.
func flashDisplaySide(pop int) int {
	ray := getRayMax(float64(pop)) + 1
	dif := float64(csSide) - ray*2
	margin := int(math.Max(0, dif*0.5)) // Haxe Std.int toward zero
	side := csSide - 2*margin
	if side < 1 {
		side = 1
	}
	if side > csSide {
		side = csSide
	}
	return side
}

// activeSquareSide is how many of bitown's displaySide squares are live
// (Game.hx viewport, capped to our PNG crop).
func activeSquareSide(pop int) int {
	side := flashDisplaySide(pop)
	if side > displaySide {
		side = displaySide
	}
	if side < 1 {
		side = 1
	}
	return side
}

// blurDensity approximates flash.filters.BlurFilter(blurX=2, blurY=2) with a
// separable 3-tap box blur on the SIDE×SIDE field.
func blurDensity(src [][]int) [][]int {
	side := len(src)
	if side == 0 {
		return src
	}
	tmp := make([][]int, side)
	out := make([][]int, side)
	for y := 0; y < side; y++ {
		tmp[y] = make([]int, side)
		out[y] = make([]int, side)
	}
	// Horizontal.
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			sum, n := 0, 0
			for dx := -1; dx <= 1; dx++ {
				xx := x + dx
				if xx < 0 || xx >= side {
					continue
				}
				sum += src[y][xx]
				n++
			}
			tmp[y][x] = (sum + n/2) / n
		}
	}
	// Vertical.
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			sum, n := 0, 0
			for dy := -1; dy <= 1; dy++ {
				yy := y + dy
				if yy < 0 || yy >= side {
					continue
				}
				sum += tmp[yy][x]
				n++
			}
			v := (sum + n/2) / n
			if v > 255 {
				v = 255
			}
			out[y][x] = v
		}
	}
	return out
}

// genMapPop mirrors Game.hx on a virtual Cs.SIDE grid (with blur), then crops
// the center displaySide×displaySide squares (bitown's map.png viewport).
func genMapPop(pop int, rng *mapRNG) popDensity {
	full := make([][]int, csSide)
	for y := 0; y < csSide; y++ {
		full[y] = make([]int, csSide)
	}
	densityMax := 0
	if pop > 0 && rng != nil {
		n := 0
		cx := float64(csSide) * 0.5
		cy := float64(csSide) * 0.5
		for n < pop {
			inc := 1
			if n > 50 {
				inc = 5
			}
			rayMax := getRayMax(float64(n))
			ray := rng.Float() * rayMax
			a := rng.Float() * 6.28
			x := int(cx + math.Cos(a)*ray)
			y := int(cy + math.Sin(a)*ray)
			if x < 0 {
				x = 0
			}
			if y < 0 {
				y = 0
			}
			if y >= len(full) {
				y = len(full) - 1
			}
			row := full[y]
			if x >= len(row) {
				x = len(row) - 1
			}
			v := row[x] + inc
			if v > 255 {
				v = 255
			}
			row[x] = v
			if v > densityMax {
				densityMax = v
			}
			n += inc
		}
		raw := full
		full = blurDensity(full)
		densityMax = 0
		for y := 0; y < csSide; y++ {
			for x := 0; x < csSide; x++ {
				if full[y][x] > densityMax {
					densityMax = full[y][x]
				}
			}
		}
		// Integer 3-tap blur rounds 1-pixel deposits to 0. Flash BlurFilter
		// keeps 8-bit mass, so Game.hx neighbor farms exist at city pop=3.
		// Restore raw deposits only when blur wiped everything; pop=1 stays
		// empty (Townzzy initial 6×6 dalle).
		if densityMax == 0 && farmsEnabled(pop) {
			rawMax := 0
			for y := 0; y < csSide; y++ {
				for x := 0; x < csSide; x++ {
					if raw[y][x] > rawMax {
						rawMax = raw[y][x]
					}
				}
			}
			if rawMax > 0 {
				full = raw
				densityMax = rawMax
			}
		}
	}

	// Crop center displaySide squares (same idea as Game.hx displayMargin).
	margin := (csSide - displaySide) / 2
	if margin < 0 {
		margin = 0
	}
	cells := make([][]int, displaySide)
	for y := 0; y < displaySide; y++ {
		cells[y] = make([]int, displaySide)
		fy := y + margin
		if fy >= csSide {
			continue
		}
		for x := 0; x < displaySide; x++ {
			fx := x + margin
			if fx >= csSide {
				continue
			}
			cells[y][x] = full[fy][fx]
		}
	}
	return popDensity{cells: cells, max: densityMax}
}

func (d popDensity) at(sx, sy int) int {
	if sy < 0 || sx < 0 || sy >= len(d.cells) || sx >= len(d.cells[sy]) {
		return 0
	}
	return d.cells[sy][sx]
}

// squareOf returns the Game.hx square indices for a mini-cell.
func squareOf(x, y int) (sx, sy int) {
	return x / squareSide, y / squareSide
}

// localDensityAt returns the square density covering mini-cell (x,y).
func localDensityAt(d popDensity, x, y int) int {
	sx, sy := squareOf(x, y)
	return d.at(sx, sy)
}

// updateLibHouseUnlocked mirrors Game.hx updateLib size-frame skips.
// densityMax is the primary gate; cityPop is a fallback because bitown's ~500
// pop scale on SIDE=30 rarely reaches POP_HUGE the way Flash megacities do.
func updateLibHouseUnlocked(libraryID, densityMax, cityPop int) bool {
	switch libraryID {
	case libHouse1:
		return true
	case libHouse2:
		return densityMax >= csPopBig || cityPop >= houseBandPop
	case libHouse3:
		return densityMax >= csPopHuge || cityPop >= cityHugePop
	default:
		return true
	}
}

// maxTierForLocalDensity maps Game.hx genMiniSquare thresholds onto catalog
// tiers. updateLib (densityMax) gates which house libraries exist; local density
// still shapes per-lot caps.
func maxTierForLocalDensity(local, cityPop int) int {
	t := 0
	switch {
	case local <= 0:
		return 0
	case local < densityHut:
		t = 0
	case local < csPopBig:
		t = 1
	case local < csPopHuge:
		t = 2
	default:
		t = 3
	}
	if cityPop >= cityHugePop && local > 0 && t < 3 {
		t = 3
	} else if cityPop >= houseBandPop && local > 0 && t < 2 {
		t = 2
	} else if cityPop >= hutBandPop && local > 0 && t < 1 {
		t = 1
	}
	return t
}

// maxTierForLotWithLocal combines Game.hx local square density with the
// city-wide periphery cap (big_city house belt).
func maxTierForLotWithLocal(local, cityPop, ind, com, x, y int, tag string) int {
	max := maxTierForLocalDensity(local, cityPop)
	// Rim industrial/commercial zones sit on the crop edge where genMapPop is
	// often sparse; keep warehouses/shops placeable when sectors unlock them.
	const sectorMid = 50 // matches buildings.json min_ind/min_com for tier≥2
	if max < 2 && tag == TagIndustrial && cityPop >= houseBandPop && ind >= sectorMid {
		max = 2
	}
	if max < 2 && tag == TagCommercial && cityPop >= houseBandPop && com >= sectorMid {
		max = 2
	}
	// City-wide unlock still gates landmark-scale art at low total pop.
	if cityMax := maxTierForPop(cityPop); max > cityMax {
		max = cityMax
	}
	cx, cy := plateIslandCenter(cityPop)
	dx, dy := x-cx, y-cy
	dist2 := dx*dx + dy*dy
	outer := outerLotDist2ForPop(cityPop)
	if tag == TagResidential && cityPop < cityHugePop {
		outer = outer / 2
		if outer < 1 {
			outer = 1
		}
	}
	if tag != TagIndustrial && dist2 > outer && max > 0 {
		max--
	}
	if tag != TagIndustrial && dist2 >= outer*2 && max > 1 {
		max = 1
	}
	return max
}

// getBatType mirrors Game.hx weighted type pick (0=pop … 4=com).
func getBatType(city *citycore.City, x, y int, rng *mapRNG) int {
	if city == nil || rng == nil {
		return 0
	}
	cx, cy := plateIslandCenter(city.Pop.Int())
	dx := float64(x) - float64(cx)
	dy := float64(y) - float64(cy)
	dist := math.Sqrt(dx*dx + dy*dy)
	ray := getRayMax(float64(city.Pop.Int())) * float64(squareSide)
	coef := 0.0
	if ray > 0 {
		coef = dist / ray
	}
	weights := []int{
		city.Pop.Int(),
		int(float64(city.Ind.Int()) * (coef * 5)),
		city.Env.Int() * 2,
		int(float64(city.Sec.Int()) * 0.2),
		int(float64(city.Com.Int()) * 0.75),
	}
	sum := 0
	for _, w := range weights {
		if w > 0 {
			sum += w
		}
	}
	if sum <= 0 {
		return 0
	}
	rnd := int(rng.Float() * float64(sum))
	acc := 0
	typ := 0
	for i, w := range weights {
		if w <= 0 {
			continue
		}
		acc += w
		if acc > rnd {
			typ = i
			break
		}
		typ = i
	}
	return typ
}

func batTypeToTag(typ int) string {
	switch typ {
	case 1:
		return TagIndustrial
	case 2:
		return TagTree // env → park lot
	case 4:
		return TagCommercial
	default:
		// 0 pop, 3 sec (landmark mix handled at pick time)
		return TagResidential
	}
}
