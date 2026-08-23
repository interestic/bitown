package citycore

import (
	"errors"
	"testing"
)

func TestParseCountryCode(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    CountryCode
		wantErr bool
	}{
		{name: "JP", raw: "JP", want: "JP"},
		{name: "lowercase", raw: "jp", want: "JP"},
		{name: "trim", raw: " jp ", want: "JP"},
		{name: "mixed case", raw: "jP", want: "JP"},
		{name: "too short", raw: "J", wantErr: true},
		{name: "too long", raw: "JPN", wantErr: true},
		{name: "empty", raw: "", wantErr: true},
		{name: "digits", raw: "12", wantErr: true},
		{name: "letter digit", raw: "J1", wantErr: true},
		{name: "space only", raw: "  ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCountryCode(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseCountryCode(%q) = %q, want error", tt.raw, got)
				}
				if !errors.Is(err, ErrInvalidCountryCode) {
					t.Fatalf("err = %v, want ErrInvalidCountryCode", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseCountryCode(%q) unexpected error: %v", tt.raw, err)
			}
			if got != tt.want {
				t.Fatalf("ParseCountryCode(%q) = %q, want %q", tt.raw, got, tt.want)
			}
			if got.String() != string(tt.want) {
				t.Fatalf("String() = %q, want %q", got.String(), tt.want)
			}
		})
	}
}
