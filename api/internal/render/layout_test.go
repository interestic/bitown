package render

import (
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestBuildCityGridConnectsRoads(t *testing.T) {
	grid := buildCityGridForCity(&citycore.City{Slug: "testcity", Pop: 500})

	roadCount := 0
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] != cellRoad {
				continue
			}
			roadCount++
		}
	}
	if roadCount == 0 {
		t.Fatal("expected roads")
	}

	again := buildCityGridForCity(&citycore.City{Slug: "testcity", Pop: 500})
	if again[3][3] != grid[3][3] {
		t.Fatal("expected deterministic grid for same slug")
	}
}

func TestLotsStayInsideSquares(t *testing.T) {
	grid := buildCityGridForCity(&citycore.City{Slug: "lot-check", Pop: 500})
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if x%squareSide == 0 || y%squareSide == 0 {
				continue
			}
			if grid[y][x] == cellRoad {
				t.Fatalf("interior lot %d,%d should not be a road cell", x, y)
			}
		}
	}
}

func TestCityGridUsesSquareSideNesting(t *testing.T) {
	if mapCols != displaySide*squareSide || mapRows != displaySide*squareSide {
		t.Fatalf("map %dx%d, want displaySide(%d)*squareSide(%d)", mapCols, mapRows, displaySide, squareSide)
	}
	plan := planRoadsForCity(&citycore.City{Slug: "nest-check", Pop: 500})
	for _, st := range plan.stamps {
		if st.sx < 0 || st.sy < 0 || st.sx >= displaySide || st.sy >= displaySide {
			t.Fatalf("stamp square (%d,%d) out of crop", st.sx, st.sy)
		}
		x0, y0 := st.sx*squareSide, st.sy*squareSide
		if plan.grid[y0][x0] != cellRoad {
			t.Fatalf("square origin (%d,%d) should be a road cell", x0, y0)
		}
	}
}

func TestRoadlessCityHasNoArterials(t *testing.T) {
	grid := buildCityGridForCity(&citycore.City{Slug: "roadless", Pop: 1})
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] == cellRoad {
				t.Fatalf("roadless map should have no roads, found at (%d,%d)", x, y)
			}
		}
	}
}

func TestTransportUnlocksArterials(t *testing.T) {
	// pop<80 would be roadless (no streets) without Tra; Game.hx c*roadCoef still
	// needs some square density, so this is a small town rather than pop=1.
	grid := buildCityGridForCity(&citycore.City{Slug: "tra", Pop: 40, Tra: 10})
	found := false
	for y := 0; y < mapRows && !found; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] == cellRoad {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("Tra>0 should unlock arterial roads even at low pop")
	}
}

func TestPlanRoadsUsesOneStampPerEdge(t *testing.T) {
	plan := planRoadsForCity(&citycore.City{Slug: "testcity", Pop: 80})
	if len(plan.stamps) == 0 {
		t.Fatal("expected square-edge stamps")
	}
	counts := map[[2]int]int{}
	for _, st := range plan.stamps {
		counts[[2]int{st.sx, st.sy}]++
		if counts[[2]int{st.sx, st.sy}] > 2 {
			t.Fatalf("square (%d,%d) has more than 2 edge stamps", st.sx, st.sy)
		}
	}
}

func TestRoadStyleFromScoreThresholds(t *testing.T) {
	if roadStyleFromScore(0.9) != -1 {
		t.Fatal("score < 1 should skip the axis")
	}
	if roadStyleFromScore(1) != roadStyleThin {
		t.Fatal("score 1 should be 細暗")
	}
	if roadStyleFromScore(1.5) != roadStyleDirt {
		t.Fatal("score 1.5 should be 土")
	}
	if roadStyleFromScore(5) != roadStylePave {
		t.Fatal("score 5 should be アスファルト")
	}
}

func TestPlanRoadsUsesDirtThenAsphalt(t *testing.T) {
	dens := uniformDensity(2)
	// Dense cells must sit inside the live island (Game.hx-aligned), not the
	// geometric corner of the displaySide crop.
	o80 := activeSquareOrigin(80)
	o500 := activeSquareOrigin(500)
	dens.cells[o80][o80] = 8
	a500 := activeSquareSide(500)
	cx := o500 + a500/2
	dens.cells[cx][cx] = 40
	dens.max = 40
	low := planRoads(&citycore.City{Slug: "dirt-town", Pop: 80}, dens)
	if !planHasStyle(low, roadStyleDirt) && !planHasStyle(low, roadStyleThin) {
		t.Fatalf("low-density squares should get dirt/thin roads, stamps=%v", stampStyles(low))
	}

	high := planRoads(&citycore.City{Slug: "pave-town", Pop: 500}, dens)
	if !planHasStyle(high, roadStylePave) {
		t.Fatalf("dense downtown square should pave, stamps=%v", stampStyles(high))
	}
	if !planHasStyle(high, roadStyleDirt) && !planHasStyle(high, roadStyleThin) {
		t.Fatal("crop rim should keep dirt/thin roads")
	}
}

func uniformDensity(v int) popDensity {
	cells := make([][]int, displaySide)
	for y := 0; y < displaySide; y++ {
		cells[y] = make([]int, displaySide)
		for x := 0; x < displaySide; x++ {
			cells[y][x] = v
		}
	}
	return popDensity{cells: cells, max: v}
}

func planHasStyle(plan roadPlan, style int) bool {
	for _, st := range plan.stamps {
		if st.style == style {
			return true
		}
	}
	return false
}

func stampStyles(plan roadPlan) []int {
	out := make([]int, 0, len(plan.stamps))
	for _, st := range plan.stamps {
		out = append(out, st.style)
	}
	return out
}

func buildCityGridForCity(city *citycore.City) cityGrid {
	return planRoadsForCity(city).grid
}

func planRoadsForCity(city *citycore.City) roadPlan {
	if city == nil || !arterialsEnabled(city) {
		return emptyRoadPlan()
	}
	dens := genMapPop(city.Pop.Int(), newMapRNG(city.Slug.String()))
	return planRoads(city, dens)
}
