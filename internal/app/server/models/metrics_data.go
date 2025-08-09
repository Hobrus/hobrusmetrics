package models

// MetricsData описывает снимок всех метрик в удобном для сериализации виде.
type MetricsData struct {
	Gauges   map[string]string `json:"gauges"`   // gaugeName -> "123.45"
	Counters map[string]int64  `json:"counters"` // counterName -> 10
}
