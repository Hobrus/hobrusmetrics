package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

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

		var value string
		switch metric.MType {
		case CounterMetric:
			if metric.Delta == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "delta is required for counter"})
				return
			}
			value = strconv.FormatInt(*metric.Delta, 10)
		case GaugeMetric:
			if metric.Value == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "value is required for gauge"})
				return
			}
			value = strconv.FormatFloat(*metric.Value, 'f', -1, 64)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric type"})
			return
		}

		if err := metricsService.UpdateMetric(string(metric.MType), metric.ID, value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updatedValue, err := metricsService.GetMetricValue(string(metric.MType), metric.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get updated value"})
			return
		}

		response := MetricsJSON{
			ID:    metric.ID,
			MType: metric.MType,
		}
		switch metric.MType {
		case CounterMetric:
			delta, _ := strconv.ParseInt(updatedValue, 10, 64)
			response.Delta = &delta
		case GaugeMetric:
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

		value, err := metricsService.GetMetricValue(string(metric.MType), metric.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "metric not found"})
			return
		}

		response := MetricsJSON{
			ID:    metric.ID,
			MType: metric.MType,
		}
		switch metric.MType {
		case CounterMetric:
			delta, _ := strconv.ParseInt(value, 10, 64)
			response.Delta = &delta
		case GaugeMetric:
			floatVal, _ := strconv.ParseFloat(value, 64)
			response.Value = &floatVal
		}

		c.JSON(http.StatusOK, response)
	}
}
