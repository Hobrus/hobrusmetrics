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

// UpdateMetric обрабатывает обновление одной метрики по типу и имени.
// Для counter значения накапливаются, для gauge значение перезаписывается.
func (ms *MetricsService) UpdateMetric(metricType, metricName, metricValue string) error {
	if metricName == "" {
		return errors.New("metric name is required")
	}
	mt := strings.ToLower(metricType)

	switch mt {
	case GaugeMetric:
		// Проверим, что metricValue действительно float
		if _, err := strconv.ParseFloat(metricValue, 64); err != nil {
			return fmt.Errorf("invalid gauge value: %w", err)
		}
		// Сохраняем как «сырую» строку (но позже будем возвращать в каноническом формате)
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

// UpdateMetricsBatch обрабатывает пакетное обновление метрик.
// Возвращает уже «актуальные» значения метрик после обновления.
func (ms *MetricsService) UpdateMetricsBatch(batch []middleware.MetricsJSON) ([]middleware.MetricsJSON, error) {
	if err := ms.Storage.UpdateMetricsBatch(batch); err != nil {
		return nil, err
	}

	// Формируем ответ с актуальными значениями.
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
				fv, _ := strconv.ParseFloat(raw, 64) // не ожидается ошибка, т.к. ранее проверяли
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

// GetMetricValue возвращает текущее значение одной метрики (в виде строки).
// Для gauge мы теперь приводим число к каноническому формату через %g, чтобы убрать лишние ".0".
func (ms *MetricsService) GetMetricValue(metricType, metricName string) (string, error) {
	mt := strings.ToLower(metricType)

	switch mt {
	case GaugeMetric:
		raw, ok := ms.Storage.GetGaugeRaw(metricName)
		if !ok {
			return "", errors.New("metric not found")
		}
		val, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return "", fmt.Errorf("invalid stored gauge value: %w", err)
		}
		// Используем %g, чтобы "42.0" форматировать как "42".
		return fmt.Sprintf("%g", val), nil

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

// GetAllMetrics возвращает все метрики в виде "имя -> строковое представление".
// Для gauge аналогично используем канонический формат через %g, чтобы убрать ненужные ".0".
func (ms *MetricsService) GetAllMetrics() map[string]string {
	result := make(map[string]string)

	// Обрабатываем gauges
	for name, raw := range ms.Storage.GetAllGauges() {
		if val, err := strconv.ParseFloat(raw, 64); err == nil {
			result[name] = fmt.Sprintf("%g", val)
		} else {
			// Если вдруг парсинг не удался, вернём как есть — но в норме такое не должно происходить
			result[name] = raw
		}
	}

	// Обрабатываем counters
	for name, c := range ms.Storage.GetAllCounters() {
		result[name] = strconv.FormatInt(int64(c), 10)
	}

	return result
}
