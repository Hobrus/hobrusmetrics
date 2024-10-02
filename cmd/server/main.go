package main

import (
	"log"
	"net/http"

	"github.com/Hobrus/hobrusmetrics.git/internal/handlers"
	"github.com/Hobrus/hobrusmetrics.git/internal/service"
)

func main() {
	storage := service.NewMemStorage()

	metricsService := &service.MetricsService{Storage: storage}

	http.HandleFunc("/update/", handlers.UpdateHandler(metricsService))

	log.Println("Сервер запущен на :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Не удалось запустить сервер: %v\n", err)
	}
}
