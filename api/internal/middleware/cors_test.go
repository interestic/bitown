package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestCORS_Wildcard(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGIN", "")

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	CORS(okHandler()).ServeHTTP(w, r)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected *, got %q", got)
	}
}

func TestCORS_SpecificOrigin_Match(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGIN", "https://bitown.dev")

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "https://bitown.dev")
	w := httptest.NewRecorder()

	CORS(okHandler()).ServeHTTP(w, r)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://bitown.dev" {
		t.Errorf("expected https://bitown.dev, got %q", got)
	}
}

func TestCORS_SpecificOrigin_NoMatch(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGIN", "https://bitown.dev")

	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()

	CORS(okHandler()).ServeHTTP(w, r)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no ACAO header, got %q", got)
	}
}

func TestCORS_Preflight(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGIN", "")

	r := httptest.NewRequest("OPTIONS", "/", nil)
	w := httptest.NewRecorder()

	CORS(okHandler()).ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}
