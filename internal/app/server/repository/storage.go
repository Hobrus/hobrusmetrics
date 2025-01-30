package repository

import (
	"strconv"
	"strings"
	"sync"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/middleware"
)

type Counter int64

// Storage — интерфейс к любому типу хранилища (memory, file-backed, postgres)
type Storage interface {
	UpdateGaugeRaw(name, rawValue string) error
	GetGaugeRaw(name string) (string, bool)
	UpdateCounter(name string, value Counter)
	GetCounter(name string) (Counter, bool)
	GetAllGauges() map[string]string
	GetAllCounters() map[string]Counter
	UpdateMetricsBatch(batch []middleware.MetricsJSON) error
	Shutdown() error
}

// Вспомогательный тип для хранения int64 в памяти
type MetricStorage[T ~int64] struct {
	mu     sync.RWMutex
	values map[string]T
}

func NewMetricStorage[T ~int64]() *MetricStorage[T] {
	return &MetricStorage[T]{
		values: make(map[string]T),
	}
}

func (m *MetricStorage[T]) Update(name string, value T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Для counter складываем, т.к. по условию counter накапливается
	m.values[name] += value
}

func (m *MetricStorage[T]) Get(name string) (T, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.values[name]
	return val, ok
}

func (m *MetricStorage[T]) GetAll() map[string]T {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cpy := make(map[string]T, len(m.values))
	for k, v := range m.values {
		cpy[k] = v
	}
	return cpy
}

// MemStorage — в памяти храним:
// 1) gauges как map[string]string
// 2) counters (int64) в MetricStorage
type MemStorage struct {
	gauges   sync.Map // ключ string -> значение string
	counters *MetricStorage[Counter]
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		counters: NewMetricStorage[Counter](),
	}
}

func (m *MemStorage) UpdateGaugeRaw(name, rawValue string) error {
	_, err := parseGaugeOrFail(rawValue)
	if err != nil {
		return err
	}
	m.gauges.Store(name, rawValue)
	return nil
}

func (m *MemStorage) GetGaugeRaw(name string) (string, bool) {
	v, ok := m.gauges.Load(name)
	if !ok {
		return "", false
	}
	return v.(string), true
}

func (m *MemStorage) UpdateCounter(name string, value Counter) {
	m.counters.Update(name, value)
}

func (m *MemStorage) GetCounter(name string) (Counter, bool) {
	return m.counters.Get(name)
}

func (m *MemStorage) GetAllGauges() map[string]string {
	result := make(map[string]string)
	m.gauges.Range(func(key, value any) bool {
		k := key.(string)
		v := value.(string)
		result[k] = v
		return true
	})
	return result
}

func (m *MemStorage) GetAllCounters() map[string]Counter {
	return m.counters.GetAll()
}

func (m *MemStorage) UpdateMetricsBatch(batch []middleware.MetricsJSON) error {
	for _, metric := range batch {
		mType := strings.ToLower(string(metric.MType))
		switch mType {
		case "counter":
			if metric.Delta != nil {
				m.UpdateCounter(metric.ID, Counter(*metric.Delta))
			}
		case "gauge":
			if metric.Value != nil {
				raw := floatToString(*metric.Value)
				_ = m.UpdateGaugeRaw(metric.ID, raw)
			}
		default:
		}
	}
	return nil
}

func (m *MemStorage) Shutdown() error {
	return nil
}

func parseGaugeOrFail(raw string) (float64, error) {
	return strconv.ParseFloat(raw, 64)
}

func floatToString(val float64) string {
	return strconv.FormatFloat(val, 'g', 17, 64)
}
