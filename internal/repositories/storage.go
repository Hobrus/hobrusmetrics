package repositories

type Gauge float64
type Counter int64

type Storage interface {
	UpdateGauge(name string, value Gauge)
	UpdateCounter(name string, value Counter)
}
