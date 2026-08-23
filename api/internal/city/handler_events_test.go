package city

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestEvents_ReturnsRows(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	created := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(cityBySlugQuery).WithArgs("tokyo").WillReturnRows(cityRow(10))
	mock.ExpectQuery(`SELECT id, event_type, delta, created_at\s+FROM events_log WHERE city_slug = \$1\s+ORDER BY created_at DESC, id DESC LIMIT \$2`).
		WithArgs("tokyo", 20).
		WillReturnRows(pgxmock.NewRows([]string{"id", "event_type", "delta", "created_at"}).
			AddRow(int64(1), "support", []byte(`{"sector":"pop","delta":1}`), created))

	h := NewHandler(mock, nil, "test-salt")
	req := withSlugParam(httptest.NewRequest(http.MethodGet, "/api/cities/tokyo/events", nil), "tokyo")
	rec := httptest.NewRecorder()
	h.Events(rec, req)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%q", rec.Code, rec.Body.String())
	}
	var events []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &events); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(events) != 1 || events[0]["event_type"] != "support" {
		t.Fatalf("events = %v", events)
	}
}

func TestEvents_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(cityBySlugQuery).WithArgs("nope").WillReturnError(pgx.ErrNoRows)

	h := NewHandler(mock, nil, "test-salt")
	req := withSlugParam(httptest.NewRequest(http.MethodGet, "/api/cities/nope/events", nil), "nope")
	rec := httptest.NewRecorder()
	h.Events(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
