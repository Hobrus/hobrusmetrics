package main

import (
	"log"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/agent"
	"github.com/Hobrus/hobrusmetrics.git/internal/pkg/buildinfo"
)

// Точка входа агента сбора метрик.
func main() {
	buildinfo.PrintSelf()
	myAgent := agent.NewAgent()
	log.Println("Agent is starting...")
	myAgent.Run()
}
