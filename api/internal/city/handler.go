package city

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/interestic/bitown/internal/citycore"
	"github.com/interestic/bitown/internal/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// slugRe requires 2-40 chars, lowercase alphanumeric and hyphens,
// with no leading or trailing hyphens.
// The mandatory [a-z0-9] at both ends ensures minimum 2 chars.
var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,38}[a-z0-9]$`)

type Handler struct {
	db       *pgxpool.Pool
	rdb      *redis.Client
	saltSeed string
}

func NewHandler(db *pgxpool.Pool, rdb *redis.Client, saltSeed string) *Handler {
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

	req.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	if !slugRe.MatchString(req.Slug) {
		http.Error(w, "slug must be 2-40 lowercase alphanumeric/hyphen", http.StatusBadRequest)
		return
	}
	if l := utf8.RuneCountInString(req.Name); l < 2 || l > 40 {
		http.Error(w, "name must be 2-40 characters", http.StatusBadRequest)
		return
	}
	cc := strings.ToUpper(strings.TrimSpace(req.CountryCode))
	if len(cc) != 2 {
		http.Error(w, "country_code must be 2 chars", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	var city citycore.City
	err := h.db.QueryRow(ctx,
		`INSERT INTO cities (slug, name, country_code) VALUES ($1, $2, $3)
		 RETURNING slug, name, country_code, owner_id, pop, ind, tra, sec, env, com, created_at`,
		req.Slug, req.Name, cc,
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
	slug := chi.URLParam(r, "slug")
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

// POST /api/cities/{slug}/support
func (h *Handler) Support(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var req struct {
		Sector string `json:"sector"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Sector == "" {
		req.Sector = citycore.SectorPop
	}
	if !citycore.ValidSectors[req.Sector] {
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

	if !citycore.IsUnlocked(city, req.Sector) {
		http.Error(w, fmt.Sprintf("sector %s not yet unlocked", req.Sector), http.StatusForbidden)
		return
	}

	now := time.Now().UTC()
	clientIP := middleware.GetClientIP(r)
	hash := citycore.VisitorHashFromValues(clientIP, r.UserAgent(), h.saltSeed, now)
	date := now.Format("2006-01-02")
	key := citycore.VisitKey(date, slug, hash)

	endOfDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	ttl := endOfDay.Sub(now)

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

	col := citycore.SectorColumn(req.Sector)
	_, err = h.db.Exec(ctx,
		fmt.Sprintf(`UPDATE cities SET %s = %s + 1 WHERE slug = $1`, col, col),
		slug)
	if err != nil {
		if delErr := h.rdb.Del(ctx, key).Err(); delErr != nil {
			slog.Warn("failed to roll back Redis visit key", "key", key, "err", delErr)
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if _, logErr := h.db.Exec(ctx,
		`INSERT INTO visites_log (city_slug, sector, visitor_hash) VALUES ($1, $2, $3)`,
		slug, req.Sector, hash); logErr != nil {
		slog.Warn("failed to insert visites_log", "city", slug, "err", logErr)
	}

	city, _ = h.getCity(ctx, slug)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"city":    city,
		"message": fmt.Sprintf("+1 %s!", req.Sector),
	})
}

// GET /api/rankings
func (h *Handler) Rankings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cc := r.URL.Query().Get("country")
	limit := 20

	var rows pgx.Rows
	var err error
	if cc != "" {
		rows, err = h.db.Query(ctx,
			`SELECT slug, name, country_code, pop, ind, tra, sec, env, com, created_at
			 FROM cities WHERE country_code = $1 ORDER BY pop DESC LIMIT $2`,
			strings.ToUpper(cc), limit)
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
	slug := chi.URLParam(r, "slug")
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

func (h *Handler) getCity(ctx context.Context, slug string) (*citycore.City, error) {
	var c citycore.City
	err := h.db.QueryRow(ctx,
		`SELECT slug, name, country_code, owner_id, pop, ind, tra, sec, env, com, created_at
		 FROM cities WHERE slug = $1`, slug,
	).Scan(&c.Slug, &c.Name, &c.CountryCode, &c.OwnerID,
		&c.Pop, &c.Ind, &c.Tra, &c.Sec, &c.Env, &c.Com, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
