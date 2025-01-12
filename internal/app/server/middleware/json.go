package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type MetricType string

const (
	CounterMetric MetricType = "counter"
	GaugeMetric   MetricType = "gauge"
)

type MetricsJSON struct {
	ID    string     `json:"id"`
	MType MetricType `json:"type"`
	Delta *int64     `json:"delta,omitempty"`
	Value *float64   `json:"value,omitempty"`
}

func formatGauge(value float64) string {
	s := fmt.Sprintf("%.15f", value)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

func (m MetricsJSON) MarshalJSON() ([]byte, error) {
	// Если это gauge, то хотим вывести поле "value" в "красивом" виде
	if strings.ToLower(string(m.MType)) == string(GaugeMetric) && m.Value != nil {
		valStr := formatGauge(*m.Value)
		// Сформируем JSON руками
		return []byte(fmt.Sprintf(`{"id":"%s","type":"%s","value":%s}`,
			m.ID, m.MType, valStr,
		)), nil
	}

	if strings.ToLower(string(m.MType)) == string(CounterMetric) && m.Delta != nil {
		return []byte(fmt.Sprintf(`{"id":"%s","type":"%s","delta":%d}`,
			m.ID, m.MType, *m.Delta,
		)), nil
	}

	type alias MetricsJSON
	return json.Marshal(alias(m))
}

func JSONUpdateMiddleware(metricsService interface {
	UpdateMetric(metricType, metricName, metricValue string) error
	GetMetricValue(metricType, metricName string) (string, error)
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		var metric MetricsJSON

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		if err := json.Unmarshal(body, &metric); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format"})
			return
		}

		if metric.ID == "" || metric.MType == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id and type are required"})
			return
		}

		mt := strings.ToLower(string(metric.MType))
		switch mt {
		case string(CounterMetric):
			if metric.Delta == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "delta is required for counter"})
				return
			}
			if err := metricsService.UpdateMetric(mt, metric.ID, strconv.FormatInt(*metric.Delta, 10)); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

		case string(GaugeMetric):
			if metric.Value == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "value is required for gauge"})
				return
			}
			if err := metricsService.UpdateMetric(mt, metric.ID, formatGauge(*metric.Value)); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric type"})
			return
		}

		updatedValue, err := metricsService.GetMetricValue(mt, metric.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get updated value"})
			return
		}

		response := MetricsJSON{
			ID:    metric.ID,
			MType: metric.MType,
		}
		if mt == string(CounterMetric) {
			d, _ := strconv.ParseInt(updatedValue, 10, 64)
			response.Delta = &d
		} else {
			f, _ := strconv.ParseFloat(updatedValue, 64)
			response.Value = &f
		}

		c.JSON(http.StatusOK, response)
	}
}

func JSONValueMiddleware(metricsService interface {
	GetMetricValue(metricType, metricName string) (string, error)
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		var metric MetricsJSON

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		if err := json.Unmarshal(body, &metric); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format"})
			return
		}
		if metric.ID == "" || metric.MType == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id and type are required"})
			return
		}

		mt := strings.ToLower(string(metric.MType))
		val, err := metricsService.GetMetricValue(mt, metric.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "metric not found"})
			return
		}

		response := MetricsJSON{
			ID:    metric.ID,
			MType: metric.MType,
		}
		if mt == string(CounterMetric) {
			d, _ := strconv.ParseInt(val, 10, 64)
			response.Delta = &d
		} else {
			f, _ := strconv.ParseFloat(val, 64)
			response.Value = &f
		}

		c.JSON(http.StatusOK, response)
	}
}
