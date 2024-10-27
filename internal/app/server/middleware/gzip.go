package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	gzipEncoding = "gzip"
)

type gzipWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.writer.Write(data)
}

func shouldCompress(c *gin.Context) bool {
	// Check if client accepts gzip
	if !strings.Contains(c.Request.Header.Get("Accept-Encoding"), gzipEncoding) {
		return false
	}

	// Check content type
	contentType := c.Writer.Header().Get("Content-Type")
	return strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "text/html")
}

// GzipMiddleware handles both compression and decompression
func GzipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Handle compressed requests
		if c.Request.Header.Get("Content-Encoding") == gzipEncoding {
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

		// Prepare for compressed response if needed
		if shouldCompress(c) {
			gz := gzip.NewWriter(c.Writer)
			defer gz.Close()

			gzipWriter := &gzipWriter{
				ResponseWriter: c.Writer,
				writer:         gz,
			}

			c.Writer = gzipWriter
			c.Header("Content-Encoding", gzipEncoding)
			c.Header("Vary", "Accept-Encoding")
		}

		c.Next()
	}
}
