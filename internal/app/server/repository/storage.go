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

type MemStorage struct {
	mu       sync.RWMutex
	gauges   map[string]Gauge
	counters map[string]Counter
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]Gauge),
		counters: make(map[string]Counter),
	}
}

func (m *MemStorage) UpdateGauge(name string, value Gauge) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

func (m *MemStorage) UpdateCounter(name string, value Counter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
}

func (m *MemStorage) GetGauge(name string) (Gauge, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, exists := m.gauges[name]
	return value, exists
}

func (m *MemStorage) GetCounter(name string) (Counter, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, exists := m.counters[name]
	return value, exists
}

func (m *MemStorage) GetAllGauges() map[string]Gauge {
	m.mu.RLock()
	defer m.mu.RUnlock()
	gauges := make(map[string]Gauge)
	for k, v := range m.gauges {
		gauges[k] = v
	}
	return gauges
}

func (m *MemStorage) GetAllCounters() map[string]Counter {
	m.mu.RLock()
	defer m.mu.RUnlock()
	counters := make(map[string]Counter)
	for k, v := range m.counters {
		counters[k] = v
	}
	return counters
}
