package render

import "image"

// farmsEnabled mirrors Townzzy / Game.hx: empty cells get champs stamps
// once population reaches 3 (Beach City–class peon towns).
func farmsEnabled(pop int) bool {
	return pop >= 3
}

// farmMiniKeys are 4-cell field clips (~WW×4 × HH×4), Townzzy auth_champs_gfx.
// Arterial empty minis only — do not pick via building/commercial catalog pools.
var farmMiniKeys = []string{
	"sprites/DefineSprite_521/1_v00.png", // yellow fill
	"sprites/DefineSprite_521/2_v00.png", // grass fill
	"sprites/DefineSprite_521/3_v00.png", // soil furrow
	"sprites/DefineSprite_521/4_v00.png", // quad mix
	"sprites/DefineSprite_521/5_v00.png", // grass + hut + yellow
	"sprites/DefineSprite_503/1_v00.png", // yellow furrow EW
}

// farmBigKeys are 10-cell field clips, Townzzy auth_big_champs_gfx (size=0 type=15).
// Peon plates — explicit list, not the commercial building pool.
var farmBigKeys = []string{
	"sprites/DefineSprite_401/1_v00.png", // soil grid + hut
	"sprites/DefineSprite_401/2_v00.png", // soil grid
	"sprites/DefineSprite_401/3_v00.png", // pumpkins
	"sprites/DefineSprite_401/4_v00.png", // yellow fill
	"sprites/DefineSprite_401/5_v00.png", // grass fill
	"sprites/DefineSprite_401/6_v00.png", // checkered grass
}

// farmGrassLift raises peon big-champs so the field top sits on mcDalle grass.
// 401 is ~116px with foot 5px above the SE cell; dalle soil sides are ~20px
// (#83). Without the lift the clip hangs off the island bottom the same way
// trees used to sit on the brown rim.
const farmGrassLift = dalleGrassLift

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

// miniSquareFoot is Townzzy's empty-mini stamp: origin + (4,4).
func miniSquareFoot(ox, oy int) (int, int) {
	return ox + 4, oy + 4
}

// squareFarmFoot is the SE cell of a 10×10 square, matching drawGroundBlocks.
func squareFarmFoot(bx, by int) (int, int) {
	return bx + squareSide - 1, by + squareSide - 1
}

func cellBlocksFarm(ctx mapRenderCtx, x, y int) bool {
	if x < 0 || y < 0 || x >= mapCols || y >= mapRows {
		return false
	}
	if ctx.grid[y][x] == cellRoad {
		return true
	}
	lot, ok := ctx.occupancy[[2]int{x, y}]
	if !ok {
		return false
	}
	return lot.use == lotBuilding || lot.use == lotPark
}

// miniSquareOccupied reports whether the 4×4 mini or its Townzzy farm foot
// (origin+4,+4) already has a building, park, or road. The foot often sits in
// the 1-cell gap between minis — the same cell peon plate anchors use — so it
// must be checked explicitly (not only the 4×4).
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
}

func collectFarmStamps(ctx mapRenderCtx) []farmStamp {
	if ctx.atlas == nil || !farmsEnabled(ctx.pop) {
		return nil
	}
	if ctx.peon {
		return collectPeonBigFarmStamps(ctx)
	}
	return collectArterialMiniFarmStamps(ctx)
}

func collectArterialMiniFarmStamps(ctx mapRenderCtx) []farmStamp {
	out := make([]farmStamp, 0, 64)
	active := activeSquareSide(ctx.pop)
	origin := (displaySide - active) / 2
	for sy := origin; sy < origin+active; sy++ {
		for sx := origin; sx < origin+active; sx++ {
			baseX := sx * squareSide
			baseY := sy * squareSide
			for i := 0; i < 4; i++ {
				ox, oy := miniSquareOrigin(baseX, baseY, i)
				if miniSquareOccupied(ctx, ox, oy) {
					continue
				}
				fx, fy := miniSquareFoot(ox, oy)
				key := ctx.atlas.PickFarmKey(hashCell(ctx.slug, fx, fy))
				if key == "" {
					continue
				}
				out = append(out, farmStamp{fx: fx, fy: fy, key: key})
			}
		}
	}
	return out
}

func collectPeonBigFarmStamps(ctx mapRenderCtx) []farmStamp {
	out := make([]farmStamp, 0, 36)
	o := peonIslandOriginFor(ctx.pop)
	e := peonIslandExtentFor(ctx.pop)
	x1, y1 := o+e, o+e
	for by := o; by < y1; by += groundBlock {
		for bx := o; bx < x1; bx += groundBlock {
			fx, fy := squareFarmFoot(bx, by)
			if fx >= x1 {
				fx = x1 - 1
			}
			if fy >= y1 {
				fy = y1 - 1
			}
			if !inPeonIslandFor(ctx.pop, fx, fy) {
				continue
			}
			key := ctx.atlas.PickBigFarmKey(hashCell(ctx.slug, fx, fy))
			if key == "" {
				continue
			}
			out = append(out, farmStamp{fx: fx, fy: fy, key: key})
		}
	}
	return out
}

// drawFarmBlocks stamps champs after dalle plates and before arterial roads /
// object sprites. Peon plates get full-square big champs as floor (buildings
// and trees draw later). Arterial maps keep 4-cell mini champs on empty minis.
func drawFarmBlocks(img *image.RGBA, ctx mapRenderCtx) {
	for _, s := range collectFarmStamps(ctx) {
		topX, topY := isoCell(s.fx, s.fy)
		footX, footY := topX, topY+isoTileH
		if ctx.peon {
			footY -= farmGrassLift
			_ = ctx.atlas.drawFrameOnPeonGrass(img, s.key, footX, footY, ctx.grass)
		} else {
			_ = ctx.atlas.drawFrameAtFoot(img, s.key, footX, footY)
		}
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

// PickFarmKey chooses a deterministic mini champs frame from farmMiniKeys.
func (a *Atlas) PickFarmKey(seed uint32) string {
	return pickFarmFrame(a, farmMiniKeys, seed)
}

// PickBigFarmKey chooses a deterministic full-square champs frame from farmBigKeys.
func (a *Atlas) PickBigFarmKey(seed uint32) string {
	return pickFarmFrame(a, farmBigKeys, seed)
}
