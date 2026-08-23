package render

import "sort"

type mapCoord struct {
	x, y int
}

type mapObject struct {
	x, y   int
	depth  int
	height int
	seed   uint32
	tag    string
	key    string
	kind   objectKind
}

type objectKind int

const (
	objectRoad objectKind = iota
	objectPark
	objectBuilding
)

// mapDrawOrder returns cell coordinates back-to-front for 2:1 isometric overlap.
func mapDrawOrder() []mapCoord {
	cells := make([]mapCoord, 0, mapCols*mapRows)
	for y := 0; y < mapRows; y++ {
		for x := 0; x < mapCols; x++ {
			cells = append(cells, mapCoord{x: x, y: y})
		}
	}
	sort.Slice(cells, func(i, j int) bool {
		a, b := cells[i], cells[j]
		da, db := a.x+a.y, b.x+b.y
		if da != db {
			return da < db
		}
		if a.y != b.y {
			return a.y < b.y
		}
		return a.x < b.x
	})
	return cells
}

// sortMapObjects orders sprites back-to-front: depth (x+y), then sprite height,
// so taller frames on the same diagonal paint over shorter ones.
func sortMapObjects(objs []mapObject) {
	sort.SliceStable(objs, func(i, j int) bool {
		a, b := objs[i], objs[j]
		if a.depth != b.depth {
			return a.depth < b.depth
		}
		// Same diagonal: roads under parks under buildings (kind order).
		if a.kind != b.kind {
			return a.kind < b.kind
		}
		if a.height != b.height {
			return a.height < b.height
		}
		if a.y != b.y {
			return a.y < b.y
		}
		return a.x < b.x
	})
}

func (a *Atlas) frameHeight(key string) int {
	if a == nil || key == "" {
		return 0
	}
	rect, ok := a.Frames[key]
	if !ok {
		return 0
	}
	return rect.H
}
