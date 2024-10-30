package middleware

import (
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

var compressibleMIMETypes = map[string]bool{
	"text/html":                true,
	"text/css":                 true,
	"text/plain":               true,
	"text/javascript":          true,
	"application/javascript":   true,
	"application/x-javascript": true,
	"application/json":         true,
	"application/xml":          true,
	"application/x-yaml":       true,
	"image/svg+xml":            true,
}

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

func shouldCompress(c *gin.Context) bool {
	// 1. Check if client accepts gzip encoding
	if !strings.Contains(strings.ToLower(c.Request.Header.Get("Accept-Encoding")), "gzip") {
		return false
	}

	// 2. Get content type
	contentType := c.Writer.Header().Get("Content-Type")
	if contentType == "" {
		// Try to detect content type from file extension if present
		if path := c.Request.URL.Path; path != "" {
			ext := filepath.Ext(path)
			if ext != "" {
				if mimeType := mime.TypeByExtension(ext); mimeType != "" {
					contentType = mimeType
				}
			}
		}

		// Fallback to Accept header
		if contentType == "" {
			contentType = c.Request.Header.Get("Accept")
		}
	}

	// Extract base MIME type without parameters
	baseType := strings.Split(contentType, ";")[0]

	// 3. Check if content type is compressible
	return compressibleMIMETypes[baseType]
}

func GzipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Handle incoming compressed data
		if strings.Contains(c.Request.Header.Get("Content-Encoding"), "gzip") {
			reader, err := gzip.NewReader(c.Request.Body)
			if err != nil {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}
			defer reader.Close()

			body, err := io.ReadAll(reader)
			if err != nil {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			c.Request.Body = io.NopCloser(strings.NewReader(string(body)))
			c.Request.Header.Del("Content-Encoding")
			c.Request.ContentLength = int64(len(body))
		}

		if shouldCompress(c) {
			gz, err := gzip.NewWriterLevel(c.Writer, gzip.BestCompression)
			if err != nil {
				c.Next()
				return
			}

			c.Writer = &gzipWriter{
				ResponseWriter: c.Writer,
				writer:         gz,
			}

			c.Header("Content-Encoding", "gzip")
			c.Header("Vary", "Accept-Encoding")

			defer func() {
				gz.Close()
			}()
		}

		c.Next()
	}
}
