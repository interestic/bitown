package render

import "image"

// farmsEnabled is Townzzy's city-pop gate (Game.hx has none). Empty cells then
// follow genSquare neighbor density, not a city-wide carpet.
func farmsEnabled(pop int) bool {
	return pop >= 3
}

// farmMiniKeys are 4-cell field clips (~WW×4 × HH×4), Townzzy auth_champs_gfx.
// Empty minis of density squares (Game.hx size=1 type=2) — not catalog pools.
var farmMiniKeys = []string{
	"sprites/DefineSprite_521/1_v00.png", // yellow fill
	"sprites/DefineSprite_521/2_v00.png", // grass fill
	"sprites/DefineSprite_521/3_v00.png", // soil furrow
	"sprites/DefineSprite_521/4_v00.png", // quad mix
	// 521/5 (grass+hut+yellow) omitted from farms (#113); drawn via parkDecoKeys.
	"sprites/DefineSprite_503/1_v00.png", // yellow furrow EW
}

// farmBigKeys are 10-cell field clips, Townzzy auth_big_champs_gfx (size=0 type=15).
// Density-0 fringe squares — explicit list, not the commercial building pool.
var farmBigKeys = []string{
	// 401/1 (soil+hut) omitted — baked hut reads as a building on the field (#113)
	// 401/2 (soil grid) and 401/6 (checkered grass) omitted — baked grids read as
	// the same yellow-green mesh bug as per-cell clip seams (#116).
	"sprites/DefineSprite_401/3_v00.png", // pumpkins
	"sprites/DefineSprite_401/4_v00.png", // yellow fill
	"sprites/DefineSprite_401/5_v00.png", // grass fill
}

// miniSquareOrigin returns the genMiniSquare origin inside a 10×10 square for
// quadrant i in 0..3 (NW, NE, SW, SE), matching zone_density.go.
func miniSquareOrigin(baseX, baseY, i int) (int, int) {
	px := (i % 2) * 4
	py := (i / 2) * 4
	if px > 1 {
		px++
	}
	if py > 1 {
		py++
	}
	return baseX + px, baseY + py
}

// Mini quadrant indices inside a Game.hx square (viewer-facing names).
const (
	miniNW = 0
	miniNE = 1 // right
	miniSW = 2 // left
	miniSE = 3 // front
)

// westMiniStampNudgeX shifts SW-mini (viewer-left) building stamps right in
// screen space so peon yards stay on the plate west tip (sandbox map-base #07).
const westMiniStampNudgeX = 10

// northMiniStampNudgeX / Y shift NW-mini (viewer-top) building stamps in
// screen space so peon yards stay on the plate north tip (sandbox map-base #08).
// +Y keeps yards on-plate; +X nudges toward the upper-right of the tip.
const northMiniStampNudgeX = 3
const northMiniStampNudgeY = 10

// seMiniStampNudgeY shifts SE-mini buildings toward the viewer tip (screen +Y)
// so yards stay in the front quadrant (sandbox map-base #15).
const seMiniStampNudgeY = 3

// ewMiniStampNudgeY shifts SW/NE (left/right) buildings down in screen space
// so yards sit below the CROSS arms (sandbox map-base #15).
const ewMiniStampNudgeY = 5

// eastMiniStampNudgeX shifts NE (right) buildings left in screen space
// (negative = toward CROSS / plate center; sandbox map-base #15).
const eastMiniStampNudgeX = -1

// Arterial yard stamp extras (sandbox map-base #17 + #20 mid-rise fine → core).
// Applied after the shared mini nudges so mid-rise yards clear CROSS / arterial
// axes. Screen space: +Y = down (viewer), −Y = 上, −X = 左.
// #20 refined #17 on DefineSprite_493 mid-rise around a plate CROSS.
const arterialYardLiftY = plateGrassLift // +20 after grass-top foot
const arterialSENudgeX = -2              // 下象限
const arterialSENudgeY = -2              // 下象限
const arterialNWNudgeX = -6              // 上象限
const arterialNWNudgeY = -11             // 上象限
const arterialSWNudgeX = -11             // 左象限
const arterialSWNudgeY = -1              // 左象限
const arterialNENudgeX = -4              // 右象限
const arterialNENudgeY = -1              // 右象限

// lotInSquareMini reports whether (x,y) sits in mini i's 4×4 block.
func lotInSquareMini(x, y, i int) bool {
	if x < 0 || y < 0 || x >= mapCols || y >= mapRows {
		return false
	}
	baseX := (x / squareSide) * squareSide
	baseY := (y / squareSide) * squareSide
	ox, oy := miniSquareOrigin(baseX, baseY, i)
	return x >= ox && x < ox+4 && y >= oy && y < oy+4
}

// applyWestMiniStampNudge adds westMiniStampNudgeX when the lot is in the SW mini.
func applyWestMiniStampNudge(footX, lotX, lotY int) int {
	if lotInSquareMini(lotX, lotY, miniSW) {
		return footX + westMiniStampNudgeX
	}
	return footX
}

// applyNorthMiniStampNudgeX adds northMiniStampNudgeX when the lot is in the NW mini.
func applyNorthMiniStampNudgeX(footX, lotX, lotY int) int {
	if lotInSquareMini(lotX, lotY, miniNW) {
		return footX + northMiniStampNudgeX
	}
	return footX
}

// applyNorthMiniStampNudge adds northMiniStampNudgeY when the lot is in the NW mini.
func applyNorthMiniStampNudge(footY, lotX, lotY int) int {
	if lotInSquareMini(lotX, lotY, miniNW) {
		return footY + northMiniStampNudgeY
	}
	return footY
}

// applySEMiniStampNudge adds seMiniStampNudgeY when the lot is in the SE mini.
func applySEMiniStampNudge(footY, lotX, lotY int) int {
	if lotInSquareMini(lotX, lotY, miniSE) {
		return footY + seMiniStampNudgeY
	}
	return footY
}

// applyEWMiniStampNudge adds ewMiniStampNudgeY when the lot is in SW or NE mini.
func applyEWMiniStampNudge(footY, lotX, lotY int) int {
	if lotInSquareMini(lotX, lotY, miniSW) || lotInSquareMini(lotX, lotY, miniNE) {
		return footY + ewMiniStampNudgeY
	}
	return footY
}

// applyEastMiniStampNudge adds eastMiniStampNudgeX when the lot is in the NE mini.
func applyEastMiniStampNudge(footX, lotX, lotY int) int {
	if lotInSquareMini(lotX, lotY, miniNE) {
		return footX + eastMiniStampNudgeX
	}
	return footX
}

// applyArterialYardStampNudgeForMini adds sandbox #17/#20 yard lift + per-mini fine
// nudges by mini index. Use when the caller already knows the mini (map-base lab
// plates are not always aligned to the global 10×10 square grid).
func applyArterialYardStampNudgeForMini(footX, footY, mini int) (int, int) {
	footY += arterialYardLiftY
	switch mini {
	case miniSE:
		footX += arterialSENudgeX
		footY += arterialSENudgeY
	case miniNW:
		footX += arterialNWNudgeX
		footY += arterialNWNudgeY
	case miniSW:
		footX += arterialSWNudgeX
		footY += arterialSWNudgeY
	case miniNE:
		footX += arterialNENudgeX
		footY += arterialNENudgeY
	}
	return footX, footY
}

// applyArterialYardStampNudge adds sandbox #17/#20 yard lift + per-mini fine
// nudges. Call only for arterial (non-roadless) building stamps.
func applyArterialYardStampNudge(footX, footY, lotX, lotY int) (int, int) {
	for _, mi := range []int{miniSE, miniNW, miniSW, miniNE} {
		if lotInSquareMini(lotX, lotY, mi) {
			return applyArterialYardStampNudgeForMini(footX, footY, mi)
		}
	}
	footY += arterialYardLiftY
	return footX, footY
}

// miniSquareFoot is the SE cell of the 4×4 mini (ox+3, oy+3).
// Older Townzzy notes used origin+(4,4) (plate gap); that put farm stamps on
// the shared center and painted under houses in sibling minis (#116).
func miniSquareFoot(ox, oy int) (int, int) {
	return ox + 3, oy + 3
}

// squareFarmFoot is the SE cell of a 10×10 square, matching drawGroundBlocks.
func squareFarmFoot(bx, by int) (int, int) {
	return bx + squareSide - 1, by + squareSide - 1
}

func cellBlocksFarm(ctx mapRenderCtx, x, y int) bool {
	return cellBlocksFarmOcc(ctx.occupancy, ctx.grid, x, y)
}

// markFarmLots reserves Game.hx farm cells as lotFarm before env trees so
// parks never sit on type 15 / mini type 2 cover (farm XOR forest).
func markFarmLots(occ map[[2]int]lotCell, slug string, pop int, dens popDensity, grid cityGrid, roadless bool) {
	if !farmsEnabled(pop) {
		return
	}
	active := activeSquareSide(pop)
	origin := activeSquareOrigin(pop)
	for sy := origin; sy < origin+active; sy++ {
		for sx := origin; sx < origin+active; sx++ {
			sqPop := dens.at(sx, sy)
			if sqPop >= csPopHuge {
				continue
			}
			if sqPop <= 0 {
				if !bigFarmEligibleOcc(occ, dens, grid, sx, sy) {
					continue
				}
				markSquareFarm(occ, pop, roadless, sx, sy)
				continue
			}
			markMiniFarms(occ, grid, slug, pop, roadless, sx, sy, sqPop)
		}
	}
}

func markSquareFarm(occ map[[2]int]lotCell, pop int, roadless bool, sx, sy int) {
	bx := sx * squareSide
	by := sy * squareSide
	fx, fy := squareFarmFoot(bx, by)
	if roadless {
		o := plateIslandOrigin(pop)
		e := plateIslandExtent(pop)
		x1, y1 := o+e, o+e
		if fx >= x1 {
			fx = x1 - 1
		}
		if fy >= y1 {
			fy = y1 - 1
		}
		if !inPlateIsland(pop, fx, fy) {
			return
		}
	}
	for y := by; y < by+squareSide && y < mapRows; y++ {
		if y < 0 {
			continue
		}
		for x := bx; x < bx+squareSide && x < mapCols; x++ {
			if x < 0 {
				continue
			}
			if !inPlateIsland(pop, x, y) {
				continue
			}
			markEmptyAsFarm(occ, x, y)
		}
	}
}

func markMiniFarms(occ map[[2]int]lotCell, grid cityGrid, slug string, pop int, roadless bool, sx, sy, sqPop int) {
	rep := squareMiniPops(slug, sx, sy, sqPop)
	baseX := sx * squareSide
	baseY := sy * squareSide
	any := false
	for i := 0; i < 4; i++ {
		if rep[i] > 0 {
			continue
		}
		ox, oy := miniSquareOrigin(baseX, baseY, i)
		if miniSquareOccupiedOcc(occ, grid, ox, oy) {
			continue
		}
		fx, fy := miniSquareFoot(ox, oy)
		if !inPlateIsland(pop, fx, fy) {
			continue
		}
		// Cover the 4×4 mini plus the 1-cell seam toward the plate gap so env
		// trees cannot sit where mini farm still read as field edge.
		for dy := 0; dy <= 4; dy++ {
			for dx := 0; dx <= 4; dx++ {
				markEmptyAsFarm(occ, ox+dx, oy+dy)
			}
		}
		any = true
	}
	if !any {
		return
	}
	// Mini farm clips spill across the square; keep remaining empty cells in
	// this square tree-free (Game.hx empty minis are farm, not forest).
	for y := baseY; y < baseY+squareSide && y < mapRows; y++ {
		if y < 0 {
			continue
		}
		for x := baseX; x < baseX+squareSide && x < mapCols; x++ {
			if x < 0 {
				continue
			}
			if !inPlateIsland(pop, x, y) {
				continue
			}
			markEmptyAsFarm(occ, x, y)
		}
	}
}

func markEmptyAsFarm(occ map[[2]int]lotCell, x, y int) {
	if x < 0 || y < 0 || x >= mapCols || y >= mapRows {
		return
	}
	lot, ok := occ[[2]int{x, y}]
	if !ok || lot.use != lotEmpty {
		return
	}
	lot.use = lotFarm
	lot.tag = ""
	occ[[2]int{x, y}] = lot
}

// expandFarmMargins keeps env trees off neighboring squares that large 401
// farm clips visually cover (238px ≈ one square, but foot spill remains).
func expandFarmMargins(occ map[[2]int]lotCell) {
	seeds := make([][2]int, 0, 128)
	for pos, lot := range occ {
		if lot.use == lotFarm {
			seeds = append(seeds, pos)
		}
	}
	for _, pos := range seeds {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				markEmptyAsFarm(occ, pos[0]+dx, pos[1]+dy)
			}
		}
	}
}

func miniSquareOccupiedOcc(occ map[[2]int]lotCell, grid cityGrid, ox, oy int) bool {
	for dy := 0; dy < 4; dy++ {
		for dx := 0; dx < 4; dx++ {
			if cellBlocksFarmOcc(occ, grid, ox+dx, oy+dy) {
				return true
			}
		}
	}
	fx, fy := miniSquareFoot(ox, oy)
	return cellBlocksFarmOcc(occ, grid, fx, fy)
}

func cellBlocksFarmOcc(occ map[[2]int]lotCell, grid cityGrid, x, y int) bool {
	if x < 0 || y < 0 || x >= mapCols || y >= mapRows {
		return false
	}
	if grid[y][x] == cellRoad {
		return true
	}
	lot, ok := occ[[2]int{x, y}]
	if !ok {
		return false
	}
	// lotFarm is farm cover itself — stamps still draw there.
	return lot.use == lotBuilding || lot.use == lotPark
}

func bigFarmEligibleOcc(occ map[[2]int]lotCell, dens popDensity, grid cityGrid, sx, sy int) bool {
	if dens.at(sx, sy) != 0 {
		return false
	}
	side := squareSidePop(dens, sx, sy)
	if side < farmBigSidePopMin || side >= farmBigSidePopMax {
		return false
	}
	if squareHasRoad(grid, sx, sy) {
		return false
	}
	if squareHasBuilding(occ, sx, sy) {
		return false
	}
	return true
}

// miniSquareOccupied reports whether the 4×4 mini (or its SE stamp foot)
// already has a building, park, or road.
func miniSquareOccupied(ctx mapRenderCtx, ox, oy int) bool {
	for dy := 0; dy < 4; dy++ {
		for dx := 0; dx < 4; dx++ {
			if cellBlocksFarm(ctx, ox+dx, oy+dy) {
				return true
			}
		}
	}
	fx, fy := miniSquareFoot(ox, oy)
	return cellBlocksFarm(ctx, fx, fy)
}

type farmStamp struct {
	fx, fy int
	key    string
	// cells is the iso block size to clip when painting (4 = mini, 10 = big).
	cells  int
	ox, oy int // block origin for clip (mini origin or square origin)
}

// Game.hx empty-square big farm: 2 ≤ sidePop < 50 and no roads.
const (
	farmBigSidePopMin = 2
	farmBigSidePopMax = 50
)

func collectFarmStamps(ctx mapRenderCtx) []farmStamp {
	if ctx.atlas == nil || !farmsEnabled(ctx.pop) {
		return nil
	}
	out := make([]farmStamp, 0, 32)
	active := activeSquareSide(ctx.pop)
	origin := activeSquareOrigin(ctx.pop)
	for sy := origin; sy < origin+active; sy++ {
		for sx := origin; sx < origin+active; sx++ {
			sqPop := ctx.dens.at(sx, sy)
			if sqPop >= csPopHuge {
				continue
			}
			if sqPop <= 0 {
				if stamp, ok := collectBigFarmStamp(ctx, sx, sy); ok {
					out = append(out, stamp)
				}
				continue
			}
			out = append(out, collectMiniFarmStamps(ctx, sx, sy, sqPop)...)
		}
	}
	return out
}

func collectMiniFarmStamps(ctx mapRenderCtx, sx, sy, sqPop int) []farmStamp {
	// Roadless hut feet sit inside their 4×4 (#118), so empty minis can stamp
	// beside sibling houses. Arterial maps keep Game.hx edge feet; skip the
	// whole square there so 521 cannot paint under those houses (#116).
	if !ctx.roadless && squareHasBuilding(ctx.occupancy, sx, sy) {
		return nil
	}
	rep := squareMiniPops(ctx.slug, sx, sy, sqPop)
	baseX := sx * squareSide
	baseY := sy * squareSide
	out := make([]farmStamp, 0, 4)
	for i := 0; i < 4; i++ {
		if rep[i] > 0 {
			continue
		}
		ox, oy := miniSquareOrigin(baseX, baseY, i)
		if miniSquareOccupied(ctx, ox, oy) {
			continue
		}
		fx, fy := miniSquareFoot(ox, oy)
		if !inPlateIsland(ctx.pop, fx, fy) {
			continue
		}
		if grassMaskRimDist(ctx.pop, fx, fy) < 1 {
			continue
		}
		key := ctx.atlas.PickFarmKey(hashCell(ctx.slug, fx, fy))
		if key == "" {
			continue
		}
		out = append(out, farmStamp{fx: fx, fy: fy, key: key, cells: 4, ox: ox, oy: oy})
	}
	return out
}

func collectBigFarmStamp(ctx mapRenderCtx, sx, sy int) (farmStamp, bool) {
	if !bigFarmEligible(ctx, sx, sy) {
		return farmStamp{}, false
	}
	bx := sx * squareSide
	by := sy * squareSide
	fx, fy := squareFarmFoot(bx, by)
	if ctx.roadless {
		o := plateIslandOrigin(ctx.pop)
		e := plateIslandExtent(ctx.pop)
		x1, y1 := o+e, o+e
		if fx >= x1 {
			fx = x1 - 1
		}
		if fy >= y1 {
			fy = y1 - 1
		}
		if !inPlateIsland(ctx.pop, fx, fy) {
			return farmStamp{}, false
		}
	}
	if grassMaskRimDist(ctx.pop, fx, fy) < 1 {
		return farmStamp{}, false
	}
	key := ctx.atlas.PickBigFarmKey(hashCell(ctx.slug, fx, fy))
	if key == "" {
		return farmStamp{}, false
	}
	return farmStamp{fx: fx, fy: fy, key: key, cells: squareSide, ox: bx, oy: by}, true
}

func squareSidePop(dens popDensity, sx, sy int) int {
	return dens.at(sx-1, sy) + dens.at(sx+1, sy) + dens.at(sx, sy-1) + dens.at(sx, sy+1)
}

func squareHasRoad(grid cityGrid, sx, sy int) bool {
	x0 := sx * squareSide
	y0 := sy * squareSide
	for y := y0; y < y0+squareSide && y < mapRows; y++ {
		if y < 0 {
			continue
		}
		for x := x0; x < x0+squareSide && x < mapCols; x++ {
			if x < 0 {
				continue
			}
			if grid[y][x] == cellRoad {
				return true
			}
		}
	}
	return false
}

func squareHasBuilding(occ map[[2]int]lotCell, sx, sy int) bool {
	x0 := sx * squareSide
	y0 := sy * squareSide
	for y := y0; y < y0+squareSide && y < mapRows; y++ {
		if y < 0 {
			continue
		}
		for x := x0; x < x0+squareSide && x < mapCols; x++ {
			if x < 0 {
				continue
			}
			lot, ok := occ[[2]int{x, y}]
			if ok && lot.use == lotBuilding {
				return true
			}
		}
	}
	return false
}

func bigFarmEligible(ctx mapRenderCtx, sx, sy int) bool {
	return bigFarmEligibleOcc(ctx.occupancy, ctx.dens, ctx.grid, sx, sy)
}

// drawFarmBlocks stamps farm after plates and before arterial roads /
// object sprites. Density-0 fringe squares get full-square big farm; density
// squares get 4-cell mini farm on empty minis (Game.hx genSquare).
//
// Clip with one continuous iso block diamond (not a per-cell union — integer
// diamond gaps read as yellow-green grid lines on roadless grass). Hut feet sit
// inside their mini (#118), so neighbor 401 / mini stamps stay off house cells
// without a punch-out around sprites.
func drawFarmBlocks(img *image.RGBA, ctx mapRenderCtx) {
	for _, s := range collectFarmStamps(ctx) {
		// Farm panels (401/521) and roads share the full soil lip. Clip to
		// plate grass so arterial stamps stay on the city diamond.
		footX, footY := overlayFoot(s.fx, s.fy, farmGrassLift)
		dy := -farmGrassLift
		n := s.cells
		if n <= 0 {
			n = 4
		}
		ox, oy := s.ox, s.oy
		_ = ctx.atlas.drawFrameMasked(img, s.key, footX, footY, func(px, py int) bool {
			if !grassTopPixelSupported(ctx.grass, px, py) {
				return false
			}
			return pointInIsoBlockOffset(px, py, ox, oy, n, dy, 0)
		})
	}
}

func pickFarmFrame(a *Atlas, keys []string, seed uint32) string {
	if a == nil || len(keys) == 0 {
		return ""
	}
	start := int(seed % uint32(len(keys))) //#nosec G115 -- pool length is compile-time fixed
	for i := 0; i < len(keys); i++ {
		key := keys[(start+i)%len(keys)]
		if _, ok := a.Frames[key]; ok {
			return key
		}
	}
	return ""
}

// PickFarmKey chooses a deterministic mini farm frame from farmMiniKeys.
func (a *Atlas) PickFarmKey(seed uint32) string {
	return pickFarmFrame(a, farmMiniKeys, seed)
}

// PickBigFarmKey chooses a deterministic full-square farm frame from farmBigKeys.
func (a *Atlas) PickBigFarmKey(seed uint32) string {
	return pickFarmFrame(a, farmBigKeys, seed)
}
