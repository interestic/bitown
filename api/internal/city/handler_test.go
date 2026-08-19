package city

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
			wantBody:   "country_code must be 2 chars",
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

func TestSupport_InvalidSector(t *testing.T) {
	h := NewHandler(nil, nil, "test-salt")
	req := httptest.NewRequest(http.MethodPost, "/api/cities/tokyo/support", strings.NewReader(`{"sector":"hack"}`))
	rec := httptest.NewRecorder()

	h.Support(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), "invalid sector") {
		t.Fatalf("body = %q, want to contain %q", rec.Body.String(), "invalid sector")
	}
}
