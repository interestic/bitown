package citycore

import (
	"errors"
	"fmt"
)

// SectorValue is a non-negative integer sector score (pop, ind, tra, …).
type SectorValue int

// ErrInvalidSectorValue is returned when ParseSectorValue rejects input.
var ErrInvalidSectorValue = errors.New("invalid sector value")

// ParseSectorValue validates a non-negative sector value.
// Prefer this constructor for domain writes; JSON/DB Scan may populate City
// fields directly as numeric wire values without re-validation.
func ParseSectorValue(n int) (SectorValue, error) {
	if n < 0 {
		return 0, fmt.Errorf("%w: must be >= 0", ErrInvalidSectorValue)
	}
	return SectorValue(n), nil
}

// Int returns the underlying int for boundaries and APIs that need a plain int.
func (v SectorValue) Int() int {
	return int(v)
}
