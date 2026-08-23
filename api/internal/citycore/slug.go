package citycore

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Slug is a city identifier used in URLs, Redis keys, and the cities PK.
type Slug string

// slugRe requires 2-40 chars, lowercase alphanumeric and hyphens,
// with no leading or trailing hyphens.
// The mandatory [a-z0-9] at both ends ensures minimum 2 chars.
var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]$`)

// ErrInvalidSlug is returned when ParseSlug rejects input.
var ErrInvalidSlug = errors.New("invalid slug")

// ParseSlug normalizes (trim + lower) and validates a city slug.
func ParseSlug(raw string) (Slug, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if !slugRe.MatchString(s) {
		return "", fmt.Errorf("%w: must be 2-40 lowercase alphanumeric/hyphen", ErrInvalidSlug)
	}
	return Slug(s), nil
}

// String returns the underlying slug string.
func (s Slug) String() string {
	return string(s)
}
