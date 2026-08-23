package city

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/interestic/bitown/internal/citycore"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/redis/go-redis/v9"
)

const cityBySlugQuery = `SELECT slug, name, country_code, owner_id, pop, ind, tra, sec, env, com, created_at\s+FROM cities WHERE slug = \$1`

func cityRow(pop int) *pgxmock.Rows {
	created := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	return pgxmock.NewRows([]string{
		"slug", "name", "country_code", "owner_id", "pop", "ind", "tra", "sec", "env", "com", "created_at",
	}).AddRow("tokyo", "Tokyo", "JP", nil, pop, 0, 0, 0, 0, 0, created)
}

func TestSupport_LockedSector(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(cityBySlugQuery).
		WithArgs("tokyo").
		WillReturnRows(cityRow(0))

	h := NewHandler(mock, nil, "test-salt")
	req := withSlugParam(
		httptest.NewRequest(http.MethodPost, "/api/cities/tokyo/support", strings.NewReader(`{"sector":"ind"}`)),
		"tokyo",
	)
	rec := httptest.NewRecorder()

	h.Support(rec, req)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if !strings.Contains(rec.Body.String(), "sector ind not yet unlocked") {
		t.Fatalf("body = %q, want locked sector message", rec.Body.String())
	}
}

func TestSupport_EmptySectorDefaultsToPop(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	mock.ExpectQuery(cityBySlugQuery).WithArgs("tokyo").WillReturnRows(cityRow(10))
	mock.ExpectExec(`UPDATE cities SET pop = pop \+ 1 WHERE slug = \$1`).
		WithArgs("tokyo").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`INSERT INTO visites_log \(city_slug, sector, visitor_hash\) VALUES \(\$1, \$2, \$3\)`).
		WithArgs("tokyo", citycore.SectorPop.String(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectQuery(cityBySlugQuery).WithArgs("tokyo").WillReturnRows(cityRow(11))

	h := NewHandler(mock, rdb, "test-salt")
	req := withSlugParam(
		httptest.NewRequest(http.MethodPost, "/api/cities/tokyo/support", strings.NewReader(`{}`)),
		"tokyo",
	)
	req.RemoteAddr = "192.0.2.1:12345"
	rec := httptest.NewRecorder()

	h.Support(rec, req)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "+1 pop!") {
		t.Fatalf("body = %q, want +1 pop message", rec.Body.String())
	}
}

func TestSupport_UnlockedSectorIncrements(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	mock.ExpectQuery(cityBySlugQuery).WithArgs("tokyo").WillReturnRows(cityRow(50))
	mock.ExpectExec(`UPDATE cities SET ind = ind \+ 1 WHERE slug = \$1`).
		WithArgs("tokyo").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`INSERT INTO visites_log \(city_slug, sector, visitor_hash\) VALUES \(\$1, \$2, \$3\)`).
		WithArgs("tokyo", citycore.SectorInd.String(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectQuery(cityBySlugQuery).WithArgs("tokyo").WillReturnRows(cityRow(50))

	h := NewHandler(mock, rdb, "test-salt")
	req := withSlugParam(
		httptest.NewRequest(http.MethodPost, "/api/cities/tokyo/support", strings.NewReader(`{"sector":"ind"}`)),
		"tokyo",
	)
	req.RemoteAddr = "192.0.2.1:12345"
	rec := httptest.NewRecorder()

	h.Support(rec, req)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "+1 ind!") {
		t.Fatalf("body = %q, want +1 ind message", rec.Body.String())
	}
}
