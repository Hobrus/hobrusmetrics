package agent

import (
	"time"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/agent/collector"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/agent/config"
	"github.com/Hobrus/hobrusmetrics.git/internal/app/agent/sender"
)

type Agent struct {
	Config    *config.Config
	Metrics   *collector.Metrics
	Sender    *sender.Sender
	PollCount int64
}

func NewAgent() *Agent {
	cfg := config.NewConfig()
	metrics := collector.NewMetrics()
	localSender := sender.NewSender(cfg.ServerAddress)

	return &Agent{
		Config:  cfg,
		Metrics: metrics,
		Sender:  localSender,
	}
}

func (a *Agent) Run() {
	pollTicker := time.NewTicker(a.Config.PollInterval)
	reportTicker := time.NewTicker(a.Config.ReportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	for {
		select {
		case <-pollTicker.C:
			a.Metrics.Collect(&a.PollCount)

		case <-reportTicker.C:
			data := a.Metrics.GetAll()
			a.Sender.SendBatch(data)
		}
	}
}
