package models

type MetricsData struct {
	Gauges   map[string]float64 `json:"gauges"`   // Changed from Gauge type
	Counters map[string]int64   `json:"counters"` // Changed from Counter type
}
