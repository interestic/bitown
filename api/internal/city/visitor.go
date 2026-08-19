package city

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"time"
)

// visitorHash returns a privacy-safe daily hash for rate-limiting.
// It combines IP, User-Agent, daily UTC date, and a server-side salt seed.
// Raw IP is never stored.
//
// IP is taken solely from r.RemoteAddr, which the chimw.RealIP middleware
// already resolves from trusted proxy headers (X-Forwarded-For / X-Real-IP).
// Reading those headers again here would allow clients to spoof their IP and
// bypass the per-day deduplication limit.
func visitorHash(r *http.Request, salt string) string {
	// RemoteAddr is "host:port"; strip the port.
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}

	ua := r.UserAgent()
	date := time.Now().UTC().Format("2006-01-02")

	raw := fmt.Sprintf("%s:%s:%s:%s", ip, ua, date, salt)
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", h)
}

// visitKey returns the Redis key for deduplication.
func visitKey(date, citySlug, hash string) string {
	return fmt.Sprintf("visit:%s:%s:%s", date, citySlug, hash)
}
