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
	// Check if client accepts gzip encoding
	if !strings.Contains(strings.ToLower(c.Request.Header.Get("Accept-Encoding")), "gzip") {
		return false
	}

	// Check content type after the next handlers have run
	contentType := c.Writer.Header().Get("Content-Type")

	// List of content types that should be compressed
	compressibleTypes := []string{
		"application/json",
		"text/html",
		"text/plain",
		"text/xml",
		"text/css",
		"text/javascript",
		"application/javascript",
		"application/x-javascript",
	}

	// Check if content type matches any compressible type
	for _, t := range compressibleTypes {
		if strings.Contains(contentType, t) {
			return true
		}
	}

	// If content type is not set yet but path suggests JSON
	if contentType == "" && (strings.Contains(c.Request.URL.Path, "/update/") ||
		strings.Contains(c.Request.URL.Path, "/value/")) {
		return true
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

		// Process the request through other handlers first
		c.Next()

		// Check if response should be gzipped
		if !shouldGzip(c) {
			return
		}

		gz := gzip.NewWriter(c.Writer)
		defer gz.Close()

		gzipWriter := &gzipWriter{
			ResponseWriter: c.Writer,
			writer:         gz,
		}

		c.Writer = gzipWriter
		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")
	}
}
