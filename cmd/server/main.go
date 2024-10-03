package main

import (
	"log"

	"github.com/Hobrus/hobrusmetrics.git/internal/handlers"
	"github.com/Hobrus/hobrusmetrics.git/internal/repositories"
	"github.com/Hobrus/hobrusmetrics.git/internal/service"
	"github.com/gin-gonic/gin"
)

func main() {
	var storage repositories.Storage = service.NewMemStorage()
	metricsService := &service.MetricsService{Storage: storage}
	handler := handlers.NewHandler(metricsService)

	router := gin.Default()

	handler.SetupRoutes(router)

	log.Println("Сервер запущен на :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Не удалось запустить сервер: %v\n", err)
	}
}
