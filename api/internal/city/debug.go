package city

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/interestic/bitown/internal/citycore"
)

// DebugModeEnabled reports whether DEBUG_MODE is on (true/1/yes).
func DebugModeEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("DEBUG_MODE")))
	return v == "true" || v == "1" || v == "yes"
}

var mapDebugSectorParams = []struct {
	name string
	set  func(*citycore.City, citycore.SectorValue)
}{
	{"pop", func(c *citycore.City, v citycore.SectorValue) { c.Pop = v }},
	{"ind", func(c *citycore.City, v citycore.SectorValue) { c.Ind = v }},
	{"tra", func(c *citycore.City, v citycore.SectorValue) { c.Tra = v }},
	{"sec", func(c *citycore.City, v citycore.SectorValue) { c.Sec = v }},
	{"env", func(c *citycore.City, v citycore.SectorValue) { c.Env = v }},
	{"com", func(c *citycore.City, v citycore.SectorValue) { c.Com = v }},
}

// applyMapDebugOverrides copies city and applies sector query overrides (pop, ind, …).
// Returns the (possibly overridden) city, whether any override was applied, and a parse error.
func applyMapDebugOverrides(city *citycore.City, q url.Values) (*citycore.City, bool, error) {
	out := *city
	applied := false
	for _, p := range mapDebugSectorParams {
		raw, ok := q[p.name]
		if !ok || len(raw) == 0 {
			continue
		}
		n, err := strconv.Atoi(raw[0])
		if err != nil {
			return nil, false, fmt.Errorf("invalid %s: must be an integer", p.name)
		}
		v, err := citycore.ParseSectorValue(n)
		if err != nil {
			return nil, false, fmt.Errorf("invalid %s: must be >= 0", p.name)
		}
		p.set(&out, v)
		applied = true
	}
	return &out, applied, nil
}
