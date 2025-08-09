package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/middleware"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/repository"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/service"
	"github.com/gin-gonic/gin"
)

// Example showing basic usage of JSON endpoints /update/ and /value/.
func ExampleHandler_JSONEndpoints() {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	ms := &service.MetricsService{Storage: repository.NewMemStorage()}
	h := NewHandler(ms)
	h.SetupRoutes(router)

	// Update a gauge via JSON
	body := middleware.MetricsJSON{ID: "Alloc", MType: middleware.GaugeMetric, Value: floatPtr(42.0)}
	b, _ := json.Marshal(body)
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/update/", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rr, req)

	// Read it back via JSON
	rr = httptest.NewRecorder()
	q := middleware.MetricsJSON{ID: "Alloc", MType: middleware.GaugeMetric}
	qb, _ := json.Marshal(q)
	req, _ = http.NewRequest(http.MethodPost, "/value/", bytes.NewReader(qb))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rr, req)

	fmt.Println(rr.Code)
	// Output: 200
}

func floatPtr(v float64) *float64 { return &v }
