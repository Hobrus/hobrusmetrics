package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// computeHMAC вычисляет HMAC‑SHA256 от data с использованием key.
func computeHMAC(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// HashRequestMiddleware проверяет, что тело запроса подписано корректно.
func HashRequestMiddleware(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Проверяем только для методов с телом (POST, PUT, PATCH)
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodPatch {
			bodyBytes, err := ioutil.ReadAll(c.Request.Body)
			if err != nil {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}
			// Восстанавливаем тело запроса для последующих обработчиков
			c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

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

// hashResponseWriter — кастомный ResponseWriter для буферизации ответа.
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

// HashResponseMiddleware вычисляет HMAC от сформированного ответа и добавляет его в заголовок.
func HashResponseMiddleware(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origWriter := c.Writer
		writer := &hashResponseWriter{ResponseWriter: origWriter, body: new(bytes.Buffer)}
		c.Writer = writer

		c.Next()

		responseData := writer.body.Bytes()
		if len(responseData) > 0 {
			hashValue := computeHMAC(responseData, key)
			origWriter.Header().Set("HashSHA256", hashValue)
		}
		origWriter.WriteHeaderNow()
		origWriter.Write(responseData)
	}
}
