package main

import (
	"log"

	"github.com/Hobrus/hobrusmetrics.git/internal/handlers"
	"github.com/Hobrus/hobrusmetrics.git/internal/service"
	"github.com/valyala/fasthttp"
)

func main() {
	storage := service.NewMemStorage()
	metricsService := &service.MetricsService{Storage: storage}
	handler := handlers.NewRouter(metricsService)

	log.Println("Server is running on :8080")
	if err := fasthttp.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("Error in ListenAndServe: %v", err)
	}
}
