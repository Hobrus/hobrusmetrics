package service

import (
	"errors"
	"fmt"
	"github.com/Hobrus/hobrusmetrics.git/internal/repository"
	"strconv"
)

type MetricsService struct {
	Storage repository.Storage
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
		ms.Storage.UpdateGauge(metricName, repository.Gauge(value))
	case "counter":
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
	case "gauge":
		if value, ok := ms.Storage.GetGauge(metricName); ok {
			return strconv.FormatFloat(float64(value), 'f', -1, 64), nil
		}
	case "counter":
		if value, ok := ms.Storage.GetCounter(metricName); ok {
			return fmt.Sprintf("%d", value), nil
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
