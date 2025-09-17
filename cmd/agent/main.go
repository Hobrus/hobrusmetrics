package main

import (
	"context"
	"log"
	"os/signal"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/agent"
	"github.com/Hobrus/hobrusmetrics.git/internal/pkg/buildinfo"
)

// Точка входа агента сбора метрик.
func main() {
	buildinfo.PrintSelf()
	myAgent := agent.NewAgent()
	log.Println("Agent is starting...")

	ctx, stop := signal.NotifyContext(context.Background(), shutdownSignals()...)
	defer stop()

	myAgent.Run(ctx)
	log.Println("Agent stopped gracefully")
}
