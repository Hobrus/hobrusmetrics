package service

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/middleware"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/repository"
)

const (
	GaugeMetric   = "gauge"
	CounterMetric = "counter"
)

type MetricsService struct {
	Storage repository.Storage
}

func roundFloat(val float64) float64 {
	const digits = 15
	factor := math.Pow10(digits)
	return math.Round(val*factor) / factor
}

func formatGauge(value float64) string {
	value = roundFloat(value)

	s := fmt.Sprintf("%.15f", value)

	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

func (ms *MetricsService) UpdateMetric(metricType, metricName, metricValue string) error {
	if metricName == "" {
		return errors.New("metric name is required")
	}
	mt := strings.ToLower(metricType)

	switch mt {
	case GaugeMetric:
		val, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return fmt.Errorf("invalid gauge value: %w", err)
		}
		ms.Storage.UpdateGauge(metricName, repository.Gauge(val))

	case CounterMetric:
		val, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid counter value: %w", err)
		}
		ms.Storage.UpdateCounter(metricName, repository.Counter(val))

	default:
		return errors.New("unsupported metric type")
	}
	return nil
}

func (ms *MetricsService) GetMetricValue(metricType, metricName string) (string, error) {
	mt := strings.ToLower(metricType)
	switch mt {
	case GaugeMetric:
		g, ok := ms.Storage.GetGauge(metricName)
		if !ok {
			return "", errors.New("metric not found")
		}
		return formatGauge(float64(g)), nil

	case CounterMetric:
		c, ok := ms.Storage.GetCounter(metricName)
		if !ok {
			return "", errors.New("metric not found")
		}
		return strconv.FormatInt(int64(c), 10), nil

	default:
		return "", errors.New("unsupported metric type")
	}
}

func (ms *MetricsService) GetAllMetrics() map[string]string {
	result := make(map[string]string)
	for name, g := range ms.Storage.GetAllGauges() {
		result[name] = formatGauge(float64(g))
	}
	for name, c := range ms.Storage.GetAllCounters() {
		result[name] = strconv.FormatInt(int64(c), 10)
	}
	return result
}

func (ms *MetricsService) UpdateMetricsBatch(batch []middleware.MetricsJSON) ([]middleware.MetricsJSON, error) {
	if err := ms.Storage.UpdateMetricsBatch(batch); err != nil {
		return nil, err
	}
	var result []middleware.MetricsJSON
	for _, m := range batch {
		mt := strings.ToLower(string(m.MType))
		switch mt {
		case CounterMetric:
			val, ok := ms.Storage.GetCounter(m.ID)
			if !ok {
				continue
			}
			delta := int64(val)
			result = append(result, middleware.MetricsJSON{
				ID:    m.ID,
				MType: m.MType,
				Delta: &delta,
			})
		case GaugeMetric:
			val, ok := ms.Storage.GetGauge(m.ID)
			if !ok {
				continue
			}
			fv := float64(val)
			result = append(result, middleware.MetricsJSON{
				ID:    m.ID,
				MType: m.MType,
				Value: &fv,
			})
		}
	}
	return result, nil
}
