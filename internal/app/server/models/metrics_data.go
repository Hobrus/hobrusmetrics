package models

type MetricsData struct {
	Gauges   map[string]string `json:"gauges"`   // gaugeName -> "123.45"
	Counters map[string]int64  `json:"counters"` // counterName -> 10
}
