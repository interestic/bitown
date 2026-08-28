package render

import "image"

// groundDecoScatterKeys are cell-sized pebble clusters (no fence diamond).
// 514 alone — larger 516/518/520/521 frames need a mini footprint or the
// fence outline is clipped away and only white dots remain.
var groundDecoScatterKeys = []string{
	"sprites/DefineSprite_514/1_v00.png",
}

// groundDecoMiniKeys are ~92×50 Storybook ground clips with an iso fence
// diamond (516/518/520/521/6–8) plus recreational minis (521/5, 616, 627).
// Stamped like mini farms on vacant 4×4 grass.
var groundDecoMiniKeys = []string{
	"sprites/DefineSprite_516/1_v00.png",
	"sprites/DefineSprite_518/1_v00.png",
	"sprites/DefineSprite_520/1_v00.png",
	"sprites/DefineSprite_521/5_v00.png",
	"sprites/DefineSprite_521/6_v00.png",
	"sprites/DefineSprite_521/7_v00.png",
	"sprites/DefineSprite_521/8_v00.png",
	"sprites/DefineSprite_616/1_v00.png", // tennis
	"sprites/DefineSprite_627/1_v00.png", // ponds
}

type groundDecoStamp struct {
	fx, fy int
	ox, oy int
	cells  int
	key    string
}

// collectGroundDecoStamps places Storybook ground deco on vacant grass:
// no roads, plate rim, farms, or buildings. Scatter may sit under lotPark trees.
func collectGroundDecoStamps(ctx mapRenderCtx) []groundDecoStamp {
	if ctx.atlas == nil || ctx.city == nil {
		return nil
	}
	env := ctx.city.Env.Int()
	if env <= 0 {
		return nil
	}
	out := make([]groundDecoStamp, 0, 64)
	out = append(out, collectGroundDecoMinis(ctx, env)...)
	out = append(out, collectGroundDecoScatter(ctx, env)...)
	return out
}

func collectGroundDecoMinis(ctx mapRenderCtx, env int) []groundDecoStamp {
	// Prefer fenced/recreational minis on vacant 4×4; high enough to read at env=20+.
	chance := 220 + env
	if chance > 450 {
		chance = 450
	}
	active := activeSquareSide(ctx.pop)
	origin := activeSquareOrigin(ctx.pop)
	out := make([]groundDecoStamp, 0, 16)
	for sy := origin; sy < origin+active; sy++ {
		for sx := origin; sx < origin+active; sx++ {
			baseX := sx * squareSide
			baseY := sy * squareSide
			for i := 0; i < 4; i++ {
				ox, oy := miniSquareOrigin(baseX, baseY, i)
				if !miniSquareVacantForGroundDeco(ctx, ox, oy) {
					continue
				}
				fx, fy := miniSquareFoot(ox, oy)
				seed := hashCell(ctx.slug, fx, fy)
				if int(seed%1000) >= chance { //#nosec G115
					continue
				}
				key := pickFarmFrame(ctx.atlas, groundDecoMiniKeys, seed)
				if key == "" {
					continue
				}
				out = append(out, groundDecoStamp{fx: fx, fy: fy, ox: ox, oy: oy, cells: 4, key: key})
			}
		}
	}
	return out
}

func collectGroundDecoScatter(ctx mapRenderCtx, env int) []groundDecoStamp {
	out := make([]groundDecoStamp, 0, 32)
	for pos, lot := range ctx.occupancy {
		if lot.use != lotEmpty && lot.use != lotPark {
			continue
		}
		x, y := pos[0], pos[1] //#nosec G602 -- map key is [2]int
		if !groundDecoCellOK(ctx, x, y) {
			continue
		}
		// Keep 514 rare on open grass; slightly more under trees (trees-plot).
		chance := 3
		if lot.use == lotPark {
			chance = 25
		}
		_ = env
		seed := hashCell(ctx.slug, x+17, y+31)
		if int(seed%1000) >= chance { //#nosec G115
			continue
		}
		key := pickFarmFrame(ctx.atlas, groundDecoScatterKeys, seed)
		if key == "" {
			continue
		}
		out = append(out, groundDecoStamp{fx: x, fy: y, ox: x, oy: y, cells: 1, key: key})
	}
	return out
}

func miniSquareVacantForGroundDeco(ctx mapRenderCtx, ox, oy int) bool {
	for dy := 0; dy < 4; dy++ {
		for dx := 0; dx < 4; dx++ {
			if !groundDecoCellOK(ctx, ox+dx, oy+dy) {
				return false
			}
			lot, ok := ctx.occupancy[[2]int{ox + dx, oy + dy}]
			if !ok || lot.use != lotEmpty {
				return false
			}
		}
	}
	fx, fy := miniSquareFoot(ox, oy)
	return groundDecoCellOK(ctx, fx, fy)
}

func groundDecoCellOK(ctx mapRenderCtx, x, y int) bool {
	if x < 0 || y < 0 || x >= mapCols || y >= mapRows {
		return false
	}
	if ctx.grid[y][x] == cellRoad {
		return false
	}
	if !inPlateIsland(ctx.pop, x, y) {
		return false
	}
	if !grassTopCell(ctx.pop, x, y) {
		return false
	}
	return true
}

// drawGroundDecoBlocks stamps recreational / scatter ground after farms and
// before arterial roads so clips sit on the plate like farm mats, not trees.
func drawGroundDecoBlocks(img *image.RGBA, ctx mapRenderCtx) {
	for _, s := range collectGroundDecoStamps(ctx) {
		footX, footY := overlayFoot(s.fx, s.fy, farmGrassLift)
		dy := -farmGrassLift
		n := s.cells
		if n <= 0 {
			n = 1
		}
		ox, oy := s.ox, s.oy
		// Mini plots (516 fence, 521/5, tennis) overhang a tight 4×4 diamond;
		// expand from center so the NE corner is not shaved off.
		expand := 0.0
		if n >= 4 {
			expand = 0.55
		}
		_ = ctx.atlas.drawFrameMasked(img, s.key, footX, footY, func(px, py int) bool {
			if !grassTopPixelSupported(ctx.grass, px, py) {
				return false
			}
			return pointInIsoBlockOffset(px, py, ox, oy, n, dy, expand)
		})
	}
}
