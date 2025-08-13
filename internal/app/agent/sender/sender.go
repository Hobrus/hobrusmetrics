package sender

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Hobrus/hobrusmetrics.git/internal/pkg/retry"
)

// Metrics описывает метрику для передачи от агента на сервер.
type Metrics struct {
	ID    string   `json:"id"`   // metric name
	MType string   `json:"type"` // "counter" или "gauge"
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

// Sender отвечает за отправку метрик на сервер с учётом gzip и HMAC-подписи.
type Sender struct {
	ServerAddress string
	Client        *http.Client
	// Поле ключа для подписи.
	Key string
	// Использовать HTTPS
	UseHTTPS bool
	// RSA публичный ключ для шифрования
	RSAPublicKey *rsa.PublicKey
}

// NewSender создаёт новый экземпляр отправителя.
func NewSender(serverAddress, key string) *Sender {
	return &Sender{
		ServerAddress: serverAddress,
		Client:        &http.Client{},
		Key:           key,
	}
}

// EnableHTTPS включает переключение на https:// схему.
func (s *Sender) EnableHTTPS() {
	s.UseHTTPS = true
}

// LoadRSAPublicKey читает PEM-файл публичного ключа.
func (s *Sender) LoadRSAPublicKey(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read public key: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return errors.New("invalid PEM public key")
	}
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}
	pub, ok := pubAny.(*rsa.PublicKey)
	if !ok {
		return errors.New("not an RSA public key")
	}
	s.RSAPublicKey = pub
	return nil
}

// encryptHybrid выполняет гибридное шифрование: AES-GCM + RSA-OAEP над ключом.
// Возвращает JSON-структуру: {"ek":"base64(RSA(aesKey))","n":"base64(nonce)","ct":"base64(ciphertext)"}
func (s *Sender) encryptHybrid(plain []byte) ([]byte, error) {
	if s.RSAPublicKey == nil {
		return plain, nil
	}
	// generate AES-256 key
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nil, nonce, plain, nil)
	// encrypt AES key by RSA-OAEP
	label := []byte("hobrusmetrics")
	encKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, s.RSAPublicKey, aesKey, label)
	if err != nil {
		return nil, err
	}
	payload := map[string]string{
		"ek": base64.StdEncoding.EncodeToString(encKey),
		"n":  base64.StdEncoding.EncodeToString(nonce),
		"ct": base64.StdEncoding.EncodeToString(ciphertext),
	}
	return json.Marshal(payload)
}

// computeHMAC вычисляет HMAC‑SHA256 от data с использованием key и возвращает base64 строку.
func computeHMAC(data []byte, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func compressData(data []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write to gzip: %w", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip: %w", err)
	}
	return &buf, nil
}

// sendRequestWithRetry выполняет HTTP-запрос с повторными попытками.
// Теперь тело запроса считывается один раз и восстанавливается для каждой попытки.
func (s *Sender) sendRequestWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response

	// Считываем тело запроса для последующих попыток.
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	err = retry.DoWithRetry(func() error {
		// Восстанавливаем тело запроса для каждой попытки.
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		r, doErr := s.Client.Do(req)
		if doErr != nil {
			return doErr
		}
		if r.StatusCode >= 500 {
			r.Body.Close()
			return fmt.Errorf("server responded with %d", r.StatusCode)
		}
		resp = r
		return nil
	})

	return resp, err
}

// Send отправляет единичные метрики по одному в эндпоинт /update/.
func (s *Sender) Send(metrics map[string]interface{}) {
	for name, val := range metrics {
		var m Metrics
		switch v := val.(type) {
		case int64:
			d := v
			m = Metrics{ID: name, MType: "counter", Delta: &d}
		case float64:
			f := v
			m = Metrics{ID: name, MType: "gauge", Value: &f}
		default:
			continue
		}

		data, err := json.Marshal(m)
		if err != nil {
			log.Printf("marshal error: %v\n", err)
			continue
		}
		encrypted, err := s.encryptHybrid(data)
		if err != nil {
			log.Printf("encrypt error: %v\n", err)
			continue
		}
		compressed, err := compressData(encrypted)
		if err != nil {
			log.Printf("compress error: %v\n", err)
			continue
		}
		// Подписываем именно по байтам, идущим по сети (gzip-данные)
		var hashHeader string
		if s.Key != "" {
			hashHeader = computeHMAC(compressed.Bytes(), s.Key)
		}

		scheme := "http"
		if s.UseHTTPS {
			scheme = "https"
		}
		url := fmt.Sprintf("%s://%s/update/", scheme, s.ServerAddress)
		req, err := http.NewRequest(http.MethodPost, url, compressed)
		if err != nil {
			log.Printf("request error: %v\n", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")
		if hashHeader != "" {
			req.Header.Set("HashSHA256", hashHeader)
		}

		resp, err := s.sendRequestWithRetry(req)
		if err != nil {
			log.Printf("send error after retries: %v\n", err)
			continue
		}
		resp.Body.Close()
	}
}

// SendBatch отправляет набор метрик одним запросом в эндпоинт /updates/.
func (s *Sender) SendBatch(metrics map[string]interface{}) {
	if len(metrics) == 0 {
		return
	}

	batch := make([]Metrics, 0, len(metrics))
	for name, val := range metrics {
		switch v := val.(type) {
		case int64:
			d := v
			batch = append(batch, Metrics{ID: name, MType: "counter", Delta: &d})
		case float64:
			f := v
			batch = append(batch, Metrics{ID: name, MType: "gauge", Value: &f})
		default:
			continue
		}
	}
	if len(batch) == 0 {
		return
	}

	data, err := json.Marshal(batch)
	if err != nil {
		log.Printf("marshal batch error: %v\n", err)
		return
	}
	encrypted, err := s.encryptHybrid(data)
	if err != nil {
		log.Printf("encrypt batch error: %v\n", err)
		return
	}
	compressed, err := compressData(encrypted)
	if err != nil {
		log.Printf("compress batch error: %v\n", err)
		return
	}
	var hashHeader string
	if s.Key != "" {
		hashHeader = computeHMAC(compressed.Bytes(), s.Key)
	}

	scheme := "http"
	if s.UseHTTPS {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s/updates/", scheme, s.ServerAddress)
	req, err := http.NewRequest(http.MethodPost, url, compressed)
	if err != nil {
		log.Printf("batch request error: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	if hashHeader != "" {
		req.Header.Set("HashSHA256", hashHeader)
	}

	resp, err := s.sendRequestWithRetry(req)
	if err != nil {
		log.Printf("batch send error after retries: %v\n", err)
		return
	}
	resp.Body.Close()
}
