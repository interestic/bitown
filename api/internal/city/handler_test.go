package city

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func TestCreate_ValidationErrors(t *testing.T) {
	h := NewHandler(nil, nil, "test-salt")

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "invalid json",
			body:       `{"name":"Tokyo"`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid JSON",
		},
		{
			name:       "invalid slug too short",
			body:       `{"name":"Tokyo","country_code":"JP","slug":"a"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "slug must be 2-40 lowercase alphanumeric/hyphen",
		},
		{
			name:       "invalid name too short",
			body:       `{"name":"A","country_code":"JP","slug":"tokyo"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "name must be 2-40 characters",
		},
		{
			name:       "invalid country code length",
			body:       `{"name":"Tokyo","country_code":"JPN","slug":"tokyo"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "country_code must be 2 letters A-Z",
		},
		{
			name:       "invalid country code digits",
			body:       `{"name":"Tokyo","country_code":"12","slug":"tokyo"}`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "country_code must be 2 letters A-Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/cities", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			h.Create(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Fatalf("body = %q, want to contain %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestCreate_SuccessNormalizesSlugAndCountry(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	created := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`INSERT INTO cities \(slug, name, country_code\) VALUES \(\$1, \$2, \$3\)\s+RETURNING slug, name, country_code, owner_id, pop, ind, tra, sec, env, com, created_at`).
		WithArgs("tokyo", "Tokyo", "JP").
		WillReturnRows(pgxmock.NewRows([]string{
			"slug", "name", "country_code", "owner_id", "pop", "ind", "tra", "sec", "env", "com", "created_at",
		}).AddRow("tokyo", "Tokyo", "JP", nil, 0, 0, 0, 0, 0, 0, created))

	h := NewHandler(mock, nil, "test-salt")
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/cities",
		strings.NewReader(`{"name":"Tokyo","country_code":"jp","slug":"  Tokyo  "}`),
	)
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusCreated, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"slug":"tokyo"`) {
		t.Fatalf("body = %q, want normalized slug tokyo", body)
	}
	if !strings.Contains(body, `"country_code":"JP"`) {
		t.Fatalf("body = %q, want normalized country JP", body)
	}
}

func TestGet_InvalidPathSlug(t *testing.T) {
	h := NewHandler(nil, nil, "test-salt")
	req := withSlugParam(httptest.NewRequest(http.MethodGet, "/api/cities/Bad_Slug", nil), "Bad_Slug")
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), "slug must be 2-40") {
		t.Fatalf("body = %q, want slug validation message", rec.Body.String())
	}
}

func TestGet_NormalizesPathSlug(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	created := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT slug, name, country_code, owner_id, pop, ind, tra, sec, env, com, created_at\s+FROM cities WHERE slug = \$1`).
		WithArgs("tokyo").
		WillReturnRows(pgxmock.NewRows([]string{
			"slug", "name", "country_code", "owner_id", "pop", "ind", "tra", "sec", "env", "com", "created_at",
		}).AddRow("tokyo", "Tokyo", "JP", nil, 10, 0, 0, 0, 0, 0, created))

	h := NewHandler(mock, nil, "test-salt")
	// Path params used to be looked up raw; ParseSlug now lowercases before DB.
	req := withSlugParam(httptest.NewRequest(http.MethodGet, "/api/cities/Tokyo", nil), "Tokyo")
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"slug":"tokyo"`) {
		t.Fatalf("body = %q, want slug tokyo", rec.Body.String())
	}
}

func TestSupport_InvalidSector(t *testing.T) {
	h := NewHandler(nil, nil, "test-salt")
	req := withSlugParam(
		httptest.NewRequest(http.MethodPost, "/api/cities/tokyo/support", strings.NewReader(`{"sector":"hack"}`)),
		"tokyo",
	)
	rec := httptest.NewRecorder()

	h.Support(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), "invalid sector") {
		t.Fatalf("body = %q, want to contain %q", rec.Body.String(), "invalid sector")
	}
}

func TestRankings_InvalidCountry(t *testing.T) {
	h := NewHandler(nil, nil, "test-salt")
	req := httptest.NewRequest(http.MethodGet, "/api/rankings?country=12", nil)
	rec := httptest.NewRecorder()

	h.Rankings(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), "country must be 2 letters A-Z") {
		t.Fatalf("body = %q, want country validation message", rec.Body.String())
	}
}
