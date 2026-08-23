package citycore

import (
	"errors"
	"testing"
)

func TestParseSectorValue(t *testing.T) {
	tests := []struct {
		name    string
		n       int
		want    SectorValue
		wantErr bool
	}{
		{name: "zero", n: 0, want: 0},
		{name: "positive", n: 42, want: 42},
		{name: "large", n: 1_000_000, want: 1_000_000},
		{name: "negative", n: -1, wantErr: true},
		{name: "large negative", n: -100, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSectorValue(tt.n)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseSectorValue(%d) = %d, want error", tt.n, got)
				}
				if !errors.Is(err, ErrInvalidSectorValue) {
					t.Fatalf("err = %v, want ErrInvalidSectorValue", err)
				}
				if got != 0 {
					t.Fatalf("ParseSectorValue(%d) value = %d, want 0", tt.n, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSectorValue(%d) unexpected error: %v", tt.n, err)
			}
			if got != tt.want {
				t.Fatalf("ParseSectorValue(%d) = %d, want %d", tt.n, got, tt.want)
			}
			if got.Int() != tt.n {
				t.Fatalf("Int() = %d, want %d", got.Int(), tt.n)
			}
		})
	}
}
