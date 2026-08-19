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
func VisitorHash(r *http.Request, salt string) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}

	return VisitorHashFromValues(ip, r.UserAgent(), salt, time.Now().UTC())
}

// VisitorHashFromValues is the deterministic hash builder used by VisitorHash.
func VisitorHashFromValues(ip, userAgent, salt string, now time.Time) string {
	date := now.Format("2006-01-02")
	raw := fmt.Sprintf("%s:%s:%s:%s", ip, userAgent, date, salt)
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum)
}

// VisitKey returns the Redis key for deduplication.
func VisitKey(date, citySlug, hash string) string {
	return fmt.Sprintf("visit:%s:%s:%s", date, citySlug, hash)
}
