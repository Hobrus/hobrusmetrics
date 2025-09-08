package middleware

import (
	"bytes"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// responseWriter — обёртка для записи ответа, позволяющая измерять размер тела.
// Используется для логирования характеристик ответа (статус, размер, длительность).
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// LoggingMiddleware возвращает middleware для логирования запросов и ответов.
func LoggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Create a custom response writer to capture the response size
		buf := &bytes.Buffer{}
		writer := responseWriter{
			ResponseWriter: c.Writer,
			body:           buf,
		}
		c.Writer = writer

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Get response size
		responseSize := c.Writer.Size()

		// Log request and response details
		logger.WithFields(logrus.Fields{
			"method":       c.Request.Method,
			"uri":          c.Request.RequestURI,
			"status":       c.Writer.Status(),
			"duration":     duration,
			"responseSize": responseSize,
		}).Info("HTTP request handled")
	}
}
