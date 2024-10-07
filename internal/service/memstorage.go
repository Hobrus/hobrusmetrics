package service

import (
	"github.com/Hobrus/hobrusmetrics.git/internal/repository"
	"sync"
)

type MemStorage struct {
	mu       sync.RWMutex
	gauges   map[string]repository.Gauge
	counters map[string]repository.Counter
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]repository.Gauge),
		counters: make(map[string]repository.Counter),
	}
}

func (m *MemStorage) UpdateGauge(name string, value repository.Gauge) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

func (m *MemStorage) UpdateCounter(name string, value repository.Counter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
}

func (m *MemStorage) GetGauge(name string) (repository.Gauge, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, exists := m.gauges[name]
	return value, exists
}

func (m *MemStorage) GetCounter(name string) (repository.Counter, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, exists := m.counters[name]
	return value, exists
}

func (m *MemStorage) GetAllGauges() map[string]repository.Gauge {
	m.mu.RLock()
	defer m.mu.RUnlock()
	gauges := make(map[string]repository.Gauge)
	for k, v := range m.gauges {
		gauges[k] = v
	}
	return gauges
}

func (m *MemStorage) GetAllCounters() map[string]repository.Counter {
	m.mu.RLock()
	defer m.mu.RUnlock()
	counters := make(map[string]repository.Counter)
	for k, v := range m.counters {
		counters[k] = v
	}
	return counters
}
