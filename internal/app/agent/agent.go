package agent

import (
	"context"
	"sync"
	"time"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/agent/collector"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/agent/config"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/agent/sender"
)

// Agent инкапсулирует конфигурацию, сбор метрик и отправку данных.
// Экземпляр агента периодически собирает метрики и отправляет их на сервер.
type Agent struct {
	Config    *config.Config
	Metrics   *collector.Metrics
	Sender    *sender.Sender
	PollCount int64
}

// NewAgent создаёт и настраивает новый экземпляр агента.
func NewAgent() *Agent {
	cfg := config.NewConfig()
	metrics := collector.NewMetrics()
	// Передаём ключ в конструктор Sender
	localSender := sender.NewSender(cfg.ServerAddress, cfg.Key)
	if cfg.EnableHTTPS {
		localSender.EnableHTTPS()
	}
	if cfg.CryptoKeyPath != "" {
		_ = localSender.LoadRSAPublicKey(cfg.CryptoKeyPath)
	}

	return &Agent{
		Config:  cfg,
		Metrics: metrics,
		Sender:  localSender,
	}
}

// Run запускает фоновые задачи агента для сбора и отправки метрик
// и блокирует текущую горутину.
func (a *Agent) Run(ctx context.Context) {
	// Канал для отправки снимков метрик
	sendCh := make(chan map[string]interface{}, a.Config.RateLimit)

	var workersWG sync.WaitGroup
	var producersWG sync.WaitGroup

	// Запускаем worker pool для отправки запросов,
	// количество воркеров ограничено a.Config.RateLimit.
	for i := 0; i < a.Config.RateLimit; i++ {
		workersWG.Add(1)
		go func(workerID int) {
			defer workersWG.Done()
			for task := range sendCh {
				a.Sender.SendBatch(task)
			}
		}(i)
	}

	// Горутин для сбора runtime-метрик
	producersWG.Add(1)
	go func() {
		defer producersWG.Done()
		pollTicker := time.NewTicker(a.Config.PollInterval)
		defer pollTicker.Stop()
		for {
			select {
			case <-pollTicker.C:
				a.Metrics.Collect(&a.PollCount)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Горутин для сбора системных метрик (gopsutil)
	producersWG.Add(1)
	go func() {
		defer producersWG.Done()
		systemTicker := time.NewTicker(a.Config.PollInterval)
		defer systemTicker.Stop()
		for {
			select {
			case <-systemTicker.C:
				a.Metrics.CollectSystemMetrics()
			case <-ctx.Done():
				return
			}
		}
	}()

	// Горутин для формирования отчёта и помещения снимка метрик в очередь на отправку
	producersWG.Add(1)
	go func() {
		defer producersWG.Done()
		reportTicker := time.NewTicker(a.Config.ReportInterval)
		defer reportTicker.Stop()
		for {
			select {
			case <-reportTicker.C:
				snapshot := a.Metrics.GetAll()
				sendCh <- snapshot
			case <-ctx.Done():
				return
			}
		}
	}()

	// Ожидаем сигнал завершения через контекст
	<-ctx.Done()

	// Останавливаем продьюсеров (они завершатся по ctx.Done())
	producersWG.Wait()
	// Закрываем канал, чтобы воркеры дообработали очередь и завершились
	close(sendCh)
	// Дожидаемся завершения всех воркеров
	workersWG.Wait()
}
