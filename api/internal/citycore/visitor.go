package citycore

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/http"
	"time"
)

// VisitorHash returns a privacy-safe daily hash for rate-limiting.
// It combines IP, User-Agent, daily UTC date, and a server-side salt seed.
// Raw IP is never stored.
//
// IP is taken solely from r.RemoteAddr, which the chimw.RealIP middleware
// already resolves from trusted proxy headers (X-Forwarded-For / X-Real-IP).
func VisitorHash(r *http.Request, salt string) string {
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

// VisitKey returns the Redis key for deduplication.
func VisitKey(date, citySlug, hash string) string {
	return fmt.Sprintf("visit:%s:%s:%s", date, citySlug, hash)
}
