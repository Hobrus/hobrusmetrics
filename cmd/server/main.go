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

	var storage repository.Storage = repository.NewMemStorage()
	metricsService := &service.MetricsService{Storage: storage}
	handler := handlers.NewHandler(metricsService)

	// Create router with middleware
	router := gin.New()

	// Important: Add gzip middleware before others
	router.Use(middleware.GzipMiddleware())
	router.Use(gin.Recovery())
	router.Use(middleware.LoggingMiddleware(logger))

	handler.SetupRoutes(router)

	logger.Infof("Server is running on %s", *serverAddress)
	if err := router.Run(*serverAddress); err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}
}
