package middleware

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// computeHMAC вычисляет HMAC‑SHA256 от data с использованием key и возвращает шестнадцатеричную строку.
func computeHMAC(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// HashRequestMiddleware проверяет подпись входящего запроса.
// Если в заголовке "Content-Encoding" указан gzip, тело сначала распаковывается,
// чтобы вычислить хеш от исходного (не сжатого) содержимого.
func HashRequestMiddleware(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Если ключ отсутствует или равен "none", пропускаем проверку
		if key == "" || key == "none" {
			c.Next()
			return
		}
		// Для эндпоинта получения значения пропускаем проверку
		if c.Request.URL.Path == "/value/" {
			c.Next()
			return
		}
		// Обрабатываем только запросы с телом (POST, PUT, PATCH)
		if c.Request.Method == http.MethodPost ||
			c.Request.Method == http.MethodPut ||
			c.Request.Method == http.MethodPatch {

			var bodyBytes []byte
			var err error

			// Если запрос зашифрован (gzip) – распаковываем тело
			if c.Request.Header.Get("Content-Encoding") == "gzip" {
				gr, err := gzip.NewReader(c.Request.Body)
				if err != nil {
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
				bodyBytes, err = io.ReadAll(gr)
				gr.Close()
				if err != nil {
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
				// Убираем заголовок, чтобы последующие middleware не пытались снова распаковывать тело
				c.Request.Header.Del("Content-Encoding")
				c.Request.ContentLength = int64(len(bodyBytes))
			} else {
				bodyBytes, err = io.ReadAll(c.Request.Body)
				if err != nil {
					c.AbortWithStatus(http.StatusBadRequest)
					return
				}
			}

			// Восстанавливаем тело запроса для последующих обработчиков
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			// Вычисляем хеш от распакованных данных
			computedHash := computeHMAC(bodyBytes, key)
			receivedHash := c.GetHeader("HashSHA256")
			// Если заголовок присутствует – проверяем корректность; если его нет – пропускаем проверку.
			if receivedHash != "" && receivedHash != computedHash {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}
		}
		c.Next()
	}
}

// hashResponseWriter – обёртка для ResponseWriter, которая буферизует ответ.
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
		// Генерируем подпись только если ключ задан и не равен "none"
		if key != "" && key != "none" && len(responseData) > 0 {
			hashValue := computeHMAC(responseData, key)
			origWriter.Header().Set("HashSHA256", hashValue)
		}
		if origWriter.Header().Get("Content-Type") == "" {
			origWriter.Header().Set("Content-Type", http.DetectContentType(responseData))
		}

		origWriter.WriteHeaderNow()
		_, _ = origWriter.Write(responseData)
	}
}
