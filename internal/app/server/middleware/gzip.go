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

// gzipWriter – обёртка для gin.ResponseWriter, которая пишет сжатые данные.
type gzipWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func (g *gzipWriter) WriteHeader(code int) {
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipWriter) WriteHeaderNow() {
	g.ResponseWriter.WriteHeaderNow()
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	// Если Content-Type ещё не установлен, определяем его по данным
	if g.ResponseWriter.Header().Get("Content-Type") == "" {
		g.ResponseWriter.Header().Set("Content-Type", http.DetectContentType(data))
	}
	return g.writer.Write(data)
}

func (g *gzipWriter) WriteString(s string) (int, error) {
	if g.ResponseWriter.Header().Get("Content-Type") == "" {
		g.ResponseWriter.Header().Set("Content-Type", http.DetectContentType([]byte(s)))
	}
	return g.writer.Write([]byte(s))
}

// isGzipCompatible проверяет, подходит ли ответ для сжатия.
func isGzipCompatible(c *gin.Context) bool {
	// Проверяем, что клиент поддерживает gzip
	if !strings.Contains(strings.ToLower(c.Request.Header.Get("Accept-Encoding")), "gzip") {
		return false
	}
	// Пробуем получить Content-Type из заголовков ответа
	contentType := c.Writer.Header().Get("Content-Type")
	if contentType == "" {
		// Если нет – пытаемся определить по расширению URL
		if path := c.Request.URL.Path; path != "" {
			ext := filepath.Ext(path)
			if ext != "" {
				if mimeType := mime.TypeByExtension(ext); mimeType != "" {
					contentType = mimeType
				}
			}
		}
		// Если всё ещё пусто – берем Accept-заголовок
		if contentType == "" {
			contentType = c.Request.Header.Get("Accept")
		}
	}
	baseType := strings.Split(contentType, ";")[0]
	return compressibleMIMETypes[baseType]
}

// GzipMiddleware сжимает входящие ответы (а также распаковывает входящие запросы, если они зашифрованы gzip).
func GzipMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Если запрос пришёл с заголовком Content-Encoding: gzip – распаковываем тело
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

		// Если ответ можно сжать – оборачиваем ResponseWriter в gzipWriter
		if isGzipCompatible(c) {
			gz, err := gzip.NewWriterLevel(c.Writer, gzip.BestCompression)
			if err != nil {
				c.Next()
				return
			}

			gw := &gzipWriter{
				ResponseWriter: c.Writer,
				writer:         gz,
			}
			c.Writer = gw

			c.Header("Content-Encoding", "gzip")
			c.Header("Vary", "Accept-Encoding")

			defer func() {
				gz.Close()
			}()
		}

		c.Next()
	}
}
