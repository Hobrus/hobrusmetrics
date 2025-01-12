package service

import (
	"errors"
	"fmt"
	"strconv"

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

	switch metricType {
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
	err := ms.Storage.UpdateMetricsBatch(batch)
	if err != nil {
		return nil, err
	}

	var out []middleware.MetricsJSON
	for _, m := range batch {
		var res middleware.MetricsJSON
		res.ID = m.ID
		res.MType = m.MType

		switch m.MType {
		case CounterMetric:
			v, ok := ms.Storage.GetCounter(m.ID)
			if !ok {
				continue
			}
			delta := int64(v)
			res.Delta = &delta
		case GaugeMetric:
			v, ok := ms.Storage.GetGauge(m.ID)
			if !ok {
				continue
			}
			value := float64(v)
			res.Value = &value
		}

		out = append(out, res)
	}

	return out, nil
}

func (ms *MetricsService) GetMetricValue(metricType, metricName string) (string, error) {
	switch metricType {
	case GaugeMetric:
		value, ok := ms.Storage.GetGauge(metricName)
		if !ok {
			return "", errors.New("metric not found")
		}
		return strconv.FormatFloat(float64(value), 'f', -1, 64), nil

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
	metrics := make(map[string]string)

	for name, g := range ms.Storage.GetAllGauges() {
		metrics[name] = strconv.FormatFloat(float64(g), 'f', -1, 64)
	}
	for name, c := range ms.Storage.GetAllCounters() {
		metrics[name] = fmt.Sprintf("%d", c)
	}
	return metrics
}
