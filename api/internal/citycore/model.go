// Package citycore contains pure domain types and logic with no external dependencies.
package citycore

import "time"

type City struct {
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	CountryCode string    `json:"country_code"`
	OwnerID     *string   `json:"owner_id,omitempty"`
	Pop         int       `json:"pop"`
	Ind         int       `json:"ind"`
	Tra         int       `json:"tra"`
	Sec         int       `json:"sec"`
	Env         int       `json:"env"`
	Com         int       `json:"com"`
	CreatedAt   time.Time `json:"created_at"`
}

// Sector constants.
const (
	SectorPop = "pop"
	SectorInd = "ind"
	SectorTra = "tra"
	SectorSec = "sec"
	SectorEnv = "env"
	SectorCom = "com"
)

var ValidSectors = map[string]bool{
	SectorPop: true,
	SectorInd: true,
	SectorTra: true,
	SectorSec: true,
	SectorEnv: true,
	SectorCom: true,
}

// unlockThresholds: sector is available when pop >= threshold.
var unlockThresholds = map[string]int{
	SectorPop: 0,
	SectorInd: 50,
	SectorTra: 100,
	SectorSec: 300,
	SectorEnv: 500,
	SectorCom: 1000,
}

func IsUnlocked(c *City, sector string) bool {
	threshold, ok := unlockThresholds[sector]
	if !ok {
		return false
	}
	return c.Pop >= threshold
}

// SectorColumn returns the SQL column name for a sector.
// Using a switch instead of string interpolation prevents SQL injection.
func SectorColumn(sector string) string {
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
		return "pop"
	}
}
