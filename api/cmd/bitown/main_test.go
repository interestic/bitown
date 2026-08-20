package main

import (
	"testing"
)

func TestIsDebugModeEnabled(t *testing.T) {
	tests := []struct {
		name string
		env  string
		val  string
		want bool
	}{
		{name: "true", val: "true", want: true},
		{name: "1", val: "1", want: true},
		{name: "yes", val: "yes", want: true},
		{name: "case and spaces", val: "  TRUE  ", want: true},
		{name: "false", val: "false", want: false},
		{name: "empty", val: "", want: false},
		{name: "production blocks debug", env: "production", val: "true", want: false},
		{name: "prod alias blocks debug", env: "prod", val: "1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ENV", tt.env)
			t.Setenv("DEBUG_MODE", tt.val)
			if got := isDebugModeEnabled(); got != tt.want {
				t.Fatalf("isDebugModeEnabled() = %v, want %v (ENV=%q DEBUG_MODE=%q)", got, tt.want, tt.env, tt.val)
			}
		})
	}
}
