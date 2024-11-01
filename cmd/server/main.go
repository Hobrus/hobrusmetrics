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
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/repository"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/service"
)

func setupLogger() *logrus.Logger {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)
	return log
}

func setupServer(handler *gin.Engine, addr string) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: handler,
	}
}

func main() {
	logger := setupLogger()
	cfg := config.NewConfig()

	storage, err := repository.NewFileBackedStorage(cfg.FileStoragePath, cfg.StoreInterval, cfg.Restore, logger)
	if err != nil {
		logger.Fatal("Failed to initialize storage:", err)
	}

	app := gin.New()
	app.Use(gin.Recovery())
	handlers.NewHandler(&service.MetricsService{Storage: storage}).SetupRoutes(app)

	srv := setupServer(app, cfg.ServerAddress)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		logger.Info("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("Server shutdown error:", err)
		}

		if err := storage.Shutdown(); err != nil {
			logger.Error("Storage shutdown error:", err)
		}
	}()

	logger.Info("Server is running on ", cfg.ServerAddress)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("Server error:", err)
	}
}
