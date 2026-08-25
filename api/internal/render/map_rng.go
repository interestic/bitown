package render

import "hash/fnv"

// mapRNG is a tiny deterministic PRNG seeded like Game.hx mt.Random usage
// (slug-derived). Not crypto; map layout only.
type mapRNG struct {
	state uint32
}

func newMapRNG(slug string) *mapRNG {
	h := fnv.New32a()
	_, _ = h.Write([]byte(slug))
	_, _ = h.Write([]byte("/map-pop"))
	s := h.Sum32()
	if s == 0 {
		s = 1
	}
	return &mapRNG{state: s}
}

func newMapRNGKeyed(slug string, key uint32) *mapRNG {
	h := fnv.New32a()
	_, _ = h.Write([]byte(slug))
	_, _ = h.Write([]byte{byte(key), byte(key >> 8), byte(key >> 16), byte(key >> 24)})
	s := h.Sum32()
	if s == 0 {
		s = 1
	}
	return &mapRNG{state: s}
}

func (r *mapRNG) next() uint32 {
	// xorshift32
	x := r.state
	x ^= x << 13
	x ^= x >> 17
	x ^= x << 5
	r.state = x
	return x
}

func (r *mapRNG) Float() float64 {
	return float64(r.next()>>8) / float64(1<<24)
}

func (r *mapRNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(r.next() % uint32(n)) //#nosec G115
}
