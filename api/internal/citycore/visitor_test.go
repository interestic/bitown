package citycore

import (
	"testing"
	"time"
)

func TestVisitKey(t *testing.T) {
	key := VisitKey("2026-08-19", Slug("tokyo"), VisitorHash("abc123"))
	want := "visit:2026-08-19:tokyo:abc123"
	if key != want {
		t.Errorf("VisitKey = %q, want %q", key, want)
	}
}

func TestVisitorHash_Deterministic(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	h1 := VisitorHashFromValues("192.0.2.1", "TestAgent/1.0", "salt1", now)
	h2 := VisitorHashFromValues("192.0.2.1", "TestAgent/1.0", "salt1", now)
	if h1 != h2 {
		t.Errorf("VisitorHash should be deterministic: %q != %q", h1, h2)
	}
	if h1.String() == "" {
		t.Error("VisitorHash.String() should be non-empty")
	}
}

func TestVisitorHash_DifferentSalt(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	h1 := VisitorHashFromValues("192.0.2.1", "TestAgent/1.0", "salt1", now)
	h2 := VisitorHashFromValues("192.0.2.1", "TestAgent/1.0", "salt2", now)
	if h1 == h2 {
		t.Error("VisitorHash with different salts should differ")
	}
}

func TestVisitorHashFromValues_String(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	h := VisitorHashFromValues("192.0.2.1", "ua", "salt", now)
	if len(h.String()) != 64 {
		t.Fatalf("hash length = %d, want 64 hex chars", len(h.String()))
	}
}

func TestVisitTTLUntilUTCMidnight(t *testing.T) {
	cases := []struct {
		name    string
		now     time.Time
		wantTTL time.Duration
	}{
		{
			name:    "noon UTC",
			now:     time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC),
			wantTTL: 12 * time.Hour,
		},
		{
			name:    "one second before midnight",
			now:     time.Date(2026, 8, 19, 23, 59, 59, 0, time.UTC),
			wantTTL: time.Second,
		},
		{
			name:    "just after midnight",
			now:     time.Date(2026, 8, 20, 0, 0, 1, 0, time.UTC),
			wantTTL: 24*time.Hour - time.Second,
		},
		{
			name:    "month boundary",
			now:     time.Date(2026, 8, 31, 18, 0, 0, 0, time.UTC),
			wantTTL: 6 * time.Hour,
		},
		{
			name: "non-UTC location still uses UTC midnight",
			// 2026-08-20 01:00 JST == 2026-08-19 16:00 UTC → 8h until Aug 20 00:00 UTC
			now:     time.Date(2026, 8, 20, 1, 0, 0, 0, time.FixedZone("JST", 9*3600)),
			wantTTL: 8 * time.Hour,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := VisitTTLUntilUTCMidnight(tc.now)
			if got != tc.wantTTL {
				t.Errorf("VisitTTLUntilUTCMidnight(%v) = %v, want %v", tc.now, got, tc.wantTTL)
			}
		})
	}
}
