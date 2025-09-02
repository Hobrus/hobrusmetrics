package config

import "testing"

func TestNewConfig_Defaults(t *testing.T) {
	cfg := NewConfig()
	if cfg.ServerAddress == "" {
		t.Fatalf("ServerAddress must not be empty")
	}
	if cfg.ReportInterval <= 0 || cfg.PollInterval <= 0 {
		t.Fatalf("intervals must be positive")
	}
	if cfg.RateLimit <= 0 {
		t.Fatalf("RateLimit must be positive")
	}
}
