package handlers

import (
	"embed"
	"html/template"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/middleware"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/service"
)

// Встроенные html-шаблоны страницы метрик.
//
//go:embed template/*.html
var templatesFS embed.FS

type Handler struct {
	ms *service.MetricsService
}

// Handler предоставляет HTTP-обработчики для работы с метриками.
// NewHandler создаёт хендлеры поверх сервиса метрик.
func NewHandler(ms *service.MetricsService) *Handler {
	return &Handler{ms: ms}
}

// SetupRoutes регистрирует HTTP-маршруты сервиса метрик.
func (h *Handler) SetupRoutes(router *gin.Engine) {
	router.POST("/update/:type/:name/:value", h.updateHandler)
	router.GET("/value/:type/:name", h.getValueHandler)
	router.GET("/", h.getAllMetricsHandler)

	router.POST("/update/", middleware.JSONUpdateMiddleware(h.ms))
	router.POST("/value/", middleware.JSONValueMiddleware(h.ms))
	router.POST("/updates/", h.updateBatchHandler)
}

// updateHandler обрабатывает обновление одной метрики через path-параметры.
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

// getValueHandler возвращает значение одной метрики по типу и имени.
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

// getAllMetricsHandler возвращает HTML-страницу со всеми метриками.
func (h *Handler) getAllMetricsHandler(c *gin.Context) {
	metrics := h.ms.GetAllMetrics()

	c.Header("Content-Type", "text/html")
	if err := getTemplate().Execute(c.Writer, metrics); err != nil {
		c.String(http.StatusInternalServerError, "Error rendering template")
	}
}

var (
	tmplOnce     sync.Once
	tmplCompiled *template.Template
)

// getTemplate компилирует и кэширует встроенный HTML-шаблон для страницы метрик.
func getTemplate() *template.Template {
	tmplOnce.Do(func() {
		tmplCompiled = template.Must(template.ParseFS(templatesFS, "template/metrics.html"))
	})
	return tmplCompiled
}

// updateBatchHandler принимает массив метрик и выполняет пакетное обновление.
func (h *Handler) updateBatchHandler(c *gin.Context) {
	var metricsBatch []middleware.MetricsJSON
	if err := c.ShouldBindJSON(&metricsBatch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format"})
		return
	}
	if len(metricsBatch) == 0 {
		c.Status(http.StatusOK)
		return
	}

	updated, err := h.ms.UpdateMetricsBatch(metricsBatch)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}
