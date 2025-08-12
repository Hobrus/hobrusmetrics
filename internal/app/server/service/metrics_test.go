package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/middleware"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/repository"
)

func TestUpdateMetricAndGetMetricValue(t *testing.T) {
	storage := repository.NewMemStorage()
	ms := &MetricsService{Storage: storage}

	// Gauge valid
	if err := ms.UpdateMetric("gauge", "G1", "42.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v, err := ms.GetMetricValue("gauge", "G1")
	if err != nil || v != "42" { // formatted via %g
		t.Fatalf("expected 42, got %q, err=%v", v, err)
	}

	// Gauge invalid
	if err = ms.UpdateMetric("gauge", "G2", "not-a-float"); err == nil {
		t.Fatalf("expected error for invalid gauge value")
	}

	// Counter accumulation
	if err = ms.UpdateMetric("counter", "C1", "10"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err = ms.UpdateMetric("counter", "C1", "5"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cv, err := ms.GetMetricValue("counter", "C1")
	if err != nil || cv != "15" {
		t.Fatalf("expected 15, got %q, err=%v", cv, err)
	}

	// Unsupported
	if err := ms.UpdateMetric("unknown", "X", "1"); err == nil {
		t.Fatalf("expected unsupported metric type error")
	}
}

func TestUpdateMetricsBatchAndGetAll(t *testing.T) {
	storage := repository.NewMemStorage()
	ms := &MetricsService{Storage: storage}

	g := 12.5
	d1 := int64(7)
	d2 := int64(3)
	batch := []middleware.MetricsJSON{
		{ID: "G", MType: middleware.GaugeMetric, Value: &g},
		{ID: "C", MType: middleware.CounterMetric, Delta: &d1},
		{ID: "C", MType: middleware.CounterMetric, Delta: &d2},
	}

	updated, err := ms.UpdateMetricsBatch(batch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expect two metrics in response
	if len(updated) != 2 && len(updated) != 3 {
		t.Fatalf("expected 2 or 3 updated metrics, got %d", len(updated))
	}

	// Verify GetAllMetrics string formats
	all := ms.GetAllMetrics()
	if all["G"] != "12.5" {
		t.Fatalf("expected G=12.5, got %q", all["G"])
	}
	if all["C"] != "10" {
		t.Fatalf("expected C=10, got %q", all["C"])
	}
}

func TestHandlersIntegrationWithService(t *testing.T) {
	// Minimal integration: JSON middlewares over the service
	storage := repository.NewMemStorage()
	ms := &MetricsService{Storage: storage}

	// Build a small router with JSON update and value endpoints
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/update/", middleware.JSONUpdateMiddleware(ms))
	router.POST("/value/", middleware.JSONValueMiddleware(ms))

	// Update gauge via JSON
	body := middleware.MetricsJSON{ID: "T", MType: middleware.GaugeMetric, Value: floatPtr(42.0)}
	rr := httptest.NewRecorder()
	req := newJSONRequest(t, http.MethodPost, "/update/", body)
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("update status = %d", rr.Code)
	}

	// Read it back
	rr = httptest.NewRecorder()
	req = newJSONRequest(t, http.MethodPost, "/value/", middleware.MetricsJSON{ID: "T", MType: middleware.GaugeMetric})
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("value status = %d", rr.Code)
	}
}

func floatPtr(v float64) *float64 { return &v }

func newJSONRequest(t *testing.T, method, url string, v any) *http.Request {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}
