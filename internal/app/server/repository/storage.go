package repository

import (
	"sync"
)

type Gauge float64
type Counter int64

type Storage interface {
	UpdateGauge(name string, value Gauge)
	UpdateCounter(name string, value Counter)
	GetGauge(name string) (Gauge, bool)
	GetCounter(name string) (Counter, bool)
	GetAllGauges() map[string]Gauge
	GetAllCounters() map[string]Counter
}

type Numeric interface {
	~int64 | ~float64
}

type MetricStorage[T Numeric] struct {
	mu     sync.RWMutex
	values map[string]T
}

func NewMetricStorage[T Numeric]() *MetricStorage[T] {
	return &MetricStorage[T]{
		values: make(map[string]T),
	}
}

func (m *MetricStorage[T]) Update(name string, value T, accumulate bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if accumulate {
		m.values[name] += value
	} else {
		m.values[name] = value
	}
}

func (m *MetricStorage[T]) Get(name string) (T, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, exists := m.values[name]
	return value, exists
}

func (m *MetricStorage[T]) GetAll() map[string]T {
	m.mu.RLock()
	defer m.mu.RUnlock()
	valuesCopy := make(map[string]T, len(m.values))
	for k, v := range m.values {
		valuesCopy[k] = v
	}
	return valuesCopy
}

type MemStorage struct {
	gauges   *MetricStorage[Gauge]
	counters *MetricStorage[Counter]
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   NewMetricStorage[Gauge](),
		counters: NewMetricStorage[Counter](),
	}
}

func (m *MemStorage) UpdateGauge(name string, value Gauge) {
	m.gauges.Update(name, value, false)
}

func (m *MemStorage) UpdateCounter(name string, value Counter) {
	m.counters.Update(name, value, true)
}

func (m *MemStorage) GetGauge(name string) (Gauge, bool) {
	return m.gauges.Get(name)
}

func (m *MemStorage) GetCounter(name string) (Counter, bool) {
	return m.counters.Get(name)
}

func (m *MemStorage) GetAllGauges() map[string]Gauge {
	return m.gauges.GetAll()
}

func (m *MemStorage) GetAllCounters() map[string]Counter {
	return m.counters.GetAll()
}
