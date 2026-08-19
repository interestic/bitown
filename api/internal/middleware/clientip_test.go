package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP_UsesRemoteAddrByDefault(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_CIDRS", "")

	var got string
	h := ClientIP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = GetClientIP(r)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.9:4321"
	req.Header.Set("CF-Connecting-IP", "198.51.100.10")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got != "203.0.113.9" {
		t.Fatalf("client ip = %q, want %q", got, "203.0.113.9")
	}
}

func TestClientIP_UsesTrustedProxyHeader(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")

	var got string
	h := ClientIP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = GetClientIP(r)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.1.2.3:1234"
	req.Header.Set("CF-Connecting-IP", "198.51.100.10")
	req.Header.Set("X-Forwarded-For", "198.51.100.11, 10.1.2.3")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got != "198.51.100.10" {
		t.Fatalf("client ip = %q, want %q", got, "198.51.100.10")
	}
}

func TestClientIP_UsesFirstXForwardedForWhenTrusted(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8")

	var got string
	h := ClientIP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = GetClientIP(r)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.1.2.3:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.11, 10.1.2.3")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got != "198.51.100.11" {
		t.Fatalf("client ip = %q, want %q", got, "198.51.100.11")
	}
}
