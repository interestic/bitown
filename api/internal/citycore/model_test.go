package citycore

import "testing"

func TestIsUnlocked(t *testing.T) {
	cases := []struct {
		name   string
		pop    int
		sector string
		want   bool
	}{
		{"pop always unlocked", 0, SectorPop, true},
		{"ind locked below 50", 49, SectorInd, false},
		{"ind unlocked at 50", 50, SectorInd, true},
		{"tra locked below 100", 99, SectorTra, false},
		{"tra unlocked at 100", 100, SectorTra, true},
		{"sec locked below 300", 299, SectorSec, false},
		{"sec unlocked at 300", 300, SectorSec, true},
		{"env locked below 500", 499, SectorEnv, false},
		{"env unlocked at 500", 500, SectorEnv, true},
		{"com locked below 1000", 999, SectorCom, false},
		{"com unlocked at 1000", 1000, SectorCom, true},
		{"unknown sector", 9999, "unknown", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &City{Pop: tc.pop}
			if got := IsUnlocked(c, tc.sector); got != tc.want {
				t.Errorf("IsUnlocked(pop=%d, sector=%q) = %v, want %v", tc.pop, tc.sector, got, tc.want)
			}
		})
	}
}

func TestSectorColumn(t *testing.T) {
	cases := map[string]string{
		SectorPop: "pop",
		SectorInd: "ind",
		SectorTra: "tra",
		SectorSec: "sec",
		SectorEnv: "env",
		SectorCom: "com",
		"unknown": "pop",
	}
	for sector, want := range cases {
		if got := SectorColumn(sector); got != want {
			t.Errorf("SectorColumn(%q) = %q, want %q", sector, got, want)
		}
	}
}
