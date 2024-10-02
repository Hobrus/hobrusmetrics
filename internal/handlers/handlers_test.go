package handlers

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Hobrus/hobrusmetrics.git/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestRouter(t *testing.T) {
	storage := service.NewMemStorage()
	metricsService := &service.MetricsService{Storage: storage}
	router := NewRouter(metricsService)

	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
		expectedBody   string
		checkResponse  func(*testing.T, *fasthttp.RequestCtx)
	}{
		{
			name:           "Valid gauge metric update",
			method:         "POST",
			url:            "/update/gauge/Alloc/123.45",
			expectedStatus: fasthttp.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "Valid counter metric update",
			method:         "POST",
			url:            "/update/counter/PollCount/10",
			expectedStatus: fasthttp.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "Get existing gauge metric",
			method:         "GET",
			url:            "/value/gauge/Alloc",
			expectedStatus: fasthttp.StatusOK,
			expectedBody:   "123.45",
		},
		{
			name:           "Get existing counter metric",
			method:         "GET",
			url:            "/value/counter/PollCount",
			expectedStatus: fasthttp.StatusOK,
			expectedBody:   "10",
		},
		{
			name:           "Get non-existent metric",
			method:         "GET",
			url:            "/value/gauge/NonExistent",
			expectedStatus: fasthttp.StatusNotFound,
			expectedBody:   "Metric not found",
		},
		{
			name:           "Invalid metric type",
			method:         "POST",
			url:            "/update/invalid/Alloc/123.45",
			expectedStatus: fasthttp.StatusBadRequest,
			expectedBody:   "unsupported metric type",
		},
		{
			name:           "Invalid metric value",
			method:         "POST",
			url:            "/update/gauge/Alloc/abc",
			expectedStatus: fasthttp.StatusBadRequest,
			expectedBody:   "invalid gauge value",
		},
		{
			name:           "Metric name missing",
			method:         "POST",
			url:            "/update/gauge//123.45",
			expectedStatus: fasthttp.StatusNotFound,
			expectedBody:   "Not found",
		},
		{
			name:           "Invalid URL path",
			method:         "POST",
			url:            "/invalidpath/gauge/Alloc/123.45",
			expectedStatus: fasthttp.StatusNotFound,
			expectedBody:   "Not found",
		},
		{
			name:           "Get all metrics",
			method:         "GET",
			url:            "/",
			expectedStatus: fasthttp.StatusOK,
			checkResponse: func(t *testing.T, ctx *fasthttp.RequestCtx) {
				assert.Equal(t, "text/html", string(ctx.Response.Header.ContentType()))
				body := string(ctx.Response.Body())
				assert.Contains(t, body, "Alloc")
				assert.Contains(t, body, "123.45")
				assert.Contains(t, body, "PollCount")
				assert.Contains(t, body, "10")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			ctx.Request.Header.SetMethod(tc.method)
			ctx.Request.SetRequestURI(tc.url)

			router(ctx)

			assert.Equal(t, tc.expectedStatus, ctx.Response.StatusCode(), "Unexpected status code")
			if tc.expectedBody != "" {
				assert.Contains(t, string(ctx.Response.Body()), tc.expectedBody, "Unexpected response body")
			}
			if tc.checkResponse != nil {
				tc.checkResponse(t, ctx)
			}
		})
	}
}

func TestHandleRoot(t *testing.T) {
	storage := service.NewMemStorage()
	metricsService := &service.MetricsService{Storage: storage}
	router := NewRouter(metricsService)

	// Add some test metrics
	require.NoError(t, metricsService.UpdateMetric("gauge", "TestGauge", "123.45"))
	require.NoError(t, metricsService.UpdateMetric("counter", "TestCounter", "10"))

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/")

	router(ctx)

	assert.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode())
	assert.Equal(t, "text/html", string(ctx.Response.Header.ContentType()))
	body := string(ctx.Response.Body())
	assert.Contains(t, body, "TestGauge")
	assert.Contains(t, body, "123.45")
	assert.Contains(t, body, "TestCounter")
	assert.Contains(t, body, "10")
}

func TestJSONHandlers(t *testing.T) {
	storage := service.NewMemStorage()
	metricsService := &service.MetricsService{Storage: storage}
	router := NewRouter(metricsService)

	tests := []struct {
		name           string
		method         string
		url            string
		body           string
		expectedStatus int
		checkResponse  func(*testing.T, *fasthttp.RequestCtx)
	}{
		{
			name:           "Update gauge metric via JSON",
			method:         "POST",
			url:            "/update/",
			body:           `{"id":"TestGauge","type":"gauge","value":123.45}`,
			expectedStatus: fasthttp.StatusOK,
		},
		{
			name:           "Update counter metric via JSON",
			method:         "POST",
			url:            "/update/",
			body:           `{"id":"TestCounter","type":"counter","delta":10}`,
			expectedStatus: fasthttp.StatusOK,
		},
		{
			name:           "Get gauge metric via JSON",
			method:         "POST",
			url:            "/value/",
			body:           `{"id":"TestGauge","type":"gauge"}`,
			expectedStatus: fasthttp.StatusOK,
			checkResponse: func(t *testing.T, ctx *fasthttp.RequestCtx) {
				var response MetricResponse
				err := json.Unmarshal(ctx.Response.Body(), &response)
				require.NoError(t, err)
				assert.Equal(t, "TestGauge", response.ID)
				assert.Equal(t, "gauge", response.MType)
				require.NotNil(t, response.Value)
				assert.Equal(t, 123.45, *response.Value)
			},
		},
		{
			name:           "Get counter metric via JSON",
			method:         "POST",
			url:            "/value/",
			body:           `{"id":"TestCounter","type":"counter"}`,
			expectedStatus: fasthttp.StatusOK,
			checkResponse: func(t *testing.T, ctx *fasthttp.RequestCtx) {
				var response MetricResponse
				err := json.Unmarshal(ctx.Response.Body(), &response)
				require.NoError(t, err)
				assert.Equal(t, "TestCounter", response.ID)
				assert.Equal(t, "counter", response.MType)
				require.NotNil(t, response.Delta)
				assert.Equal(t, int64(10), *response.Delta)
			},
		},
		{
			name:           "Update gauge metric with invalid JSON",
			method:         "POST",
			url:            "/update/",
			body:           `{"id":"TestGauge","type":"gauge","value":"invalid"}`,
			expectedStatus: fasthttp.StatusBadRequest,
		},
		{
			name:           "Update counter metric with missing delta",
			method:         "POST",
			url:            "/update/",
			body:           `{"id":"TestCounter","type":"counter"}`,
			expectedStatus: fasthttp.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			ctx.Request.Header.SetMethod(tc.method)
			ctx.Request.SetRequestURI(tc.url)
			ctx.Request.Header.SetContentType("application/json")
			ctx.Request.SetBody([]byte(tc.body))

			router(ctx)

			assert.Equal(t, tc.expectedStatus, ctx.Response.StatusCode(), "Unexpected status code")
			if tc.checkResponse != nil {
				tc.checkResponse(t, ctx)
			}
		})
	}
}

func BenchmarkRouter(b *testing.B) {
	storage := service.NewMemStorage()
	metricsService := &service.MetricsService{Storage: storage}
	router := NewRouter(metricsService)

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("POST")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Request.SetRequestURI(fmt.Sprintf("/update/gauge/TestGauge/%d", i))
		router(ctx)
		ctx.Response.Reset()
	}
}
