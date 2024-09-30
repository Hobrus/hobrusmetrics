package storage

import (
	"sync"

	"github.com/Hobrus/hobrusmetrics.git/internal/repositories"
)

type MemStorage struct {
	mu       sync.Mutex
	gauges   map[string]repositories.Gauge
	counters map[string]repositories.Counter
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]repositories.Gauge),
		counters: make(map[string]repositories.Counter),
	}
}

func (m *MemStorage) UpdateGauge(name string, value repositories.Gauge) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

func (m *MemStorage) UpdateCounter(name string, value repositories.Counter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
}
