package handlers

import (
	"html/template"
	"net/http"

	"github.com/Hobrus/hobrusmetrics.git/internal/service"
	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine, ms *service.MetricsService) {
	router.POST("/update/:type/:name/:value", updateHandler(ms))
	router.GET("/value/:type/:name", getValueHandler(ms))
	router.GET("/", getAllMetricsHandler(ms))
}

func updateHandler(ms *service.MetricsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		metricType := c.Param("type")
		metricName := c.Param("name")
		metricValue := c.Param("value")

		err := ms.UpdateMetric(metricType, metricName, metricValue)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}

		c.Status(http.StatusOK)
	}
}

func getValueHandler(ms *service.MetricsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		metricType := c.Param("type")
		metricName := c.Param("name")

		value, err := ms.GetMetricValue(metricType, metricName)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		c.String(http.StatusOK, value)
	}
}

func getAllMetricsHandler(ms *service.MetricsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		metrics := ms.GetAllMetrics()

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
}
