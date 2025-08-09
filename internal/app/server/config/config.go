package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerAddress   string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool

	DatabaseDSN string
	// Новый параметр ключа для подписи:
	Key string
}

// NewConfig читает флаги и переменные окружения и возвращает конфигурацию сервера.
func NewConfig() *Config {
	cfg := &Config{
		ServerAddress:   "localhost:8080",
		StoreInterval:   300 * time.Second,
		FileStoragePath: "/tmp/metrics-db.json",
		Restore:         true,
		DatabaseDSN:     "",
		Key:             "",
	}

	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "HTTP server address")
	storeInterval := flag.Int("i", int(cfg.StoreInterval.Seconds()), "Store interval in seconds")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "File storage path")
	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "Restore metrics from file")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "Database DSN for PostgreSQL connection")
	// Добавляем флаг для ключа:
	flag.StringVar(&cfg.Key, "k", cfg.Key, "Key for HMAC SHA256 signing")
	flag.Parse()

	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		cfg.ServerAddress = envAddress
	}

	if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" {
		if si, err := strconv.Atoi(envStoreInterval); err == nil {
			cfg.StoreInterval = time.Duration(si) * time.Second
		}
	} else {
		cfg.StoreInterval = time.Duration(*storeInterval) * time.Second
	}

	if envFilePath := os.Getenv("FILE_STORAGE_PATH"); envFilePath != "" {
		cfg.FileStoragePath = envFilePath
	}

	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		cfg.Restore, _ = strconv.ParseBool(envRestore)
	}

	if envDatabaseDSN := os.Getenv("DATABASE_DSN"); envDatabaseDSN != "" {
		cfg.DatabaseDSN = envDatabaseDSN
	}

	// Читаем ключ из переменной окружения KEY (если задан)
	if envKey := os.Getenv("KEY"); envKey != "" {
		cfg.Key = envKey
	}

	return cfg
}
