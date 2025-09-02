package config

import "testing"

func TestNewConfig_Defaults(t *testing.T) {
	cfg := NewConfig()
	if cfg.ServerAddress == "" {
		t.Fatalf("ServerAddress must not be empty")
	}
	_ = cfg.EnableHTTPS
	_ = cfg.CryptoKeyPath
}
