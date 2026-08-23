package city

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/interestic/bitown/internal/citycore"
	"github.com/interestic/bitown/internal/middleware"
	"github.com/interestic/bitown/internal/render"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
)

type dbPool interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type Handler struct {
	db       dbPool
	rdb      *redis.Client
	saltSeed string
}

type debugSupportLog struct {
	Sector     string    `json:"sector"`
	CreatedAt  time.Time `json:"created_at"`
	VisitorTag string    `json:"visitor_tag"`
}

func NewHandler(db dbPool, rdb *redis.Client, saltSeed string) *Handler {
	return &Handler{db: db, rdb: rdb, saltSeed: saltSeed}
}

// POST /api/cities
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		CountryCode string `json:"country_code"`
		Slug        string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	slug, err := citycore.ParseSlug(req.Slug)
	if err != nil {
		http.Error(w, "slug must be 2-40 lowercase alphanumeric/hyphen", http.StatusBadRequest)
		return
	}
	if l := utf8.RuneCountInString(req.Name); l < 2 || l > 40 {
		http.Error(w, "name must be 2-40 characters", http.StatusBadRequest)
		return
	}
	cc, err := citycore.ParseCountryCode(req.CountryCode)
	if err != nil {
		http.Error(w, "country_code must be 2 letters A-Z", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	var city citycore.City
	err = h.db.QueryRow(ctx,
		`INSERT INTO cities (slug, name, country_code) VALUES ($1, $2, $3)
		 RETURNING slug, name, country_code, owner_id, pop, ind, tra, sec, env, com, created_at`,
		slug.String(), req.Name, cc.String(),
	).Scan(&city.Slug, &city.Name, &city.CountryCode, &city.OwnerID,
		&city.Pop, &city.Ind, &city.Tra, &city.Sec, &city.Env, &city.Com, &city.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			http.Error(w, "slug already taken", http.StatusConflict)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(city)
}

// GET /api/cities/{slug}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	slug, ok := pathSlugOrBadRequest(w, r)
	if !ok {
		return
	}
	city, err := h.getCity(r.Context(), slug)
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "city not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(city)
}

// GET /api/debug/cities/{slug}
func (h *Handler) DebugGet(w http.ResponseWriter, r *http.Request) {
	slug, ok := pathSlugOrBadRequest(w, r)
	if !ok {
		return
	}
	ctx := r.Context()

	city, err := h.getCity(ctx, slug)
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "city not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	todayBySector := make(map[string]int, len(citycore.AllSectors))
	for _, s := range citycore.AllSectors {
		todayBySector[s.String()] = 0
	}

	rows, err := h.db.Query(ctx,
		`SELECT sector, COUNT(*)::int
		 FROM visites_log
		 WHERE city_slug = $1
		   AND (created_at AT TIME ZONE 'UTC')::date = (now() AT TIME ZONE 'UTC')::date
		 GROUP BY sector`,
		slug.String())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var sector string
		var cnt int
		if scanErr := rows.Scan(&sector, &cnt); scanErr != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		todayBySector[sector] = cnt
	}
	if err := rows.Err(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	logRows, err := h.db.Query(ctx,
		`SELECT sector, created_at, visitor_hash
		 FROM visites_log
		 WHERE city_slug = $1
		 ORDER BY created_at DESC
		 LIMIT 20`,
		slug.String())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer logRows.Close()

	recent := make([]debugSupportLog, 0, 20)
	for logRows.Next() {
		var item debugSupportLog
		var visitorHash string
		if scanErr := logRows.Scan(&item.Sector, &item.CreatedAt, &visitorHash); scanErr != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if len(visitorHash) >= 8 {
			item.VisitorTag = visitorHash[:8]
		} else {
			item.VisitorTag = visitorHash
		}
		recent = append(recent, item)
	}
	if err := logRows.Err(); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	unlockedSectors := citycore.UnlockedSectors(city)
	unlocked := make([]string, 0, len(unlockedSectors))
	for _, sector := range unlockedSectors {
		unlocked = append(unlocked, sector.String())
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"city":                  city,
		"unlocked_sectors":      unlocked,
		"today_support_by_kind": todayBySector,
		"recent_support_logs":   recent,
		"debug": map[string]any{
			"utc_date": time.Now().UTC().Format("2006-01-02"),
		},
	})
}

// POST /api/cities/{slug}/support
func (h *Handler) Support(w http.ResponseWriter, r *http.Request) {
	slug, ok := pathSlugOrBadRequest(w, r)
	if !ok {
		return
	}

	var req struct {
		Sector string `json:"sector"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	rawSector := req.Sector
	if rawSector == "" {
		rawSector = citycore.SectorPop.String()
	}
	sector, err := citycore.ParseSector(rawSector)
	if err != nil {
		http.Error(w, "invalid sector", http.StatusBadRequest)
		return
	}
	col := citycore.SectorColumn(sector)
	if col == "" {
		http.Error(w, "invalid sector", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	city, err := h.getCity(ctx, slug)
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "city not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if !citycore.IsUnlocked(city, sector) {
		http.Error(w, fmt.Sprintf("sector %s not yet unlocked", sector), http.StatusForbidden)
		return
	}

	now := time.Now().UTC()
	clientIP := middleware.GetClientIP(r)
	hash := citycore.VisitorHashFromValues(clientIP, r.UserAgent(), h.saltSeed, now)
	date := now.Format("2006-01-02")
	key := citycore.VisitKey(date, slug, hash)

	ttl := citycore.VisitTTLUntilUTCMidnight(now)

	set, err := h.rdb.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !set {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"already_voted":true,"message":"You already supported this city today"}`))
		return
	}

	_, err = h.db.Exec(ctx,
		fmt.Sprintf(`UPDATE cities SET %s = %s + 1 WHERE slug = $1`, col, col),
		slug.String())
	if err != nil {
		if delErr := h.rdb.Del(ctx, key).Err(); delErr != nil {
			slog.Warn("failed to roll back Redis visit key", "key", key, "err", delErr)
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if _, logErr := h.db.Exec(ctx,
		`INSERT INTO visites_log (city_slug, sector, visitor_hash) VALUES ($1, $2, $3)`,
		slug.String(), sector.String(), hash.String()); logErr != nil {
		slog.Warn("failed to insert visites_log", "city", slug.String(), "err", logErr)
	}

	city, _ = h.getCity(ctx, slug)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"city":    city,
		"message": fmt.Sprintf("+1 %s!", sector),
	})
}

// GET /api/rankings
func (h *Handler) Rankings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ccRaw := r.URL.Query().Get("country")
	limit := 20

	var rows pgx.Rows
	var err error
	if ccRaw != "" {
		cc, parseErr := citycore.ParseCountryCode(ccRaw)
		if parseErr != nil {
			http.Error(w, "country must be 2 letters A-Z", http.StatusBadRequest)
			return
		}
		rows, err = h.db.Query(ctx,
			`SELECT slug, name, country_code, pop, ind, tra, sec, env, com, created_at
			 FROM cities WHERE country_code = $1 ORDER BY pop DESC LIMIT $2`,
			cc.String(), limit)
	} else {
		rows, err = h.db.Query(ctx,
			`SELECT slug, name, country_code, pop, ind, tra, sec, env, com, created_at
			 FROM cities ORDER BY pop DESC LIMIT $1`,
			limit)
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	cities := []citycore.City{}
	for rows.Next() {
		var c citycore.City
		if err := rows.Scan(&c.Slug, &c.Name, &c.CountryCode,
			&c.Pop, &c.Ind, &c.Tra, &c.Sec, &c.Env, &c.Com, &c.CreatedAt); err != nil {
			slog.Warn("rankings: failed to scan row", "err", err)
			continue
		}
		cities = append(cities, c)
	}
	if err := rows.Err(); err != nil {
		slog.Error("rankings: row iteration error", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(cities)
}

var badgeTmpl = template.Must(template.New("badge").Parse(`<svg xmlns="http://www.w3.org/2000/svg" width="160" height="20">
  <linearGradient id="s" x2="0" y2="100%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <rect rx="3" width="160" height="20" fill="#555"/>
  <rect rx="3" x="80" width="80" height="20" fill="#4c9e4c"/>
  <rect rx="3" width="160" height="20" fill="url(#s)"/>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
    <text x="40" y="15" fill="#010101" fill-opacity=".3">{{.Name}}</text>
    <text x="40" y="14">{{.Name}}</text>
    <text x="120" y="15" fill="#010101" fill-opacity=".3">pop {{.Pop}}</text>
    <text x="120" y="14">pop {{.Pop}}</text>
  </g>
</svg>`))

// GET /badge/{slug}.svg
func (h *Handler) Badge(w http.ResponseWriter, r *http.Request) {
	slug, ok := pathSlugOrBadRequest(w, r)
	if !ok {
		return
	}
	city, err := h.getCity(r.Context(), slug)
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "city not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=7200")
	_ = badgeTmpl.Execute(w, city)
}

// GET /api/cities/{slug}/map.png
func (h *Handler) MapPNG(w http.ResponseWriter, r *http.Request) {
	slug, ok := pathSlugOrBadRequest(w, r)
	if !ok {
		return
	}
	city, err := h.getCity(r.Context(), slug)
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "city not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	simulating := false
	if DebugModeEnabled() {
		overridden, applied, overrideErr := applyMapDebugOverrides(city, r.URL.Query())
		if overrideErr != nil {
			http.Error(w, overrideErr.Error(), http.StatusBadRequest)
			return
		}
		if applied {
			city = overridden
			simulating = true
		}
	}

	etag, err := render.MapEntityTag(city)
	if err != nil {
		http.Error(w, "failed to render map", http.StatusInternalServerError)
		return
	}
	if render.MatchIfNoneMatch(r.Header.Get("If-None-Match"), etag) {
		setMapPNGCacheHeaders(w, etag, simulating)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	pngBytes, err := render.BuildCityMapPNG(city)
	if err != nil {
		http.Error(w, "failed to render map", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	setMapPNGCacheHeaders(w, etag, simulating)
	_, _ = w.Write(pngBytes)
}

const mapPNGCacheControl = "public, max-age=300"
const mapPNGDebugCacheControl = "no-store"

func setMapPNGCacheHeaders(w http.ResponseWriter, etag string, simulating bool) {
	if simulating {
		w.Header().Set("Cache-Control", mapPNGDebugCacheControl)
		w.Header().Set("X-Bitown-Map-Debug", "1")
	} else {
		w.Header().Set("Cache-Control", mapPNGCacheControl)
	}
	w.Header().Set("ETag", etag)
}

func pathSlugOrBadRequest(w http.ResponseWriter, r *http.Request) (citycore.Slug, bool) {
	slug, err := citycore.ParseSlug(chi.URLParam(r, "slug"))
	if err != nil {
		http.Error(w, "slug must be 2-40 lowercase alphanumeric/hyphen", http.StatusBadRequest)
		return "", false
	}
	return slug, true
}

func (h *Handler) getCity(ctx context.Context, slug citycore.Slug) (*citycore.City, error) {
	var c citycore.City
	err := h.db.QueryRow(ctx,
		`SELECT slug, name, country_code, owner_id, pop, ind, tra, sec, env, com, created_at
		 FROM cities WHERE slug = $1`, slug.String(),
	).Scan(&c.Slug, &c.Name, &c.CountryCode, &c.OwnerID,
		&c.Pop, &c.Ind, &c.Tra, &c.Sec, &c.Env, &c.Com, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
