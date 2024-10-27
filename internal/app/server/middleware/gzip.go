package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type gzipWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data)
}

func (g *gzipWriter) WriteString(s string) (int, error) {
	return g.writer.Write([]byte(s))
}

// Implement the Pusher interface
func (g *gzipWriter) Pusher() http.Pusher {
	if pusher, ok := g.ResponseWriter.(http.Pusher); ok {
		return pusher
	}
	return nil
}

func shouldGzip(c *gin.Context) bool {
	if !strings.Contains(strings.ToLower(c.Request.Header.Get("Accept-Encoding")), "gzip") {
		return false
	}

	contentType := c.Writer.Header().Get("Content-Type")
	switch {
	case strings.Contains(contentType, "application/json"):
		return true
	case strings.Contains(contentType, "text/html"):
		return true
	case contentType == "":
		// If content type not yet set, check path for typical JSON endpoints
		if strings.Contains(c.Request.URL.Path, "/update/") ||
			strings.Contains(c.Request.URL.Path, "/value/") {
			return true
		}
	}
	return false
}

// GzipMiddleware handles both compression and decompression
func GzipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Handle incoming compressed data
		if c.Request.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(c.Request.Body)
			if err != nil {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}
			defer gz.Close()

			body, err := io.ReadAll(gz)
			if err != nil {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			c.Request.Body = io.NopCloser(strings.NewReader(string(body)))
			c.Request.Header.Del("Content-Encoding")
			c.Request.ContentLength = int64(len(body))
		}

		if !shouldGzip(c) {
			c.Next()
			return
		}

		gz := gzip.NewWriter(c.Writer)
		defer gz.Close()

		c.Writer = &gzipWriter{
			ResponseWriter: c.Writer,
			writer:         gz,
		}
		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")

		c.Next()
	}
}
