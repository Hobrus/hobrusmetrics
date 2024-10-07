package handlers

import (
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/repository"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/service"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRouter() (*gin.Engine, *service.MetricsService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	var storage repository.Storage = repository.NewMemStorage()
	metricsService := &service.MetricsService{Storage: storage}
	handler := NewHandler(metricsService)
	handler.SetupRoutes(router)
	return router, metricsService
}

func TestUpdateHandler(t *testing.T) {
	router, _ := setupRouter()

	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "Valid gauge metric update",
			url:            "/update/gauge/Alloc/123.45",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid counter metric update",
			url:            "/update/counter/PollCount/10",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid metric type",
			url:            "/update/invalid/Alloc/123.45",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid metric value",
			url:            "/update/gauge/Alloc/abc",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, tc.url, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestGetValueHandler(t *testing.T) {
	router, ms := setupRouter()

	// Setup some initial data
	_ = ms.UpdateMetric("gauge", "Alloc", "123.45")
	_ = ms.UpdateMetric("counter", "PollCount", "10")

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Get existing gauge metric",
			url:            "/value/gauge/Alloc",
			expectedStatus: http.StatusOK,
			expectedBody:   "123.45",
		},
		{
			name:           "Get existing counter metric",
			url:            "/value/counter/PollCount",
			expectedStatus: http.StatusOK,
			expectedBody:   "10",
		},
		{
			name:           "Get non-existent metric",
			url:            "/value/gauge/NonExistent",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid metric type",
			url:            "/value/invalid/Metric",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, tc.url, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			if tc.expectedStatus == http.StatusOK {
				assert.Equal(t, tc.expectedBody, w.Body.String())
			}
		})
	}
}

func TestGetAllMetricsHandler(t *testing.T) {
	router, ms := setupRouter()

	// Setup some initial data
	_ = ms.UpdateMetric("gauge", "Alloc", "123.45")
	_ = ms.UpdateMetric("counter", "PollCount", "10")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Alloc: 123.45")
	assert.Contains(t, w.Body.String(), "PollCount: 10")
}

func TestMetricsServiceIntegration(t *testing.T) {
	router, ms := setupRouter()

	// Test updating a metric
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/update/gauge/TestMetric/42.0", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test getting the updated metric
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/value/gauge/TestMetric", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "42", w.Body.String())

	// Test getting all metrics
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "TestMetric: 42")

	// Verify the metric is stored correctly
	value, err := ms.GetMetricValue("gauge", "TestMetric")
	require.NoError(t, err)
	assert.Equal(t, "42", value)
}
