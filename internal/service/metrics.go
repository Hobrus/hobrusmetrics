package service

import (
	"errors"
	"fmt"
	"github.com/Hobrus/hobrusmetrics.git/internal/repositories"
	"strconv"
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
