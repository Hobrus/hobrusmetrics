package main

import (
	"flag"
	"github.com/Hobrus/hobrusmetrics.git/internal/handlers"
	"github.com/Hobrus/hobrusmetrics.git/internal/repository"
	"github.com/Hobrus/hobrusmetrics.git/internal/service"
	"github.com/gin-gonic/gin"
	"log"
	"os"
)

func main() {
	serverAddress := flag.String("a", "localhost:8080", "HTTP server address")
	flag.Parse()

	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		*serverAddress = envAddress
	}

	if flag.NArg() > 0 {
		log.Fatalf("Unknown argument: %s", flag.Arg(0))
	}

	var storage repository.Storage = service.NewMemStorage()
	metricsService := &service.MetricsService{Storage: storage}
	handler := handlers.NewHandler(metricsService)

	router := gin.Default()

	handler.SetupRoutes(router)

	log.Printf("Server is running on %s\n", *serverAddress)
	if err := router.Run(*serverAddress); err != nil {
		log.Fatalf("Failed to start server: %v\n", err)
	}
}
