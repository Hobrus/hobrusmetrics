package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/agent"
	"github.com/Hobrus/hobrusmetrics.git/internal/pkg/buildinfo"
)

// Точка входа агента сбора метрик.
func main() {
	buildinfo.PrintSelf()
	myAgent := agent.NewAgent()
	log.Println("Agent is starting...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, shutdownSignals()...)
	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v. Shutting down agent...", sig)
		cancel()
	}()

	myAgent.Run(ctx)
	log.Println("Agent stopped gracefully")
}
