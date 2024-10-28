package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/config"
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

	// Load configuration
	cfg := config.NewConfig()

	// Setup storage
	storage, err := repository.NewFileBackedStorage(cfg.FileStoragePath, cfg.StoreInterval, cfg.Restore, logger)
	if err != nil {
		logger.Fatalf("Failed to initialize storage: %v", err)
	}

	// Setup services and handlers
	metricsService := &service.MetricsService{Storage: storage}
	handler := handlers.NewHandler(metricsService)

	// Create router with middleware
	router := gin.New()
	router.Use(middleware.GzipMiddleware())
	router.Use(gin.Recovery())
	router.Use(middleware.LoggingMiddleware(logger))

	// Setup routes
	handler.SetupRoutes(router)

	// Setup graceful shutdown
	srv := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: router,
	}

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutting down server...")

		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Shutdown HTTP server
		if err := srv.Shutdown(ctx); err != nil {
			logger.Errorf("Server shutdown error: %v", err)
		}

		// Save metrics before exit
		if err := storage.Shutdown(); err != nil {
			logger.Errorf("Failed to save metrics during shutdown: %v", err)
		}

		os.Exit(0)
	}()

	// Start server
	logger.Infof("Server is running on %s", cfg.ServerAddress)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Failed to start server: %v", err)
	}
}
