package render

import "github.com/interestic/bitown/internal/citycore"

// landmarkStampLocal is the 10×10 square cell for mcHouse3 (sandbox #19).
// Interior grass, not rim / SE dalle foot / packed hut feet.
const landmarkStampLocal = 4

// landmarkStampNudgeY shifts mcHouse3 down in screen space (sandbox #19).
const landmarkStampNudgeY = 62

type landmarkStamp struct {
	x, y int
	key  string
}

func squareLandmarkCell(bx, by int) (x, y int) {
	return bx + landmarkStampLocal, by + landmarkStampLocal
}

func landmarkStampFoot(x, y int) (footX, footY int) {
	footX, footY = overlayFoot(x, y, plateGrassLift)
	footY += landmarkStampNudgeY
	return footX, footY
}

func squareIndex(x, y int) (sx, sy int) {
	return x / squareSide, y / squareSide
}

func squareHasLandmark(squares map[[2]int]struct{}, x, y int) bool {
	if len(squares) == 0 {
		return false
	}
	sx, sy := squareIndex(x, y)
	_, ok := squares[[2]int{sx, sy}]
	return ok
}

// planSquareLandmarks places at most one mcHouse3 per live square at the
// sandbox #19 foot. High pop + sec/com mix; skips packed yards on that square.
func planSquareLandmarks(city *citycore.City, atlas *Atlas, dens popDensity) ([]landmarkStamp, map[[2]int]struct{}) {
	if city == nil || atlas == nil {
		return nil, nil
	}
	pop := city.Pop.Int()
	chance := landmarkMixPermille(pop, city.Sec.Int(), city.Com.Int())
	if chance <= 0 {
		return nil, nil
	}
	slug := city.Slug.String()
	o := plateIslandOrigin(pop)
	e := plateIslandExtent(pop)
	type placed struct {
		sx, sy int
		folder string
	}
	var out []landmarkStamp
	var seen []placed
	squares := make(map[[2]int]struct{})
	for by := o; by+landmarkStampLocal < o+e; by += squareSide {
		for bx := o; bx+landmarkStampLocal < o+e; bx += squareSide {
			x, y := squareLandmarkCell(bx, by)
			if !inPlateIsland(pop, x, y) || !grassTopCell(pop, x, y) {
				continue
			}
			sx, sy := squareIndex(bx, by)
			seed := hashCell(slug, sx, sy)
			roll := int(((seed >> 16) ^ (seed >> 8) ^ seed) % 1000) //#nosec G115
			if roll >= chance {
				continue
			}
			avoid := map[string]struct{}{}
			for _, p := range seen {
				dx, dy := p.sx-sx, p.sy-sy
				if dx < 0 {
					dx = -dx
				}
				if dy < 0 {
					dy = -dy
				}
				if dx <= 1 && dy <= 1 {
					avoid[p.folder] = struct{}{}
				}
			}
			key := atlas.pickBuildingFrameForTagAvoiding(city, TagLandmark, 3, pop, dens.max, seed^0x9e3779b9, avoid)
			if key == "" {
				continue
			}
			out = append(out, landmarkStamp{x: x, y: y, key: key})
			squares[[2]int{sx, sy}] = struct{}{}
			seen = append(seen, placed{sx: sx, sy: sy, folder: spriteFolderBase(key)})
		}
	}
	return out, squares
}
