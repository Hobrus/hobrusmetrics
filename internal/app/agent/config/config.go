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
	// Ключ для подписи
	Key string
	// Максимальное число одновременно выполняемых исходящих запросов
	RateLimit int
}

func NewConfig() *Config {
	cfg := &Config{
		ServerAddress:  "localhost:8080",
		ReportInterval: 10 * time.Second,
		PollInterval:   2 * time.Second,
		Key:            "",
		RateLimit:      5, // значение по умолчанию (можно изменить)
	}

	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "HTTP server address")
	reportInterval := flag.Int("r", int(cfg.ReportInterval.Seconds()), "Report interval in seconds")
	pollInterval := flag.Int("p", int(cfg.PollInterval.Seconds()), "Poll interval in seconds")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "Key for HMAC SHA256 signing")
	flag.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "Maximum number of concurrent outgoing requests (rate limit)")
	flag.Parse()

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
	if envKey := os.Getenv("KEY"); envKey != "" {
		cfg.Key = envKey
	}
	if envRateLimit := os.Getenv("RATE_LIMIT"); envRateLimit != "" {
		if rl, err := strconv.Atoi(envRateLimit); err == nil {
			cfg.RateLimit = rl
		}
	}
	if flag.NArg() > 0 {
		flag.Usage()
		os.Exit(1)
	}

	return cfg
}
