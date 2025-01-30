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
		// Сначала проверим, что metricValue действительно float
		if _, err := strconv.ParseFloat(metricValue, 64); err != nil {
			return fmt.Errorf("invalid gauge value: %w", err)
		}
		// Сохраняем как «сырую» строку
		return ms.Storage.UpdateGaugeRaw(metricName, metricValue)

	case CounterMetric:
		val, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid counter value: %w", err)
		}
		ms.Storage.UpdateCounter(metricName, repository.Counter(val))
		return nil

	default:
		return errors.New("unsupported metric type")
	}
}

func (ms *MetricsService) UpdateMetricsBatch(batch []middleware.MetricsJSON) ([]middleware.MetricsJSON, error) {
	if err := ms.Storage.UpdateMetricsBatch(batch); err != nil {
		return nil, err
	}

	// Соберём «обновлённые» значения, чтобы вернуть клиенту
	var result []middleware.MetricsJSON
	for _, m := range batch {
		mt := strings.ToLower(string(m.MType))
		switch mt {
		case CounterMetric:
			val, ok := ms.Storage.GetCounter(m.ID)
			if ok {
				delta := int64(val)
				result = append(result, middleware.MetricsJSON{
					ID:    m.ID,
					MType: m.MType,
					Delta: &delta,
				})
			}
		case GaugeMetric:
			raw, ok := ms.Storage.GetGaugeRaw(m.ID)
			if ok {
				// Возвращаем то, что лежит в raw (но т.к. response ждет float — парсим)
				fv, _ := strconv.ParseFloat(raw, 64) // не ожидается ошибка
				result = append(result, middleware.MetricsJSON{
					ID:    m.ID,
					MType: m.MType,
					Value: &fv,
				})
			}
		}
	}
	return result, nil
}

func (ms *MetricsService) GetMetricValue(metricType, metricName string) (string, error) {
	mt := strings.ToLower(metricType)

	switch mt {
	case GaugeMetric:
		raw, ok := ms.Storage.GetGaugeRaw(metricName)
		if !ok {
			return "", errors.New("metric not found")
		}
		// Возвращаем «сырой» текст gauge — он ровно такой, как был при update
		return raw, nil

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

	// gauges
	for name, raw := range ms.Storage.GetAllGauges() {
		result[name] = raw
	}

	// counters
	for name, c := range ms.Storage.GetAllCounters() {
		result[name] = strconv.FormatInt(int64(c), 10)
	}

	return result
}
