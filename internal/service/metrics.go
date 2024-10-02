package service

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/Hobrus/hobrusmetrics.git/internal/repositories"
)

type MetricsService struct {
	Storage repositories.Storage
}

func (ms *MetricsService) UpdateMetric(metricType, metricName, metricValue string) error {
	if metricName == "" {
		return errors.New("metric name is required")
	}

	switch metricType {
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return fmt.Errorf("invalid gauge value: %w", err)
		}
		ms.Storage.UpdateGauge(metricName, repositories.Gauge(value))
	case "counter":
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid counter value: %w", err)
		}
		ms.Storage.UpdateCounter(metricName, repositories.Counter(value))
	default:
		return errors.New("unsupported metric type")
	}
	return nil
}

func (ms *MetricsService) GetMetricValue(metricType, metricName string) (interface{}, error) {
	switch metricType {
	case "gauge":
		value, exists := ms.Storage.GetGauge(metricName)
		if !exists {
			return nil, errors.New("gauge metric not found")
		}
		return value, nil
	case "counter":
		value, exists := ms.Storage.GetCounter(metricName)
		if !exists {
			return nil, errors.New("counter metric not found")
		}
		return value, nil
	default:
		return nil, errors.New("unsupported metric type")
	}
}

func (ms *MetricsService) GetAllMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	for name, value := range ms.Storage.GetAllGauges() {
		metrics[fmt.Sprintf("gauge_%s", name)] = value
	}

	for name, value := range ms.Storage.GetAllCounters() {
		metrics[fmt.Sprintf("counter_%s", name)] = value
	}

	return metrics
}
