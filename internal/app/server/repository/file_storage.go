package repository

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type FileBackedStorage struct {
	MemStorage
	filePath      string
	storeInterval time.Duration
	stopChan      chan struct{}
	storeMutex    sync.Mutex
	logger        *logrus.Logger
}

type MetricsData struct {
	Gauges   map[string]Gauge   `json:"gauges"`
	Counters map[string]Counter `json:"counters"`
}

func NewFileBackedStorage(filePath string, storeInterval time.Duration, restore bool, logger *logrus.Logger) (*FileBackedStorage, error) {
	storage := &FileBackedStorage{
		MemStorage:    *NewMemStorage(),
		filePath:      filePath,
		storeInterval: storeInterval,
		stopChan:      make(chan struct{}),
		logger:        logger,
	}

	if restore {
		if err := storage.LoadFromFile(); err != nil {
			storage.logger.Warnf("Failed to load metrics from file: %v", err)
		}
	}

	// Start periodic saving if interval > 0
	if storeInterval > 0 {
		go storage.periodicSave()
	}

	return storage, nil
}

func (s *FileBackedStorage) LoadFromFile() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, not an error
		}
		return err
	}

	var metricsData MetricsData
	if err := json.Unmarshal(data, &metricsData); err != nil {
		return err
	}

	s.storeMutex.Lock()
	defer s.storeMutex.Unlock()

	// Update gauges
	for name, value := range metricsData.Gauges {
		s.MemStorage.UpdateGauge(name, value)
	}

	// Update counters
	for name, value := range metricsData.Counters {
		s.MemStorage.UpdateCounter(name, value)
	}

	return nil
}

func (s *FileBackedStorage) SaveToFile() error {
	s.storeMutex.Lock()
	defer s.storeMutex.Unlock()

	metricsData := MetricsData{
		Gauges:   s.MemStorage.GetAllGauges(),
		Counters: s.MemStorage.GetAllCounters(),
	}

	data, err := json.Marshal(metricsData)
	if err != nil {
		return err
	}

	// Create temporary file
	tempFile := s.filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return err
	}

	// Atomically rename temporary file to target file
	if err := os.Rename(tempFile, s.filePath); err != nil {
		os.Remove(tempFile) // Clean up temp file if rename fails
		return err
	}

	return nil
}

func (s *FileBackedStorage) periodicSave() {
	ticker := time.NewTicker(s.storeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.SaveToFile(); err != nil {
				s.logger.Errorf("Failed to save metrics to file: %v", err)
			}
		case <-s.stopChan:
			return
		}
	}
}

func (s *FileBackedStorage) Shutdown() error {
	close(s.stopChan)
	return s.SaveToFile()
}

// Override base methods to implement synchronous saving when storeInterval is 0
func (s *FileBackedStorage) UpdateGauge(name string, value Gauge) {
	s.MemStorage.UpdateGauge(name, value)
	if s.storeInterval == 0 {
		if err := s.SaveToFile(); err != nil {
			s.logger.Errorf("Failed to save metrics after gauge update: %v", err)
		}
	}
}

func (s *FileBackedStorage) UpdateCounter(name string, value Counter) {
	s.MemStorage.UpdateCounter(name, value)
	if s.storeInterval == 0 {
		if err := s.SaveToFile(); err != nil {
			s.logger.Errorf("Failed to save metrics after counter update: %v", err)
		}
	}
}
