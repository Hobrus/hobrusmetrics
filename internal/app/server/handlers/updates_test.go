package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/middleware"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/repository"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/service"
)

func TestUpdateBatchHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	ms := &service.MetricsService{Storage: repository.NewMemStorage()}
	h := NewHandler(ms)
	h.SetupRoutes(router)

	g := 3.14
	d := int64(5)
	batch := []middleware.MetricsJSON{{ID: "G", MType: middleware.GaugeMetric, Value: &g}, {ID: "C", MType: middleware.CounterMetric, Delta: &d}}
	body, _ := json.Marshal(batch)

	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/updates/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d", rr.Code)
	}
}
