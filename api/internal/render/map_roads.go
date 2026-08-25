package render

import (
	"image"
	"image/color"
	"sort"
)

func isoTileCorners(topX, topY int) (t, r, b, l image.Point) {
	hw := isoTileW / 2
	hh := isoTileH / 2
	return image.Pt(topX, topY),
		image.Pt(topX+hw, topY+hh),
		image.Pt(topX, topY+isoTileH),
		image.Pt(topX-hw, topY+hh)
}

// drawRoadUnderlay paints continuous road diamonds (quad fill + seam stitch)
// beneath mcRoad sprites. Quad edges follow 2:1 iso geometry more cleanly than
// the row-span diamond helper, which reduces stair-step sparkle on long runs.
func drawRoadUnderlay(img *image.RGBA, grid cityGrid, dy int) {
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] != cellRoad {
				continue
			}
			topX, topY := isoCell(x, y)
			topY += dy
			t, r, b, l := isoTileCorners(topX, topY)
			fillConvexQuad(img, t, r, b, l, roadColor)
		}
	}
	stitchRoadSeams(img, grid, dy)
	fillInteriorRoadSeams(img)
}

func drawRoadNetwork(img *image.RGBA, grid cityGrid) {
	drawRoadUnderlay(img, grid, 0)
	softenRoadGrassBoundary(img)
}

// softenRoadGrassBoundary adds a mid-tone fringe between road and grass so
// long iso diagonals read less stair-stepped against the bright ground.
func softenRoadGrassBoundary(img *image.RGBA) {
	b := img.Bounds()
	type pix struct {
		x, y int
		c    color.RGBA
	}
	var fringe []pix
	dirs := [][2]int{
		{-1, 0}, {1, 0}, {0, -1}, {0, 1},
		{-1, -1}, {-1, 1}, {1, -1}, {1, 1},
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			here := img.RGBAAt(x, y)
			if here != roadColor {
				continue
			}
			for _, d := range dirs {
				nx, ny := x+d[0], y+d[1]
				if !image.Pt(nx, ny).In(b) {
					continue
				}
				n := img.RGBAAt(nx, ny)
				if !isGrassLike(n) {
					continue
				}
				weight := uint8(140)
				if d[0] != 0 && d[1] != 0 {
					weight = 90
				}
				fringe = append(fringe, pix{nx, ny, blendRGBA(roadColor, n, weight)})
			}
		}
	}
	for _, p := range fringe {
		img.SetRGBA(p.x, p.y, p.c)
	}
}

func blendRGBA(a, b color.RGBA, aWeight uint8) color.RGBA {
	bw := uint16(255 - aWeight)
	aw := uint16(aWeight)
	// Channel blend is bounded: max (255*255)/255 = 255.
	return color.RGBA{
		R: uint8((uint16(a.R)*aw + uint16(b.R)*bw) / 255), //#nosec G115 -- see above
		G: uint8((uint16(a.G)*aw + uint16(b.G)*bw) / 255), //#nosec G115 -- see above
		B: uint8((uint16(a.B)*aw + uint16(b.B)*bw) / 255), //#nosec G115 -- see above
		A: 255,
	}
}

// stitchRoadSeams repaints shared edges between adjacent road cells so integer
// iso diamond rasterization cannot leave 1px grass-colored gaps.
func stitchRoadSeams(img *image.RGBA, grid cityGrid, dy int) {
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			if grid[y][x] != cellRoad {
				continue
			}
			topX, topY := isoCell(x, y)
			topY += dy
			t, r, b, l := isoTileCorners(topX, topY)
			if x+1 < mapCols && grid[y][x+1] == cellRoad {
				drawThickLine(img, r, b, roadColor, 1)
			}
			if y+1 < mapRows && grid[y+1][x] == cellRoad {
				drawThickLine(img, l, b, roadColor, 1)
			}
			if x > 0 && grid[y][x-1] == cellRoad {
				drawThickLine(img, t, l, roadColor, 1)
			}
			if y > 0 && grid[y-1][x] == cellRoad {
				drawThickLine(img, t, r, roadColor, 1)
			}
		}
	}
}

func fillConvexQuad(img *image.RGBA, t, r, b, l image.Point, c color.RGBA) {
	fillTriangle(img, t, r, b, c)
	fillTriangle(img, t, l, b, c)
}

func fillTriangle(img *image.RGBA, p0, p1, p2 image.Point, c color.RGBA) {
	pts := []image.Point{p0, p1, p2}
	sort.Slice(pts, func(i, j int) bool { return pts[i].Y < pts[j].Y })
	yMin, yMax := pts[0].Y, pts[2].Y
	for y := yMin; y <= yMax; y++ {
		var xs []int
		for i := 0; i < 3; i++ {
			a, b := pts[i], pts[(i+1)%3]
			if a.Y == b.Y {
				continue
			}
			if y < min(a.Y, b.Y) || y > max(a.Y, b.Y) {
				continue
			}
			x := a.X + (y-a.Y)*(b.X-a.X)/(b.Y-a.Y)
			xs = append(xs, x)
		}
		if len(xs) < 2 {
			continue
		}
		sort.Ints(xs)
		for x := xs[0]; x <= xs[len(xs)-1]; x++ {
			if !image.Pt(x, y).In(img.Bounds()) {
				continue
			}
			img.SetRGBA(x, y, c)
		}
	}
}

func drawThickLine(img *image.RGBA, p0, p1 image.Point, c color.RGBA, radius int) {
	dx := abs(p1.X - p0.X)
	dy := -abs(p1.Y - p0.Y)
	sx, sy := 1, 1
	if p0.X > p1.X {
		sx = -1
	}
	if p0.Y > p1.Y {
		sy = -1
	}
	err := dx + dy
	x, y := p0.X, p0.Y
	for {
		stampDisk(img, x, y, radius, c)
		if x == p1.X && y == p1.Y {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x += sx
		}
		if e2 <= dx {
			err += dx
			y += sy
		}
	}
}

func stampDisk(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
	r2 := radius * radius
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy > r2 {
				continue
			}
			px, py := cx+dx, cy+dy
			if !image.Pt(px, py).In(img.Bounds()) {
				continue
			}
			img.SetRGBA(px, py, c)
		}
	}
}

func fillInteriorRoadSeams(img *image.RGBA) {
	b := img.Bounds()
	for pass := 0; pass < 2; pass++ {
		changed := false
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if !shouldFillRoadSeam(img, x, y) {
					continue
				}
				img.SetRGBA(x, y, roadColor)
				changed = true
			}
		}
		if !changed {
			break
		}
	}
}

func roadNeighbors4(img image.Image, x, y int) (n, s, e, w bool) {
	if colorAt(img, x, y-1) == roadColor {
		n = true
	}
	if colorAt(img, x, y+1) == roadColor {
		s = true
	}
	if colorAt(img, x+1, y) == roadColor {
		e = true
	}
	if colorAt(img, x-1, y) == roadColor {
		w = true
	}
	return n, s, e, w
}

func shouldFillRoadSeam(img image.Image, x, y int) bool {
	if !isGrassLike(colorAt(img, x, y)) {
		return false
	}
	n, s, e, w := roadNeighbors4(img, x, y)
	// Only fill true 1px sandwiches between opposite road edges.
	// The old ">=3 neighbors" rule bled road paint into adjacent lots.
	return (n && s) || (e && w)
}

func colorAt(img image.Image, x, y int) color.RGBA {
	if !image.Pt(x, y).In(img.Bounds()) {
		return color.RGBA{}
	}
	c := color.RGBAModel.Convert(img.At(x, y))
	if rgba, ok := c.(color.RGBA); ok {
		return rgba
	}
	return color.RGBA{A: 255}
}

func isGrassLike(c color.RGBA) bool {
	if c == grassColor {
		return true
	}
	return c.A == 255 && c.G >= 190 && c.G <= 220 && c.R >= 135 && c.R <= 170 && c.B >= 70 && c.B <= 95
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
