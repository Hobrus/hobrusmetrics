package service

import (
	"errors"
	"fmt"
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
		default:
		}
	}
	return result, nil
}

func (ms *MetricsService) GetMetricValue(metricType, metricName string) (string, error) {
	mt := strings.ToLower(metricType)

	switch mt {
	case GaugeMetric:
		value, ok := ms.Storage.GetGauge(metricName)
		if !ok {
			return "", errors.New("metric not found")
		}
		return strconv.FormatFloat(float64(value), 'G', -1, 64), nil

	case CounterMetric:
		value, ok := ms.Storage.GetCounter(metricName)
		if !ok {
			return "", errors.New("metric not found")
		}
		return strconv.FormatInt(int64(value), 10), nil

	default:
		return "", errors.New("unsupported metric type")
	}
}

func (ms *MetricsService) GetAllMetrics() map[string]string {
	result := make(map[string]string)

	for name, g := range ms.Storage.GetAllGauges() {
		result[name] = strconv.FormatFloat(float64(g), 'G', -1, 64)
	}

	for name, c := range ms.Storage.GetAllCounters() {
		result[name] = fmt.Sprintf("%d", c)
	}

	return result
}
