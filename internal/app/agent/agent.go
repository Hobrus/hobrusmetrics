package agent

import (
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
func (a *Agent) Run() {
	// Канал для отправки снимков метрик
	sendCh := make(chan map[string]interface{}, a.Config.RateLimit)

	// Запускаем worker pool для отправки запросов,
	// количество воркеров ограничено a.Config.RateLimit.
	for i := 0; i < a.Config.RateLimit; i++ {
		go func(workerID int) {
			for task := range sendCh {
				a.Sender.SendBatch(task)
			}
		}(i)
	}

	// Горутин для сбора runtime-метрик
	go func() {
		pollTicker := time.NewTicker(a.Config.PollInterval)
		defer pollTicker.Stop()
		for {
			<-pollTicker.C
			a.Metrics.Collect(&a.PollCount)
		}
	}()

	// Горутин для сбора системных метрик (gopsutil)
	go func() {
		systemTicker := time.NewTicker(a.Config.PollInterval)
		defer systemTicker.Stop()
		for {
			<-systemTicker.C
			a.Metrics.CollectSystemMetrics()
		}
	}()

	// Горутин для формирования отчёта и помещения снимка метрик в очередь на отправку
	go func() {
		reportTicker := time.NewTicker(a.Config.ReportInterval)
		defer reportTicker.Stop()
		for {
			<-reportTicker.C
			snapshot := a.Metrics.GetAll()
			sendCh <- snapshot
		}
	}()

	// Блокируем основную горутину (или можно обрабатывать сигналы завершения)
	select {}
}
