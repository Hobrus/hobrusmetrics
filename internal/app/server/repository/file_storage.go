package repository

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/server/models"
	"github.com/Hobrus/hobrusmetrics.git/internal/pkg/retry"
)

type FileBackedStorage struct {
	MemStorage
	filePath      string
	storeInterval time.Duration
	stopChan      chan struct{}
	storeMutex    sync.Mutex
	logger        *logrus.Logger
}

func NewFileBackedStorage(filePath string, storeInterval time.Duration, restore bool, logger *logrus.Logger) (*FileBackedStorage, error) {
	storage := &FileBackedStorage{
		MemStorage:    *NewMemStorage(), // базируемся на памяти
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

	if storeInterval > 0 {
		go storage.periodicSave()
	}

	return storage, nil
}

func (s *FileBackedStorage) LoadFromFile() error {
	var loadErr error
	err := retry.DoWithRetry(func() error {
		data, err := os.ReadFile(s.filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return err
			}
			return err
		}

		var metricsData models.MetricsData
		if err := json.Unmarshal(data, &metricsData); err != nil {
			return err
		}

		s.storeMutex.Lock()
		defer s.storeMutex.Unlock()

		// Восстанавливаем gauges (strings)
		for name, raw := range metricsData.Gauges {
			_ = s.MemStorage.UpdateGaugeRaw(name, raw) // можно логировать ошибку
		}
		// Восстанавливаем counters
		for name, cval := range metricsData.Counters {
			s.MemStorage.UpdateCounter(name, Counter(cval))
		}

		return nil
	})
	if err != nil {
		loadErr = err
	}
	return loadErr
}

func (s *FileBackedStorage) SaveToFile() error {
	s.storeMutex.Lock()
	defer s.storeMutex.Unlock()

	var saveErr error
	err := retry.DoWithRetry(func() error {
		gauges := s.MemStorage.GetAllGauges()     // map[string]string
		counters := s.MemStorage.GetAllCounters() // map[string]Counter

		// Преобразуем Counters в int64
		intCounters := make(map[string]int64, len(counters))
		for k, v := range counters {
			intCounters[k] = int64(v)
		}

		metricsData := models.MetricsData{
			Gauges:   gauges,
			Counters: intCounters,
		}

		data, err := json.Marshal(metricsData)
		if err != nil {
			return err
		}

		tempFile := s.filePath + ".tmp"
		if err := os.WriteFile(tempFile, data, 0644); err != nil {
			return err
		}
		if err := os.Rename(tempFile, s.filePath); err != nil {
			_ = os.Remove(tempFile)
			return err
		}
		return nil
	})
	if err != nil {
		saveErr = err
	}
	return saveErr
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

func (s *FileBackedStorage) UpdateGaugeRaw(name, rawValue string) error {
	if err := s.MemStorage.UpdateGaugeRaw(name, rawValue); err != nil {
		return err
	}
	if s.storeInterval == 0 {
		if err := s.SaveToFile(); err != nil {
			s.logger.Errorf("Failed to save metrics after gauge update: %v", err)
		}
	}
	return nil
}

func (s *FileBackedStorage) UpdateCounter(name string, value Counter) {
	s.MemStorage.UpdateCounter(name, value)
	if s.storeInterval == 0 {
		if err := s.SaveToFile(); err != nil {
			s.logger.Errorf("Failed to save metrics after counter update: %v", err)
		}
	}
}
