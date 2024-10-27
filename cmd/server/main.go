package main

import (
	"flag"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/handlers"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/middleware"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/repository"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/service"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	serverAddress := flag.String("a", "localhost:8080", "HTTP server address")
	flag.Parse()

	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		*serverAddress = envAddress
	}

	if flag.NArg() > 0 {
		logger.Fatalf("Unknown argument: %s", flag.Arg(0))
	}

	// Setup storage and services
	var storage repository.Storage = repository.NewMemStorage()
	metricsService := &service.MetricsService{Storage: storage}
	handler := handlers.NewHandler(metricsService)

	// Create router with middleware
	router := gin.New()

	// Add middlewares in the correct order
	router.Use(middleware.GzipMiddleware())          // Gzip middleware first
	router.Use(gin.Recovery())                       // Then recovery
	router.Use(middleware.LoggingMiddleware(logger)) // Then logging

	// Setup routes
	handler.SetupRoutes(router)

	// Start server
	logger.Infof("Server is running on %s", *serverAddress)
	if err := router.Run(*serverAddress); err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}
}
