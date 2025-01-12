package middleware

import (
	"bytes"
	"encoding/json"
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

func (m MetricsJSON) MarshalJSON() ([]byte, error) {
	type alias MetricsJSON
	a := alias(m)

	if a.MType == GaugeMetric && a.Value != nil {
		valStr := strconv.FormatFloat(*a.Value, 'f', 10, 64)
		return []byte(`{"id":"` + a.ID + `","type":"` + string(a.MType) + `","value":` + valStr + `}`), nil
	}

	if a.MType == CounterMetric && a.Delta != nil {
		deltaStr := strconv.FormatInt(*a.Delta, 10)
		return []byte(`{"id":"` + a.ID + `","type":"` + string(a.MType) + `","delta":` + deltaStr + `}`), nil
	}

	return json.Marshal(a)
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
		var value string
		switch mt {
		case string(CounterMetric):
			if metric.Delta == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "delta is required for counter"})
				return
			}
			value = strconv.FormatInt(*metric.Delta, 10)
		case string(GaugeMetric):
			if metric.Value == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "value is required for gauge"})
				return
			}
			value = strconv.FormatFloat(*metric.Value, 'f', 10, 64) // можно 'f', 10, 64
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric type"})
			return
		}

		if err := metricsService.UpdateMetric(mt, metric.ID, value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		switch mt {
		case string(CounterMetric):
			delta, _ := strconv.ParseInt(updatedValue, 10, 64)
			response.Delta = &delta
		case string(GaugeMetric):
			fv, _ := strconv.ParseFloat(updatedValue, 64)
			response.Value = &fv
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
		value, err := metricsService.GetMetricValue(mt, metric.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "metric not found"})
			return
		}

		response := MetricsJSON{
			ID:    metric.ID,
			MType: metric.MType,
		}
		switch mt {
		case string(CounterMetric):
			delta, _ := strconv.ParseInt(value, 10, 64)
			response.Delta = &delta
		case string(GaugeMetric):
			fv, _ := strconv.ParseFloat(value, 64)
			response.Value = &fv
		}

		c.JSON(http.StatusOK, response)
	}
}
