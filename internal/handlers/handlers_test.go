package handlers

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Hobrus/hobrusmetrics.git/internal/service"
)

func TestUpdateHandler(t *testing.T) {
	// Создаем новое хранилище в памяти
	storage := service.NewMemStorage()

	// Создаем экземпляр MetricsService с использованием хранилища
	metricsService := &service.MetricsService{Storage: storage}

	// Создаем хендлер
	handler := UpdateHandler(metricsService)

	tests := []struct {
		name           string
		method         string
		contentType    string
		url            string
		expectedStatus int
	}{
		{
			name:           "Valid gauge metric update",
			method:         http.MethodPost,
			contentType:    "text/plain",
			url:            "/update/gauge/Alloc/123.45",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid counter metric update",
			method:         http.MethodPost,
			contentType:    "text/plain",
			url:            "/update/counter/PollCount/10",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Unsupported method",
			method:         http.MethodGet,
			contentType:    "text/plain",
			url:            "/update/gauge/Alloc/123.45",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Unsupported content type",
			method:         http.MethodPost,
			contentType:    "application/json",
			url:            "/update/gauge/Alloc/123.45",
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "Invalid metric type",
			method:         http.MethodPost,
			contentType:    "text/plain",
			url:            "/update/invalid/Alloc/123.45",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid metric value",
			method:         http.MethodPost,
			contentType:    "text/plain",
			url:            "/update/gauge/Alloc/abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Metric name missing",
			method:         http.MethodPost,
			contentType:    "text/plain",
			url:            "/update/gauge//123.45",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid URL path",
			method:         http.MethodPost,
			contentType:    "text/plain",
			url:            "/invalidpath/gauge/Alloc/123.45",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.url, nil)
			req.Header.Set("Content-Type", tc.contentType)
			w := httptest.NewRecorder()

			handler(w, req)

			resp := w.Result()
			defer resp.Body.Close() // Ensure the response body is always closed

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			}

			if resp.StatusCode == http.StatusOK {
				parts := strings.Split(strings.TrimPrefix(tc.url, "/"), "/")
				metricType := parts[1]
				metricName := parts[2]
				metricValue := parts[3]

				switch metricType {
				case "gauge":
					repoGauge, exists := storage.GetGauge(metricName)
					if !exists {
						t.Errorf("Gauge metric %s was not stored", metricName)
					} else {
						value, _ := strconv.ParseFloat(metricValue, 64)
						if float64(repoGauge) != value {
							t.Errorf("Gauge metric %s has value %v, expected %v", metricName, repoGauge, value)
						}
					}
				case "counter":
					repoCounter, exists := storage.GetCounter(metricName)
					if !exists {
						t.Errorf("Counter metric %s was not stored", metricName)
					} else {
						value, _ := strconv.ParseInt(metricValue, 10, 64)
						if int64(repoCounter) != value {
							t.Errorf("Counter metric %s has value %v, expected %v", metricName, repoCounter, value)
						}
					}
				}
			}
		})
	}
}
