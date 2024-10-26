package service

import (
	"errors"
	"fmt"
	"strconv"

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
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return fmt.Errorf("invalid gauge value: %w", err)
		}
		ms.Storage.UpdateGauge(metricName, repository.Gauge(value))
	case CounterMetric:
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid counter value: %w", err)
		}
		ms.Storage.UpdateCounter(metricName, repository.Counter(value))
	default:
		return errors.New("unsupported metric type")
	}
	return nil
}

func (ms *MetricsService) GetMetricValue(metricType, metricName string) (string, error) {
	switch metricType {
	case GaugeMetric:
		if value, ok := ms.Storage.GetGauge(metricName); ok {
			return strconv.FormatFloat(float64(value), 'f', -1, 64), nil
		}
	case CounterMetric:
		if value, ok := ms.Storage.GetCounter(metricName); ok {
			return strconv.FormatInt(int64(value), 10), nil
		}
	default:
		return "", errors.New("unsupported metric type")
	}
	return "", errors.New("metric not found")
}

func (ms *MetricsService) GetAllMetrics() map[string]string {
	metrics := make(map[string]string)

	for name, value := range ms.Storage.GetAllGauges() {
		metrics[name] = strconv.FormatFloat(float64(value), 'f', -1, 64)
	}

	for name, value := range ms.Storage.GetAllCounters() {
		metrics[name] = fmt.Sprintf("%d", value)
	}

	return metrics
}
