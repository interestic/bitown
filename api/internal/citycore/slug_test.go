package citycore

import (
	"errors"
	"strings"
	"testing"
)

func TestParseSlug(t *testing.T) {
	valid := []string{
		"ab",
		"tokyo",
		"new-york",
		"to-kyo",
		strings.Repeat("a", 40),
	}
	for _, in := range valid {
		got, err := ParseSlug(in)
		if err != nil {
			t.Errorf("ParseSlug(%q) unexpected error: %v", in, err)
			continue
		}
		if got.String() != strings.ToLower(strings.TrimSpace(in)) {
			t.Errorf("ParseSlug(%q) = %q, want normalized form", in, got)
		}
	}

	invalid := []string{
		"a",
		"-tokyo",
		"tokyo-",
		"to kyo",
		"",
		" ",
		strings.Repeat("a", 41),
		"tokyo_city",
		".tokyo",
		"TOKYO!",
		"a\nb",
	}
	for _, in := range invalid {
		if _, err := ParseSlug(in); !errors.Is(err, ErrInvalidSlug) {
			t.Errorf("ParseSlug(%q) error = %v, want ErrInvalidSlug", in, err)
		}
	}

	got, err := ParseSlug("  Tokyo  ")
	if err != nil {
		t.Fatalf("ParseSlug trim/lower: %v", err)
	}
	if got != Slug("tokyo") {
		t.Errorf("ParseSlug trim/lower = %q, want tokyo", got)
	}
}
