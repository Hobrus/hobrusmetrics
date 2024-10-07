package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerAddress  string
	ReportInterval time.Duration
	PollInterval   time.Duration
}

func NewConfig() *Config {
	cfg := &Config{
		ServerAddress:  "localhost:8080",
		ReportInterval: 10 * time.Second,
		PollInterval:   2 * time.Second,
	}

	// Парсинг флагов
	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "HTTP server address")
	reportInterval := flag.Int("r", int(cfg.ReportInterval.Seconds()), "Report interval in seconds")
	pollInterval := flag.Int("p", int(cfg.PollInterval.Seconds()), "Poll interval in seconds")
	flag.Parse()

	// Переопределение из переменных окружения
	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		cfg.ServerAddress = envAddress
	}
	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" {
		if ri, err := strconv.Atoi(envReportInterval); err == nil {
			cfg.ReportInterval = time.Duration(ri) * time.Second
		}
	} else {
		cfg.ReportInterval = time.Duration(*reportInterval) * time.Second
	}
	if envPollInterval := os.Getenv("POLL_INTERVAL"); envPollInterval != "" {
		if pi, err := strconv.Atoi(envPollInterval); err == nil {
			cfg.PollInterval = time.Duration(pi) * time.Second
		}
	} else {
		cfg.PollInterval = time.Duration(*pollInterval) * time.Second
	}

	if flag.NArg() > 0 {
		flag.Usage()
		os.Exit(1)
	}

	return cfg
}
