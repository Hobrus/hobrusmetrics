package config

import (
	"encoding/json"
	"flag"
	"os"
	"strconv"
	"strings"
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
	// Включение HTTPS для запросов агента
	EnableHTTPS bool
	// Путь до файла с публичным ключом RSA
	CryptoKeyPath string
}

// agentJSONConfig описывает формат JSON-конфига для агента.
// Поля указаны как указатели для различения отсутствующих значений.
type agentJSONConfig struct {
	Address        *string `json:"address"`
	ReportInterval *string `json:"report_interval"`
	PollInterval   *string `json:"poll_interval"`
	CryptoKey      *string `json:"crypto_key"`
	// Дополнительные действующие опции приложения
	Key         *string `json:"key"`
	RateLimit   *int    `json:"rate_limit"`
	EnableHTTPS *bool   `json:"enable_https"`
}

func findConfigPathFromArgs() string {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-c" || arg == "-config" || arg == "--config" {
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		}
		if strings.HasPrefix(arg, "-c=") {
			return strings.TrimPrefix(arg, "-c=")
		}
		if strings.HasPrefix(arg, "-config=") || strings.HasPrefix(arg, "--config=") {
			if idx := strings.Index(arg, "="); idx != -1 {
				return arg[idx+1:]
			}
		}
	}
	return ""
}

func applyJSONToConfig(cfg *Config, jc agentJSONConfig) {
	if jc.Address != nil && *jc.Address != "" {
		cfg.ServerAddress = *jc.Address
	}
	if jc.ReportInterval != nil && *jc.ReportInterval != "" {
		if d, err := time.ParseDuration(*jc.ReportInterval); err == nil {
			cfg.ReportInterval = d
		}
	}
	if jc.PollInterval != nil && *jc.PollInterval != "" {
		if d, err := time.ParseDuration(*jc.PollInterval); err == nil {
			cfg.PollInterval = d
		}
	}
	if jc.CryptoKey != nil {
		cfg.CryptoKeyPath = *jc.CryptoKey
	}
	if jc.Key != nil {
		cfg.Key = *jc.Key
	}
	if jc.RateLimit != nil {
		cfg.RateLimit = *jc.RateLimit
	}
	if jc.EnableHTTPS != nil {
		cfg.EnableHTTPS = *jc.EnableHTTPS
	}
}

// NewConfig читает флаги и переменные окружения и возвращает конфигурацию агента.
func NewConfig() *Config {
	cfg := &Config{
		ServerAddress:  "localhost:8080",
		ReportInterval: 10 * time.Second,
		PollInterval:   2 * time.Second,
		Key:            "",
		RateLimit:      5, // значение по умолчанию (можно изменить)
		EnableHTTPS:    false,
		CryptoKeyPath:  "",
	}

	// 1) Ищем путь к JSON-конфигу в аргументах или окружении
	configPath := findConfigPathFromArgs()
	if configPath == "" {
		configPath = os.Getenv("CONFIG")
	}

	// 2) Применяем значения из JSON как дефолты
	if configPath != "" {
		if data, err := os.ReadFile(configPath); err == nil {
			var jc agentJSONConfig
			if err := json.Unmarshal(data, &jc); err == nil {
				applyJSONToConfig(cfg, jc)
			}
		}
	}

	// 3) Флаги (поверх дефолтов/JSON)
	var configPathFlag string
	flag.StringVar(&configPathFlag, "c", configPathFlag, "Path to JSON config file")
	flag.StringVar(&configPathFlag, "config", configPathFlag, "Path to JSON config file")

	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "HTTP server address")
	reportInterval := flag.Int("r", int(cfg.ReportInterval.Seconds()), "Report interval in seconds")
	pollInterval := flag.Int("p", int(cfg.PollInterval.Seconds()), "Poll interval in seconds")
	flag.StringVar(&cfg.Key, "k", cfg.Key, "Key for HMAC SHA256 signing")
	flag.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "Maximum number of concurrent outgoing requests (rate limit)")
	flag.BoolVar(&cfg.EnableHTTPS, "s", cfg.EnableHTTPS, "Use HTTPS to connect to server")
	flag.StringVar(&cfg.CryptoKeyPath, "crypto-key", cfg.CryptoKeyPath, "Path to RSA public key (PEM)")
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
	if envEnableHTTPS := os.Getenv("ENABLE_HTTPS"); envEnableHTTPS != "" {
		if v, err := strconv.ParseBool(envEnableHTTPS); err == nil {
			cfg.EnableHTTPS = v
		}
	}
	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		cfg.CryptoKeyPath = envCryptoKey
	}
	// Игнорируем позиционные аргументы: библиотечный код не должен завершать процесс.

	return cfg
}
