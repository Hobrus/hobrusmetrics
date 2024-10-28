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
}

func NewConfig() *Config {
	cfg := &Config{
		ServerAddress:   "localhost:8080",
		StoreInterval:   300 * time.Second,
		FileStoragePath: "/tmp/metrics-db.json",
		Restore:         true,
	}

	// Parse flags
	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "HTTP server address")
	storeInterval := flag.Int("i", int(cfg.StoreInterval.Seconds()), "Store interval in seconds")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "File storage path")
	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "Restore metrics from file")
	flag.Parse()

	// Override with environment variables if present
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

	return cfg
}
