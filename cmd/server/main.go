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
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	cfg := config.NewConfig()

	dbConn, err := repository.NewDBConnection(cfg.DatabaseDSN)
	if err != nil {
		logger.Warnf("Failed to connect to database, fallback to file or memory: %v", err)
		dbConn = nil
	}

	// Выбираем хранилище:
	var storage repository.Storage

	switch {
	case dbConn != nil:
		logger.Infof("Using PostgreSQL storage at DSN=%s", cfg.DatabaseDSN)
		pStorage, err := repository.NewPostgresStorage(dbConn)
		if err != nil {
			logger.Warnf("Failed to create PostgresStorage, fallback to file or memory: %v", err)
			dbConn.Close()
			dbConn = nil
		} else {
			storage = pStorage
		}
	}

	if storage == nil {
		if cfg.FileStoragePath != "" {
			logger.Infof("Using file-backed storage at file=%s", cfg.FileStoragePath)
			fStorage, err := repository.NewFileBackedStorage(
				cfg.FileStoragePath,
				cfg.StoreInterval,
				cfg.Restore,
				logger,
			)
			if err != nil {
				logger.Warnf("Failed to initialize file storage, fallback to memory: %v", err)
				storage = repository.NewMemStorage()
			} else {
				storage = fStorage
			}
		} else {
			logger.Info("Using in-memory storage")
			storage = repository.NewMemStorage()
		}
	}

	metricsService := &service.MetricsService{Storage: storage}
	handler := handlers.NewHandler(metricsService)

	// Изменили порядок middleware: сначала Recovery, Logging и (при наличии ключа) хэширование,
	// затем GzipMiddleware – чтобы подпись вычислялась от распакованного тела.
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.LoggingMiddleware(logger))
	if cfg.Key != "" {
		router.Use(middleware.HashRequestMiddleware(cfg.Key))
		router.Use(middleware.HashResponseMiddleware(cfg.Key))
	}
	router.Use(middleware.GzipMiddleware())

	handler.SetupRoutes(router)

	router.GET("/ping", func(c *gin.Context) {
		if dbConn == nil {
			c.String(http.StatusInternalServerError, "database not configured")
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		if err := dbConn.Ping(ctx); err != nil {
			c.String(http.StatusInternalServerError, "database ping error: %v", err)
			return
		}
		c.String(http.StatusOK, "OK")
	})

	srv := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: router,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Errorf("Server shutdown error: %v", err)
		}

		if err := storage.Shutdown(); err != nil {
			logger.Errorf("Failed to save metrics during shutdown: %v", err)
		}

		if dbConn != nil {
			dbConn.Close()
		}
	}()

	logger.Infof("Server is running on %s", cfg.ServerAddress)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Failed to start server: %v", err)
	}
}
