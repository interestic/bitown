package render

import "testing"

func TestLotInSquareMini_SW(t *testing.T) {
	// Production squares are aligned to squareSide.
	bx, by := 20, 20
	ox, oy := miniSquareOrigin(bx, by, miniSW)
	x, y := miniHutFoot(ox, oy, 0, true)
	if !lotInSquareMini(x, y, miniSW) {
		t.Fatalf("SW hut foot (%d,%d) not in miniSW (ox,oy)=(%d,%d)", x, y, ox, oy)
	}
	if lotInSquareMini(x, y, miniNE) {
		t.Fatalf("SW hut foot (%d,%d) must not be in miniNE", x, y)
	}
}

func TestApplyWestMiniStampNudge(t *testing.T) {
	if westMiniStampNudgeX != 10 {
		t.Fatalf("westMiniStampNudgeX=%d, want 10", westMiniStampNudgeX)
	}
	bx, by := 20, 20
	ox, oy := miniSquareOrigin(bx, by, miniSW)
	sx, sy := miniHutFoot(ox, oy, 0, true)
	got := applyWestMiniStampNudge(100, sx, sy)
	if got != 110 {
		t.Fatalf("SW nudge: got %d, want 110 (lot %d,%d)", got, sx, sy)
	}
	ox, oy = miniSquareOrigin(bx, by, miniNE)
	nx, ny := miniHutFoot(ox, oy, 0, true)
	got = applyWestMiniStampNudge(100, nx, ny)
	if got != 100 {
		t.Fatalf("NE must not west-nudge: got %d, want 100", got)
	}
}

func TestApplySEEWEastMiniStampNudges(t *testing.T) {
	if seMiniStampNudgeY != 3 || ewMiniStampNudgeY != 5 || eastMiniStampNudgeX != -1 {
		t.Fatalf("nudge consts SE=%d EW=%d east=%d", seMiniStampNudgeY, ewMiniStampNudgeY, eastMiniStampNudgeX)
	}
	bx, by := 20, 20
	ox, oy := miniSquareOrigin(bx, by, miniSE)
	sx, sy := miniHutFoot(ox, oy, 0, true)
	if got := applySEMiniStampNudge(100, sx, sy); got != 103 {
		t.Fatalf("SE +Y: got %d, want 103", got)
	}
	ox, oy = miniSquareOrigin(bx, by, miniNE)
	nx, ny := miniHutFoot(ox, oy, 0, true)
	if got := applyEWMiniStampNudge(100, nx, ny); got != 105 {
		t.Fatalf("NE EW +Y: got %d, want 105", got)
	}
	if got := applyEastMiniStampNudge(100, nx, ny); got != 99 {
		t.Fatalf("NE east −X: got %d, want 99", got)
	}
	ox, oy = miniSquareOrigin(bx, by, miniNW)
	wx, wy := miniHutFoot(ox, oy, 0, true)
	if got := applySEMiniStampNudge(100, wx, wy); got != 100 {
		t.Fatalf("NW must not SE-nudge: got %d", got)
	}
	if got := applyEastMiniStampNudge(100, wx, wy); got != 100 {
		t.Fatalf("NW must not east-nudge: got %d", got)
	}
}
