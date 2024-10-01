package service

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

func (m *MemStorage) GetGauge(name string) (repositories.Gauge, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, exists := m.gauges[name]
	return value, exists
}

func (m *MemStorage) GetCounter(name string) (repositories.Counter, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	value, exists := m.counters[name]
	return value, exists
}
