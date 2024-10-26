package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/models"
)

func (h *Handler) updateJSONMetric(c *gin.Context) {
	var metric models.Metrics

	if err := c.ShouldBindJSON(&metric); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format"})
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

	// Return the updated value
	updatedValue, err := h.ms.GetMetricValue(metric.MType, metric.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get updated value"})
		return
	}

	result := models.Metrics{
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

	c.JSON(http.StatusOK, result)
}

func (h *Handler) getJSONMetric(c *gin.Context) {
	var metric models.Metrics

	if err := c.ShouldBindJSON(&metric); err != nil {
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

	result := models.Metrics{
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

	c.JSON(http.StatusOK, result)
}

func parseInt64(s string) int64 {
	val, _ := strconv.ParseInt(s, 10, 64)
	return val
}

func parseFloat64(s string) float64 {
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

func formatFloat64(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func format64(v int64) string {
	return strconv.FormatInt(v, 10)
}
