package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

func main() {
	pollInterval := 2 * time.Second
	reportInterval := 10 * time.Second
	serverAddress := "http://localhost:8080"

	pollCount := int64(0)

	metricsTicker := time.NewTicker(pollInterval)
	reportTicker := time.NewTicker(reportInterval)
	defer metricsTicker.Stop()
	defer reportTicker.Stop()

	// Initialize metrics storage
	var m runtime.MemStats
	metrics := make(map[string]interface{})

	for {
		select {
		case <-metricsTicker.C:
			// Update runtime metrics
			runtime.ReadMemStats(&m)
			metrics["Alloc"] = float64(m.Alloc)
			metrics["BuckHashSys"] = float64(m.BuckHashSys)
			metrics["Frees"] = float64(m.Frees)
			metrics["GCCPUFraction"] = m.GCCPUFraction
			metrics["GCSys"] = float64(m.GCSys)
			metrics["HeapAlloc"] = float64(m.HeapAlloc)
			metrics["HeapIdle"] = float64(m.HeapIdle)
			metrics["HeapInuse"] = float64(m.HeapInuse)
			metrics["HeapObjects"] = float64(m.HeapObjects)
			metrics["HeapReleased"] = float64(m.HeapReleased)
			metrics["HeapSys"] = float64(m.HeapSys)
			metrics["LastGC"] = float64(m.LastGC)
			metrics["Lookups"] = float64(m.Lookups)
			metrics["MCacheInuse"] = float64(m.MCacheInuse)
			metrics["MCacheSys"] = float64(m.MCacheSys)
			metrics["MSpanInuse"] = float64(m.MSpanInuse)
			metrics["MSpanSys"] = float64(m.MSpanSys)
			metrics["Mallocs"] = float64(m.Mallocs)
			metrics["NextGC"] = float64(m.NextGC)
			metrics["NumForcedGC"] = float64(m.NumForcedGC)
			metrics["NumGC"] = float64(m.NumGC)
			metrics["OtherSys"] = float64(m.OtherSys)
			metrics["PauseTotalNs"] = float64(m.PauseTotalNs)
			metrics["StackInuse"] = float64(m.StackInuse)
			metrics["StackSys"] = float64(m.StackSys)
			metrics["Sys"] = float64(m.Sys)
			metrics["TotalAlloc"] = float64(m.TotalAlloc)

			// Update PollCount
			pollCount++
			metrics["PollCount"] = pollCount

			// Update RandomValue
			metrics["RandomValue"] = rand.Float64()
		case <-reportTicker.C:
			// Send metrics to server
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

				url := fmt.Sprintf("%s/update/%s/%s/%s", serverAddress, metricType, name, valueStr)
				req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte{}))
				if err != nil {
					log.Printf("Failed to create request: %v\n", err)
					continue
				}
				req.Header.Set("Content-Type", "text/plain")

				client := &http.Client{}
				resp, err := client.Do(req)
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
	}
}
