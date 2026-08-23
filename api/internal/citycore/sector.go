package citycore

import (
	"errors"
	"fmt"
)

// Sector is a city support / growth dimension (pop, ind, …).
type Sector string

// Sector constants.
const (
	SectorPop Sector = "pop"
	SectorInd Sector = "ind"
	SectorTra Sector = "tra"
	SectorSec Sector = "sec"
	SectorEnv Sector = "env"
	SectorCom Sector = "com"
)

// AllSectors lists sectors in display / unlock order.
var AllSectors = []Sector{
	SectorPop,
	SectorInd,
	SectorTra,
	SectorSec,
	SectorEnv,
	SectorCom,
}

// ValidSectors is the set of known sector values (for O(1) lookup).
// Derived from AllSectors so the two cannot drift.
var ValidSectors = func() map[Sector]bool {
	m := make(map[Sector]bool, len(AllSectors))
	for _, s := range AllSectors {
		m[s] = true
	}
	return m
}()

// ErrInvalidSector is returned when ParseSector rejects input.
var ErrInvalidSector = errors.New("invalid sector")

// ParseSector validates a sector string. Empty input is invalid (handlers may
// default empty to SectorPop before calling ParseSector).
func ParseSector(raw string) (Sector, error) {
	s := Sector(raw)
	if !ValidSectors[s] {
		return "", fmt.Errorf("%w: %q", ErrInvalidSector, raw)
	}
	return s, nil
}

// String returns the underlying sector string.
func (s Sector) String() string {
	return string(s)
}

// unlockThresholds: sector is available when pop >= threshold.
var unlockThresholds = map[Sector]SectorValue{
	SectorPop: 0,
	SectorInd: 50,
	SectorTra: 100,
	SectorSec: 300,
	SectorEnv: 500,
	SectorCom: 1000,
}

func IsUnlocked(c *City, sector Sector) bool {
	threshold, ok := unlockThresholds[sector]
	if !ok {
		return false
	}
	return c.Pop >= threshold
}

// UnlockedSectors returns the subset of AllSectors that are unlocked for city c.
func UnlockedSectors(c *City) []Sector {
	out := make([]Sector, 0, len(AllSectors))
	for _, s := range AllSectors {
		if IsUnlocked(c, s) {
			out = append(out, s)
		}
	}
	return out
}

// SectorColumn returns the SQL column name for a sector.
// Using a switch instead of string interpolation prevents SQL injection.
// Only valid sectors have a column; callers must ParseSector first.
func SectorColumn(sector Sector) string {
	switch sector {
	case SectorPop:
		return "pop"
	case SectorInd:
		return "ind"
	case SectorTra:
		return "tra"
	case SectorSec:
		return "sec"
	case SectorEnv:
		return "env"
	case SectorCom:
		return "com"
	default:
		return ""
	}
}

// ApplySectorDelta bumps a sector value on c in memory (best-effort response shaping).
func ApplySectorDelta(c *City, sector Sector, delta SectorValue) {
	if c == nil {
		return
	}
	switch sector {
	case SectorPop:
		c.Pop += delta
	case SectorInd:
		c.Ind += delta
	case SectorTra:
		c.Tra += delta
	case SectorSec:
		c.Sec += delta
	case SectorEnv:
		c.Env += delta
	case SectorCom:
		c.Com += delta
	}
}
