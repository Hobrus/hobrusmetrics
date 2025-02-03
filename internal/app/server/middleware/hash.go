package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// computeHMAC вычисляет HMAC‑SHA256 от data с использованием key и возвращает шестнадцатеричную строку.
func computeHMAC(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// HashRequestMiddleware проверяет, что тело запроса подписано корректно.
// Если ключ равен "none", проверка пропускается.
func HashRequestMiddleware(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Бypass hash check если key == "none"
		if key == "none" {
			c.Next()
			return
		}
		// Пропускаем проверку для POST-запросов к /value/
		if c.Request.URL.Path == "/value/" {
			c.Next()
			return
		}

		// Проверяем для методов с телом запроса.
		if c.Request.Method == http.MethodPost ||
			c.Request.Method == http.MethodPut ||
			c.Request.Method == http.MethodPatch {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}
			// Восстанавливаем тело запроса для последующих обработчиков.
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			computedHash := computeHMAC(bodyBytes, key)
			receivedHash := c.GetHeader("HashSHA256")
			if receivedHash == "" || receivedHash != computedHash {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}
		}
		c.Next()
	}
}

// hashResponseWriter – кастомный ResponseWriter для буферизации ответа.
type hashResponseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *hashResponseWriter) Write(data []byte) (int, error) {
	return w.body.Write(data)
}

func (w *hashResponseWriter) WriteString(s string) (int, error) {
	return w.body.WriteString(s)
}

// HashResponseMiddleware вычисляет HMAC от сформированного ответа и добавляет его в заголовок "HashSHA256".
// Если key == "none", подпись не добавляется.
func HashResponseMiddleware(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origWriter := c.Writer
		writer := &hashResponseWriter{
			ResponseWriter: origWriter,
			body:           new(bytes.Buffer),
		}
		c.Writer = writer

		c.Next()

		responseData := writer.body.Bytes()
		if key != "none" && len(responseData) > 0 {
			hashValue := computeHMAC(responseData, key)
			origWriter.Header().Set("HashSHA256", hashValue)
		}
		if origWriter.Header().Get("Content-Type") == "" {
			origWriter.Header().Set("Content-Type", http.DetectContentType(responseData))
		}

		origWriter.WriteHeaderNow()
		if _, err := origWriter.Write(responseData); err != nil {
			log.Printf("failed to write response data: %v", err)
		}
	}
}
