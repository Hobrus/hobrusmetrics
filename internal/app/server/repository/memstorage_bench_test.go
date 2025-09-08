package repository

import (
	"strconv"
	"testing"
)

func BenchmarkMemStorage_UpdateCounter(b *testing.B) {
	storage := NewMemStorage()
	const numKeys = 1024
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "counter_" + strconv.Itoa(i%numKeys)
		storage.UpdateCounter(key, Counter(1))
	}
}

func BenchmarkMemStorage_UpdateGaugeRaw(b *testing.B) {
	storage := NewMemStorage()
	const numKeys = 1024
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "gauge_" + strconv.Itoa(i%numKeys)
		// Небольшая вариативность значений без роста памяти из-за количества ключей
		val := float64(i%1000) / 10.0
		raw := strconv.FormatFloat(val, 'f', -1, 64)
		_ = storage.UpdateGaugeRaw(key, raw)
	}
}

func BenchmarkMemStorage_GetAll(b *testing.B) {
	storage := NewMemStorage()
	// Заполним тестовыми данными
	for i := 0; i < 2048; i++ {
		storage.UpdateCounter("counter_"+strconv.Itoa(i), Counter(i))
		_ = storage.UpdateGaugeRaw("gauge_"+strconv.Itoa(i), strconv.FormatFloat(float64(i), 'f', -1, 64))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.GetAllGauges()
		_ = storage.GetAllCounters()
	}
}
