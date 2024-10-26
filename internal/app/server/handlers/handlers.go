package handlers

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/service"

	"github.com/gin-gonic/gin"
)

//go:embed template/*.html
var templatesFS embed.FS

type Handler struct {
	ms *service.MetricsService
}

func NewHandler(ms *service.MetricsService) *Handler {
	return &Handler{ms: ms}
}

func (h *Handler) SetupRoutes(router *gin.Engine) {
	// Existing routes for backward compatibility
	router.POST("/update/:type/:name/:value", h.updateHandler)
	router.GET("/value/:type/:name", h.getValueHandler)
	router.GET("/", h.getAllMetricsHandler)

	// New JSON API routes
	router.POST("/update/", h.updateJSONMetric)
	router.POST("/value/", h.getJSONMetric)
}

func (h *Handler) updateHandler(c *gin.Context) {
	metricType := c.Param("type")
	metricName := c.Param("name")
	metricValue := c.Param("value")

	err := h.ms.UpdateMetric(metricType, metricName, metricValue)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) getValueHandler(c *gin.Context) {
	metricType := c.Param("type")
	metricName := c.Param("name")

	value, err := h.ms.GetMetricValue(metricType, metricName)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.String(http.StatusOK, value)
}

func (h *Handler) getAllMetricsHandler(c *gin.Context) {
	metrics := h.ms.GetAllMetrics()

	// Parse the embedded template
	tmpl, err := template.ParseFS(templatesFS, "template/metrics.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Error rendering template")
		return
	}

	c.Header("Content-Type", "text/html")
	err = tmpl.Execute(c.Writer, metrics)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error rendering template")
		return
	}
}
