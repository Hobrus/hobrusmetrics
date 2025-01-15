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

type MetricService interface {
	UpdateMetric(metricType, metricName, metricValue string) error
	GetMetricValue(metricType, metricName string) (string, error)
}

func JSONUpdateMiddleware(metricsService MetricService) gin.HandlerFunc {
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
			value = strconv.FormatFloat(*metric.Value, 'g', -1, 64)

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
			MType: MetricType(mt),
		}
		switch mt {
		case string(CounterMetric):
			delta, _ := strconv.ParseInt(updatedValue, 10, 64)
			response.Delta = &delta
		case string(GaugeMetric):
			val, _ := strconv.ParseFloat(updatedValue, 64)
			response.Value = &val
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
			MType: MetricType(mt),
		}
		switch mt {
		case string(CounterMetric):
			delta, _ := strconv.ParseInt(value, 10, 64)
			response.Delta = &delta
		case string(GaugeMetric):
			val, _ := strconv.ParseFloat(value, 64)
			response.Value = &val
		}

		c.JSON(http.StatusOK, response)
	}
}
