package city

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestGet_IncludesMetrics(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock: %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(cityBySlugQuery).WithArgs("tokyo").WillReturnRows(cityRow(1))

	h := NewHandler(mock, nil, "test-salt")
	req := withSlugParam(httptest.NewRequest(http.MethodGet, "/api/cities/tokyo", nil), "tokyo")
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 body=%q", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v", err)
	}
	metrics, ok := body["metrics"].(map[string]any)
	if !ok {
		t.Fatalf("missing metrics: %v", body)
	}
	if _, ok := metrics["income"]; !ok {
		t.Fatalf("metrics missing income: %v", metrics)
	}
}
