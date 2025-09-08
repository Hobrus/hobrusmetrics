package middleware

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// encryptedPayload соответствует формату, который отправляет агент после гибридного шифрования
type encryptedPayload struct {
	EK string `json:"ek"`
	N  string `json:"n"`
	CT string `json:"ct"`
}

// loadRSAPrivateKey читает приватный ключ RSA из PEM-файла
func loadRSAPrivateKey(path string) (*rsa.PrivateKey, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid PEM private key")
	}
	// support PKCS1 and PKCS8
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	priv, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA private key")
	}
	return priv, nil
}

// DecryptRequestMiddleware расшифровывает тело запроса, если на сервере задан приватный ключ.
// Ожидается, что тело до gzip представляет JSON encryptedPayload. Сначала middleware проверки подписи
// читают исходные байты и могут распаковать gzip. Здесь мы предполагаем, что к нам пришло распакованное тело.
func DecryptRequestMiddleware(privateKeyPath string) gin.HandlerFunc {
	if privateKeyPath == "" {
		return func(c *gin.Context) { c.Next() }
	}
	priv, err := loadRSAPrivateKey(privateKeyPath)
	if err != nil {
		// если ключ не загрузился, продолжаем без расшифровки
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPost && c.Request.Method != http.MethodPut && c.Request.Method != http.MethodPatch {
			c.Next()
			return
		}
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		// Пытаемся распарсить как зашифрованный полезный груз
		var ep encryptedPayload
		if err := json.Unmarshal(body, &ep); err != nil || ep.EK == "" || ep.N == "" || ep.CT == "" {
			// не наш формат – возвращаем тело назад как есть
			c.Request.Body = io.NopCloser(bytes.NewReader(body))
			c.Next()
			return
		}
		encKey, err := base64.StdEncoding.DecodeString(ep.EK)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		nonce, err := base64.StdEncoding.DecodeString(ep.N)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		ct, err := base64.StdEncoding.DecodeString(ep.CT)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		// Расшифровываем AES ключ через RSA-OAEP
		label := []byte("hobrusmetrics")
		aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, encKey, label)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		block, err := aes.NewCipher(aesKey)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		plain, err := gcm.Open(nil, nonce, ct, nil)
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(plain))
		c.Request.ContentLength = int64(len(plain))
		c.Next()
	}
}
