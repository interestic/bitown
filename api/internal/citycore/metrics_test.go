package citycore

import "testing"

func TestComputeMetrics_TownzzyBat(t *testing.T) {
	// Townzzy Bat: income $792,300 (元仕様調査 §4).
	// High ind/com/sec keep unemployment & crime at 0; roads saturated.
	c := &City{
		Pop: 1325, Ind: 971, Tra: 1219, Sec: 877, Env: 765, Com: 877,
	}
	m := ComputeMetrics(c)
	if m.Income != 792300 {
		t.Fatalf("income = %d, want 792300", m.Income)
	}
	if m.Unemployment != 0 {
		t.Fatalf("unemployment = %d, want 0", m.Unemployment)
	}
	if m.Roads != 100 {
		t.Fatalf("roads = %d, want 100", m.Roads)
	}
	if m.Pollution != 0 {
		t.Fatalf("pollution = %d, want 0", m.Pollution)
	}
	if m.Crime != 0 {
		t.Fatalf("crime = %d, want 0", m.Crime)
	}
}

func TestComputeMetrics_StarterCity(t *testing.T) {
	c := &City{Pop: 1}
	m := ComputeMetrics(c)
	if m.Income != 150 { // (1*1.5)*100
		t.Fatalf("income = %d, want 150", m.Income)
	}
	// Base buffers keep early cities healthy.
	if m.Unemployment != 0 {
		t.Fatalf("unemployment = %d, want 0", m.Unemployment)
	}
	if m.Roads != 100 {
		t.Fatalf("roads = %d, want 100", m.Roads)
	}
	if m.Pollution != 0 {
		t.Fatalf("pollution = %d, want 0 (no industry)", m.Pollution)
	}
	if m.Crime != 0 {
		t.Fatalf("crime = %d, want 0", m.Crime)
	}
}

func TestComputeMetrics_Nil(t *testing.T) {
	m := ComputeMetrics(nil)
	if m != (Metrics{}) {
		t.Fatalf("nil city metrics = %+v", m)
	}
}

func TestComputeMetrics_HighUnemployment(t *testing.T) {
	// pop grows without jobs → unemployment approaches 100%.
	c := &City{Pop: 1000, Ind: 0, Com: 0}
	m := ComputeMetrics(c)
	// 1 - min((0+0+50)/1000,1) = 1 - 0.05 = 0.95 → 95%
	if m.Unemployment != 95 {
		t.Fatalf("unemployment = %d, want 95", m.Unemployment)
	}
}
