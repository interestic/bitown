package city

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/interestic/bitown/internal/render"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func withSlugParam(req *http.Request, slug string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("slug", slug)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestMapPNG_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT slug, name, country_code, owner_id, pop, ind, tra, sec, env, com, created_at\s+FROM cities WHERE slug = \$1`).
		WithArgs("missing").
		WillReturnError(pgx.ErrNoRows)

	h := NewHandler(mock, nil, "test-salt")
	req := withSlugParam(httptest.NewRequest(http.MethodGet, "/api/cities/missing/map.png", nil), "missing")
	rec := httptest.NewRecorder()

	h.MapPNG(rec, req)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
func TestMapPNG_NotModified(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	created := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	const q = `SELECT slug, name, country_code, owner_id, pop, ind, tra, sec, env, com, created_at\s+FROM cities WHERE slug = \$1`
	mock.ExpectQuery(q).WithArgs("testcity").WillReturnRows(pgxmock.NewRows([]string{
		"slug", "name", "country_code", "owner_id", "pop", "ind", "tra", "sec", "env", "com", "created_at",
	}).AddRow("testcity", "TestCity", "JP", nil, 42, 0, 0, 0, 0, 0, created))
	mock.ExpectQuery(q).WithArgs("testcity").WillReturnRows(pgxmock.NewRows([]string{
		"slug", "name", "country_code", "owner_id", "pop", "ind", "tra", "sec", "env", "com", "created_at",
	}).AddRow("testcity", "TestCity", "JP", nil, 42, 0, 0, 0, 0, 0, created))

	h := NewHandler(mock, nil, "test-salt")

	firstReq := withSlugParam(httptest.NewRequest(http.MethodGet, "/api/cities/testcity/map.png", nil), "testcity")
	firstRec := httptest.NewRecorder()
	h.MapPNG(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first status = %d, want 200", firstRec.Code)
	}
	etag := firstRec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag on first response")
	}

	secondReq := withSlugParam(httptest.NewRequest(http.MethodGet, "/api/cities/testcity/map.png", nil), "testcity")
	secondReq.Header.Set("If-None-Match", etag)
	secondRec := httptest.NewRecorder()
	h.MapPNG(secondRec, secondReq)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
	if secondRec.Code != http.StatusNotModified {
		t.Fatalf("second status = %d, want 304", secondRec.Code)
	}
	if len(secondRec.Body.Bytes()) != 0 {
		t.Fatalf("expected empty 304 body, got %d bytes", len(secondRec.Body.Bytes()))
	}
	if got := secondRec.Header().Get("ETag"); got != etag {
		t.Fatalf("304 ETag = %q, want %q", got, etag)
	}
	if cc := secondRec.Header().Get("Cache-Control"); !strings.Contains(cc, "max-age=300") {
		t.Fatalf("304 Cache-Control = %q, want max-age=300", cc)
	}
}

func TestMapPNG_IfNoneMatchList(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	created := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	const q = `SELECT slug, name, country_code, owner_id, pop, ind, tra, sec, env, com, created_at\s+FROM cities WHERE slug = \$1`
	rows := func() *pgxmock.Rows {
		return pgxmock.NewRows([]string{
			"slug", "name", "country_code", "owner_id", "pop", "ind", "tra", "sec", "env", "com", "created_at",
		}).AddRow("testcity", "TestCity", "JP", nil, 42, 0, 0, 0, 0, 0, created)
	}
	mock.ExpectQuery(q).WithArgs("testcity").WillReturnRows(rows())
	mock.ExpectQuery(q).WithArgs("testcity").WillReturnRows(rows())

	h := NewHandler(mock, nil, "test-salt")

	firstReq := withSlugParam(httptest.NewRequest(http.MethodGet, "/api/cities/testcity/map.png", nil), "testcity")
	firstRec := httptest.NewRecorder()
	h.MapPNG(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first status = %d, want 200", firstRec.Code)
	}
	etag := firstRec.Header().Get("ETag")

	secondReq := withSlugParam(httptest.NewRequest(http.MethodGet, "/api/cities/testcity/map.png", nil), "testcity")
	secondReq.Header.Set("If-None-Match", `W/"stale", `+etag)
	secondRec := httptest.NewRecorder()
	h.MapPNG(secondRec, secondReq)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
	if secondRec.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want 304", secondRec.Code)
	}
	if got := secondRec.Header().Get("ETag"); got != etag {
		t.Fatalf("304 ETag = %q, want %q", got, etag)
	}
}

func TestMapPNG_OK(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	created := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	rows := pgxmock.NewRows([]string{
		"slug", "name", "country_code", "owner_id", "pop", "ind", "tra", "sec", "env", "com", "created_at",
	}).AddRow("testcity", "TestCity", "JP", nil, 42, 0, 0, 0, 0, 0, created)

	mock.ExpectQuery(`SELECT slug, name, country_code, owner_id, pop, ind, tra, sec, env, com, created_at\s+FROM cities WHERE slug = \$1`).
		WithArgs("testcity").
		WillReturnRows(rows)

	h := NewHandler(mock, nil, "test-salt")
	req := withSlugParam(httptest.NewRequest(http.MethodGet, "/api/cities/testcity/map.png", nil), "testcity")
	rec := httptest.NewRecorder()

	h.MapPNG(rec, req)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "max-age=300") {
		t.Fatalf("Cache-Control = %q, want max-age=300", cc)
	}
	if rec.Header().Get("ETag") == "" {
		t.Fatal("expected ETag header")
	}
	if len(rec.Body.Bytes()) == 0 {
		t.Fatal("expected png body")
	}
}

func TestMapPNG_AtlasRequiredFailure(t *testing.T) {
	render.ResetAtlasCacheForTest()
	t.Setenv("BITOWN_ATLAS_REQUIRED", "true")
	t.Setenv("BITOWN_ASSETS_DIR", t.TempDir())

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	created := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT slug, name, country_code, owner_id, pop, ind, tra, sec, env, com, created_at\s+FROM cities WHERE slug = \$1`).
		WithArgs("testcity").
		WillReturnRows(pgxmock.NewRows([]string{
			"slug", "name", "country_code", "owner_id", "pop", "ind", "tra", "sec", "env", "com", "created_at",
		}).AddRow("testcity", "TestCity", "JP", nil, 42, 0, 0, 0, 0, 0, created))

	h := NewHandler(mock, nil, "test-salt")
	req := withSlugParam(httptest.NewRequest(http.MethodGet, "/api/cities/testcity/map.png", nil), "testcity")
	rec := httptest.NewRecorder()

	h.MapPNG(rec, req)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
