package sender

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

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

	_, err := gz.Write(data)
	if err != nil {
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
		var valueStr string

		switch v := value.(type) {
		case int64:
			metricType = "counter"
			valueStr = strconv.FormatInt(v, 10)
		case float64:
			metricType = "gauge"
			valueStr = strconv.FormatFloat(v, 'f', -1, 64)
		default:
			log.Printf("Unsupported metric type for %s: %T\n", name, value)
			continue
		}

		url := fmt.Sprintf("http://%s/update/%s/%s/%s", s.ServerAddress, metricType, name, valueStr)

		// Create empty body for POST request
		body := &bytes.Buffer{}

		// Compress the body if it's not empty
		if body.Len() > 0 {
			compressedBody, err := compressData(body.Bytes())
			if err != nil {
				log.Printf("Failed to compress request body: %v\n", err)
				continue
			}
			body = compressedBody
		}

		req, err := http.NewRequest(http.MethodPost, url, body)
		if err != nil {
			log.Printf("Failed to create request: %v\n", err)
			continue
		}

		// Set appropriate headers
		req.Header.Set("Content-Type", "text/plain")
		if body.Len() > 0 {
			req.Header.Set("Content-Encoding", "gzip")
		}

		resp, err := s.Client.Do(req)
		if err != nil {
			log.Printf("Failed to send metric %s: %v\n", name, err)
			continue
		}
		err = resp.Body.Close()
		if err != nil {
			log.Printf("Failed to close response body: %v\n", err)
			continue
		}
	}
}
