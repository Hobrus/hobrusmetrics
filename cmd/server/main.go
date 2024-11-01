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

func setupServer(logger *logrus.Logger, cfg *config.Config) (*http.Server, repository.Storage, error) {
	storage, err := repository.NewFileBackedStorage(cfg.FileStoragePath, cfg.StoreInterval, cfg.Restore, logger)
	if err != nil {
		return nil, nil, err
	}

	metricsService := &service.MetricsService{Storage: storage}
	handler := handlers.NewHandler(metricsService)

	router := gin.New()
	router.Use(middleware.GzipMiddleware(), gin.Recovery(), middleware.LoggingMiddleware(logger))
	handler.SetupRoutes(router)

	return &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: router,
	}, storage, nil
}

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)

	srv, storage, err := setupServer(logger, config.NewConfig())
	if err != nil {
		logger.Fatal(err)
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("Server shutdown error: %v", err)
	}

	if err := storage.Shutdown(); err != nil {
		logger.Errorf("Failed to save metrics during shutdown: %v", err)
	}

	logger.Info("Server stopped gracefully")
}
