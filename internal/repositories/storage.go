package repositories

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
