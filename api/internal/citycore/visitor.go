package citycore

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// VisitorHash is an opaque daily visitor identifier for rate-limiting.
// Raw IP is never stored; only this hash is persisted / keyed in Redis.
type VisitorHash string

// VisitorHashFromValues is the deterministic hash builder for rate-limiting.
// It combines IP, User-Agent, daily UTC date, and a server-side salt seed.
func VisitorHashFromValues(ip, userAgent, salt string, now time.Time) VisitorHash {
	date := now.Format("2006-01-02")
	raw := fmt.Sprintf("%s:%s:%s:%s", ip, userAgent, date, salt)
	sum := sha256.Sum256([]byte(raw))
	return VisitorHash(fmt.Sprintf("%x", sum))
}

// VisitKey returns the Redis key for deduplication.
func VisitKey(date string, citySlug Slug, hash VisitorHash) string {
	return fmt.Sprintf("visit:%s:%s:%s", date, citySlug, hash)
}

// String returns the underlying hash string (for Redis / DB).
func (h VisitorHash) String() string {
	return string(h)
}

// VisitTTLUntilUTCMidnight returns the duration from now until the next UTC midnight.
// now is normalized to UTC so callers may pass any location.
func VisitTTLUntilUTCMidnight(now time.Time) time.Duration {
	now = now.UTC()
	endOfDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	return endOfDay.Sub(now)
}
