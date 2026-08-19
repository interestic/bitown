package middleware

import (
	"net/http"
	"os"
	"strings"
)

// CORS handles cross-origin requests.
//
// In production the allowed origin is read from CORS_ALLOW_ORIGIN (e.g.
// "https://bitown.dev"). Wildcard "*" is intentionally avoided when the
// Authorization header is used, because browsers reject credentialed requests
// with a wildcard origin.
//
// For the badge/og endpoints (public <img> tags) the responses don't carry
// credentials, so "*" is fine there — but we apply the same policy uniformly
// and let CloudFront handle CDN-level caching.
func CORS(next http.Handler) http.Handler {
	allowedOrigin := os.Getenv("CORS_ALLOW_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "*" // dev fallback only
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if allowedOrigin == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" && strings.EqualFold(origin, allowedOrigin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
