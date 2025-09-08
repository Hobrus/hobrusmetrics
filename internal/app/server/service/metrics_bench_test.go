package service

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/middleware"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/repository"
)

func setupService() *MetricsService {
	return &MetricsService{Storage: repository.NewMemStorage()}
}

func BenchmarkMetricsService_UpdateMetric_Counter(b *testing.B) {
	ms := setupService()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := "PollCount_" + strconv.Itoa(i%1024)
		_ = ms.UpdateMetric("counter", name, "1")
	}
}

func BenchmarkMetricsService_UpdateMetric_Gauge(b *testing.B) {
	ms := setupService()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := "Alloc_" + strconv.Itoa(i%1024)
		val := strconv.FormatFloat(rand.Float64()*1000, 'f', -1, 64)
		_ = ms.UpdateMetric("gauge", name, val)
	}
}

func BenchmarkMetricsService_UpdateMetricsBatch(b *testing.B) {
	ms := setupService()
	// подготовим батч фиксированного размера
	const size = 512
	batch := make([]middleware.MetricsJSON, 0, size)
	for i := 0; i < size; i++ {
		d := int64(i + 1)
		v := rand.Float64() * 1000
		if i%2 == 0 {
			batch = append(batch, middleware.MetricsJSON{ID: "c_" + strconv.Itoa(i), MType: middleware.CounterMetric, Delta: &d})
		} else {
			batch = append(batch, middleware.MetricsJSON{ID: "g_" + strconv.Itoa(i), MType: middleware.GaugeMetric, Value: &v})
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ms.UpdateMetricsBatch(batch)
	}
}
