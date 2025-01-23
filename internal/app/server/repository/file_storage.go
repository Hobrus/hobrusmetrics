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
		MemStorage:    *NewMemStorage(),
		filePath:      filePath,
		storeInterval: storeInterval,
		stopChan:      make(chan struct{}),
		logger:        logger,
	}

	if restore {
		// Загрузка из файла при старте.
		if err := storage.LoadFromFile(); err != nil {
			storage.logger.Warnf("Failed to load metrics from file: %v", err)
		}
	}

	if storeInterval > 0 {
		go storage.periodicSave()
	}

	return storage, nil
}

// LoadFromFile теперь обёрнут в retry.DoWithRetry.
// Если, например, файл временно заблокирован, мы попробуем повторить до 4 раз.
func (s *FileBackedStorage) LoadFromFile() error {
	var loadErr error
	err := retry.DoWithRetry(func() error {
		data, err := os.ReadFile(s.filePath)
		if err != nil {
			// Если файл не найден, это не временная ошибка — прерываем сразу.
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

		// Update gauges
		for name, value := range metricsData.Gauges {
			s.MemStorage.UpdateGauge(name, Gauge(value))
		}

		// Update counters
		for name, value := range metricsData.Counters {
			s.MemStorage.UpdateCounter(name, Counter(value))
		}

		return nil
	})
	if err != nil {
		loadErr = err
	}
	return loadErr
}

// SaveToFile тоже обёрнут в retry. Например, если файл заблокирован.
func (s *FileBackedStorage) SaveToFile() error {
	s.storeMutex.Lock()
	defer s.storeMutex.Unlock()

	var saveErr error
	err := retry.DoWithRetry(func() error {
		// Преобразуем данные
		gauges := make(map[string]float64)
		for k, v := range s.MemStorage.GetAllGauges() {
			gauges[k] = float64(v)
		}
		counters := make(map[string]int64)
		for k, v := range s.MemStorage.GetAllCounters() {
			counters[k] = int64(v)
		}
		metricsData := models.MetricsData{
			Gauges:   gauges,
			Counters: counters,
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
			os.Remove(tempFile)
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

func (s *FileBackedStorage) UpdateGauge(name string, value Gauge) {
	s.MemStorage.UpdateGauge(name, value)
	if s.storeInterval == 0 {
		// Если storeInterval=0 — сохраняем на диск сразу (с ретраями).
		if err := s.SaveToFile(); err != nil {
			s.logger.Errorf("Failed to save metrics after gauge update: %v", err)
		}
	}
}

func (s *FileBackedStorage) UpdateCounter(name string, value Counter) {
	s.MemStorage.UpdateCounter(name, value)
	if s.storeInterval == 0 {
		// То же самое для counter.
		if err := s.SaveToFile(); err != nil {
			s.logger.Errorf("Failed to save metrics after counter update: %v", err)
		}
	}
}
