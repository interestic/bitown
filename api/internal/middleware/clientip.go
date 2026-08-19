package middleware

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
)

type clientIPKey struct{}

// ClientIP resolves request client IP safely.
//
// By default it uses r.RemoteAddr. Proxy headers are used only when
// REMOTE_ADDR is within TRUSTED_PROXY_CIDRS (comma-separated CIDRs), e.g.:
//   TRUSTED_PROXY_CIDRS=173.245.48.0/20,103.21.244.0/22
//
// In Cloudflare mode, only CF-Connecting-IP is trusted as client IP.
// X-Forwarded-For / X-Real-IP are intentionally ignored to avoid spoofing.
func ClientIP(next http.Handler) http.Handler {
	trusted := parseTrustedCIDRs(os.Getenv("TRUSTED_PROXY_CIDRS"))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := resolveClientIP(r, trusted)
		ctx := context.WithValue(r.Context(), clientIPKey{}, ip)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetClientIP returns the middleware-resolved IP, or falls back to r.RemoteAddr.
func GetClientIP(r *http.Request) string {
	if v, ok := r.Context().Value(clientIPKey{}).(string); ok && v != "" {
		return v
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func resolveClientIP(r *http.Request, trusted []*net.IPNet) string {
	remoteIP := GetClientIP(r)
	parsedRemote := net.ParseIP(remoteIP)
	if parsedRemote == nil || !isTrustedProxy(parsedRemote, trusted) {
		return remoteIP
	}

	cfIP := strings.TrimSpace(r.Header.Get("CF-Connecting-IP"))
	if net.ParseIP(cfIP) != nil {
		return cfIP
	}

	return remoteIP
}

func parseTrustedCIDRs(raw string) []*net.IPNet {
	var out []*net.IPNet
	for _, part := range strings.Split(raw, ",") {
		cidr := strings.TrimSpace(part)
		if cidr == "" {
			continue
		}
		_, n, err := net.ParseCIDR(cidr)
		if err == nil {
			out = append(out, n)
		}
	}
	return out
}

func isTrustedProxy(ip net.IP, trusted []*net.IPNet) bool {
	for _, n := range trusted {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
