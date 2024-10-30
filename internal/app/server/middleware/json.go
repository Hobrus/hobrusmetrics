package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// MetricsJSON represents the structure for JSON metrics
type MetricsJSON struct {
	ID    string   `json:"id"`              // metric name
	MType string   `json:"type"`            // gauge or counter
	Delta *int64   `json:"delta,omitempty"` // counter value
	Value *float64 `json:"value,omitempty"` // gauge value
}

// JSONUpdateMiddleware handles metric updates via JSON
func JSONUpdateMiddleware(metricsService interface {
	UpdateMetric(metricType, metricName, metricValue string) error
	GetMetricValue(metricType, metricName string) (string, error)
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		var metric MetricsJSON

		// Read and restore the request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		// Parse JSON
		if err := json.Unmarshal(body, &metric); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format"})
			return
		}

		// Validate required fields
		if metric.ID == "" || metric.MType == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id and type are required"})
			return
		}

		var value string
		switch metric.MType {
		case "counter":
			if metric.Delta == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "delta is required for counter"})
				return
			}
			value = strconv.FormatInt(*metric.Delta, 10)
		case "gauge":
			if metric.Value == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "value is required for gauge"})
				return
			}
			value = strconv.FormatFloat(*metric.Value, 'f', -1, 64)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric type"})
			return
		}

		if err := metricsService.UpdateMetric(metric.MType, metric.ID, value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get updated value
		updatedValue, err := metricsService.GetMetricValue(metric.MType, metric.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get updated value"})
			return
		}

		response := MetricsJSON{
			ID:    metric.ID,
			MType: metric.MType,
		}

		switch metric.MType {
		case "counter":
			delta, _ := strconv.ParseInt(updatedValue, 10, 64)
			response.Delta = &delta
		case "gauge":
			value, _ := strconv.ParseFloat(updatedValue, 64)
			response.Value = &value
		}

		c.JSON(http.StatusOK, response)
	}
}

// JSONValueMiddleware handles retrieving metric values via JSON
func JSONValueMiddleware(metricsService interface {
	GetMetricValue(metricType, metricName string) (string, error)
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		var metric MetricsJSON

		// Read and restore the request body
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		// Parse JSON
		if err := json.Unmarshal(body, &metric); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format"})
			return
		}

		if metric.ID == "" || metric.MType == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id and type are required"})
			return
		}

		value, err := metricsService.GetMetricValue(metric.MType, metric.ID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "metric not found"})
			return
		}

		response := MetricsJSON{
			ID:    metric.ID,
			MType: metric.MType,
		}

		switch metric.MType {
		case "counter":
			delta, _ := strconv.ParseInt(value, 10, 64)
			response.Delta = &delta
		case "gauge":
			floatVal, _ := strconv.ParseFloat(value, 64)
			response.Value = &floatVal
		}

		c.JSON(http.StatusOK, response)
	}
}
