package main

import (
	"log"

	"github.com/Hobrus/hobrusmetrics.git/internal/handlers"
	"github.com/Hobrus/hobrusmetrics.git/internal/service"
	"github.com/gin-gonic/gin"
)

func main() {
	storage := service.NewMemStorage()
	metricsService := &service.MetricsService{Storage: storage}

	router := gin.Default()

	handlers.SetupRoutes(router, metricsService)

	log.Println("Сервер запущен на :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Не удалось запустить сервер: %v\n", err)
	}
}
