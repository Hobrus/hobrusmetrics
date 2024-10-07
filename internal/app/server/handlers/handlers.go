package handlers

import (
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/service"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	ms *service.MetricsService
}

func NewHandler(ms *service.MetricsService) *Handler {
	return &Handler{ms: ms}
}

func (h *Handler) SetupRoutes(router *gin.Engine) {
	router.POST("/update/:type/:name/:value", h.updateHandler)
	router.GET("/value/:type/:name", h.getValueHandler)
	router.GET("/", h.getAllMetricsHandler)
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

	tmpl, err := template.New("metrics").Parse(`
    <html>
        <body>
            <h1>Metrics</h1>
            <ul>
                {{range $name, $value := .}}
                    <li>{{$name}}: {{$value}}</li>
                {{end}}
            </ul>
        </body>
    </html>
    `)
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
