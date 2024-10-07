package collector

import (
	"math/rand"
	"runtime"
	"sync"
)

type Metrics struct {
	sync.RWMutex
	Data map[string]interface{}
}

func NewMetrics() *Metrics {
	return &Metrics{
		Data: make(map[string]interface{}),
	}
}

func (m *Metrics) Collect(pollCount *int64) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.Lock()
	defer m.Unlock()

	m.Data["Alloc"] = float64(memStats.Alloc)
	m.Data["BuckHashSys"] = float64(memStats.BuckHashSys)
	m.Data["Frees"] = float64(memStats.Frees)
	m.Data["GCCPUFraction"] = memStats.GCCPUFraction
	m.Data["GCSys"] = float64(memStats.GCSys)
	m.Data["HeapAlloc"] = float64(memStats.HeapAlloc)
	m.Data["HeapIdle"] = float64(memStats.HeapIdle)
	m.Data["HeapInuse"] = float64(memStats.HeapInuse)
	m.Data["HeapObjects"] = float64(memStats.HeapObjects)
	m.Data["HeapReleased"] = float64(memStats.HeapReleased)
	m.Data["HeapSys"] = float64(memStats.HeapSys)
	m.Data["LastGC"] = float64(memStats.LastGC)
	m.Data["Lookups"] = float64(memStats.Lookups)
	m.Data["MCacheInuse"] = float64(memStats.MCacheInuse)
	m.Data["MCacheSys"] = float64(memStats.MCacheSys)
	m.Data["MSpanInuse"] = float64(memStats.MSpanInuse)
	m.Data["MSpanSys"] = float64(memStats.MSpanSys)
	m.Data["Mallocs"] = float64(memStats.Mallocs)
	m.Data["NextGC"] = float64(memStats.NextGC)
	m.Data["NumForcedGC"] = float64(memStats.NumForcedGC)
	m.Data["NumGC"] = float64(memStats.NumGC)
	m.Data["OtherSys"] = float64(memStats.OtherSys)
	m.Data["PauseTotalNs"] = float64(memStats.PauseTotalNs)
	m.Data["StackInuse"] = float64(memStats.StackInuse)
	m.Data["StackSys"] = float64(memStats.StackSys)
	m.Data["Sys"] = float64(memStats.Sys)
	m.Data["TotalAlloc"] = float64(memStats.TotalAlloc)

	*pollCount++
	m.Data["PollCount"] = *pollCount

	m.Data["RandomValue"] = rand.Float64()
}

func (m *Metrics) GetAll() map[string]interface{} {
	m.RLock()
	defer m.RUnlock()

	copy := make(map[string]interface{})
	for k, v := range m.Data {
		copy[k] = v
	}
	return copy
}
