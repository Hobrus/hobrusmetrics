package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/config"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/handlers"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/middleware"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/repository"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/service"
	"github.com/Hobrus/hobrusmetrics.git/internal/pkg/buildinfo"

	_ "net/http/pprof"
)

// Приложение HTTP-сервера метрик. Точка входа.
func main() {
	buildinfo.PrintSelf()
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

	// Запускаем pprof-сервер на localhost:6060
	go func() {
		if err := http.ListenAndServe("localhost:6060", nil); err != nil && err != http.ErrServerClosed {
			logger.Warnf("pprof server error: %v", err)
		}
	}()

	// Порядок: Recovery, Logging, проверка подписи (если есть ключ), расшифровка (если есть приватный ключ), затем gzip.
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.LoggingMiddleware(logger))
	if cfg.Key != "" {
		router.Use(middleware.HashRequestMiddleware(cfg.Key))
		router.Use(middleware.HashResponseMiddleware(cfg.Key))
	}
	// Расшифровка после проверки подписи и до gzip-распаковки уже не требуется,
	// так как HashRequestMiddleware сам распаковывает тело для последующих обработчиков.
	router.Use(middleware.DecryptRequestMiddleware(cfg.CryptoKeyPath))
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

	// Канал ошибок сервера
	serverErr := make(chan error, 1)

	// Запускаем сервер в отдельной горутине
	go func() {
		logger.Infof("Server is running on %s", cfg.ServerAddress)
		if cfg.EnableHTTPS {
			certFile := os.Getenv("TLS_CERT_FILE")
			keyFile := os.Getenv("TLS_KEY_FILE")
			if certFile == "" || keyFile == "" {
				if _, err := os.Stat("server.crt"); err == nil {
					certFile = "server.crt"
				}
				if _, err := os.Stat("server.key"); err == nil {
					keyFile = "server.key"
				}
			}
			if certFile == "" || keyFile == "" {
				logger.Warn("ENABLE_HTTPS is set but TLS_CERT_FILE/TLS_KEY_FILE not provided; falling back to http")
				serverErr <- srv.ListenAndServe()
				return
			}
			logger.Infof("Starting HTTPS with cert=%s key=%s", filepath.Base(certFile), filepath.Base(keyFile))
			serverErr <- srv.ListenAndServeTLS(certFile, keyFile)
			return
		}
		serverErr <- srv.ListenAndServe()
	}()

	// Ожидаем сигнал завершения или ошибку сервера
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, shutdownSignals()...)

	select {
	case sig := <-sigChan:
		logger.Infof("Shutting down server (signal: %v)...", sig)
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
		if err := <-serverErr; err != nil && err != http.ErrServerClosed {
			logger.Errorf("Server closed with error: %v", err)
		}
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server error: %v", err)
		}
	}
}
