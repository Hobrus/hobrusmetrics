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
			metric = Metrics{ID: name, MType: metricType, Delta: &delta}
		case float64:
			metricType = "gauge"
			val := v
			metric = Metrics{ID: name, MType: metricType, Value: &val}
		default:
			log.Printf("Unsupported metric type for %s: %T\n", name, value)
			continue
		}

		jsonData, err := json.Marshal(metric)
		if err != nil {
			log.Printf("Failed to marshal metric %s: %v\n", name, err)
			continue
		}

		compressed, err := compressData(jsonData)
		if err != nil {
			log.Printf("Failed to compress metric %s: %v\n", name, err)
			continue
		}

		url := fmt.Sprintf("http://%s/update/", s.ServerAddress)
		req, err := http.NewRequest(http.MethodPost, url, compressed)
		if err != nil {
			log.Printf("Failed to create request: %v\n", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")

		resp, err := s.Client.Do(req)
		if err != nil {
			log.Printf("Failed to send metric %s: %v\n", name, err)
			continue
		}
		resp.Body.Close()
	}
}

func (s *Sender) SendBatch(metrics map[string]interface{}) {
	if len(metrics) == 0 {
		return
	}

	var batch []Metrics
	for name, val := range metrics {
		switch v := val.(type) {
		case int64:
			delta := v
			batch = append(batch, Metrics{ID: name, MType: "counter", Delta: &delta})
		case float64:
			valCopy := v
			batch = append(batch, Metrics{ID: name, MType: "gauge", Value: &valCopy})
		default:
			log.Printf("Unsupported metric type for %s: %T\n", name, val)
		}
	}
	if len(batch) == 0 {
		return
	}

	data, err := json.Marshal(batch)
	if err != nil {
		log.Printf("Failed to marshal batch: %v\n", err)
		return
	}

	compressed, err := compressData(data)
	if err != nil {
		log.Printf("Failed to compress batch: %v\n", err)
		return
	}

	url := fmt.Sprintf("http://%s/updates/", s.ServerAddress)
	req, err := http.NewRequest(http.MethodPost, url, compressed)
	if err != nil {
		log.Printf("Failed to create batch request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := s.Client.Do(req)
	if err != nil {
		log.Printf("Failed to send batch: %v\n", err)
		return
	}
	resp.Body.Close()
}
