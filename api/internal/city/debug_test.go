package city

import (
	"net/url"
	"testing"

	"github.com/interestic/bitown/internal/citycore"
)

func TestDebugModeEnabled(t *testing.T) {
	tests := []struct {
		name string
		val  string
		want bool
	}{
		{name: "true", val: "true", want: true},
		{name: "1", val: "1", want: true},
		{name: "yes", val: "yes", want: true},
		{name: "case and spaces", val: "  TRUE  ", want: true},
		{name: "false", val: "false", want: false},
		{name: "empty", val: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DEBUG_MODE", tt.val)
			if got := DebugModeEnabled(); got != tt.want {
				t.Fatalf("DebugModeEnabled() = %v, want %v (DEBUG_MODE=%q)", got, tt.want, tt.val)
			}
		})
	}
}

func TestApplyMapDebugOverrides(t *testing.T) {
	base := &citycore.City{
		Slug: "demo",
		Pop:  10,
		Ind:  1,
		Env:  2,
		Com:  3,
	}

	t.Run("no params", func(t *testing.T) {
		got, applied, err := applyMapDebugOverrides(base, url.Values{})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if applied {
			t.Fatal("expected applied=false")
		}
		if got.Pop != 10 || got.Ind != 1 {
			t.Fatalf("unexpected city: pop=%d ind=%d", got.Pop, got.Ind)
		}
	})

	t.Run("overrides pop and env", func(t *testing.T) {
		got, applied, err := applyMapDebugOverrides(base, url.Values{
			"pop": {"500"},
			"env": {"400"},
		})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !applied {
			t.Fatal("expected applied=true")
		}
		if got.Pop != 500 || got.Env != 400 {
			t.Fatalf("got pop=%d env=%d", got.Pop, got.Env)
		}
		if base.Pop != 10 || base.Env != 2 {
			t.Fatal("base city must not be mutated")
		}
		if got.Ind != 1 || got.Com != 3 {
			t.Fatalf("unspecified fields should stay: ind=%d com=%d", got.Ind, got.Com)
		}
	})

	t.Run("invalid negative", func(t *testing.T) {
		_, _, err := applyMapDebugOverrides(base, url.Values{"pop": {"-1"}})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid non-integer", func(t *testing.T) {
		_, _, err := applyMapDebugOverrides(base, url.Values{"ind": {"abc"}})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
