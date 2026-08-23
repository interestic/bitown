package citycore

import (
	"errors"
	"fmt"
	"strings"
)

// CountryCode is an ISO 3166-1 alpha-2 country code.
type CountryCode string

// ErrInvalidCountryCode is returned when ParseCountryCode rejects input.
var ErrInvalidCountryCode = errors.New("invalid country code")

// ParseCountryCode normalizes (trim + upper) and validates a 2-letter A-Z code.
func ParseCountryCode(raw string) (CountryCode, error) {
	s := strings.ToUpper(strings.TrimSpace(raw))
	if len(s) != 2 || s[0] < 'A' || s[0] > 'Z' || s[1] < 'A' || s[1] > 'Z' {
		return "", fmt.Errorf("%w: must be 2 letters A-Z", ErrInvalidCountryCode)
	}
	return CountryCode(s), nil
}

// String returns the underlying country code string.
func (c CountryCode) String() string {
	return string(c)
}
