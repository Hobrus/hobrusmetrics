package sender

import (
	"bytes"
	"compress/gzip"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Hobrus/hobrusmetrics.git/internal/pkg/retry"
)

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"` // "counter" или "gauge"
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

type Sender struct {
	ServerAddress string
	Client        *http.Client
	// Добавляем поле ключа:
	Key string
}

func NewSender(serverAddress, key string) *Sender {
	return &Sender{
		ServerAddress: serverAddress,
		Client:        &http.Client{},
		Key:           key,
	}
}

// computeHMAC вычисляет HMAC‑SHA256 от data с использованием key и возвращает шестнадцатеричную строку.
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

func (s *Sender) sendRequestWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response

	err := retry.DoWithRetry(func() error {
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
		// Если ключ задан – вычисляем хеш от исходных (JSON) данных.
		var hashHeader string
		if s.Key != "" {
			hashHeader = computeHMAC(data, s.Key)
		}

		compressed, err := compressData(data)
		if err != nil {
			log.Printf("compress error: %v\n", err)
			continue
		}

		url := fmt.Sprintf("http://%s/update/", s.ServerAddress)
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
	var hashHeader string
	if s.Key != "" {
		hashHeader = computeHMAC(data, s.Key)
	}

	compressed, err := compressData(data)
	if err != nil {
		log.Printf("compress batch error: %v\n", err)
		return
	}

	url := fmt.Sprintf("http://%s/updates/", s.ServerAddress)
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
