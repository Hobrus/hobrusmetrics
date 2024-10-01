package handlers

import (
	"net/http"
	"strings"

	"github.com/Hobrus/hobrusmetrics.git/internal/service"
)

func UpdateHandler(ms *service.MetricsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if r.Header.Get("Content-Type") != "text/plain" {
			http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/")
		parts := strings.Split(path, "/")

		if len(parts) != 4 || parts[0] != "update" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		metricType := parts[1]
		metricName := parts[2]
		metricValue := parts[3]

		err := ms.UpdateMetric(metricType, metricName, metricValue)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
