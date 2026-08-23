// Package citycore contains pure domain types and logic with no external dependencies.
package citycore

import "time"

type City struct {
	Slug        Slug        `json:"slug"`
	Name        string      `json:"name"`
	CountryCode CountryCode `json:"country_code"`
	OwnerID     *string     `json:"owner_id,omitempty"`
	Pop         SectorValue `json:"pop"`
	Ind         SectorValue `json:"ind"`
	Tra         SectorValue `json:"tra"`
	Sec         SectorValue `json:"sec"`
	Env         SectorValue `json:"env"`
	Com         SectorValue `json:"com"`
	CreatedAt   time.Time   `json:"created_at"`
}
