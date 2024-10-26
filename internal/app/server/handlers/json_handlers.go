package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Metrics struct {
	ID    string   `json:"id"`              // metric name
	MType string   `json:"type"`            // gauge or counter
	Delta *int64   `json:"delta,omitempty"` // counter value
	Value *float64 `json:"value,omitempty"` // gauge value
}

func parseInt64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func parseFloat64(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func format64(v int64) string {
	return strconv.FormatInt(v, 10)
}

func formatFloat64(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func (h *Handler) updateJSONMetric(c *gin.Context) {
	var metric Metrics

	// Read the body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// Parse JSON using encoding/json
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
		value = format64(*metric.Delta)
	case "gauge":
		if metric.Value == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "value is required for gauge"})
			return
		}
		value = formatFloat64(*metric.Value)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid metric type"})
		return
	}

	if err := h.ms.UpdateMetric(metric.MType, metric.ID, value); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get updated value
	updatedValue, err := h.ms.GetMetricValue(metric.MType, metric.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get updated value"})
		return
	}

	result := Metrics{
		ID:    metric.ID,
		MType: metric.MType,
	}

	switch metric.MType {
	case "counter":
		deltaVal := parseInt64(updatedValue)
		result.Delta = &deltaVal
	case "gauge":
		floatVal := parseFloat64(updatedValue)
		result.Value = &floatVal
	}

	// Use encoding/json for response
	response, err := json.Marshal(result)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode response"})
		return
	}

	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, string(response))
}

func (h *Handler) getJSONMetric(c *gin.Context) {
	var metric Metrics

	// Read the body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// Parse JSON using encoding/json
	if err := json.Unmarshal(body, &metric); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format"})
		return
	}

	if metric.ID == "" || metric.MType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id and type are required"})
		return
	}

	value, err := h.ms.GetMetricValue(metric.MType, metric.ID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "metric not found"})
		return
	}

	result := Metrics{
		ID:    metric.ID,
		MType: metric.MType,
	}

	switch metric.MType {
	case "counter":
		deltaVal := parseInt64(value)
		result.Delta = &deltaVal
	case "gauge":
		floatVal := parseFloat64(value)
		result.Value = &floatVal
	}

	// Use encoding/json for response
	response, err := json.Marshal(result)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode response"})
		return
	}

	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, string(response))
}
