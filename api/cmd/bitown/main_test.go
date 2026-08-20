package main

import (
	"testing"
)

func TestIsDebugModeEnabled(t *testing.T) {
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
			if got := isDebugModeEnabled(); got != tt.want {
				t.Fatalf("isDebugModeEnabled() = %v, want %v (DEBUG_MODE=%q)", got, tt.want, tt.val)
			}
		})
	}
}
