package storage

import "sync"

type Gauge float64
type Counter int64

type MemStorage struct {
	mu       sync.Mutex
	Gauges   map[string]Gauge
	Counters map[string]Counter
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		Gauges:   make(map[string]Gauge),
		Counters: make(map[string]Counter),
	}
}

type Storage interface {
	UpdateGauge(name string, value Gauge)
	UpdateCounter(name string, value Counter)
}

func (m *MemStorage) UpdateGauge(name string, value Gauge) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Gauges[name] = value
}

// UpdateCounter обновляет метрику типа Counter
func (m *MemStorage) UpdateCounter(name string, value Counter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Counters[name] += value
}
