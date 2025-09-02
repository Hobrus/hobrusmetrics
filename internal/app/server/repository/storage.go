package repository

import (
	"strconv"
	"strings"
	"sync"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/middleware"
)

type Counter int64

// Storage — интерфейс к любому типу хранилища (memory, file-backed, postgres).
// Реализации должны быть потокобезопасными.
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

// MetricStorage — вспомогательное потокобезопасное хранилище значений int64 по ключу.
type MetricStorage[T ~int64] struct {
	mu     sync.RWMutex
	values map[string]T
}

// NewMetricStorage создаёт пустое хранилище для значений типа T.
func NewMetricStorage[T ~int64]() *MetricStorage[T] {
	return &MetricStorage[T]{
		values: make(map[string]T),
	}
}

// Update добавляет значение к существующему.
func (m *MetricStorage[T]) Update(name string, value T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Для counter складываем, т.к. по условию counter накапливается
	m.values[name] += value
}

// Get возвращает текущее значение по имени.
func (m *MetricStorage[T]) Get(name string) (T, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.values[name]
	return val, ok
}

// GetAll возвращает копию всех значений.
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

// NewMemStorage создаёт хранилище метрик в оперативной памяти.
func NewMemStorage() *MemStorage {
	return &MemStorage{
		counters: NewMetricStorage[Counter](),
	}
}

// UpdateGaugeRaw валидирует и сохраняет gauge как строку.
func (m *MemStorage) UpdateGaugeRaw(name, rawValue string) error {
	_, err := parseGaugeOrFail(rawValue)
	if err != nil {
		return err
	}
	m.gauges.Store(name, rawValue)
	return nil
}

// GetGaugeRaw возвращает строковое значение gauge и признак наличия.
func (m *MemStorage) GetGaugeRaw(name string) (string, bool) {
	v, ok := m.gauges.Load(name)
	if !ok {
		return "", false
	}
	return v.(string), true
}

// UpdateCounter накапливает значение counter по ключу.
func (m *MemStorage) UpdateCounter(name string, value Counter) {
	m.counters.Update(name, value)
}

// GetCounter возвращает текущее значение counter и признак наличия.
func (m *MemStorage) GetCounter(name string) (Counter, bool) {
	return m.counters.Get(name)
}

// GetAllGauges возвращает копию всех gauge.
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

// GetAllCounters возвращает копию всех counter.
func (m *MemStorage) GetAllCounters() map[string]Counter {
	return m.counters.GetAll()
}

// UpdateMetricsBatch применяет пакет обновлений к памяти.
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

// Shutdown для памяти ничего не делает.
func (m *MemStorage) Shutdown() error {
	return nil
}

// parseGaugeOrFail парсит строковое представление gauge в float64.
func parseGaugeOrFail(raw string) (float64, error) {
	return strconv.ParseFloat(raw, 64)
}

// floatToString форматирует число в строку без лишних нулей.
func floatToString(val float64) string {
	return strconv.FormatFloat(val, 'f', -1, 64)
}
