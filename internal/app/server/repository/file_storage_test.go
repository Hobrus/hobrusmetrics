package repository

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/models"
)

func TestFileBackedStorage_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "metrics.json")
	logger := logrus.New()

	// storeInterval = 0 to save on each update
	s, err := NewFileBackedStorage(file, 0, false, logger)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if err = s.UpdateGaugeRaw("G", "12.5"); err != nil {
		t.Fatalf("update gauge: %v", err)
	}
	s.UpdateCounter("C", 3)

	// Ensure file exists and content valid
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	var md models.MetricsData
	if err = json.Unmarshal(data, &md); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if md.Gauges["G"] != "12.5" {
		t.Fatalf("expected gauge saved")
	}
	if md.Counters["C"] != 3 {
		t.Fatalf("expected counter saved")
	}

	// Now restore into a new storage
	s2, err := NewFileBackedStorage(file, 0, true, logger)
	if err != nil {
		t.Fatalf("create2: %v", err)
	}
	if v, ok := s2.GetGaugeRaw("G"); !ok || v != "12.5" {
		t.Fatalf("restored gauge mismatch: %q ok=%v", v, ok)
	}
	if v, ok := s2.GetCounter("C"); !ok || v != 3 {
		t.Fatalf("restored counter mismatch: %d ok=%v", v, ok)
	}
}

func TestFileBackedStorage_ShutdownSaves(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "metrics.json")
	logger := logrus.New()

	s, err := NewFileBackedStorage(file, time.Millisecond*10, false, logger)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_ = s.UpdateGaugeRaw("G", "1")
	s.UpdateCounter("C", 1)

	// call shutdown to force save
	if err := s.Shutdown(); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	// verify file exists
	if _, err := os.Stat(file); err != nil {
		t.Fatalf("file not present after shutdown: %v", err)
	}
}
