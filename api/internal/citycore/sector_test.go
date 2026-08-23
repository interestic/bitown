package citycore

import (
	"errors"
	"testing"
)

func TestParseSector(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		for _, want := range AllSectors {
			got, err := ParseSector(want.String())
			if err != nil {
				t.Fatalf("ParseSector(%q) unexpected error: %v", want, err)
			}
			if got != want {
				t.Fatalf("ParseSector(%q) = %q, want %q", want, got, want)
			}
		}
	})

	t.Run("invalid", func(t *testing.T) {
		cases := []string{"", "hack", "POP", "unknown", " pop "}
		for _, raw := range cases {
			got, err := ParseSector(raw)
			if err == nil {
				t.Fatalf("ParseSector(%q) = %q, want error", raw, got)
			}
			if !errors.Is(err, ErrInvalidSector) {
				t.Fatalf("ParseSector(%q) error = %v, want ErrInvalidSector", raw, err)
			}
			if got != "" {
				t.Fatalf("ParseSector(%q) value = %q, want empty", raw, got)
			}
		}
	})
}

func TestIsUnlocked(t *testing.T) {
	cases := []struct {
		name   string
		pop    int
		sector Sector
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
		{"unknown sector", 9999, Sector("unknown"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &City{Pop: SectorValue(tc.pop)}
			if got := IsUnlocked(c, tc.sector); got != tc.want {
				t.Errorf("IsUnlocked(pop=%d, sector=%q) = %v, want %v", tc.pop, tc.sector, got, tc.want)
			}
		})
	}
}

func TestSectorColumn(t *testing.T) {
	cases := map[Sector]string{
		SectorPop: "pop",
		SectorInd: "ind",
		SectorTra: "tra",
		SectorSec: "sec",
		SectorEnv: "env",
		SectorCom: "com",
	}
	for sector, want := range cases {
		if got := SectorColumn(sector); got != want {
			t.Errorf("SectorColumn(%q) = %q, want %q", sector, got, want)
		}
	}
	if got := SectorColumn(Sector("unknown")); got != "" {
		t.Errorf("SectorColumn(unknown) = %q, want empty", got)
	}
}

func TestUnlockedSectors(t *testing.T) {
	cases := []struct {
		name string
		pop  int
		want []Sector
	}{
		{"pop=0: only pop", 0, []Sector{SectorPop}},
		{"pop=50: pop+ind", 50, []Sector{SectorPop, SectorInd}},
		{"pop=99: pop+ind", 99, []Sector{SectorPop, SectorInd}},
		{"pop=100: pop+ind+tra", 100, []Sector{SectorPop, SectorInd, SectorTra}},
		{"pop=300: pop+ind+tra+sec", 300, []Sector{SectorPop, SectorInd, SectorTra, SectorSec}},
		{"pop=500: pop+ind+tra+sec+env", 500, []Sector{SectorPop, SectorInd, SectorTra, SectorSec, SectorEnv}},
		{"pop=1000: all", 1000, AllSectors},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &City{Pop: SectorValue(tc.pop)}
			got := UnlockedSectors(c)
			if len(got) != len(tc.want) {
				t.Fatalf("UnlockedSectors(pop=%d) = %v, want %v", tc.pop, got, tc.want)
			}
			for i, s := range got {
				if s != tc.want[i] {
					t.Errorf("UnlockedSectors(pop=%d)[%d] = %q, want %q", tc.pop, i, s, tc.want[i])
				}
			}
		})
	}
}

func TestSectorDefsConsistent(t *testing.T) {
	if len(ValidSectors) != len(AllSectors) {
		t.Fatalf("ValidSectors len = %d, AllSectors len = %d", len(ValidSectors), len(AllSectors))
	}
	for _, s := range AllSectors {
		if !ValidSectors[s] {
			t.Errorf("AllSectors entry %q missing from ValidSectors", s)
		}
		if _, ok := unlockThresholds[s]; !ok {
			t.Errorf("AllSectors entry %q missing unlockThresholds", s)
		}
		if SectorColumn(s) == "" {
			t.Errorf("AllSectors entry %q has empty SectorColumn", s)
		}
	}
}
