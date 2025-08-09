package main

import (
	"log"

	"github.com/Hobrus/hobrusmetrics.git/internal/app/agent"
)

// Точка входа агента сбора метрик.
func main() {
	myAgent := agent.NewAgent()
	log.Println("Agent is starting...")
	myAgent.Run()
}
