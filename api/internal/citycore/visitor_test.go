package citycore

import (
	"net/http/httptest"
	"testing"
)

func TestVisitKey(t *testing.T) {
	key := VisitKey("2026-08-19", "tokyo", "abc123")
	want := "visit:2026-08-19:tokyo:abc123"
	if key != want {
		t.Errorf("VisitKey = %q, want %q", key, want)
	}
}

func TestVisitorHash_Deterministic(t *testing.T) {
	r := httptest.NewRequest("POST", "/", nil)
	r.RemoteAddr = "192.0.2.1:12345"
	r.Header.Set("User-Agent", "TestAgent/1.0")

	h1 := VisitorHash(r, "salt1")
	h2 := VisitorHash(r, "salt1")
	if h1 != h2 {
		t.Errorf("VisitorHash should be deterministic: %q != %q", h1, h2)
	}
}

func TestVisitorHash_DifferentSalt(t *testing.T) {
	r := httptest.NewRequest("POST", "/", nil)
	r.RemoteAddr = "192.0.2.1:12345"
	r.Header.Set("User-Agent", "TestAgent/1.0")

	h1 := VisitorHash(r, "salt1")
	h2 := VisitorHash(r, "salt2")
	if h1 == h2 {
		t.Error("VisitorHash with different salts should differ")
	}
}

func TestVisitorHash_NoPort(t *testing.T) {
	r := httptest.NewRequest("POST", "/", nil)
	r.RemoteAddr = "192.0.2.1"
	r.Header.Set("User-Agent", "TestAgent/1.0")

	_ = VisitorHash(r, "salt")
}
