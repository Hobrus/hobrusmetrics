package sender

import (
	"bytes"
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
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte{}))
		if err != nil {
			log.Printf("Failed to create request: %v\n", err)
			continue
		}
		req.Header.Set("Content-Type", "text/plain")

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
