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
	ServerAddress   string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool

	DatabaseDSN string
	// Новый параметр ключа для подписи:
	Key string
	// Включение HTTPS через -s или ENABLE_HTTPS
	EnableHTTPS bool
	// Путь к приватному ключу RSA для расшифровки входящих сообщений (CRYPTO_KEY или -crypto-key)
	CryptoKeyPath string
}

// serverJSONConfig описывает формат JSON-конфига для сервера.
// Все поля указаны как указатели, чтобы отличать отсутствующие значения от нулевых.
type serverJSONConfig struct {
	Address       *string `json:"address"`
	Restore       *bool   `json:"restore"`
	StoreInterval *string `json:"store_interval"`
	StoreFile     *string `json:"store_file"`
	DatabaseDSN   *string `json:"database_dsn"`
	CryptoKey     *string `json:"crypto_key"`
	// Дополнительные действующие опции приложения
	Key         *string `json:"key"`
	EnableHTTPS *bool   `json:"enable_https"`
}

// findConfigPathFromArgs ищет путь к JSON-файлу конфигурации в аргументах командной строки (-c, -config)
// или возвращает пустую строку, если аргумент не найден. Поддерживает формы -c path и -c=path.
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

// applyJSONToConfig применяет считанные из JSON значения к конфигурации cfg.
func applyJSONToConfig(cfg *Config, jc serverJSONConfig) {
	if jc.Address != nil && *jc.Address != "" {
		cfg.ServerAddress = *jc.Address
	}
	if jc.Restore != nil {
		cfg.Restore = *jc.Restore
	}
	if jc.StoreFile != nil && *jc.StoreFile != "" {
		cfg.FileStoragePath = *jc.StoreFile
	}
	if jc.DatabaseDSN != nil {
		cfg.DatabaseDSN = *jc.DatabaseDSN
	}
	if jc.CryptoKey != nil {
		cfg.CryptoKeyPath = *jc.CryptoKey
	}
	if jc.Key != nil {
		cfg.Key = *jc.Key
	}
	if jc.EnableHTTPS != nil {
		cfg.EnableHTTPS = *jc.EnableHTTPS
	}
	if jc.StoreInterval != nil && *jc.StoreInterval != "" {
		if d, err := time.ParseDuration(*jc.StoreInterval); err == nil {
			cfg.StoreInterval = d
		}
	}
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
		EnableHTTPS:     false,
		CryptoKeyPath:   "",
	}

	// 1) Предварительно ищем путь к JSON-конфигу в аргументах или окружении
	configPath := findConfigPathFromArgs()
	if configPath == "" {
		configPath = os.Getenv("CONFIG")
	}

	// 2) Если найден путь, читаем JSON и применяем значения как новые дефолты
	if configPath != "" {
		if data, err := os.ReadFile(configPath); err == nil {
			var jc serverJSONConfig
			if err := json.Unmarshal(data, &jc); err == nil {
				applyJSONToConfig(cfg, jc)
			}
		}
	}

	// 3) Определяем флаги с учётом дефолтов (которые уже могут быть из JSON)
	var configPathFlag string
	flag.StringVar(&configPathFlag, "c", configPathFlag, "Path to JSON config file")
	flag.StringVar(&configPathFlag, "config", configPathFlag, "Path to JSON config file")

	flag.StringVar(&cfg.ServerAddress, "a", cfg.ServerAddress, "HTTP server address")
	storeInterval := flag.Int("i", int(cfg.StoreInterval.Seconds()), "Store interval in seconds")
	flag.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "File storage path")
	flag.BoolVar(&cfg.Restore, "r", cfg.Restore, "Restore metrics from file")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "Database DSN for PostgreSQL connection")
	// Добавляем флаг для ключа:
	flag.StringVar(&cfg.Key, "k", cfg.Key, "Key for HMAC SHA256 signing")
	// Флаг включения HTTPS
	flag.BoolVar(&cfg.EnableHTTPS, "s", cfg.EnableHTTPS, "Enable HTTPS (ListenAndServeTLS)")
	// Флаг приватного ключа для асимметричного шифрования
	flag.StringVar(&cfg.CryptoKeyPath, "crypto-key", cfg.CryptoKeyPath, "Path to RSA private key (PEM)")
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

	if envEnableHTTPS := os.Getenv("ENABLE_HTTPS"); envEnableHTTPS != "" {
		if v, err := strconv.ParseBool(envEnableHTTPS); err == nil {
			cfg.EnableHTTPS = v
		}
	}

	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		cfg.CryptoKeyPath = envCryptoKey
	}

	return cfg
}
