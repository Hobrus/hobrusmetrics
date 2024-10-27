package sender

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
}

type Sender struct {
	ServerAddress string
	Client        *http.Client
}

func NewSender(serverAddress string) *Sender {
	return &Sender{
		ServerAddress: serverAddress,
		Client:        &http.Client{},
	}
}

func compressData(data []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	if _, err := gz.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write to gzip writer: %w", err)
	}

	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return &buf, nil
}

func (s *Sender) Send(metrics map[string]interface{}) {
	for name, value := range metrics {
		var metricType string
		var metric Metrics

		switch v := value.(type) {
		case int64:
			metricType = "counter"
			delta := v
			metric = Metrics{
				ID:    name,
				MType: metricType,
				Delta: &delta,
			}
		case float64:
			metricType = "gauge"
			val := v
			metric = Metrics{
				ID:    name,
				MType: metricType,
				Value: &val,
			}
		default:
			log.Printf("Unsupported metric type for %s: %T\n", name, value)
			continue
		}

		// Сериализуем метрику в JSON
		jsonData, err := json.Marshal(metric)
		if err != nil {
			log.Printf("Failed to marshal metric %s: %v\n", name, err)
			continue
		}

		// Сжимаем JSON данные
		compressedData, err := compressData(jsonData)
		if err != nil {
			log.Printf("Failed to compress metric data %s: %v\n", name, err)
			continue
		}

		// Создаем запрос с сжатыми данными
		url := fmt.Sprintf("http://%s/update/", s.ServerAddress)
		req, err := http.NewRequest(http.MethodPost, url, compressedData)
		if err != nil {
			log.Printf("Failed to create request: %v\n", err)
			continue
		}

		// Устанавливаем заголовки
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")

		// Отправляем запрос
		resp, err := s.Client.Do(req)
		if err != nil {
			log.Printf("Failed to send metric %s: %v\n", name, err)
			continue
		}

		resp.Body.Close()
	}
}
